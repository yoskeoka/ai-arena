package dungeon

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type matchState struct {
	turn              int
	playerStates      map[string]PlayerState
	uncollectedChests map[string]ChestState
	discoveredGoal    map[string]*Position
	discoveredChests  map[string]map[string]ChestState
}

func buildInitialState(layout GeneratedLayout, playerIDs []string) (matchState, []string, error) {
	if len(playerIDs) < 2 || len(playerIDs) > len(layout.SpawnPoints) {
		return matchState{}, nil, fmt.Errorf("dungeon: player count must be between 2 and %d", len(layout.SpawnPoints))
	}

	playerStates := make(map[string]PlayerState, len(playerIDs))
	playerOrder := make([]string, 0, len(playerIDs))
	for i, playerID := range playerIDs {
		if strings.TrimSpace(playerID) == "" {
			return matchState{}, nil, fmt.Errorf("dungeon: player_id is required")
		}
		if _, exists := playerStates[playerID]; exists {
			return matchState{}, nil, fmt.Errorf("dungeon: duplicate player_id %q", playerID)
		}
		spawn := layout.SpawnPoints[i]
		playerStates[playerID] = PlayerState{
			PlayerID: playerID,
			X:        spawn.X,
			Y:        spawn.Y,
		}
		playerOrder = append(playerOrder, playerID)
	}

	state := matchState{
		playerStates:      playerStates,
		uncollectedChests: make(map[string]ChestState, len(layout.InitialChests)),
		discoveredGoal:    make(map[string]*Position, len(playerOrder)),
		discoveredChests:  make(map[string]map[string]ChestState, len(playerOrder)),
	}
	for _, chest := range layout.InitialChests {
		state.uncollectedChests[chestKey(chest)] = chest
	}
	for _, playerID := range playerOrder {
		state.discoveredChests[playerID] = make(map[string]ChestState)
	}
	return state, playerOrder, nil
}

func restoreMatchState(cfg Config, ruleset Ruleset, layout GeneratedLayout, state FullState, rngSeed string) (matchState, []string, error) {
	resumed, playerOrder, err := buildInitialState(layout, append([]string(nil), cfg.PlayerIDs...))
	if err != nil {
		return matchState{}, nil, err
	}
	if state.MapID != ruleset.MapID {
		return matchState{}, nil, fmt.Errorf("dungeon: snapshot map_id %q does not match %q", state.MapID, ruleset.MapID)
	}
	if state.MaxTurns != ruleset.MaxTurns {
		return matchState{}, nil, fmt.Errorf("dungeon: snapshot max_turns %d does not match ruleset %d", state.MaxTurns, ruleset.MaxTurns)
	}
	if state.Turn < 0 || state.Turn > ruleset.MaxTurns {
		return matchState{}, nil, fmt.Errorf("dungeon: snapshot turn %d out of range", state.Turn)
	}
	stateSeed, err := normalizeSeed(state.RNGSeed)
	if err != nil {
		return matchState{}, nil, fmt.Errorf("dungeon: invalid snapshot rng_seed: %w", err)
	}
	if stateSeed != rngSeed {
		return matchState{}, nil, fmt.Errorf("dungeon: snapshot rng_seed %q does not match config %q", state.RNGSeed, rngSeed)
	}
	if !equalStringSlices(state.Tiles, layout.Tiles) {
		return matchState{}, nil, fmt.Errorf("dungeon: snapshot tiles do not match generated layout")
	}
	if !equalPositions(state.SpawnPoints, layout.SpawnPoints) {
		return matchState{}, nil, fmt.Errorf("dungeon: snapshot spawn_points do not match generated layout")
	}
	if state.Goal != layout.Goal {
		return matchState{}, nil, fmt.Errorf("dungeon: snapshot goal does not match generated layout")
	}
	if !equalChests(state.InitialChests, layout.InitialChests) {
		return matchState{}, nil, fmt.Errorf("dungeon: snapshot initial_chests do not match generated layout")
	}

	resumed.turn = state.Turn
	seenPlayers := make(map[string]struct{}, len(state.Players))
	for _, player := range state.Players {
		if _, ok := resumed.playerStates[player.PlayerID]; !ok {
			return matchState{}, nil, fmt.Errorf("dungeon: snapshot has unknown player %q", player.PlayerID)
		}
		if !resumed.isWalkable(layout, Position{X: player.X, Y: player.Y}) {
			return matchState{}, nil, fmt.Errorf("dungeon: snapshot player %q has invalid position (%d,%d)", player.PlayerID, player.X, player.Y)
		}
		resumed.playerStates[player.PlayerID] = clonePlayerState(player)
		seenPlayers[player.PlayerID] = struct{}{}
	}
	if len(seenPlayers) != len(playerOrder) {
		return matchState{}, nil, fmt.Errorf("dungeon: snapshot player count does not match config")
	}

	resumed.uncollectedChests = make(map[string]ChestState, len(state.UncollectedChests))
	for _, chest := range state.UncollectedChests {
		if !resumed.isOriginalChest(layout, chest) {
			return matchState{}, nil, fmt.Errorf("dungeon: snapshot chest at (%d,%d) is not in generated layout", chest.X, chest.Y)
		}
		resumed.uncollectedChests[chestKey(chest)] = chest
	}

	resumed.discoveredGoal = make(map[string]*Position, len(playerOrder))
	resumed.discoveredChests = make(map[string]map[string]ChestState, len(playerOrder))
	for _, playerID := range playerOrder {
		resumed.discoveredChests[playerID] = make(map[string]ChestState)
		discovery := state.Discovery[playerID]
		if discovery.KnownGoal != nil {
			pos := *discovery.KnownGoal
			resumed.discoveredGoal[playerID] = &pos
		}
		for _, chest := range discovery.KnownChests {
			if original, ok := resumed.uncollectedChests[chestKey(chest)]; ok {
				resumed.discoveredChests[playerID][chestKey(chest)] = original
			}
		}
	}
	return resumed, playerOrder, nil
}

func (s *matchState) visibleState(ruleset Ruleset, layout GeneratedLayout, playerOrder []string, playerID string) (VisibleState, error) {
	player, ok := s.playerStates[playerID]
	if !ok {
		return VisibleState{}, fmt.Errorf("dungeon: unknown player %q", playerID)
	}
	turn := s.turn + 1
	remainingTurns := ruleset.MaxTurns - s.turn
	if s.terminal(ruleset, playerOrder) {
		turn = s.turn
		remainingTurns = 0
	}
	return VisibleState{
		Turn:           turn,
		RemainingTurns: remainingTurns,
		ViewRadius:     ruleset.ViewRadius,
		Self:           clonePlayerState(player),
		VisibleTiles:   s.visibleTiles(layout, player.position(), ruleset.ViewRadius),
		KnownGoal:      clonePositionPtr(s.discoveredGoal[playerID]),
		KnownChests:    chestsFromMap(s.discoveredChests[playerID]),
		Scores:         s.scoreboard(playerOrder),
	}, nil
}

func (s *matchState) fullState(ruleset Ruleset, layout GeneratedLayout, playerOrder []string, rngSeed string) FullState {
	discovery := make(map[string]DiscoveryState, len(playerOrder))
	for _, playerID := range playerOrder {
		discovery[playerID] = DiscoveryState{
			KnownGoal:   clonePositionPtr(s.discoveredGoal[playerID]),
			KnownChests: chestsFromMap(s.discoveredChests[playerID]),
		}
	}
	return FullState{
		MapID:             ruleset.MapID,
		RNGSeed:           rngSeed,
		Turn:              s.turn,
		MaxTurns:          ruleset.MaxTurns,
		Tiles:             append([]string(nil), layout.Tiles...),
		SpawnPoints:       append([]Position(nil), layout.SpawnPoints...),
		Goal:              layout.Goal,
		InitialChests:     append([]ChestState(nil), layout.InitialChests...),
		Players:           s.scoreboardWithPositions(playerOrder),
		UncollectedChests: chestsFromMap(s.uncollectedChests),
		Discovery:         discovery,
	}
}

func (s *matchState) publicState(ruleset Ruleset, layout GeneratedLayout, playerOrder []string, rngSeed string) PublicState {
	return PublicState{
		MapID:             ruleset.MapID,
		RNGSeed:           rngSeed,
		Turn:              s.turn,
		MaxTurns:          ruleset.MaxTurns,
		Tiles:             append([]string(nil), layout.Tiles...),
		SpawnPoints:       append([]Position(nil), layout.SpawnPoints...),
		Goal:              layout.Goal,
		InitialChests:     append([]ChestState(nil), layout.InitialChests...),
		Players:           s.scoreboardWithPositions(playerOrder),
		UncollectedChests: chestsFromMap(s.uncollectedChests),
	}
}

func (s *matchState) terminal(ruleset Ruleset, playerOrder []string) bool {
	return s.turn >= ruleset.MaxTurns || s.allPlayersFinished(playerOrder)
}

func (s *matchState) allPlayersFinished(playerOrder []string) bool {
	for _, playerID := range playerOrder {
		if s.playerStates[playerID].FinishedTurn == nil {
			return false
		}
	}
	return true
}

func (s *matchState) applyGoalBonuses(ruleset Ruleset, playerOrder []string) {
	finished := make([]PlayerState, 0, len(playerOrder))
	for _, playerID := range playerOrder {
		player := s.playerStates[playerID]
		if player.FinishedTurn != nil {
			finished = append(finished, clonePlayerState(player))
		}
	}
	sort.SliceStable(finished, func(i, j int) bool {
		if *finished[i].FinishedTurn != *finished[j].FinishedTurn {
			return *finished[i].FinishedTurn < *finished[j].FinishedTurn
		}
		return finished[i].PlayerID < finished[j].PlayerID
	})

	lastTurn := -1
	lastRank := 0
	for i, player := range finished {
		if i == 0 || *player.FinishedTurn != lastTurn {
			lastRank = i + 1
			lastTurn = *player.FinishedTurn
		}
		bonus := 0
		if lastRank-1 < len(ruleset.GoalBonuses) {
			bonus = ruleset.GoalBonuses[lastRank-1]
		}
		state := s.playerStates[player.PlayerID]
		state.GoalBonus = bonus
		state.Score = state.ChestPoints + state.GoalBonus
		s.playerStates[player.PlayerID] = state
	}
	for _, playerID := range playerOrder {
		state := s.playerStates[playerID]
		if state.FinishedTurn == nil {
			state.GoalBonus = 0
			state.Score = state.ChestPoints
			s.playerStates[playerID] = state
		}
	}
}

func (s *matchState) refreshDiscoveries(ruleset Ruleset, layout GeneratedLayout, playerOrder []string) {
	for _, playerID := range playerOrder {
		state := s.playerStates[playerID]
		visible := s.visibleTiles(layout, state.position(), ruleset.ViewRadius)
		for _, tile := range visible {
			pos := Position{X: tile.X, Y: tile.Y}
			switch tile.Tile {
			case TileGoal:
				goal := pos
				s.discoveredGoal[playerID] = &goal
			case TileChest:
				if chest, ok := s.uncollectedChests[posKey(pos)]; ok {
					s.discoveredChests[playerID][posKey(pos)] = chest
				}
			}
		}
		for key := range s.discoveredChests[playerID] {
			if _, ok := s.uncollectedChests[key]; !ok {
				delete(s.discoveredChests[playerID], key)
			}
		}
	}
}

func (s *matchState) visibleTiles(layout GeneratedLayout, center Position, radius int) []VisibleTile {
	tiles := make([]VisibleTile, 0)
	for y := center.Y - radius; y <= center.Y+radius; y++ {
		for x := center.X - radius; x <= center.X+radius; x++ {
			if !s.inBounds(layout, Position{X: x, Y: y}) {
				continue
			}
			if manhattan(center, Position{X: x, Y: y}) > radius {
				continue
			}
			tiles = append(tiles, VisibleTile{
				X:    x,
				Y:    y,
				Tile: s.tileAt(layout, Position{X: x, Y: y}),
			})
		}
	}
	return tiles
}

func (s *matchState) tileAt(layout GeneratedLayout, pos Position) string {
	if pos == layout.Goal {
		return TileGoal
	}
	if _, ok := s.uncollectedChests[posKey(pos)]; ok {
		return TileChest
	}
	if layout.Tiles[pos.Y][pos.X] == '#' {
		return TileWall
	}
	return TileFloor
}

func (s *matchState) inBounds(layout GeneratedLayout, pos Position) bool {
	return pos.Y >= 0 && pos.Y < len(layout.Tiles) && pos.X >= 0 && pos.X < len(layout.Tiles[pos.Y])
}

func (s *matchState) isWalkable(layout GeneratedLayout, pos Position) bool {
	if !s.inBounds(layout, pos) {
		return false
	}
	return layout.Tiles[pos.Y][pos.X] != '#'
}

func (s *matchState) isOriginalChest(layout GeneratedLayout, chest ChestState) bool {
	for _, original := range layout.InitialChests {
		if original == chest {
			return true
		}
	}
	return false
}

func (s *matchState) scoreboard(playerOrder []string) []PlayerState {
	scores := make([]PlayerState, 0, len(playerOrder))
	for _, playerID := range playerOrder {
		state := s.playerStates[playerID]
		scores = append(scores, clonePlayerState(PlayerState{
			PlayerID:     state.PlayerID,
			Score:        state.Score,
			GoalBonus:    state.GoalBonus,
			ChestPoints:  state.ChestPoints,
			FinishedTurn: cloneIntPtr(state.FinishedTurn),
		}))
	}
	sort.SliceStable(scores, func(i, j int) bool {
		if scores[i].Score != scores[j].Score {
			return scores[i].Score > scores[j].Score
		}
		if finishOrder(scores[i].FinishedTurn) != finishOrder(scores[j].FinishedTurn) {
			return finishOrder(scores[i].FinishedTurn) < finishOrder(scores[j].FinishedTurn)
		}
		return scores[i].PlayerID < scores[j].PlayerID
	})
	return scores
}

func (s *matchState) scoreboardWithPositions(playerOrder []string) []PlayerState {
	players := make([]PlayerState, 0, len(playerOrder))
	for _, playerID := range playerOrder {
		players = append(players, clonePlayerState(s.playerStates[playerID]))
	}
	sort.SliceStable(players, func(i, j int) bool {
		return players[i].PlayerID < players[j].PlayerID
	})
	return players
}

func (p PlayerState) position() Position {
	return Position{X: p.X, Y: p.Y}
}

func clonePlayerState(p PlayerState) PlayerState {
	p.FinishedTurn = cloneIntPtr(p.FinishedTurn)
	return p
}

func cloneIntPtr(v *int) *int {
	if v == nil {
		return nil
	}
	copy := *v
	return &copy
}

func clonePositionPtr(v *Position) *Position {
	if v == nil {
		return nil
	}
	copy := *v
	return &copy
}

func chestsFromMap(values map[string]ChestState) []ChestState {
	chests := make([]ChestState, 0, len(values))
	for _, chest := range values {
		chests = append(chests, chest)
	}
	sort.Slice(chests, func(i, j int) bool {
		if chests[i].Y != chests[j].Y {
			return chests[i].Y < chests[j].Y
		}
		if chests[i].X != chests[j].X {
			return chests[i].X < chests[j].X
		}
		return chests[i].Points < chests[j].Points
	})
	return chests
}

func chestsCopy(values map[string]ChestState) map[string]ChestState {
	cloned := make(map[string]ChestState, len(values))
	for key, chest := range values {
		cloned[key] = chest
	}
	return cloned
}

func posKey(pos Position) string {
	return fmt.Sprintf("%d,%d", pos.X, pos.Y)
}

func chestKey(chest ChestState) string {
	return posKey(Position{X: chest.X, Y: chest.Y})
}

func finishOrder(turn *int) int {
	if turn == nil {
		return 1 << 30
	}
	return *turn
}

func mustJSON(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
