package service

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/yoskeoka/ai-arena/internal/platform/contract"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
)

// MatchSubmission is one admitted request to execute a single match.
type MatchSubmission struct {
	SubmissionID string                `json:"submission_id"`
	MatchID      string                `json:"match_id"`
	Game         contract.GameMetadata `json:"game"`
	Players      []SubmittedPlayer     `json:"players"`
	OutputDir    string                `json:"output_dir"`
	AttemptCount int                   `json:"attempt_count"`
}

// SubmittedPlayer binds a player id to an opaque AI artifact reference.
type SubmittedPlayer struct {
	PlayerID    string `json:"player_id"`
	ArtifactRef string `json:"artifact_ref"`
}

// LifecycleState tracks service-side queue and execution progress.
type LifecycleState string

const (
	// StateQueued is waiting for worker claim after admission passes.
	StateQueued LifecycleState = "queued"
	// StateLeased is exclusively claimed by a worker before runner start.
	StateLeased LifecycleState = "leased"
	// StateRunning means the runner has started the match.
	StateRunning LifecycleState = "running"
	// StatePersisting means terminal artifacts are being written.
	StatePersisting LifecycleState = "persisting"
	// StateCompleted means terminal persist succeeded.
	StateCompleted LifecycleState = "completed"
	// StateFailed means execution or persist failed without retry.
	StateFailed LifecycleState = "failed"
	// StateCanceled means a queued job was canceled before lease.
	StateCanceled LifecycleState = "canceled"
)

// QueueRecord is the persisted service-side lifecycle state for one submission.
type QueueRecord struct {
	Submission MatchSubmission    `json:"submission"`
	State      LifecycleState     `json:"state"`
	Lease      *WorkerLease       `json:"lease,omitempty"`
	Terminal   *TerminalArtifacts `json:"terminal,omitempty"`
}

// WorkerLease records which worker currently owns a queued submission.
type WorkerLease struct {
	WorkerID string `json:"worker_id"`
}

// ExecutionRequest is the worker-to-runner execution handoff.
type ExecutionRequest struct {
	Submission MatchSubmission `json:"submission"`
}

// ExecutionResult is the worker-visible terminal runner output plus captured stderr text.
type ExecutionResult struct {
	Record       match.Record      `json:"record"`
	PlayerStderr map[string]string `json:"player_stderr,omitempty"`
}

// TerminalArtifacts captures the minimum persisted output for one terminal match.
type TerminalArtifacts struct {
	MatchDir          string            `json:"match_dir"`
	RecordPath        string            `json:"record_path"`
	ResultSummaryPath string            `json:"result_summary_path"`
	PlayerStderrPaths map[string]string `json:"player_stderr_paths,omitempty"`
	MatchStatus       game.MatchStatus  `json:"match_status"`
	Error             string            `json:"error,omitempty"`
}

// QueueStore manages service-side queue state.
type QueueStore interface {
	Enqueue(context.Context, MatchSubmission) (QueueRecord, error)
	Claim(context.Context, string) (QueueRecord, error)
	Update(context.Context, QueueRecord) error
	CancelQueued(context.Context, string) (QueueRecord, error)
	Get(context.Context, string) (QueueRecord, error)
	List(context.Context) ([]QueueRecord, error)
}

// AdmissionValidator validates a submission before queue admission.
type AdmissionValidator interface {
	Validate(context.Context, MatchSubmission) error
}

// DryRunChecker reuses runner-facing startup checks without full match execution.
type DryRunChecker interface {
	Check(context.Context, MatchSubmission) error
}

// RunnerInvoker executes exactly one match for a leased submission.
type RunnerInvoker interface {
	Run(context.Context, ExecutionRequest) (ExecutionResult, error)
}

// TerminalPersister writes file-backed artifacts for a terminal runner result.
type TerminalPersister interface {
	Persist(context.Context, MatchSubmission, ExecutionResult) (TerminalArtifacts, error)
}

// ValidateSubmission checks the minimum service skeleton contract.
func ValidateSubmission(submission MatchSubmission) error {
	if strings.TrimSpace(submission.SubmissionID) == "" {
		return fmt.Errorf("service: submission_id is required")
	}
	if strings.TrimSpace(submission.MatchID) == "" {
		return fmt.Errorf("service: match_id is required")
	}
	if strings.TrimSpace(submission.Game.GameID) == "" {
		return fmt.Errorf("service: game.game_id is required")
	}
	if strings.TrimSpace(submission.Game.GameVersion) == "" {
		return fmt.Errorf("service: game.game_version is required")
	}
	if strings.TrimSpace(submission.Game.RulesetVersion) == "" {
		return fmt.Errorf("service: game.ruleset_version is required")
	}
	if len(submission.Players) == 0 {
		return fmt.Errorf("service: at least one player is required")
	}
	if strings.TrimSpace(submission.OutputDir) == "" {
		return fmt.Errorf("service: output_dir is required")
	}
	if submission.AttemptCount != 1 {
		return fmt.Errorf("service: attempt_count must be 1 in the initial service skeleton")
	}

	playerIDs := make([]string, 0, len(submission.Players))
	for _, player := range submission.Players {
		if strings.TrimSpace(player.PlayerID) == "" {
			return fmt.Errorf("service: player_id is required")
		}
		if strings.TrimSpace(player.ArtifactRef) == "" {
			return fmt.Errorf("service: artifact_ref is required for player %q", player.PlayerID)
		}
		if slices.Contains(playerIDs, player.PlayerID) {
			return fmt.Errorf("service: duplicate player_id %q", player.PlayerID)
		}
		playerIDs = append(playerIDs, player.PlayerID)
	}

	return nil
}

// ValidateTransition checks whether one lifecycle transition is allowed.
func ValidateTransition(from, to LifecycleState) error {
	if from == to {
		return nil
	}

	switch from {
	case StateQueued:
		if to == StateLeased || to == StateCanceled {
			return nil
		}
	case StateLeased:
		if to == StateRunning || to == StateFailed {
			return nil
		}
	case StateRunning:
		if to == StatePersisting || to == StateFailed {
			return nil
		}
	case StatePersisting:
		if to == StateCompleted || to == StateFailed {
			return nil
		}
	case StateCompleted, StateFailed, StateCanceled:
		return fmt.Errorf("service: terminal state %q cannot transition to %q", from, to)
	default:
		return fmt.Errorf("service: unknown lifecycle state %q", from)
	}

	return fmt.Errorf("service: invalid transition %q -> %q", from, to)
}
