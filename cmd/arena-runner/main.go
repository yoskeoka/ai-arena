package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/gamemaster"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
	"github.com/yoskeoka/ai-arena/internal/platform/registry"
	"github.com/yoskeoka/ai-arena/internal/platform/replay"
	"github.com/yoskeoka/ai-arena/internal/platform/runtime"
	"github.com/yoskeoka/ai-arena/internal/platform/session"
)

const defaultStderrLimitBytes = 4096
const defaultOutputDir = "arena-runner-output"

type playerSpec struct {
	PlayerID string
	Entry    string
}

type logRecord struct {
	MatchID  string          `json:"match_id"`
	Seq      int             `json:"seq"`
	Kind     string          `json:"kind"`
	Turn     int             `json:"turn"`
	PlayerID string          `json:"player_id,omitempty"`
	Payload  json.RawMessage `json:"payload,omitempty"`
}

type streamObserver struct {
	matchID string
	enc     *json.Encoder
	nextSeq int
	err     error // first encode/write error
}

type artifactLayout struct {
	BaseDir              string
	MatchDir             string
	RecordPath           string
	StructuredLogPath    string
	SnapshotPath         string
	ExportedSnapshotPath string
	HistoryPath          string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	var (
		gameName         string
		gameVersion      string
		ruleset          string
		matchID          string
		outputDir        string
		logOutput        string
		persistRecord    string
		exportedOutput   string
		recordInput      string
		snapshotInput    string
		historyInput     string
		targetTurn       int
		matchTimeout     time.Duration
		stderrLimitBytes int
		playerArgs       multiFlag
	)

	fs := flag.NewFlagSet("arena-runner", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&gameName, "game", "", "game id")
	fs.StringVar(&gameVersion, "game-version", "", "game version")
	fs.StringVar(&ruleset, "ruleset", "", "game ruleset")
	fs.StringVar(&matchID, "match-id", "", "match id")
	fs.StringVar(&outputDir, "output-dir", defaultOutputDir, "base directory for standard runner artifacts")
	fs.StringVar(&logOutput, "log-output", "stdout", "structured log output target path or stdout")
	fs.StringVar(&persistRecord, "persist-record", "", "additional source-of-truth final match-record output target path or stdout")
	fs.StringVar(&exportedOutput, "exported-snapshot-output", "", "additional exported snapshot output target path or stdout")
	fs.StringVar(&recordInput, "record-input", "", "source-of-truth final match-record input path")
	fs.StringVar(&snapshotInput, "snapshot-input", "", "snapshot input path for debug resume (hand-crafted or extracted from a record)")
	fs.StringVar(&historyInput, "history-input", "", "history input path extracted from a record event_log")
	fs.IntVar(&targetTurn, "target-turn", 0, "replay/resume boundary turn used with --history-input or --record-input")
	fs.DurationVar(&matchTimeout, "match-timeout", 0, "cancel the match after the given duration")
	fs.IntVar(&stderrLimitBytes, "stderr-limit-bytes", defaultStderrLimitBytes, "captured stderr bytes per player")
	fs.Var(&playerArgs, "player", "player_id=entry-path")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(playerArgs) == 0 {
		return fmt.Errorf("at least one --player is required")
	}
	if strings.TrimSpace(outputDir) == "" {
		return fmt.Errorf("--output-dir must not be empty")
	}
	if matchID == "" {
		matchID = "match-" + uuid.NewString()
	}
	layout := newArtifactLayout(outputDir, matchID)
	if err := ensureArtifactLayout(layout); err != nil {
		return err
	}

	playersForGame, err := parsePlayersForGame(playerArgs)
	if err != nil {
		return err
	}

	debugInputCount := 0
	for _, value := range []string{recordInput, snapshotInput, historyInput} {
		if value != "" {
			debugInputCount++
		}
	}
	if snapshotInput != "" && historyInput != "" {
		return fmt.Errorf("--snapshot-input and --history-input cannot be combined")
	}
	if snapshotInput != "" && recordInput != "" {
		return fmt.Errorf("--snapshot-input and --record-input cannot be combined")
	}
	if historyInput != "" && recordInput != "" {
		return fmt.Errorf("--history-input and --record-input cannot be combined")
	}
	if debugInputCount > 1 {
		return fmt.Errorf("at most one debug input source can be selected")
	}
	if historyInput != "" && targetTurn <= 0 {
		return fmt.Errorf("--target-turn is required with --history-input")
	}
	if targetTurn < 0 {
		return fmt.Errorf("--target-turn must be non-negative")
	}

	var (
		recordSource   *match.Record
		resumeSnapshot *game.Snapshot
	)
	if recordInput != "" {
		record, err := replay.LoadRecord(recordInput)
		if err != nil {
			return err
		}
		recordSource = &record
		if gameName == "" {
			gameName = record.Game.GameID
		}
		if gameVersion == "" {
			gameVersion = record.Game.GameVersion
		}
		if ruleset == "" {
			ruleset = record.Game.RulesetVersion
		}
		if snapshotInput == "" && historyInput == "" && targetTurn == 0 {
			snapshot := record.Snapshot
			resumeSnapshot = &snapshot
		}
	}
	if snapshotInput != "" {
		snapshot, err := replay.LoadSnapshot(snapshotInput)
		if err != nil {
			return err
		}
		resumeSnapshot = &snapshot
		if gameName == "" {
			gameName = snapshot.GameID
		}
		if gameVersion == "" {
			gameVersion = snapshot.GameVersion
		}
		if ruleset == "" {
			ruleset = snapshot.RulesetVersion
		}
	}
	if gameName == "" {
		return fmt.Errorf("--game is required")
	}
	if gameVersion == "" {
		return fmt.Errorf("--game-version is required")
	}
	if ruleset == "" {
		return fmt.Errorf("--ruleset is required")
	}
	descriptor, err := registry.Default().LookupVersion(context.Background(), gameName, gameVersion)
	if err != nil {
		return err
	}

	var metaOverride *catalog.GameMetadata
	if historyInput != "" {
		history, err := replay.LoadHistory(historyInput)
		if err != nil {
			return err
		}
		buildSpec := registry.BuildSpec{
			GameVersion: gameVersion,
			Ruleset:     ruleset,
			Players:     append([]game.Player(nil), playersForGame...),
		}
		master, err := descriptor.BuildSession(buildSpec)
		if err != nil {
			return err
		}
		metaOverride = ptr(master.Metadata())
		_ = master.Shutdown(context.Background())
		snapshot, err := descriptor.SnapshotFromHistory(buildSpec, history, targetTurn)
		if err != nil {
			return err
		}
		resumeSnapshot = &snapshot
	} else if recordSource != nil && targetTurn > 0 {
		snapshot, err := replay.SnapshotFromHistory(recordSource.Game, recordSource.Players, recordSource.EventLog, targetTurn)
		if err != nil {
			return err
		}
		resumeSnapshot = &snapshot
		metaOverride = &recordSource.Game
	}

	master, err := newGameMasterSession(descriptor, registry.BuildSpec{
		GameVersion: gameVersion,
		Ruleset:     ruleset,
		Players:     append([]game.Player(nil), playersForGame...),
	}, resumeSnapshot)
	if err != nil {
		return err
	}
	masterOwnedByRunner := false
	defer func() {
		if !masterOwnedByRunner {
			_ = master.Shutdown(context.Background())
		}
	}()
	if meta := master.Metadata(); meta.GameVersion != gameVersion {
		return fmt.Errorf("selected game version %q does not match implementation version %q", gameVersion, meta.GameVersion)
	}
	if metaOverride != nil {
		if err := catalog.Compatible(*metaOverride, master.Metadata()); err != nil {
			return fmt.Errorf("resume metadata incompatible: %w", err)
		}
	}
	if exportedOutput != "" && resumeSnapshot != nil {
		exported, err := master.CurrentExportedSnapshot(context.Background())
		if err != nil {
			return err
		}
		exported.MatchID = matchID
		exported.Status = game.StatusRunning
		if err := writeJSONToTarget(exportedOutput, exported, os.Stdout, "exported snapshot"); err != nil {
			return err
		}
	}

	players, sessions, err := loadPlayersAndSessions(master.Metadata(), playerArgs, stderrLimitBytes)
	if err != nil {
		return err
	}

	ctx := context.Background()
	if matchTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, matchTimeout)
		defer cancel()
	}

	logWriter, closeLog, err := openLogOutputs(layout.StructuredLogPath, logOutput, os.Stdout)
	if err != nil {
		closeSessions(sessions)
		return err
	}
	defer closeLog()

	observer := &streamObserver{
		matchID: matchID,
		enc:     json.NewEncoder(logWriter),
	}

	opts := []match.RunnerOption{match.WithObserver(observer)}
	if resumeSnapshot != nil {
		opts = append(opts, match.WithResumeState(*resumeSnapshot))
	}
	masterOwnedByRunner = true

	record, runErr := match.NewRunnerWithOptions(
		matchID,
		players,
		master,
		sessions,
		opts...,
	).Run(ctx)
	if observer.err != nil {
		return fmt.Errorf("stream log: %w", observer.err)
	}
	if exportedOutput != "" && resumeSnapshot == nil {
		if err := writeJSONToTarget(exportedOutput, record.ExportedSnapshot, os.Stdout, "exported snapshot"); err != nil {
			return err
		}
	}
	if err := writeStandardArtifacts(layout, record); err != nil {
		return err
	}
	if err := persistRecordToTarget(persistRecord, record, os.Stdout); err != nil {
		return err
	}
	if runErr != nil {
		fmt.Fprintln(os.Stderr, runErr)
	}
	return nil
}

func (o *streamObserver) OnEvent(event match.Event) {
	if o.err != nil {
		return
	}
	o.nextSeq = event.Seq
	o.err = o.enc.Encode(logRecord{
		MatchID:  o.matchID,
		Seq:      event.Seq,
		Kind:     event.Kind,
		Turn:     event.Turn,
		PlayerID: event.PlayerID,
		Payload:  event.Payload,
	})
}

func (o *streamObserver) OnRecordBuilt(record match.Record) {
	o.emitTerminalRecord("terminal_snapshot", record.Snapshot.Turn, mustMarshal(record.Snapshot))
	o.emitTerminalRecord("terminal_exported_snapshot", record.ExportedSnapshot.Turn, mustMarshal(record.ExportedSnapshot))
	o.emitTerminalRecord("terminal_summary", record.Snapshot.Turn, mustMarshal(terminalSummary(record)))
}

func (o *streamObserver) emitTerminalRecord(kind string, turn int, payload json.RawMessage) {
	if o.err != nil {
		return
	}
	o.nextSeq++
	o.err = o.enc.Encode(logRecord{
		MatchID: o.matchID,
		Seq:     o.nextSeq,
		Kind:    kind,
		Turn:    turn,
		Payload: payload,
	})
}

func terminalSummary(record match.Record) map[string]any {
	summary := map[string]any{
		"status": record.Status,
		"result": record.Result,
	}
	if errMsg := terminalError(record); errMsg != "" {
		summary["error"] = errMsg
	}
	return summary
}

func terminalError(record match.Record) string {
	for i := len(record.EventLog) - 1; i >= 0; i-- {
		event := record.EventLog[i]
		if event.Kind != "match_failed" && event.Kind != "match_canceled" {
			continue
		}
		var payload struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err == nil {
			return payload.Error
		}
	}
	return ""
}

func persistRecordToTarget(target string, record match.Record, stdout io.Writer) error {
	if target == "" {
		return nil
	}
	return writeJSONToTarget(target, record, stdout, "persist target")
}

func newArtifactLayout(outputDir, matchID string) artifactLayout {
	matchDir := filepath.Join(outputDir, matchID)
	return artifactLayout{
		BaseDir:              outputDir,
		MatchDir:             matchDir,
		RecordPath:           filepath.Join(matchDir, "record.json"),
		StructuredLogPath:    filepath.Join(matchDir, "structured-log.ndjson"),
		SnapshotPath:         filepath.Join(matchDir, "snapshot.json"),
		ExportedSnapshotPath: filepath.Join(matchDir, "exported-snapshot.json"),
		HistoryPath:          filepath.Join(matchDir, "history.json"),
	}
}

func ensureArtifactLayout(layout artifactLayout) error {
	if err := os.MkdirAll(layout.MatchDir, 0o750); err != nil {
		return fmt.Errorf("create artifact directory %s: %w", layout.MatchDir, err)
	}
	return nil
}

func openLogOutputs(standardPath, extraTarget string, stdout io.Writer) (io.Writer, func() error, error) {
	writers := []io.Writer{stdout}
	closers := make([]io.Closer, 0, 2)

	standardFile, err := createFileOutput(standardPath)
	if err != nil {
		return nil, nil, fmt.Errorf("create standard log output %s: %w", standardPath, err)
	}
	writers = append(writers, standardFile)
	closers = append(closers, standardFile)

	if extraTarget != "" && extraTarget != "stdout" && !sameCleanPath(extraTarget, standardPath) {
		extraFile, err := createFileOutput(extraTarget)
		if err != nil {
			_ = closeAll(closers)
			return nil, nil, fmt.Errorf("create extra log output %s: %w", extraTarget, err)
		}
		writers = append(writers, extraFile)
		closers = append(closers, extraFile)
	}

	return io.MultiWriter(writers...), func() error { return closeAll(closers) }, nil
}

func writeStandardArtifacts(layout artifactLayout, record match.Record) error {
	if err := writeJSONFile(layout.RecordPath, record, "record"); err != nil {
		return err
	}
	if err := writeJSONFile(layout.SnapshotPath, record.Snapshot, "snapshot"); err != nil {
		return err
	}
	if err := writeJSONFile(layout.ExportedSnapshotPath, record.ExportedSnapshot, "exported snapshot"); err != nil {
		return err
	}
	if err := writeJSONFile(layout.HistoryPath, record.EventLog, "history"); err != nil {
		return err
	}
	return nil
}

func mustMarshal(v any) json.RawMessage {
	raw, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`{"marshal_error":"failed to encode log payload"}`)
	}
	return raw
}

func newGameMasterSession(descriptor registry.GameDescriptor, spec registry.BuildSpec, snapshot *game.Snapshot) (gamemaster.Session, error) {
	if snapshot != nil {
		return descriptor.BuildSessionFromSnapshot(spec, *snapshot)
	}
	return descriptor.BuildSession(spec)
}

func parsePlayersForGame(args []string) ([]game.Player, error) {
	players := make([]game.Player, 0, len(args))
	seenPlayerIDs := make(map[string]struct{}, len(args))
	for _, arg := range args {
		spec, err := parsePlayerSpec(arg)
		if err != nil {
			return nil, err
		}
		if _, exists := seenPlayerIDs[spec.PlayerID]; exists {
			return nil, fmt.Errorf("duplicate player_id %q", spec.PlayerID)
		}
		seenPlayerIDs[spec.PlayerID] = struct{}{}
		players = append(players, game.Player{
			PlayerID: spec.PlayerID,
			AIID:     spec.PlayerID,
		})
	}
	return players, nil
}

func loadPlayersAndSessions(meta catalog.GameMetadata, args []string, stderrLimitBytes int) ([]game.Player, map[string]match.PlayerSession, error) {
	players := make([]game.Player, 0, len(args))
	sessions := make(map[string]match.PlayerSession, len(args))
	seenPlayerIDs := make(map[string]struct{}, len(args))
	for _, arg := range args {
		spec, err := parsePlayerSpec(arg)
		if err != nil {
			return nil, nil, err
		}
		if _, exists := seenPlayerIDs[spec.PlayerID]; exists {
			closeSessions(sessions)
			return nil, nil, fmt.Errorf("duplicate player_id %q", spec.PlayerID)
		}
		seenPlayerIDs[spec.PlayerID] = struct{}{}
		loaded, err := loadEntry(meta, spec)
		if err != nil {
			closeSessions(sessions)
			return nil, nil, err
		}
		players = append(players, game.Player{
			PlayerID: spec.PlayerID,
			AIID:     loaded.AIID,
		})
		cfg := loaded.Runtime
		cfg.Dir = repoRoot()
		cfg.StderrLimitBytes = stderrLimitBytes
		adapter, err := runtime.Start(context.Background(), cfg)
		if err != nil {
			closeSessions(sessions)
			return nil, nil, err
		}
		sessions[spec.PlayerID] = session.New(adapter)
	}
	return players, sessions, nil
}

type loadedEntry struct {
	Runtime runtime.Config
	AIID    string
}

func loadEntry(matchMeta catalog.GameMetadata, spec playerSpec) (loadedEntry, error) {
	sidecarPath := spec.Entry + ".arena.json"
	if _, err := os.Stat(sidecarPath); err == nil {
		var manifest catalog.SidecarManifest
		if err := readJSON(sidecarPath, &manifest); err != nil {
			return loadedEntry{}, err
		}
		aiMeta := catalog.GameMetadata{
			GameID:         manifest.Protocol.GameID,
			GameVersion:    manifest.Protocol.GameVersion,
			RulesetVersion: manifest.Protocol.RulesetVersion,
		}
		if err := catalog.Compatible(matchMeta, aiMeta); err != nil {
			return loadedEntry{}, fmt.Errorf("%s metadata incompatible: %w", spec.PlayerID, err)
		}
		if manifest.Protocol.Transport != "" && manifest.Protocol.Transport != "stdio-jsonrpc-ndjson" {
			return loadedEntry{}, fmt.Errorf("%s transport %q is unsupported", spec.PlayerID, manifest.Protocol.Transport)
		}
		runtimeCfg, err := catalog.ResolveRuntime(spec.Entry, manifest.Runtime)
		if err != nil {
			return loadedEntry{}, fmt.Errorf("%s runtime invalid: %w", spec.PlayerID, err)
		}
		return loadedEntry{Runtime: runtimeCfg, AIID: manifest.AIID}, nil
	} else if err != nil && !os.IsNotExist(err) {
		return loadedEntry{}, err
	}

	if err := catalog.ValidateMetadata(matchMeta); err != nil {
		return loadedEntry{}, err
	}
	return loadedEntry{Runtime: catalog.FallbackRuntime(spec.Entry), AIID: filepath.Base(spec.Entry)}, nil
}

func parsePlayerSpec(raw string) (playerSpec, error) {
	playerID, entry, ok := strings.Cut(raw, "=")
	if !ok || playerID == "" || entry == "" {
		return playerSpec{}, fmt.Errorf("invalid --player %q", raw)
	}
	return playerSpec{PlayerID: playerID, Entry: entry}, nil
}

func readJSON(path string, dest any) error {
	root, err := os.OpenRoot(repoRoot())
	if err != nil {
		return err
	}
	defer root.Close()

	data, err := root.ReadFile(strings.TrimPrefix(filepath.Clean(path), "./"))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func openOutputTarget(target string, stdout io.Writer) (io.Writer, func() error, error) {
	switch target {
	case "", "stdout":
		return stdout, func() error { return nil }, nil
	default:
		file, err := createFileOutput(target)
		if err != nil {
			return nil, nil, fmt.Errorf("create output target %s: %w", target, err)
		}
		return file, file.Close, nil
	}
}

func createFileOutput(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, err
	}
	// #nosec G304 -- the caller explicitly selects the local output target path.
	return os.Create(path)
}

func writeJSONToTarget(target string, value any, stdout io.Writer, label string) error {
	writer, closeWriter, err := openOutputTarget(target, stdout)
	if err != nil {
		return err
	}
	defer closeWriter()
	if err := json.NewEncoder(writer).Encode(value); err != nil {
		return fmt.Errorf("write %s %s: %w", label, target, err)
	}
	return nil
}

func writeJSONFile(path string, value any, label string) error {
	file, err := createFileOutput(path)
	if err != nil {
		return fmt.Errorf("create %s %s: %w", label, path, err)
	}
	defer file.Close()
	if err := json.NewEncoder(file).Encode(value); err != nil {
		return fmt.Errorf("write %s %s: %w", label, path, err)
	}
	return nil
}

func closeAll(closers []io.Closer) error {
	var firstErr error
	for _, closer := range closers {
		if err := closer.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func sameCleanPath(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}

func ptr[T any](value T) *T {
	return &value
}

func repoRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func closeSessions(sessions map[string]match.PlayerSession) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	for _, sess := range sessions {
		_ = sess.Close(ctx)
	}
}

type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}
