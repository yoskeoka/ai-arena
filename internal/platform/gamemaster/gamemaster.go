package gamemaster

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/protocol"
	"github.com/yoskeoka/ai-arena/internal/platform/runtime"
)

type Mode string

const (
	ModeInProcess             Mode = "in-process"
	ModeLocalSubprocess       Mode = "local-subprocess"
	ModeFutureExternalAdapter Mode = "future-external-adapter"
)

type Session interface {
	Metadata() catalog.GameMetadata
	InitializeMatch(context.Context) (game.InitState, error)
	NextDecisionStep(context.Context) (*game.DecisionStep, error)
	NormalizeAction(context.Context, game.DecisionRequest, game.ActionStatus) (game.ActionStatus, error)
	ApplyDecisionResults(context.Context, game.DecisionStep, []game.ActionStatus) error
	CurrentSnapshot(context.Context) (game.Snapshot, error)
	CurrentExportedSnapshot(context.Context) (game.ExportedSnapshot, error)
	CurrentResult(context.Context) (game.MatchResult, error)
	Shutdown(context.Context) error
}

type InitializeMatchParams struct {
	Players        []game.Player  `json:"players"`
	ResumeSnapshot *game.Snapshot `json:"resume_snapshot,omitempty"`
}

type InitializeMatchResult struct {
	InitState game.InitState `json:"init_state"`
}

type ApplyDecisionResultsParams struct {
	Step           game.DecisionStep   `json:"step"`
	ActionStatuses []game.ActionStatus `json:"action_statuses"`
}

type inProcessSession struct {
	master game.Master
}

func NewInProcessSession(master game.Master) Session {
	return &inProcessSession{master: master}
}

func (s *inProcessSession) Metadata() catalog.GameMetadata {
	return s.master.Metadata()
}

func (s *inProcessSession) InitializeMatch(ctx context.Context) (game.InitState, error) {
	return s.master.Init(ctx)
}

func (s *inProcessSession) NextDecisionStep(ctx context.Context) (*game.DecisionStep, error) {
	return s.master.NextStep(ctx)
}

func (s *inProcessSession) NormalizeAction(_ context.Context, req game.DecisionRequest, actionStatus game.ActionStatus) (game.ActionStatus, error) {
	return s.master.NormalizeAction(req, actionStatus), nil
}

func (s *inProcessSession) ApplyDecisionResults(ctx context.Context, step game.DecisionStep, actionStatuses []game.ActionStatus) error {
	return s.master.ApplyStep(ctx, step, actionStatuses)
}

func (s *inProcessSession) CurrentSnapshot(context.Context) (game.Snapshot, error) {
	snapshot := s.master.Snapshot()
	if snapshot.PerPlayer == nil {
		snapshot.PerPlayer = make(map[string]game.PlayerSnapshot)
	}
	exported := s.master.ExportedSnapshot()
	for _, player := range exported.Players {
		playerState := snapshot.PerPlayer[player.PlayerID]
		if len(playerState.VisibleState) == 0 {
			playerState.VisibleState = s.master.VisibleState(player.PlayerID)
		}
		if playerState.LastActionStatus.PlayerID == "" {
			playerState.LastActionStatus = player.LastActionStatus
		}
		snapshot.PerPlayer[player.PlayerID] = playerState
	}
	return snapshot, nil
}

func (s *inProcessSession) CurrentExportedSnapshot(context.Context) (game.ExportedSnapshot, error) {
	return s.master.ExportedSnapshot(), nil
}

func (s *inProcessSession) CurrentResult(context.Context) (game.MatchResult, error) {
	return s.master.Result(), nil
}

func (s *inProcessSession) Shutdown(context.Context) error {
	return nil
}

type LocalSubprocessConfig struct {
	ExpectedMetadata catalog.GameMetadata
	Command          []string
	Dir              string
	Players          []game.Player
	ResumeSnapshot   *game.Snapshot
	StderrLimitBytes int
}

type localSubprocessSession struct {
	meta           catalog.GameMetadata
	players        []game.Player
	resumeSnapshot *game.Snapshot
	adapter        *runtime.Adapter
	nextID         int
	initialized    bool
	initState      game.InitState
}

func StartLocalSubprocess(cfg LocalSubprocessConfig) (Session, error) {
	adapter, err := runtime.Start(context.Background(), runtime.Config{
		Command:          cfg.Command,
		Dir:              cfg.Dir,
		StderrLimitBytes: cfg.StderrLimitBytes,
	})
	if err != nil {
		return nil, err
	}
	return &localSubprocessSession{
		meta:           cfg.ExpectedMetadata,
		players:        append([]game.Player(nil), cfg.Players...),
		resumeSnapshot: cloneSnapshotPtr(cfg.ResumeSnapshot),
		adapter:        adapter,
	}, nil
}

func (s *localSubprocessSession) Metadata() catalog.GameMetadata {
	return s.meta
}

func (s *localSubprocessSession) InitializeMatch(ctx context.Context) (game.InitState, error) {
	if s.initialized {
		return s.initState, nil
	}
	var actual catalog.GameMetadata
	if err := s.call(ctx, "metadata", nil, &actual); err != nil {
		return game.InitState{}, fmt.Errorf("game master metadata: %w", err)
	}
	if err := catalog.Compatible(s.meta, actual); err != nil {
		return game.InitState{}, fmt.Errorf("game master metadata incompatible: %w", err)
	}
	s.meta = actual

	var result InitializeMatchResult
	if err := s.call(ctx, "initialize_match", InitializeMatchParams{
		Players:        append([]game.Player(nil), s.players...),
		ResumeSnapshot: cloneSnapshotPtr(s.resumeSnapshot),
	}, &result); err != nil {
		return game.InitState{}, fmt.Errorf("game master initialize_match: %w", err)
	}
	s.initialized = true
	s.initState = result.InitState
	return s.initState, nil
}

func (s *localSubprocessSession) NextDecisionStep(ctx context.Context) (*game.DecisionStep, error) {
	var step *game.DecisionStep
	if err := s.call(ctx, "next_decision_step", nil, &step); err != nil {
		return nil, fmt.Errorf("game master next_decision_step: %w", err)
	}
	return step, nil
}

func (s *localSubprocessSession) NormalizeAction(ctx context.Context, req game.DecisionRequest, actionStatus game.ActionStatus) (game.ActionStatus, error) {
	var normalized game.ActionStatus
	if err := s.call(ctx, "normalize_action", struct {
		Request      game.DecisionRequest `json:"request"`
		ActionStatus game.ActionStatus    `json:"action_status"`
	}{
		Request:      req,
		ActionStatus: actionStatus,
	}, &normalized); err != nil {
		return game.ActionStatus{}, fmt.Errorf("game master normalize_action: %w", err)
	}
	return normalized, nil
}

func (s *localSubprocessSession) ApplyDecisionResults(ctx context.Context, step game.DecisionStep, actionStatuses []game.ActionStatus) error {
	return s.call(ctx, "apply_decision_results", ApplyDecisionResultsParams{
		Step:           step,
		ActionStatuses: append([]game.ActionStatus(nil), actionStatuses...),
	}, nil)
}

func (s *localSubprocessSession) CurrentSnapshot(ctx context.Context) (game.Snapshot, error) {
	if _, err := s.InitializeMatch(ctx); err != nil {
		return game.Snapshot{}, err
	}
	var snapshot game.Snapshot
	if err := s.call(ctx, "current_snapshot", nil, &snapshot); err != nil {
		return game.Snapshot{}, fmt.Errorf("game master current_snapshot: %w", err)
	}
	return snapshot, nil
}

func (s *localSubprocessSession) CurrentExportedSnapshot(ctx context.Context) (game.ExportedSnapshot, error) {
	if _, err := s.InitializeMatch(ctx); err != nil {
		return game.ExportedSnapshot{}, err
	}
	var snapshot game.ExportedSnapshot
	if err := s.call(ctx, "current_exported_snapshot", nil, &snapshot); err != nil {
		return game.ExportedSnapshot{}, fmt.Errorf("game master current_exported_snapshot: %w", err)
	}
	return snapshot, nil
}

func (s *localSubprocessSession) CurrentResult(ctx context.Context) (game.MatchResult, error) {
	if _, err := s.InitializeMatch(ctx); err != nil {
		return game.MatchResult{}, err
	}
	var result game.MatchResult
	if err := s.call(ctx, "current_result", nil, &result); err != nil {
		return game.MatchResult{}, fmt.Errorf("game master current_result: %w", err)
	}
	return result, nil
}

func (s *localSubprocessSession) Shutdown(ctx context.Context) error {
	shutdownErr := s.call(ctx, "shutdown", nil, nil)
	closeErr := s.adapter.Close(ctx)
	if shutdownErr != nil && closeErr != nil {
		return fmt.Errorf("%v; close runtime: %w", shutdownErr, closeErr)
	}
	if shutdownErr != nil {
		return shutdownErr
	}
	return closeErr
}

func (s *localSubprocessSession) call(ctx context.Context, method string, params any, dest any) error {
	s.nextID++
	reqID := fmt.Sprintf("gm-%03d", s.nextID)
	req, err := protocol.NewRequest(reqID, method, params)
	if err != nil {
		return err
	}
	if err := s.adapter.Send(req); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case incoming, ok := <-s.adapter.Incoming():
			if !ok {
				return fmt.Errorf("runtime stopped")
			}
			if incoming.Err != nil {
				return incoming.Err
			}
			if incoming.Response == nil {
				continue
			}
			if err := protocol.MatchResponseID(reqID, *incoming.Response); err != nil {
				return err
			}
			if incoming.Response.Error != nil {
				return fmt.Errorf("%s", incoming.Response.Error.Message)
			}
			if dest == nil {
				return nil
			}
			if err := decodeResult(incoming.Response.Result, dest); err != nil {
				return err
			}
			return nil
		}
	}
}

func decodeResult(raw json.RawMessage, dest any) error {
	if len(raw) == 0 {
		if dest != nil {
			return fmt.Errorf("decode result: empty result")
		}
		return nil
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return fmt.Errorf("decode result: %w", err)
	}
	return nil
}

func cloneSnapshotPtr(snapshot *game.Snapshot) *game.Snapshot {
	if snapshot == nil {
		return nil
	}
	copied := *snapshot
	return &copied
}
