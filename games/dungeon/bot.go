package dungeon

import (
	"fmt"
	"sort"
)

// Bot is a deterministic exploration bot for the dungeon visible-state API.
type Bot struct {
	knownTiles    map[string]string
	knownGoal     *Position
	knownChests   map[string]Position
	exploreTarget *Position
}

// NewBot creates a bot that only depends on the public dungeon package API.
func NewBot() *Bot {
	return &Bot{
		knownTiles:  make(map[string]string),
		knownChests: make(map[string]Position),
	}
}

// Decide chooses the next action from the player's visible state.
func (b *Bot) Decide(state VisibleState) Action {
	b.observe(state)
	start := state.Self.position()

	if step, ok := b.stepTowardAny(start, positionsFromMap(b.knownChests)); ok {
		b.exploreTarget = nil
		return step
	}
	if b.knownGoal != nil {
		if step, ok := b.stepTowardAny(start, []Position{*b.knownGoal}); ok {
			b.exploreTarget = nil
			return step
		}
	}
	if step, ok := b.stepTowardExploreTarget(start); ok {
		return step
	}
	if step, ok := b.chooseFrontierTarget(start); ok {
		return step
	}
	return Action{Action: "wait"}
}

func (b *Bot) observe(state VisibleState) {
	for _, tile := range state.VisibleTiles {
		pos := Position{X: tile.X, Y: tile.Y}
		b.knownTiles[posKey(pos)] = tile.Tile
		if tile.Tile == TileGoal {
			goal := pos
			b.knownGoal = &goal
		}
		if tile.Tile == TileChest {
			b.knownChests[posKey(pos)] = pos
		}
	}
	if state.KnownGoal != nil {
		goal := *state.KnownGoal
		b.knownGoal = &goal
	}
	currentKnown := make(map[string]struct{}, len(state.KnownChests))
	for _, chest := range state.KnownChests {
		key := posKey(chest)
		currentKnown[key] = struct{}{}
		b.knownChests[key] = chest
	}
	for key := range b.knownChests {
		if _, ok := currentKnown[key]; !ok {
			delete(b.knownChests, key)
		}
	}
}

func (b *Bot) stepTowardAny(start Position, targets []Position) (Action, bool) {
	bestPath := []Position(nil)
	for _, target := range targets {
		path, ok := b.shortestKnownPath(start, target)
		if !ok || len(path) < 2 {
			continue
		}
		if bestPath == nil || len(path) < len(bestPath) || comparePath(path, bestPath) < 0 {
			bestPath = path
		}
	}
	if len(bestPath) < 2 {
		return Action{}, false
	}
	return directionAction(start, bestPath[1]), true
}

func (b *Bot) stepTowardExploreTarget(start Position) (Action, bool) {
	if b.exploreTarget == nil || !b.isFrontier(*b.exploreTarget) {
		b.exploreTarget = nil
		return Action{}, false
	}
	if step, ok := b.stepTowardAny(start, []Position{*b.exploreTarget}); ok {
		return step, true
	}
	b.exploreTarget = nil
	return Action{}, false
}

func (b *Bot) chooseFrontierTarget(start Position) (Action, bool) {
	type candidate struct {
		target Position
		path   []Position
	}
	candidates := make([]candidate, 0)
	for key, tile := range b.knownTiles {
		if tile == TileWall {
			continue
		}
		pos := parsePosKey(key)
		if b.isFrontier(pos) {
			path, ok := b.shortestKnownPath(start, pos)
			if !ok || len(path) < 2 {
				continue
			}
			candidates = append(candidates, candidate{target: pos, path: path})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].target.Y != candidates[j].target.Y {
			return candidates[i].target.Y > candidates[j].target.Y
		}
		if candidates[i].target.X != candidates[j].target.X {
			return candidates[i].target.X > candidates[j].target.X
		}
		if len(candidates[i].path) != len(candidates[j].path) {
			return len(candidates[i].path) < len(candidates[j].path)
		}
		return comparePath(candidates[i].path, candidates[j].path) < 0
	})
	if len(candidates) == 0 {
		return Action{}, false
	}
	target := candidates[0].target
	b.exploreTarget = &target
	return directionAction(start, candidates[0].path[1]), true
}

func (b *Bot) shortestKnownPath(from, to Position) ([]Position, bool) {
	if from == to {
		return []Position{from}, true
	}
	queue := []Position{from}
	prev := map[string]Position{}
	seen := map[string]struct{}{posKey(from): {}}
	directions := []string{"up", "left", "right", "down"}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, direction := range directions {
			next, _ := step(current, direction)
			tile, ok := b.knownTiles[posKey(next)]
			if !ok || tile == TileWall {
				continue
			}
			key := posKey(next)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			prev[key] = current
			if next == to {
				path := []Position{to}
				cursor := to
				for cursor != from {
					cursor = prev[posKey(cursor)]
					path = append(path, cursor)
				}
				reversePositions(path)
				return path, true
			}
			queue = append(queue, next)
		}
	}
	return nil, false
}

func (b *Bot) isFrontier(pos Position) bool {
	directions := []string{"up", "down", "left", "right"}
	for _, direction := range directions {
		next, _ := step(pos, direction)
		if _, ok := b.knownTiles[posKey(next)]; !ok {
			return true
		}
	}
	return false
}

func directionAction(from, to Position) Action {
	switch {
	case to.X == from.X && to.Y == from.Y-1:
		return Action{Action: "move", Direction: "up"}
	case to.X == from.X && to.Y == from.Y+1:
		return Action{Action: "move", Direction: "down"}
	case to.X == from.X-1 && to.Y == from.Y:
		return Action{Action: "move", Direction: "left"}
	default:
		return Action{Action: "move", Direction: "right"}
	}
}

func comparePath(a, b []Position) int {
	if len(a) != len(b) {
		return len(a) - len(b)
	}
	for i := range a {
		if a[i].Y != b[i].Y {
			return a[i].Y - b[i].Y
		}
		if a[i].X != b[i].X {
			return a[i].X - b[i].X
		}
	}
	return 0
}

func parsePosKey(key string) Position {
	var pos Position
	_, _ = fmt.Sscanf(key, "%d,%d", &pos.X, &pos.Y)
	return pos
}
