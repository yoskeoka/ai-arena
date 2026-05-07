package janken

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
	GameID              = "janken"
	GameVersion         = "2.1.0"
	RulesetRegular      = "regular"
	RegularRounds       = 5
	defaultTurnDeadline = 100 * time.Millisecond
)

var legalActionHint = mustRaw(map[string]any{
	"type":     "object",
	"required": []string{"action"},
	"properties": map[string]any{
		"action": map[string]any{
			"type": "string",
			"enum": []string{"rock", "paper", "scissors"},
		},
	},
})

type Config struct {
	GameVersion string
	Ruleset     string
	Players     []game.Player
}

type Master struct {
	meta          catalog.GameMetadata
	players       []game.Player
	playerIDs     []string
	rounds        int
	deadline      time.Duration
	resolved      int
	scores        map[string]score
	selfHistory   map[string][]selfRound
	publicHistory []publicRound
	lastAction    map[string]game.ActionStatus
}

type initState struct {
	Players []string `json:"players"`
	Rounds  int      `json:"rounds"`
}

type visibleState struct {
	Round         int           `json:"round"`
	Rounds        int           `json:"rounds"`
	SelfHistory   []selfRound   `json:"self_history"`
	PublicHistory []publicRound `json:"public_history"`
}

type action struct {
	Action string `json:"action"`
}

type selfRound struct {
	Round  int    `json:"round"`
	Action string `json:"action"`
	Result string `json:"result"`
}

type publicRound struct {
	Round   int               `json:"round"`
	Actions map[string]string `json:"actions"`
	Results map[string]string `json:"results"`
}

type score struct {
	Wins           int `json:"wins"`
	Losses         int `json:"losses"`
	Draws          int `json:"draws"`
	Timeouts       int `json:"timeouts"`
	InvalidActions int `json:"invalid_actions"`
}

type snapshotState struct {
	Rounds         int                    `json:"rounds"`
	ResolvedRounds int                    `json:"resolved_rounds"`
	Scores         map[string]score       `json:"scores"`
	SelfHistory    map[string][]selfRound `json:"self_history"`
	PublicHistory  []publicRound          `json:"public_history"`
}

type publicState struct {
	Rounds         int              `json:"rounds"`
	ResolvedRounds int              `json:"resolved_rounds"`
	Scores         map[string]score `json:"scores"`
	PublicHistory  []publicRound    `json:"public_history"`
}

func New(cfg Config) (*Master, error) {
	if len(cfg.Players) < 2 {
		return nil, fmt.Errorf("janken: at least two players are required")
	}
	if cfg.GameVersion == "" {
		return nil, fmt.Errorf("janken: game version is required")
	}

	meta, rounds, deadline, err := metadataForSelection(cfg.GameVersion, cfg.Ruleset)
	if err != nil {
		return nil, err
	}

	scores := make(map[string]score, len(cfg.Players))
	selfHistory := make(map[string][]selfRound, len(cfg.Players))
	lastAction := make(map[string]game.ActionStatus, len(cfg.Players))
	playerIDs := make([]string, 0, len(cfg.Players))
	for _, player := range cfg.Players {
		playerIDs = append(playerIDs, player.PlayerID)
		scores[player.PlayerID] = score{}
		selfHistory[player.PlayerID] = nil
		lastAction[player.PlayerID] = game.ActionStatus{PlayerID: player.PlayerID, ActionStatus: session.StatusNoAction}
	}

	return &Master{
		meta:        meta,
		players:     append([]game.Player(nil), cfg.Players...),
		playerIDs:   playerIDs,
		rounds:      rounds,
		deadline:    deadline,
		scores:      scores,
		selfHistory: selfHistory,
		lastAction:  lastAction,
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

func metadataForSelection(gameVersion, ruleset string) (catalog.GameMetadata, int, time.Duration, error) {
	if gameVersion != GameVersion {
		return catalog.GameMetadata{}, 0, 0, fmt.Errorf("janken: unsupported game version %q", gameVersion)
	}
	switch ruleset {
	case RulesetRegular:
		return catalog.GameMetadata{
			GameID:         GameID,
			GameVersion:    gameVersion,
			RulesetVersion: RulesetRegular,
		}, RegularRounds, defaultTurnDeadline, nil
	default:
		return catalog.GameMetadata{}, 0, 0, fmt.Errorf("janken: unsupported ruleset %q", ruleset)
	}
}

func (m *Master) Init(context.Context) (game.InitState, error) {
	state := mustRaw(initState{
		Players: append([]string(nil), m.playerIDs...),
		Rounds:  m.rounds,
	})
	perPlayer := make(map[string]json.RawMessage, len(m.players))
	for _, player := range m.players {
		perPlayer[player.PlayerID] = state
	}
	return game.InitState{PerPlayer: perPlayer}, nil
}

func (m *Master) NextStep(context.Context) (*game.DecisionStep, error) {
	if m.resolved >= m.rounds {
		return nil, nil
	}
	round := m.resolved + 1
	requests := make([]game.DecisionRequest, 0, len(m.players))
	for _, player := range m.players {
		requests = append(requests, game.DecisionRequest{
			PlayerID:        player.PlayerID,
			VisibleState:    m.VisibleState(player.PlayerID),
			LegalActionHint: legalActionHint,
			Deadline:        m.deadline,
		})
	}
	return &game.DecisionStep{
		Turn:     round,
		Mode:     game.Simultaneous,
		Requests: requests,
	}, nil
}

func (m *Master) NormalizeAction(req game.DecisionRequest, actionStatus game.ActionStatus) game.ActionStatus {
	if actionStatus.ActionStatus != session.StatusAccepted {
		actionStatus.Action = nil
		return actionStatus
	}

	var act action
	if err := json.Unmarshal(actionStatus.Action, &act); err != nil || !isPlayableAction(act.Action) {
		return game.ActionStatus{
			PlayerID:      req.PlayerID,
			ActionStatus:  session.StatusNoAction,
			FailureReason: contract.ReasonIllegalAction,
		}
	}

	actionStatus.Action = mustRaw(action{Action: act.Action})
	return actionStatus
}

func (m *Master) ApplyStep(_ context.Context, step game.DecisionStep, actionStatuses []game.ActionStatus) error {
	if step.Mode != game.Simultaneous {
		return fmt.Errorf("janken: unsupported mode %q", step.Mode)
	}
	if len(actionStatuses) != len(m.players) {
		return fmt.Errorf("janken: action count %d does not match players %d", len(actionStatuses), len(m.players))
	}

	actionsByPlayer := make(map[string]string, len(m.players))
	outcomesByPlayer := make(map[string]string, len(m.players))
	validActions := make(map[string]string, len(m.players))

	for _, status := range actionStatuses {
		m.lastAction[status.PlayerID] = status

		resolvedAction := "no_action"
		if status.ActionStatus == session.StatusAccepted {
			var act action
			if err := json.Unmarshal(status.Action, &act); err == nil && isPlayableAction(act.Action) {
				resolvedAction = act.Action
				validActions[status.PlayerID] = act.Action
			}
		}
		actionsByPlayer[status.PlayerID] = resolvedAction
		if resolvedAction == "no_action" {
			outcomesByPlayer[status.PlayerID] = "loss"
		}

		current := m.scores[status.PlayerID]
		switch status.FailureReason {
		case session.ReasonTimeout:
			current.Timeouts++
		case contract.ReasonIllegalAction:
			current.InvalidActions++
		}
		m.scores[status.PlayerID] = current
	}

	for playerID, outcome := range resolveValidOutcomes(validActions) {
		outcomesByPlayer[playerID] = outcome
	}

	for _, player := range m.players {
		playerID := player.PlayerID
		current := m.scores[playerID]
		switch outcomesByPlayer[playerID] {
		case "win":
			current.Wins++
		case "draw":
			current.Draws++
		default:
			current.Losses++
		}
		m.scores[playerID] = current
		m.selfHistory[playerID] = append(m.selfHistory[playerID], selfRound{
			Round:  step.Turn,
			Action: actionsByPlayer[playerID],
			Result: outcomesByPlayer[playerID],
		})
	}

	m.publicHistory = append(m.publicHistory, publicRound{
		Round:   step.Turn,
		Actions: cloneActionMap(actionsByPlayer),
		Results: cloneActionMap(outcomesByPlayer),
	})
	m.resolved++
	return nil
}

func (m *Master) VisibleState(playerID string) json.RawMessage {
	return mustRaw(visibleState{
		Round:         m.visibleRound(),
		Rounds:        m.rounds,
		SelfHistory:   cloneSelfHistory(m.selfHistory[playerID]),
		PublicHistory: clonePublicHistory(m.publicHistory),
	})
}

func (m *Master) Snapshot() game.Snapshot {
	return game.Snapshot{
		GameID:         m.meta.GameID,
		GameVersion:    m.meta.GameVersion,
		RulesetVersion: m.meta.RulesetVersion,
		Turn:           m.resolved,
		Status:         game.StatusRunning,
		GameState: mustRaw(snapshotState{
			Rounds:         m.rounds,
			ResolvedRounds: m.resolved,
			Scores:         cloneScores(m.scores),
			SelfHistory:    cloneAllSelfHistory(m.selfHistory),
			PublicHistory:  clonePublicHistory(m.publicHistory),
		}),
	}
}

func (m *Master) ExportedSnapshot() game.ExportedSnapshot {
	exported := game.ExportedSnapshot{
		GameID:         m.meta.GameID,
		GameVersion:    m.meta.GameVersion,
		RulesetVersion: m.meta.RulesetVersion,
		Turn:           m.resolved,
		Status:         game.StatusRunning,
		PublicState: mustRaw(publicState{
			Rounds:         m.rounds,
			ResolvedRounds: m.resolved,
			Scores:         cloneScores(m.scores),
			PublicHistory:  clonePublicHistory(m.publicHistory),
		}),
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
	ordered := append([]string(nil), m.playerIDs...)
	sort.Slice(ordered, func(i, j int) bool {
		left := m.scores[ordered[i]]
		right := m.scores[ordered[j]]
		if left.Wins != right.Wins {
			return left.Wins > right.Wins
		}
		if left.Losses != right.Losses {
			return left.Losses < right.Losses
		}
		if left.Timeouts != right.Timeouts {
			return left.Timeouts < right.Timeouts
		}
		if left.InvalidActions != right.InvalidActions {
			return left.InvalidActions < right.InvalidActions
		}
		return ordered[i] < ordered[j]
	})

	placements := make([]game.Placement, 0, len(ordered))
	lastPlace := 0
	var previous score
	for i, playerID := range ordered {
		place := i + 1
		current := m.scores[playerID]
		if i > 0 && scoreEqualForPlacement(current, previous) {
			place = lastPlace
		}
		placements = append(placements, game.Placement{PlayerID: playerID, Place: place})
		previous = current
		lastPlace = place
	}
	return game.MatchResult{Placements: placements}
}

func (m *Master) applySnapshot(snapshot game.Snapshot) error {
	if snapshot.GameID != "" && snapshot.GameID != m.meta.GameID {
		return fmt.Errorf("janken: snapshot game id %q does not match %q", snapshot.GameID, m.meta.GameID)
	}
	if snapshot.GameVersion != "" && snapshot.GameVersion != m.meta.GameVersion {
		return fmt.Errorf("janken: snapshot game version %q does not match %q", snapshot.GameVersion, m.meta.GameVersion)
	}
	if snapshot.RulesetVersion != "" && snapshot.RulesetVersion != m.meta.RulesetVersion {
		return fmt.Errorf("janken: snapshot ruleset %q does not match %q", snapshot.RulesetVersion, m.meta.RulesetVersion)
	}
	if snapshot.Turn < 0 || snapshot.Turn > m.rounds {
		return fmt.Errorf("janken: snapshot turn %d out of range 0..%d", snapshot.Turn, m.rounds)
	}

	var state snapshotState
	if err := json.Unmarshal(snapshot.GameState, &state); err != nil {
		return fmt.Errorf("janken: decode snapshot game_state: %w", err)
	}

	m.resolved = snapshot.Turn
	for _, playerID := range m.playerIDs {
		m.scores[playerID] = score{}
		m.selfHistory[playerID] = nil
		m.lastAction[playerID] = game.ActionStatus{PlayerID: playerID, ActionStatus: session.StatusNoAction}
	}

	for playerID, scoreValue := range state.Scores {
		if _, ok := m.scores[playerID]; !ok {
			return fmt.Errorf("janken: snapshot score has unknown player %q", playerID)
		}
		m.scores[playerID] = scoreValue
	}
	for playerID, history := range state.SelfHistory {
		if _, ok := m.selfHistory[playerID]; !ok {
			return fmt.Errorf("janken: snapshot self_history has unknown player %q", playerID)
		}
		m.selfHistory[playerID] = cloneSelfHistory(history)
	}
	m.publicHistory = clonePublicHistory(state.PublicHistory)

	for playerID, playerState := range snapshot.PerPlayer {
		if _, ok := m.lastAction[playerID]; !ok {
			return fmt.Errorf("janken: snapshot per_player has unknown player %q", playerID)
		}
		status := playerState.LastActionStatus
		if status.PlayerID == "" {
			status.PlayerID = playerID
		}
		if status.ActionStatus != "" {
			m.lastAction[playerID] = status
		}
	}
	return nil
}

func resolveValidOutcomes(validActions map[string]string) map[string]string {
	outcomes := make(map[string]string, len(validActions))
	choices := make(map[string]struct{}, 3)
	for _, choice := range validActions {
		choices[choice] = struct{}{}
	}
	if len(validActions) == 0 {
		return outcomes
	}
	if len(choices) == 1 || len(choices) == 3 {
		for playerID := range validActions {
			outcomes[playerID] = "draw"
		}
		return outcomes
	}

	winning := winningActionForChoices(choices)
	for playerID, choice := range validActions {
		if choice == winning {
			outcomes[playerID] = "win"
			continue
		}
		outcomes[playerID] = "loss"
	}
	return outcomes
}

func winningActionForChoices(choices map[string]struct{}) string {
	if _, ok := choices["rock"]; ok {
		if _, ok := choices["scissors"]; ok {
			return "rock"
		}
	}
	if _, ok := choices["scissors"]; ok {
		if _, ok := choices["paper"]; ok {
			return "scissors"
		}
	}
	return "paper"
}

func scoreEqualForPlacement(left, right score) bool {
	return left.Wins == right.Wins &&
		left.Losses == right.Losses &&
		left.Timeouts == right.Timeouts &&
		left.InvalidActions == right.InvalidActions
}

func isPlayableAction(value string) bool {
	return value == "rock" || value == "paper" || value == "scissors"
}

func (m *Master) visibleRound() int {
	if m.resolved >= m.rounds {
		return m.rounds
	}
	return m.resolved + 1
}

func clonePublicHistory(src []publicRound) []publicRound {
	cloned := make([]publicRound, 0, len(src))
	for _, round := range src {
		cloned = append(cloned, publicRound{
			Round:   round.Round,
			Actions: cloneActionMap(round.Actions),
			Results: cloneActionMap(round.Results),
		})
	}
	return cloned
}

func cloneSelfHistory(src []selfRound) []selfRound {
	cloned := make([]selfRound, len(src))
	copy(cloned, src)
	return cloned
}

func cloneAllSelfHistory(src map[string][]selfRound) map[string][]selfRound {
	cloned := make(map[string][]selfRound, len(src))
	for key, value := range src {
		cloned[key] = cloneSelfHistory(value)
	}
	return cloned
}

func cloneScores(src map[string]score) map[string]score {
	cloned := make(map[string]score, len(src))
	for key, value := range src {
		cloned[key] = value
	}
	return cloned
}

func cloneActionMap(src map[string]string) map[string]string {
	cloned := make(map[string]string, len(src))
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
