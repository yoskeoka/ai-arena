package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/yoskeoka/ai-arena/internal/platform/artifacts"
	"github.com/yoskeoka/ai-arena/internal/platform/contract"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
)

// ReplayResumeAuditInputs exposes the persisted inputs needed to replay, resume, or audit one match.
type ReplayResumeAuditInputs struct {
	Game                 contract.GameMetadata   `json:"game"`
	Players              []SubmittedPlayer       `json:"players"`
	RecordPath           string                  `json:"record_path,omitempty"`
	SnapshotPath         string                  `json:"snapshot_path,omitempty"`
	HistoryPath          string                  `json:"history_path,omitempty"`
	ExportedSnapshotPath string                  `json:"exported_snapshot_path,omitempty"`
	Verification         ReplayInputVerification `json:"verification"`
}

// ReplayInputVerification reports whether local artifact consistency checks ran and whether they passed.
type ReplayInputVerification struct {
	Checked    bool     `json:"checked"`
	Consistent bool     `json:"consistent"`
	Issues     []string `json:"issues,omitempty"`
}

func buildReplayResumeAuditInputs(ctx context.Context, record QueueRecord, summary *artifacts.ResultSummary, reader ArtifactReader) (*ReplayResumeAuditInputs, error) {
	if record.Terminal == nil || strings.TrimSpace(record.Terminal.RecordPath) == "" {
		return nil, nil
	}

	inputs := &ReplayResumeAuditInputs{
		Game:       record.Submission.Game,
		Players:    append([]SubmittedPlayer(nil), record.Submission.Players...),
		RecordPath: record.Terminal.RecordPath,
	}
	if strings.TrimSpace(record.Terminal.MatchDir) != "" {
		inputs.SnapshotPath = joinArtifactLocator(record.Terminal.MatchDir, artifactRef(summary, summaryPathSnapshot, "snapshot.json"))
		inputs.HistoryPath = joinArtifactLocator(record.Terminal.MatchDir, artifactRef(summary, summaryPathHistory, "history.json"))
		inputs.ExportedSnapshotPath = joinArtifactLocator(record.Terminal.MatchDir, artifactRef(summary, summaryPathExportedSnapshot, "exported-snapshot.json"))
	}

	verification, err := verifyReplayResumeAuditInputs(ctx, record, summary, inputs, reader)
	if err != nil {
		return nil, err
	}
	inputs.Verification = verification
	return inputs, nil
}

type summaryPathKind int

const (
	summaryPathSnapshot summaryPathKind = iota
	summaryPathHistory
	summaryPathExportedSnapshot
)

func artifactRef(summary *artifacts.ResultSummary, kind summaryPathKind, fallback string) string {
	if summary == nil {
		return fallback
	}
	switch kind {
	case summaryPathSnapshot:
		if summary.ArtifactPaths.Snapshot != "" {
			return summary.ArtifactPaths.Snapshot
		}
	case summaryPathHistory:
		if summary.ArtifactPaths.History != "" {
			return summary.ArtifactPaths.History
		}
	case summaryPathExportedSnapshot:
		if summary.ArtifactPaths.ExportedSnapshot != "" {
			return summary.ArtifactPaths.ExportedSnapshot
		}
	}
	return fallback
}

func verifyReplayResumeAuditInputs(ctx context.Context, record QueueRecord, summary *artifacts.ResultSummary, inputs *ReplayResumeAuditInputs, reader ArtifactReader) (ReplayInputVerification, error) {
	verification := ReplayInputVerification{}
	if inputs == nil || strings.TrimSpace(inputs.RecordPath) == "" {
		return verification, nil
	}

	verification.Checked = true
	persistedRecord, err := loadRecordFromLocator(ctx, reader, inputs.RecordPath)
	if err != nil {
		return verification, err
	}

	var issues []string
	if persistedRecord.MatchID != record.Submission.MatchID {
		issues = append(issues, fmt.Sprintf("record match_id %q does not match submission match_id %q", persistedRecord.MatchID, record.Submission.MatchID))
	}
	if persistedRecord.Game != record.Submission.Game {
		issues = append(issues, fmt.Sprintf("record game metadata %+v does not match submission metadata %+v", persistedRecord.Game, record.Submission.Game))
	}
	if !playerOrderMatches(record.Submission.Players, persistedRecord.Players) {
		issues = append(issues, "record player order does not match submitted players")
	}
	if summary != nil {
		issues = append(issues, verifyResultSummary(summary, persistedRecord, inputs)...)
	}

	if path := strings.TrimSpace(inputs.SnapshotPath); path == "" {
		issues = append(issues, "snapshot locator is not available")
	} else {
		snapshot, snapshotErr := loadSnapshotFromLocator(ctx, reader, path)
		if snapshotErr != nil {
			issues = append(issues, fmt.Sprintf("snapshot artifact could not be loaded: %v", snapshotErr))
		} else if !reflect.DeepEqual(snapshot, persistedRecord.Snapshot) {
			issues = append(issues, "snapshot artifact does not match record.json.snapshot")
		}
	}

	if path := strings.TrimSpace(inputs.HistoryPath); path == "" {
		issues = append(issues, "history locator is not available")
	} else {
		history, historyErr := loadHistoryFromLocator(ctx, reader, path)
		if historyErr != nil {
			issues = append(issues, fmt.Sprintf("history artifact could not be loaded: %v", historyErr))
		} else if !reflect.DeepEqual(history, persistedRecord.EventLog) {
			issues = append(issues, "history artifact does not match record.json.event_log")
		}
	}

	if path := strings.TrimSpace(inputs.ExportedSnapshotPath); path == "" {
		issues = append(issues, "exported snapshot locator is not available")
	} else {
		exportedSnapshot, exportedSnapshotErr := loadExportedSnapshotFromLocator(ctx, reader, path)
		if exportedSnapshotErr != nil {
			issues = append(issues, fmt.Sprintf("exported snapshot artifact could not be loaded: %v", exportedSnapshotErr))
		} else if !reflect.DeepEqual(exportedSnapshot, persistedRecord.ExportedSnapshot) {
			issues = append(issues, "exported snapshot artifact does not match record.json.exported_snapshot")
		}
	}

	verification.Consistent = len(issues) == 0
	verification.Issues = issues
	return verification, nil
}

func verifyResultSummary(summary *artifacts.ResultSummary, persistedRecord match.Record, inputs *ReplayResumeAuditInputs) []string {
	var issues []string
	if summary.MatchID != persistedRecord.MatchID {
		issues = append(issues, fmt.Sprintf("result-summary match_id %q does not match record match_id %q", summary.MatchID, persistedRecord.MatchID))
	}
	if summary.GameID != persistedRecord.Game.GameID || summary.GameVersion != persistedRecord.Game.GameVersion || summary.RulesetVersion != persistedRecord.Game.RulesetVersion {
		issues = append(issues, "result-summary game metadata does not match record metadata")
	}
	if summary.ArtifactPaths.Record != "" && locatorBase(inputs.RecordPath) != summary.ArtifactPaths.Record {
		issues = append(issues, fmt.Sprintf("result-summary record ref %q does not match resolved record locator %q", summary.ArtifactPaths.Record, locatorBase(inputs.RecordPath)))
	}
	if summary.ArtifactPaths.Snapshot != "" && locatorBase(inputs.SnapshotPath) != summary.ArtifactPaths.Snapshot {
		issues = append(issues, fmt.Sprintf("result-summary snapshot ref %q does not match resolved snapshot locator %q", summary.ArtifactPaths.Snapshot, locatorBase(inputs.SnapshotPath)))
	}
	if summary.ArtifactPaths.History != "" && locatorBase(inputs.HistoryPath) != summary.ArtifactPaths.History {
		issues = append(issues, fmt.Sprintf("result-summary history ref %q does not match resolved history locator %q", summary.ArtifactPaths.History, locatorBase(inputs.HistoryPath)))
	}
	if summary.ArtifactPaths.ExportedSnapshot != "" && locatorBase(inputs.ExportedSnapshotPath) != summary.ArtifactPaths.ExportedSnapshot {
		issues = append(issues, fmt.Sprintf("result-summary exported snapshot ref %q does not match resolved exported snapshot locator %q", summary.ArtifactPaths.ExportedSnapshot, locatorBase(inputs.ExportedSnapshotPath)))
	}
	return issues
}

func playerOrderMatches(submitted []SubmittedPlayer, persisted []game.Player) bool {
	if len(submitted) != len(persisted) {
		return false
	}
	for i := range submitted {
		if submitted[i].PlayerID != persisted[i].PlayerID {
			return false
		}
	}
	return true
}

func joinArtifactLocator(base, ref string) string {
	base = strings.TrimSpace(base)
	ref = strings.TrimSpace(ref)
	if base == "" || ref == "" {
		return ""
	}
	if isAbsoluteLocator(ref) {
		return ref
	}
	if isLocalPath(base) {
		basePath := localPath(base)
		joined := filepath.Join(basePath, filepath.Clean(ref))
		if strings.HasPrefix(base, "file://") {
			return "file://" + filepath.ToSlash(joined)
		}
		return joined
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" {
		return ""
	}
	parsed.Path = path.Join(strings.TrimSuffix(parsed.Path, "/"), ref)
	return parsed.String()
}

func isAbsoluteLocator(value string) bool {
	if filepath.IsAbs(value) {
		return true
	}
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme != ""
}

func localPath(value string) string {
	return filepath.Clean(strings.TrimPrefix(value, "file://"))
}

func locatorBase(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	if isLocalPath(value) {
		return filepath.Base(localPath(value))
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return path.Base(value)
	}
	return path.Base(parsed.Path)
}

func loadRecordFromLocator(ctx context.Context, reader ArtifactReader, locator string) (match.Record, error) {
	var record match.Record
	if err := readJSONArtifact(ctx, reader, locator, &record); err != nil {
		return match.Record{}, err
	}
	return record, nil
}

func loadSnapshotFromLocator(ctx context.Context, reader ArtifactReader, locator string) (game.Snapshot, error) {
	var snapshot game.Snapshot
	if err := readJSONArtifact(ctx, reader, locator, &snapshot); err != nil {
		return game.Snapshot{}, err
	}
	return snapshot, nil
}

func loadExportedSnapshotFromLocator(ctx context.Context, reader ArtifactReader, locator string) (game.ExportedSnapshot, error) {
	var snapshot game.ExportedSnapshot
	if err := readJSONArtifact(ctx, reader, locator, &snapshot); err != nil {
		return game.ExportedSnapshot{}, err
	}
	return snapshot, nil
}

func loadHistoryFromLocator(ctx context.Context, reader ArtifactReader, locator string) ([]match.Event, error) {
	var events []match.Event
	if err := readJSONArtifact(ctx, reader, locator, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func readJSONArtifact(ctx context.Context, reader ArtifactReader, locator string, target any) error {
	data, err := reader.Read(ctx, locator)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode artifact %s: %w", locator, err)
	}
	return nil
}
