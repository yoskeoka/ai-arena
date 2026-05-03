package echo

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
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
	Mode        string   `json:"mode"`
	Turns       int      `json:"turns"`
	PlayerOrder []string `json:"player_order"`
}

type action struct {
	Echo int `json:"echo"`
}

type publicState struct {
	Mode          string         `json:"mode"`
	ResolvedTurns int            `json:"resolved_turns"`
	Score         map[string]int `json:"score"`
}

func New(cfg Config) (*Master, error) {
	if len(cfg.Players) == 0 {
		return nil, fmt.Errorf("echo: at least one player is required")
	}
	if cfg.GameVersion == "" {
		return nil, fmt.Errorf("echo: game version is required")
	}

	meta, turns, deadline, err := metadataForSelection(cfg.GameVersion, cfg.Ruleset)
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
		turns:      turns,
		deadline:   deadline,
		score:      score,
		lastAction: lastAction,
	}, nil
}

func (m *Master) Metadata() catalog.GameMetadata {
	return m.meta
}

func metadataForSelection(gameVersion, ruleset string) (catalog.GameMetadata, int, time.Duration, error) {
	if gameVersion != GameVersion {
		return catalog.GameMetadata{}, 0, 0, fmt.Errorf("echo: unsupported game version %q", gameVersion)
	}

	switch ruleset {
	case RulesetSimultaneous3Turn:
		return catalog.GameMetadata{
			GameID:         GameID,
			GameVersion:    gameVersion,
			RulesetVersion: RulesetSimultaneous3Turn,
			TurnMode:       string(game.Simultaneous),
		}, 3, defaultTurnDeadline, nil
	case RulesetSequential3Turn:
		return catalog.GameMetadata{
			GameID:         GameID,
			GameVersion:    gameVersion,
			RulesetVersion: RulesetSequential3Turn,
			TurnMode:       string(game.Sequential),
		}, 3, defaultTurnDeadline, nil
	case RulesetSimultaneous2Turn:
		return catalog.GameMetadata{
			GameID:         GameID,
			GameVersion:    gameVersion,
			RulesetVersion: RulesetSimultaneous2Turn,
			TurnMode:       string(game.Simultaneous),
		}, 2, defaultTurnDeadline, nil
	default:
		return catalog.GameMetadata{}, 0, 0, fmt.Errorf("echo: unsupported ruleset %q", ruleset)
	}
}

func (m *Master) Init(context.Context) (game.InitState, error) {
	state := mustRaw(initState{
		Mode:        m.meta.TurnMode,
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
	switch game.DecisionMode(m.meta.TurnMode) {
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
		return nil, fmt.Errorf("echo: unsupported mode %q", m.meta.TurnMode)
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
			FailureReason: "invalid-illegal-action",
		}
	}
	if act.Echo != m.expectedForTurn(req) {
		return game.ActionStatus{
			PlayerID:      req.PlayerID,
			ActionStatus:  session.StatusNoAction,
			FailureReason: "invalid-illegal-action",
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
		Turn:      m.resolved,
		Status:    "running",
		GameState: mustRaw(m.snapshotState()),
	}
}

func (m *Master) ExportedSnapshot() game.ExportedSnapshot {
	return game.ExportedSnapshot{
		Turn:        m.resolved,
		Status:      "running",
		PublicState: mustRaw(publicState{Mode: m.meta.TurnMode, ResolvedTurns: m.resolved, Score: cloneScore(m.score)}),
	}
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
		"mode":     m.meta.TurnMode,
		"turn":     m.resolved,
		"expected": expected,
		"score":    cloneScore(m.score),
	}
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
