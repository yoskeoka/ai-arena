package echo

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/contract"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
	"github.com/yoskeoka/ai-arena/internal/platform/session"
)

const (
	// GameID is the canonical id for the echo-count game family.
	GameID = "echo-count"
	// SubprocessGameID is the game id used for the subprocess-hosted variant.
	SubprocessGameID = "echo-count-subprocess"
	// BuilderIDInProcess is the registry builder id for the in-process echo game.
	BuilderIDInProcess = "echo-count/in-process"
	// BuilderIDLocalSubprocess is the registry builder id for the subprocess echo game.
	BuilderIDLocalSubprocess = "echo-count-subprocess/local-subprocess"
	// GameVersion is the supported echo game version.
	GameVersion = "2.0.0"
	// RulesetSimultaneous3Turn runs three simultaneous turns.
	RulesetSimultaneous3Turn = "phase2-simultaneous-3turn"
	// RulesetSequential3Turn runs three sequential turns.
	RulesetSequential3Turn = "phase2-sequential-3turn"
	// RulesetSimultaneous2Turn runs two simultaneous turns.
	RulesetSimultaneous2Turn = "phase2-simultaneous-2turn"
	// RulesetSimultaneousShuffle3Turn runs three simultaneous turns with a seed-aware expected order.
	RulesetSimultaneousShuffle3Turn = "phase2-simultaneous-shuffle-3turn"
	defaultTurnDeadline             = 100 * time.Millisecond
)

// Config selects the echo game variant and participating players.
type Config struct {
	GameID      string
	GameVersion string
	Ruleset     string
	RNGSeed     string
	Players     []game.Player
}

// Master executes one echo-count match.
type Master struct {
	meta       catalog.GameMetadata
	players    []game.Player
	playerIDs  []string
	mode       game.DecisionMode
	turns      int
	deadline   time.Duration
	rngSeed    string
	seedAware  bool
	order      []int
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
	Mode          game.DecisionMode `json:"mode"`
	Turn          int               `json:"turn"`
	Expected      int               `json:"expected"`
	Score         map[string]int    `json:"score"`
	RNGSeed       string            `json:"rng_seed,omitempty"`
	ExpectedOrder []int             `json:"expected_order,omitempty"`
}

type selection struct {
	meta      catalog.GameMetadata
	mode      game.DecisionMode
	turns     int
	deadline  time.Duration
	seedAware bool
}

// New builds a fresh echo-count master.
func New(cfg Config) (*Master, error) {
	if len(cfg.Players) == 0 {
		return nil, fmt.Errorf("echo: at least one player is required")
	}
	if cfg.GameVersion == "" {
		return nil, fmt.Errorf("echo: game version is required")
	}

	selected, err := selectionFor(cfg.GameID, cfg.GameVersion, cfg.Ruleset)
	if err != nil {
		return nil, err
	}
	if selected.seedAware && cfg.RNGSeed == "" {
		return nil, fmt.Errorf("echo: rng_seed is required for ruleset %q", cfg.Ruleset)
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
		meta:       selected.meta,
		players:    append([]game.Player(nil), cfg.Players...),
		playerIDs:  playerIDs,
		mode:       selected.mode,
		turns:      selected.turns,
		deadline:   selected.deadline,
		rngSeed:    cfg.RNGSeed,
		seedAware:  selected.seedAware,
		order:      expectedOrder(selected.turns, cfg.RNGSeed, selected.seedAware),
		score:      score,
		lastAction: lastAction,
	}, nil
}

// NewFromSnapshot rebuilds an echo-count master from a persisted snapshot.
func NewFromSnapshot(cfg Config, snapshot game.Snapshot) (*Master, error) {
	if cfg.RNGSeed == "" {
		var state snapshotState
		if err := json.Unmarshal(snapshot.GameState, &state); err == nil {
			cfg.RNGSeed = state.RNGSeed
		}
	}
	master, err := New(cfg)
	if err != nil {
		return nil, err
	}
	if err := master.applySnapshot(snapshot); err != nil {
		return nil, err
	}
	return master, nil
}

// Metadata returns the selected game metadata.
func (m *Master) Metadata() catalog.GameMetadata {
	return m.meta
}

// SnapshotFromHistory rebuilds an echo snapshot from event history.
func SnapshotFromHistory(gameVersion, ruleset, rngSeed string, players []game.Player, events []match.Event, targetTurn int) (game.Snapshot, error) {
	return SnapshotFromHistoryWithGameID(GameID, gameVersion, ruleset, rngSeed, players, events, targetTurn)
}

// SnapshotFromHistoryWithGameID rebuilds an echo snapshot for an explicit game id.
func SnapshotFromHistoryWithGameID(gameID, gameVersion, ruleset, rngSeed string, players []game.Player, events []match.Event, targetTurn int) (game.Snapshot, error) {
	selected, err := selectionFor(gameID, gameVersion, ruleset)
	if err != nil {
		return game.Snapshot{}, err
	}
	if selected.seedAware && rngSeed == "" {
		return game.Snapshot{}, fmt.Errorf("echo: rng_seed is required for ruleset %q", ruleset)
	}
	if targetTurn < 0 || targetTurn > selected.turns {
		return game.Snapshot{}, fmt.Errorf("target turn %d out of range 0..%d", targetTurn, selected.turns)
	}
	order := expectedOrder(selected.turns, rngSeed, selected.seedAware)

	score := make(map[string]int, len(players))
	perPlayer := make(map[string]game.PlayerSnapshot, len(players))
	for _, player := range players {
		score[player.PlayerID] = 0
		perPlayer[player.PlayerID] = game.PlayerSnapshot{
			LastActionStatus: game.ActionStatus{
				PlayerID:     player.PlayerID,
				ActionStatus: session.StatusNoAction,
			},
		}
	}

	for _, event := range events {
		if event.Turn == 0 || event.Turn > targetTurn {
			continue
		}
		switch event.Kind {
		case "turn_result", "turn_timeout", "protocol_error", "runtime_exited":
			var actionStatus game.ActionStatus
			if err := json.Unmarshal(event.Payload, &actionStatus); err != nil {
				return game.Snapshot{}, fmt.Errorf(
					"decode history event payload seq=%d kind=%q turn=%d player_id=%q: %w",
					event.Seq,
					event.Kind,
					event.Turn,
					event.PlayerID,
					err,
				)
			}
			if _, ok := perPlayer[event.PlayerID]; !ok {
				return game.Snapshot{}, fmt.Errorf("history has unknown player %q", event.PlayerID)
			}
			if actionStatus.PlayerID == "" {
				actionStatus.PlayerID = event.PlayerID
			}
			playerState := perPlayer[event.PlayerID]
			playerState.LastActionStatus = actionStatus
			perPlayer[event.PlayerID] = playerState
			if actionStatus.ActionStatus == session.StatusAccepted {
				score[event.PlayerID]++
			}
		}
	}

	expected := expectedForResolvedCount(order, targetTurn)
	for _, player := range players {
		playerState := perPlayer[player.PlayerID]
		playerState.VisibleState = mustRaw(visibleState{
			Turn:     expected,
			Expected: expected,
			Score:    score,
		})
		perPlayer[player.PlayerID] = playerState
	}
	state := snapshotState{
		Mode:     selected.mode,
		Turn:     targetTurn,
		Expected: expected,
		Score:    score,
	}
	if selected.seedAware {
		state.RNGSeed = rngSeed
		state.ExpectedOrder = append([]int(nil), order...)
	}
	gameState, err := json.Marshal(state)
	if err != nil {
		return game.Snapshot{}, fmt.Errorf("encode replay snapshot: %w", err)
	}

	return game.Snapshot{
		GameID:         selected.meta.GameID,
		GameVersion:    selected.meta.GameVersion,
		RulesetVersion: selected.meta.RulesetVersion,
		Turn:           targetTurn,
		Status:         game.StatusRunning,
		GameState:      gameState,
		PerPlayer:      perPlayer,
	}, nil
}

// MetadataForSelection resolves metadata and turn settings for the default game id.
func MetadataForSelection(gameVersion, ruleset string) (catalog.GameMetadata, game.DecisionMode, int, time.Duration, error) {
	return MetadataForSelectionWithGameID(GameID, gameVersion, ruleset)
}

// SupportedRulesets lists the supported echo rulesets.
func SupportedRulesets() []string {
	return []string{
		RulesetSimultaneous3Turn,
		RulesetSequential3Turn,
		RulesetSimultaneous2Turn,
		RulesetSimultaneousShuffle3Turn,
	}
}

// MetadataForSelectionWithGameID resolves metadata and turn settings for one game id.
func MetadataForSelectionWithGameID(gameID, gameVersion, ruleset string) (catalog.GameMetadata, game.DecisionMode, int, time.Duration, error) {
	selected, err := selectionFor(gameID, gameVersion, ruleset)
	if err != nil {
		return catalog.GameMetadata{}, "", 0, 0, err
	}
	return selected.meta, selected.mode, selected.turns, selected.deadline, nil
}

// Init returns the per-player initialization payloads for the match.
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

// NextStep returns the next decision step or nil when the match is complete.
func (m *Master) NextStep(context.Context) (*game.DecisionStep, error) {
	if m.resolved >= m.turns {
		return nil, nil
	}

	turn := m.resolved + 1
	expected := m.expectedForResolvedCount(m.resolved)
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

// NormalizeAction validates one player action against the expected echo value.
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

// ApplyStep commits one resolved decision step into match state.
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

// Snapshot returns the internal replay/resume snapshot.
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

// VisibleState returns the visible state sent to a player.
func (m *Master) VisibleState(string) json.RawMessage {
	turn := m.resolved + 1
	if turn > m.turns {
		turn = m.turns
	}
	expected := m.expectedForResolvedCount(m.resolved)
	return mustRaw(m.currentVisibleState(turn, expected))
}

// ExportedSnapshot returns the public snapshot safe to expose externally.
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

// Result returns the final placement summary implied by the current score state.
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
	state := map[string]any{
		"mode":     m.mode,
		"turn":     m.resolved,
		"expected": m.expectedForResolvedCount(m.resolved),
		"score":    cloneScore(m.score),
	}
	if m.seedAware {
		state["rng_seed"] = m.rngSeed
		state["expected_order"] = append([]int(nil), m.order...)
	}
	return state
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
	if m.seedAware {
		if state.RNGSeed == "" {
			return fmt.Errorf("echo: snapshot rng_seed is required for ruleset %q", m.meta.RulesetVersion)
		}
		if m.rngSeed != "" && state.RNGSeed != m.rngSeed {
			return fmt.Errorf("echo: snapshot rng_seed %q does not match %q", state.RNGSeed, m.rngSeed)
		}
		m.rngSeed = state.RNGSeed
		expected := expectedOrder(m.turns, m.rngSeed, true)
		if len(state.ExpectedOrder) > 0 {
			if !sameIntSlice(state.ExpectedOrder, expected) {
				return fmt.Errorf("echo: snapshot expected_order %v does not match seed-derived order %v", state.ExpectedOrder, expected)
			}
		}
		m.order = expected
	}
	if state.Expected > 0 {
		expected := m.expectedForResolvedCount(snapshot.Turn)
		if state.Expected != expected {
			return fmt.Errorf("echo: snapshot expected %d does not match %d", state.Expected, expected)
		}
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
	return m.expectedForResolvedCount(m.resolved)
}

func (m *Master) expectedForResolvedCount(resolved int) int {
	return expectedForResolvedCount(m.order, resolved)
}

func selectionFor(gameID, gameVersion, ruleset string) (selection, error) {
	if gameVersion != GameVersion {
		return selection{}, fmt.Errorf("echo: unsupported game version %q", gameVersion)
	}
	if gameID == "" {
		gameID = GameID
	}

	newSelection := func(ruleset string, mode game.DecisionMode, turns int, seedAware bool) selection {
		return selection{
			meta: catalog.GameMetadata{
				GameID:         gameID,
				GameVersion:    gameVersion,
				RulesetVersion: ruleset,
			},
			mode:      mode,
			turns:     turns,
			deadline:  defaultTurnDeadline,
			seedAware: seedAware,
		}
	}

	switch ruleset {
	case RulesetSimultaneous3Turn:
		return newSelection(RulesetSimultaneous3Turn, game.Simultaneous, 3, false), nil
	case RulesetSequential3Turn:
		return newSelection(RulesetSequential3Turn, game.Sequential, 3, false), nil
	case RulesetSimultaneous2Turn:
		return newSelection(RulesetSimultaneous2Turn, game.Simultaneous, 2, false), nil
	case RulesetSimultaneousShuffle3Turn:
		return newSelection(RulesetSimultaneousShuffle3Turn, game.Simultaneous, 3, true), nil
	default:
		return selection{}, fmt.Errorf("echo: unsupported ruleset %q", ruleset)
	}
}

func expectedOrder(turns int, rngSeed string, seedAware bool) []int {
	order := make([]int, turns)
	for i := range turns {
		order[i] = i + 1
	}
	if !seedAware {
		return order
	}
	sort.Slice(order, func(i, j int) bool {
		left := turnRank(rngSeed, order[i])
		right := turnRank(rngSeed, order[j])
		return left < right
	})
	return order
}

func turnRank(rngSeed string, turn int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", rngSeed, turn)))
	return string(sum[:])
}

func expectedForResolvedCount(order []int, resolved int) int {
	if len(order) == 0 {
		return 0
	}
	if resolved < len(order) {
		return order[resolved]
	}
	return order[len(order)-1]
}

func sameIntSlice(left, right []int) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
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
