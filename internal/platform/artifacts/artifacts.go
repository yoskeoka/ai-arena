// Package artifacts persists the standard file-backed runner artifact layout.
package artifacts

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
)

// Layout defines the standard file-backed runner artifact paths for one match.
type Layout struct {
	BaseDir              string
	MatchDir             string
	RecordPath           string
	StructuredLogPath    string
	SnapshotPath         string
	ExportedSnapshotPath string
	HistoryPath          string
	ResultSummaryPath    string
}

// PathRefs stores match-dir-relative references used by result-summary.json.
type PathRefs struct {
	Record           string `json:"record"`
	StructuredLog    string `json:"structured_log"`
	Snapshot         string `json:"snapshot"`
	ExportedSnapshot string `json:"exported_snapshot"`
	History          string `json:"history"`
}

// ResultSummary is the compact terminal summary stored next to record.json.
type ResultSummary struct {
	MatchID        string           `json:"match_id"`
	GameID         string           `json:"game_id"`
	GameVersion    string           `json:"game_version"`
	RulesetVersion string           `json:"ruleset_version"`
	Status         game.MatchStatus `json:"status"`
	Turn           int              `json:"turn"`
	Placements     []game.Placement `json:"placements,omitempty"`
	ArtifactPaths  PathRefs         `json:"artifact_paths"`
	Error          string           `json:"error,omitempty"`
}

// NewLayout builds the standard artifact file layout under outputDir/matchID.
func NewLayout(outputDir, matchID string) Layout {
	matchDir := filepath.Join(outputDir, matchID)
	return Layout{
		BaseDir:              outputDir,
		MatchDir:             matchDir,
		RecordPath:           filepath.Join(matchDir, "record.json"),
		StructuredLogPath:    filepath.Join(matchDir, "structured-log.ndjson"),
		SnapshotPath:         filepath.Join(matchDir, "snapshot.json"),
		ExportedSnapshotPath: filepath.Join(matchDir, "exported-snapshot.json"),
		HistoryPath:          filepath.Join(matchDir, "history.json"),
		ResultSummaryPath:    filepath.Join(matchDir, "result-summary.json"),
	}
}

// EnsureLayout creates the match artifact directory if it does not exist yet.
func EnsureLayout(layout Layout) error {
	if err := os.MkdirAll(layout.MatchDir, 0o750); err != nil {
		return fmt.Errorf("create artifact directory %s: %w", layout.MatchDir, err)
	}
	return nil
}

// WriteStandardArtifacts persists the standard runner artifact set except stderr logs.
func WriteStandardArtifacts(layout Layout, record match.Record) error {
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
	summary, err := BuildResultSummary(layout, record)
	if err != nil {
		return err
	}
	if err := writeJSONFile(layout.ResultSummaryPath, summary, "result summary"); err != nil {
		return err
	}
	return nil
}

// BuildResultSummary materializes the persisted terminal summary for one record.
func BuildResultSummary(layout Layout, record match.Record) (ResultSummary, error) {
	summary := ResultSummary{
		MatchID:        record.MatchID,
		GameID:         record.Game.GameID,
		GameVersion:    record.Game.GameVersion,
		RulesetVersion: record.Game.RulesetVersion,
		Status:         record.Status,
		Turn:           record.Snapshot.Turn,
		Placements:     append([]game.Placement(nil), record.Result.Placements...),
		ArtifactPaths: PathRefs{
			Record:           filepath.Base(layout.RecordPath),
			StructuredLog:    filepath.Base(layout.StructuredLogPath),
			Snapshot:         filepath.Base(layout.SnapshotPath),
			ExportedSnapshot: filepath.Base(layout.ExportedSnapshotPath),
			History:          filepath.Base(layout.HistoryPath),
		},
	}
	if errMsg := TerminalError(record); errMsg != "" {
		summary.Error = errMsg
	}
	return summary, nil
}

// TerminalError extracts the terminal failure/cancel reason from the event log when present.
func TerminalError(record match.Record) string {
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

// CreateFileOutput opens one local output file, creating parent directories as needed.
func CreateFileOutput(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, err
	}
	// #nosec G304 -- the caller explicitly selects the local output target path.
	return os.Create(path)
}

// WriteJSONToTarget writes JSON either to stdout or to the requested local file path.
func WriteJSONToTarget(target string, value any, stdout io.Writer, label string) error {
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

func openOutputTarget(target string, stdout io.Writer) (io.Writer, func() error, error) {
	switch target {
	case "", "stdout":
		return stdout, func() error { return nil }, nil
	default:
		file, err := CreateFileOutput(target)
		if err != nil {
			return nil, nil, fmt.Errorf("create output target %s: %w", target, err)
		}
		return file, file.Close, nil
	}
}

func writeJSONFile(path string, value any, label string) error {
	file, err := CreateFileOutput(path)
	if err != nil {
		return fmt.Errorf("create %s %s: %w", label, path, err)
	}
	defer file.Close()
	if err := json.NewEncoder(file).Encode(value); err != nil {
		return fmt.Errorf("write %s %s: %w", label, path, err)
	}
	return nil
}
