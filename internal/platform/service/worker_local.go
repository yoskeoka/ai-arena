package service

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/yoskeoka/ai-arena/internal/platform/artifacts"
	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
	"github.com/yoskeoka/ai-arena/internal/platform/registry"
	"github.com/yoskeoka/ai-arena/internal/platform/runtime"
	"github.com/yoskeoka/ai-arena/internal/platform/session"
)

const defaultWorkerStderrLimitBytes = 4096

// LocalRunnerInvoker resolves local artifact refs and executes one runner invocation in-process.
type LocalRunnerInvoker struct {
	baseDir          string
	registry         *registry.Registry
	stderrLimitBytes int
	matchTimeout     time.Duration
}

// NewLocalRunnerInvoker constructs the initial local file-backed runner invoker.
func NewLocalRunnerInvoker(baseDir string, reg *registry.Registry, matchTimeout time.Duration) (*LocalRunnerInvoker, error) {
	if baseDir == "" {
		return nil, fmt.Errorf("service: base_dir is required")
	}
	if reg == nil {
		reg = registry.Default()
	}
	return &LocalRunnerInvoker{
		baseDir:          baseDir,
		registry:         reg,
		stderrLimitBytes: defaultWorkerStderrLimitBytes,
		matchTimeout:     matchTimeout,
	}, nil
}

// Run builds one game session plus player sessions and executes a single match.
func (i *LocalRunnerInvoker) Run(ctx context.Context, req ExecutionRequest) (ExecutionResult, error) {
	submission := req.Submission

	descriptor, err := i.registry.LookupVersion(ctx, submission.Game.GameID, submission.Game.GameVersion)
	if err != nil {
		return ExecutionResult{}, err
	}

	players, sessions, err := i.loadPlayersAndSessions(ctx, submission)
	if err != nil {
		return ExecutionResult{}, err
	}
	defer closeSessions(sessions)

	master, err := descriptor.BuildSession(registry.BuildSpec{
		GameVersion: submission.Game.GameVersion,
		Ruleset:     submission.Game.RulesetVersion,
		Players:     clonePlayers(players),
	})
	if err != nil {
		return ExecutionResult{}, err
	}

	runCtx := ctx
	cancel := func() {}
	if i.matchTimeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, i.matchTimeout)
	}
	defer cancel()

	record, runErr := match.NewRunner(submission.MatchID, players, master, sessions).Run(runCtx)
	return ExecutionResult{
		Record:       record,
		PlayerStderr: snapshotPlayerStderr(sessions),
	}, runErr
}

func (i *LocalRunnerInvoker) loadPlayersAndSessions(ctx context.Context, submission MatchSubmission) ([]game.Player, map[string]match.PlayerSession, error) {
	meta := catalog.GameMetadata{
		GameID:         submission.Game.GameID,
		GameVersion:    submission.Game.GameVersion,
		RulesetVersion: submission.Game.RulesetVersion,
	}
	players := make([]game.Player, 0, len(submission.Players))
	sessions := make(map[string]match.PlayerSession, len(submission.Players))
	for _, submitted := range submission.Players {
		entryPath, err := resolveLocalArtifactRef(i.baseDir, submitted.ArtifactRef)
		if err != nil {
			closeSessions(sessions)
			return nil, nil, fmt.Errorf("service: %s artifact_ref invalid: %w", submitted.PlayerID, err)
		}
		loaded, err := catalog.LoadEntry(meta, entryPath)
		if err != nil {
			closeSessions(sessions)
			return nil, nil, fmt.Errorf("service: %s runtime load failed: %w", submitted.PlayerID, err)
		}
		cfg := loaded.Runtime
		cfg.Dir = i.baseDir
		cfg.StderrLimitBytes = i.stderrLimitBytes
		adapter, err := runtime.Start(ctx, cfg)
		if err != nil {
			closeSessions(sessions)
			return nil, nil, fmt.Errorf("service: %s runtime start failed: %w", submitted.PlayerID, err)
		}
		players = append(players, game.Player{
			PlayerID: submitted.PlayerID,
			AIID:     loaded.AIID,
		})
		sessions[submitted.PlayerID] = session.New(adapter)
	}
	return players, sessions, nil
}

// LocalTerminalPersister writes standard artifacts plus per-player stderr logs under output_dir.
type LocalTerminalPersister struct{}

// Persist writes terminal artifacts under submission.output_dir/run_id.
func (LocalTerminalPersister) Persist(_ context.Context, submission MatchSubmission, result ExecutionResult) (TerminalArtifacts, error) {
	layout := artifacts.NewLayout(submission.OutputDir, submission.RunID)
	if err := artifacts.EnsureLayout(layout); err != nil {
		return TerminalArtifacts{}, err
	}
	if err := artifacts.WriteStandardArtifacts(layout, result.Record); err != nil {
		return TerminalArtifacts{}, err
	}
	if err := ensureStructuredLogFile(layout.StructuredLogPath); err != nil {
		return TerminalArtifacts{}, err
	}

	playerStderrPaths := make(map[string]string, len(result.Record.Players))
	for _, player := range result.Record.Players {
		stderr := result.PlayerStderr[player.PlayerID]
		path := filepath.Join(layout.MatchDir, player.PlayerID+"-stderr.log")
		if err := writePlayerStderr(path, stderr); err != nil {
			return TerminalArtifacts{}, err
		}
		playerStderrPaths[player.PlayerID] = path
	}

	return TerminalArtifacts{
		MatchDir:          layout.MatchDir,
		RecordPath:        layout.RecordPath,
		ResultSummaryPath: layout.ResultSummaryPath,
		PlayerStderrPaths: playerStderrPaths,
	}, nil
}

func writePlayerStderr(path, stderr string) error {
	file, err := artifacts.CreateFileOutput(path)
	if err != nil {
		return fmt.Errorf("create stderr artifact %s: %w", path, err)
	}
	defer file.Close()
	if _, err := file.WriteString(stderr); err != nil {
		return fmt.Errorf("write stderr artifact %s: %w", path, err)
	}
	return nil
}

func ensureStructuredLogFile(path string) error {
	file, err := artifacts.CreateFileOutput(path)
	if err != nil {
		return fmt.Errorf("create structured log artifact %s: %w", path, err)
	}
	return file.Close()
}

func closeSessions(sessions map[string]match.PlayerSession) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	for _, sess := range sessions {
		_ = sess.Close(ctx)
	}
}

func clonePlayers(players []game.Player) []game.Player {
	return append([]game.Player(nil), players...)
}

func snapshotPlayerStderr(sessions map[string]match.PlayerSession) map[string]string {
	snapshots := make(map[string]string, len(sessions))
	for playerID, sess := range sessions {
		stderr := sess.StderrSnapshot()
		if stderr.Output == "" {
			continue
		}
		snapshots[playerID] = stderr.Output
	}
	return snapshots
}
