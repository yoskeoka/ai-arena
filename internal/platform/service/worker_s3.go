package service

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/yoskeoka/ai-arena/internal/platform/artifacts"
)

// S3TerminalPersister writes standard artifacts plus per-player stderr logs to an S3-compatible backend.
type S3TerminalPersister struct {
	store *S3ArtifactStore
}

// NewS3TerminalPersister constructs an S3-compatible terminal persister.
func NewS3TerminalPersister(store *S3ArtifactStore) (*S3TerminalPersister, error) {
	if store == nil {
		return nil, fmt.Errorf("service: S3 artifact store is required")
	}
	return &S3TerminalPersister{store: store}, nil
}

// Persist writes terminal artifacts under output_dir/match_id in object storage.
func (p *S3TerminalPersister) Persist(ctx context.Context, submission MatchSubmission, result ExecutionResult) (TerminalArtifacts, error) {
	matchKeyPrefix := strings.Trim(strings.TrimSpace(submission.OutputDir), "/")
	matchKeyPrefix = path.Join(matchKeyPrefix, submission.MatchID)

	artifactNames := artifacts.Layout{
		RecordPath:           "record.json",
		StructuredLogPath:    "structured-log.ndjson",
		SnapshotPath:         "snapshot.json",
		ExportedSnapshotPath: "exported-snapshot.json",
		HistoryPath:          "history.json",
		ResultSummaryPath:    "result-summary.json",
	}
	summary, err := artifacts.BuildResultSummary(artifactNames, result.Record)
	if err != nil {
		return TerminalArtifacts{}, err
	}

	recordPath, err := p.putJSON(ctx, path.Join(matchKeyPrefix, "record.json"), result.Record)
	if err != nil {
		return TerminalArtifacts{}, err
	}
	if _, err := p.putJSON(ctx, path.Join(matchKeyPrefix, "snapshot.json"), result.Record.Snapshot); err != nil {
		return TerminalArtifacts{}, err
	}
	if _, err := p.putJSON(ctx, path.Join(matchKeyPrefix, "exported-snapshot.json"), result.Record.ExportedSnapshot); err != nil {
		return TerminalArtifacts{}, err
	}
	if _, err := p.putJSON(ctx, path.Join(matchKeyPrefix, "history.json"), result.Record.EventLog); err != nil {
		return TerminalArtifacts{}, err
	}
	resultSummaryPath, err := p.putJSON(ctx, path.Join(matchKeyPrefix, "result-summary.json"), summary)
	if err != nil {
		return TerminalArtifacts{}, err
	}
	if _, err := p.store.PutBytes(ctx, path.Join(matchKeyPrefix, "structured-log.ndjson"), []byte{}, "text/plain; charset=utf-8"); err != nil {
		return TerminalArtifacts{}, err
	}

	playerStderrPaths := make(map[string]string, len(result.Record.Players))
	for _, player := range result.Record.Players {
		playerKey := path.Join(matchKeyPrefix, player.PlayerID+"-stderr.log")
		locator, putErr := p.store.PutBytes(ctx, playerKey, []byte(result.PlayerStderr[player.PlayerID]), "text/plain; charset=utf-8")
		if putErr != nil {
			return TerminalArtifacts{}, putErr
		}
		playerStderrPaths[player.PlayerID] = locator
	}

	return TerminalArtifacts{
		MatchDir:          p.store.ObjectLocator(matchKeyPrefix),
		RecordPath:        recordPath,
		ResultSummaryPath: resultSummaryPath,
		PlayerStderrPaths: playerStderrPaths,
	}, nil
}

func (p *S3TerminalPersister) putJSON(ctx context.Context, key string, value any) (string, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("service: encode artifact %s: %w", key, err)
	}
	return p.store.PutBytes(ctx, key, body, "application/json")
}
