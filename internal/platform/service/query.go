package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/yoskeoka/ai-arena/internal/platform/artifacts"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
)

// ResultListItem is the compact operator-facing view for one submission.
type ResultListItem struct {
	SubmissionID      string            `json:"submission_id"`
	MatchID           string            `json:"match_id"`
	GameID            string            `json:"game_id"`
	GameVersion       string            `json:"game_version"`
	RulesetVersion    string            `json:"ruleset_version"`
	LifecycleState    LifecycleState    `json:"lifecycle_state"`
	WorkerID          string            `json:"worker_id,omitempty"`
	TerminalStatus    *game.MatchStatus `json:"terminal_status,omitempty"`
	Error             string            `json:"error,omitempty"`
	Turn              *int              `json:"turn,omitempty"`
	Placements        []game.Placement  `json:"placements,omitempty"`
	ResultSummaryPath string            `json:"result_summary_path,omitempty"`
}

// MatchDetail is the operator-facing detail view for one submission.
type MatchDetail struct {
	ResultListItem
	Players           []SubmittedPlayer        `json:"players"`
	OutputDir         string                   `json:"output_dir"`
	MatchDir          string                   `json:"match_dir,omitempty"`
	RecordPath        string                   `json:"record_path,omitempty"`
	PlayerStderrPaths map[string]string        `json:"player_stderr_paths,omitempty"`
	ResultSummary     *artifacts.ResultSummary `json:"result_summary,omitempty"`
	ReplayInputs      *ReplayResumeAuditInputs `json:"replay_inputs,omitempty"`
}

// QueryService builds operator-facing read models from queue records and persisted artifacts.
type QueryService struct {
	queue QueueStore
}

// NewQueryService constructs the read-side query service.
func NewQueryService(queue QueueStore) (*QueryService, error) {
	if queue == nil {
		return nil, fmt.Errorf("service: queue store is required")
	}
	return &QueryService{queue: queue}, nil
}

// List returns compact operator-facing rows for all known submissions.
func (s *QueryService) List(ctx context.Context) ([]ResultListItem, error) {
	records, err := s.queue.List(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]ResultListItem, 0, len(records))
	for _, record := range records {
		item, _, err := buildResultListItem(record)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// Get returns the operator-facing detail view for one submission.
func (s *QueryService) Get(ctx context.Context, submissionID string) (MatchDetail, error) {
	record, err := s.queue.Get(ctx, submissionID)
	if err != nil {
		return MatchDetail{}, err
	}
	item, summary, err := buildResultListItem(record)
	if err != nil {
		return MatchDetail{}, err
	}
	detail := MatchDetail{
		ResultListItem: item,
		Players:        append([]SubmittedPlayer(nil), record.Submission.Players...),
		OutputDir:      record.Submission.OutputDir,
		ResultSummary:  summary,
	}
	if record.Terminal != nil {
		detail.MatchDir = record.Terminal.MatchDir
		detail.RecordPath = record.Terminal.RecordPath
		if record.Terminal.PlayerStderrPaths != nil {
			detail.PlayerStderrPaths = cloneStringMap(record.Terminal.PlayerStderrPaths)
		}
	}
	detail.ReplayInputs, err = buildReplayResumeAuditInputs(record, summary)
	if err != nil {
		return MatchDetail{}, err
	}
	return detail, nil
}

func buildResultListItem(record QueueRecord) (ResultListItem, *artifacts.ResultSummary, error) {
	item := ResultListItem{
		SubmissionID:   record.Submission.SubmissionID,
		MatchID:        record.Submission.MatchID,
		GameID:         record.Submission.Game.GameID,
		GameVersion:    record.Submission.Game.GameVersion,
		RulesetVersion: record.Submission.Game.RulesetVersion,
		LifecycleState: record.State,
	}
	if record.Lease != nil {
		item.WorkerID = record.Lease.WorkerID
	}
	if record.Terminal == nil {
		return item, nil, nil
	}

	item.ResultSummaryPath = record.Terminal.ResultSummaryPath
	if record.Terminal.MatchStatus != "" {
		status := record.Terminal.MatchStatus
		item.TerminalStatus = &status
	}
	item.Error = record.Terminal.Error

	summary, err := readResultSummary(record.Terminal.ResultSummaryPath)
	if err != nil {
		return ResultListItem{}, nil, err
	}
	if summary == nil {
		return item, nil, nil
	}

	status := summary.Status
	item.TerminalStatus = &status
	item.Error = summary.Error
	turn := summary.Turn
	item.Turn = &turn
	item.Placements = append([]game.Placement(nil), summary.Placements...)
	return item, summary, nil
}

func readResultSummary(path string) (*artifacts.ResultSummary, error) {
	if path == "" || !isLocalPath(path) {
		return nil, nil
	}
	data, err := os.ReadFile(filepath.Clean(strings.TrimPrefix(path, "file://")))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("service: read result summary %s: %w", path, err)
	}
	var summary artifacts.ResultSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return nil, fmt.Errorf("service: decode result summary %s: %w", path, err)
	}
	return &summary, nil
}

func isLocalPath(path string) bool {
	parsed, err := url.Parse(path)
	if err != nil {
		return true
	}
	return parsed.Scheme == "" || parsed.Scheme == "file"
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(src))
	for key, value := range src {
		cloned[key] = value
	}
	return cloned
}
