package echo

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/contract"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/session"
)

const (
	GameID                   = "echo-count"
	GameVersion              = "2.0.0"
	RulesetSimultaneous3Turn = "phase2-simultaneous-3turn"
	RulesetSequential3Turn   = "phase2-sequential-3turn"
	RulesetSimultaneous2Turn = "phase2-simultaneous-2turn"
	defaultTurnDeadline      = 100 * time.Millisecond
)

type Config struct {
	GameVersion string
	Ruleset     string
	Players     []game.Player
}

type Master struct {
	meta       catalog.GameMetadata
	players    []game.Player
	playerIDs  []string
	mode       game.DecisionMode
	turns      int
	deadline   time.Duration
	resolved   int
	nextPlayer int
	score      map[string]int
	lastAction map[string]game.ActionStatus
}

type visibleState struct {
	Turn     int            `json:"turn"`
	Expected int            `json:"expected"`
	Score    map[string]int `json:"score"`
}

type initState struct {
	Mode        game.DecisionMode `json:"mode"`
	Turns       int               `json:"turns"`
	PlayerOrder []string          `json:"player_order"`
}

type action struct {
	Echo int `json:"echo"`
}

type publicState struct {
	Mode          game.DecisionMode `json:"mode"`
	ResolvedTurns int               `json:"resolved_turns"`
	Score         map[string]int    `json:"score"`
}

type snapshotState struct {
	Mode     game.DecisionMode `json:"mode"`
	Turn     int               `json:"turn"`
	Expected int               `json:"expected"`
	Score    map[string]int    `json:"score"`
}

func New(cfg Config) (*Master, error) {
	if len(cfg.Players) == 0 {
		return nil, fmt.Errorf("echo: at least one player is required")
	}
	if cfg.GameVersion == "" {
		return nil, fmt.Errorf("echo: game version is required")
	}

	meta, mode, turns, deadline, err := metadataForSelection(cfg.GameVersion, cfg.Ruleset)
	if err != nil {
		return nil, err
	}

	score := make(map[string]int, len(cfg.Players))
	lastAction := make(map[string]game.ActionStatus, len(cfg.Players))
	playerIDs := make([]string, 0, len(cfg.Players))
	for _, player := range cfg.Players {
		playerIDs = append(playerIDs, player.PlayerID)
		score[player.PlayerID] = 0
		lastAction[player.PlayerID] = game.ActionStatus{PlayerID: player.PlayerID, ActionStatus: session.StatusNoAction}
	}

	return &Master{
		meta:       meta,
		players:    append([]game.Player(nil), cfg.Players...),
		playerIDs:  playerIDs,
		mode:       mode,
		turns:      turns,
		deadline:   deadline,
		score:      score,
		lastAction: lastAction,
	}, nil
}

func NewFromSnapshot(cfg Config, snapshot game.Snapshot) (*Master, error) {
	master, err := New(cfg)
	if err != nil {
		return nil, err
	}
	if err := master.applySnapshot(snapshot); err != nil {
		return nil, err
	}
	return master, nil
}

func (m *Master) Metadata() catalog.GameMetadata {
	return m.meta
}

func metadataForSelection(gameVersion, ruleset string) (catalog.GameMetadata, game.DecisionMode, int, time.Duration, error) {
	if gameVersion != GameVersion {
		return catalog.GameMetadata{}, "", 0, 0, fmt.Errorf("echo: unsupported game version %q", gameVersion)
	}

	switch ruleset {
	case RulesetSimultaneous3Turn:
		return catalog.GameMetadata{
			GameID:         GameID,
			GameVersion:    gameVersion,
			RulesetVersion: RulesetSimultaneous3Turn,
		}, game.Simultaneous, 3, defaultTurnDeadline, nil
	case RulesetSequential3Turn:
		return catalog.GameMetadata{
			GameID:         GameID,
			GameVersion:    gameVersion,
			RulesetVersion: RulesetSequential3Turn,
		}, game.Sequential, 3, defaultTurnDeadline, nil
	case RulesetSimultaneous2Turn:
		return catalog.GameMetadata{
			GameID:         GameID,
			GameVersion:    gameVersion,
			RulesetVersion: RulesetSimultaneous2Turn,
		}, game.Simultaneous, 2, defaultTurnDeadline, nil
	default:
		return catalog.GameMetadata{}, "", 0, 0, fmt.Errorf("echo: unsupported ruleset %q", ruleset)
	}
}

func (m *Master) Init(context.Context) (game.InitState, error) {
	state := mustRaw(initState{
		Mode:        m.mode,
		Turns:       m.turns,
		PlayerOrder: append([]string(nil), m.playerIDs...),
	})
	perPlayer := make(map[string]json.RawMessage, len(m.players))
	for _, player := range m.players {
		perPlayer[player.PlayerID] = state
	}
	return game.InitState{PerPlayer: perPlayer}, nil
}

func (m *Master) NextStep(context.Context) (*game.DecisionStep, error) {
	if m.resolved >= m.turns {
		return nil, nil
	}

	turn := m.resolved + 1
	expected := turn
	switch m.mode {
	case game.Simultaneous:
		reqs := make([]game.DecisionRequest, 0, len(m.players))
		for _, player := range m.players {
			reqs = append(reqs, game.DecisionRequest{
				PlayerID:        player.PlayerID,
				VisibleState:    mustRaw(m.currentVisibleState(turn, expected)),
				LegalActionHint: mustRaw(map[string]any{"type": "object", "required": []string{"echo"}}),
				Deadline:        m.deadline,
			})
		}
		return &game.DecisionStep{Turn: turn, Mode: game.Simultaneous, Requests: reqs}, nil
	case game.Sequential:
		player := m.players[m.nextPlayer]
		return &game.DecisionStep{
			Turn: turn,
			Mode: game.Sequential,
			Requests: []game.DecisionRequest{{
				PlayerID:        player.PlayerID,
				VisibleState:    mustRaw(m.currentVisibleState(turn, expected)),
				LegalActionHint: mustRaw(map[string]any{"type": "object", "required": []string{"echo"}}),
				Deadline:        m.deadline,
			}},
		}, nil
	default:
		return nil, fmt.Errorf("echo: unsupported mode %q", m.mode)
	}
}

func (m *Master) NormalizeAction(req game.DecisionRequest, actionStatus game.ActionStatus) game.ActionStatus {
	if actionStatus.ActionStatus != session.StatusAccepted {
		actionStatus.Action = nil
		return actionStatus
	}

	var act action
	if err := json.Unmarshal(actionStatus.Action, &act); err != nil {
		return game.ActionStatus{
			PlayerID:      req.PlayerID,
			ActionStatus:  session.StatusNoAction,
			FailureReason: contract.ReasonIllegalAction,
		}
	}
	if act.Echo != m.expectedForTurn(req) {
		return game.ActionStatus{
			PlayerID:      req.PlayerID,
			ActionStatus:  session.StatusNoAction,
			FailureReason: contract.ReasonIllegalAction,
		}
	}
	return actionStatus
}

func (m *Master) ApplyStep(_ context.Context, step game.DecisionStep, actionStatuses []game.ActionStatus) error {
	for _, actionStatus := range actionStatuses {
		m.lastAction[actionStatus.PlayerID] = actionStatus
		if actionStatus.ActionStatus == session.StatusAccepted {
			m.score[actionStatus.PlayerID]++
		}
	}

	switch step.Mode {
	case game.Simultaneous:
		m.resolved++
	case game.Sequential:
		m.nextPlayer++
		if m.nextPlayer >= len(m.players) {
			m.nextPlayer = 0
			m.resolved++
		}
	default:
		return fmt.Errorf("echo: unsupported mode %q", step.Mode)
	}
	return nil
}

func (m *Master) Snapshot() game.Snapshot {
	return game.Snapshot{
		GameID:         m.meta.GameID,
		GameVersion:    m.meta.GameVersion,
		RulesetVersion: m.meta.RulesetVersion,
		Turn:           m.resolved,
		Status:         game.StatusRunning,
		GameState:      mustRaw(m.snapshotState()),
	}
}

func (m *Master) VisibleState(string) json.RawMessage {
	turn := m.resolved + 1
	if turn > m.turns {
		turn = m.turns
	}
	expected := turn
	return mustRaw(m.currentVisibleState(turn, expected))
}

func (m *Master) ExportedSnapshot() game.ExportedSnapshot {
	exported := game.ExportedSnapshot{
		GameID:         m.meta.GameID,
		GameVersion:    m.meta.GameVersion,
		RulesetVersion: m.meta.RulesetVersion,
		Turn:           m.resolved,
		Status:         game.StatusRunning,
		PublicState:    mustRaw(publicState{Mode: m.mode, ResolvedTurns: m.resolved, Score: cloneScore(m.score)}),
	}
	for _, playerID := range m.playerIDs {
		exported.Players = append(exported.Players, game.ExportedPlayerSnapshot{
			PlayerID:         playerID,
			LastActionStatus: m.lastAction[playerID],
		})
	}
	return exported
}

func (m *Master) Result() game.MatchResult {
	type ranked struct {
		playerID string
		score    int
	}
	rankedPlayers := make([]ranked, 0, len(m.players))
	for _, player := range m.players {
		rankedPlayers = append(rankedPlayers, ranked{playerID: player.PlayerID, score: m.score[player.PlayerID]})
	}
	sort.Slice(rankedPlayers, func(i, j int) bool {
		if rankedPlayers[i].score != rankedPlayers[j].score {
			return rankedPlayers[i].score > rankedPlayers[j].score
		}
		return rankedPlayers[i].playerID < rankedPlayers[j].playerID
	})

	placements := make([]game.Placement, 0, len(rankedPlayers))
	lastScore := 0
	lastPlace := 0
	for i, player := range rankedPlayers {
		place := i + 1
		if i > 0 && player.score == lastScore {
			place = lastPlace
		}
		placements = append(placements, game.Placement{PlayerID: player.playerID, Place: place})
		lastScore = player.score
		lastPlace = place
	}
	return game.MatchResult{Placements: placements}
}

func (m *Master) currentVisibleState(turn, expected int) visibleState {
	return visibleState{
		Turn:     turn,
		Expected: expected,
		Score:    cloneScore(m.score),
	}
}

func (m *Master) snapshotState() map[string]any {
	expected := 0
	if m.resolved < m.turns {
		expected = m.resolved + 1
	} else {
		expected = m.turns
	}
	return map[string]any{
		"mode":     m.mode,
		"turn":     m.resolved,
		"expected": expected,
		"score":    cloneScore(m.score),
	}
}

func (m *Master) applySnapshot(snapshot game.Snapshot) error {
	if snapshot.GameID != "" && snapshot.GameID != m.meta.GameID {
		return fmt.Errorf("echo: snapshot game id %q does not match %q", snapshot.GameID, m.meta.GameID)
	}
	if snapshot.GameVersion != "" && snapshot.GameVersion != m.meta.GameVersion {
		return fmt.Errorf("echo: snapshot game version %q does not match %q", snapshot.GameVersion, m.meta.GameVersion)
	}
	if snapshot.RulesetVersion != "" && snapshot.RulesetVersion != m.meta.RulesetVersion {
		return fmt.Errorf("echo: snapshot ruleset %q does not match %q", snapshot.RulesetVersion, m.meta.RulesetVersion)
	}

	var state snapshotState
	if err := json.Unmarshal(snapshot.GameState, &state); err != nil {
		return fmt.Errorf("echo: decode snapshot game_state: %w", err)
	}
	if state.Mode != "" && state.Mode != m.mode {
		return fmt.Errorf("echo: snapshot mode %q does not match %q", state.Mode, m.mode)
	}
	if snapshot.Turn < 0 || snapshot.Turn > m.turns {
		return fmt.Errorf("echo: snapshot turn %d out of range 0..%d", snapshot.Turn, m.turns)
	}

	m.resolved = snapshot.Turn
	m.nextPlayer = 0
	for _, playerID := range m.playerIDs {
		m.score[playerID] = 0
		m.lastAction[playerID] = game.ActionStatus{PlayerID: playerID, ActionStatus: session.StatusNoAction}
	}
	for playerID, score := range state.Score {
		if _, ok := m.score[playerID]; !ok {
			return fmt.Errorf("echo: snapshot score has unknown player %q", playerID)
		}
		m.score[playerID] = score
	}
	for playerID, playerState := range snapshot.PerPlayer {
		if _, ok := m.lastAction[playerID]; !ok {
			return fmt.Errorf("echo: snapshot per_player has unknown player %q", playerID)
		}
		if playerState.LastActionStatus.PlayerID == "" {
			playerState.LastActionStatus.PlayerID = playerID
		}
		if playerState.LastActionStatus.ActionStatus != "" {
			m.lastAction[playerID] = playerState.LastActionStatus
		}
	}
	return nil
}

func (m *Master) expectedForTurn(req game.DecisionRequest) int {
	var state visibleState
	if err := json.Unmarshal(req.VisibleState, &state); err == nil && state.Expected > 0 {
		return state.Expected
	}
	return m.resolved + 1
}

func cloneScore(src map[string]int) map[string]int {
	cloned := make(map[string]int, len(src))
	for key, value := range src {
		cloned[key] = value
	}
	return cloned
}

func mustRaw(v any) json.RawMessage {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return raw
}
