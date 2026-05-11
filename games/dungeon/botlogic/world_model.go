package botlogic

import (
	"fmt"
	"sort"

	"github.com/yoskeoka/ai-arena/games/dungeon"
)

// WorldModel offers path and frontier queries over already observed tiles only.
type WorldModel struct {
	memory *Memory
}

// NewWorldModel returns a world-model backed by shared observed memory.
func NewWorldModel(memory *Memory) *WorldModel {
	return &WorldModel{memory: memory}
}

// GoalPath returns a known path from start to the discovered goal when available.
func (w *WorldModel) GoalPath(start dungeon.Position) ([]dungeon.Position, bool) {
	goal := w.memory.KnownGoal()
	if goal == nil {
		return nil, false
	}
	path, ok := w.ShortestKnownPath(start, *goal)
	if !ok || len(path) < 2 {
		return nil, false
	}
	return path, true
}

// ShortestKnownPath returns a path that only uses already observed non-wall tiles.
func (w *WorldModel) ShortestKnownPath(from, to dungeon.Position) ([]dungeon.Position, bool) {
	if from == to {
		return []dungeon.Position{from}, true
	}
	queue := []dungeon.Position{from}
	prev := map[string]dungeon.Position{}
	seen := map[string]struct{}{posKey(from): {}}
	directions := []string{"up", "left", "right", "down"}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, direction := range directions {
			next, _ := step(current, direction)
			tile, ok := w.memory.knownTiles[posKey(next)]
			if !ok || tile == dungeon.TileWall {
				continue
			}
			key := posKey(next)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			prev[key] = current
			if next == to {
				path := []dungeon.Position{to}
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

// IsFrontier reports whether a known tile touches at least one unknown neighbor.
func (w *WorldModel) IsFrontier(pos dungeon.Position) bool {
	directions := []string{"up", "down", "left", "right"}
	for _, direction := range directions {
		next, _ := step(pos, direction)
		if _, ok := w.memory.knownTiles[posKey(next)]; !ok {
			return true
		}
	}
	return false
}

// StepTowardAny chooses the first move toward the best reachable target.
func (w *WorldModel) StepTowardAny(start dungeon.Position, targets []dungeon.Position) (dungeon.Action, bool) {
	bestPath := []dungeon.Position(nil)
	for _, target := range targets {
		path, ok := w.ShortestKnownPath(start, target)
		if !ok || len(path) < 2 {
			continue
		}
		if bestPath == nil || len(path) < len(bestPath) || comparePath(path, bestPath) < 0 {
			bestPath = path
		}
	}
	if len(bestPath) < 2 {
		return dungeon.Action{}, false
	}
	return directionAction(start, bestPath[1]), true
}

// ChooseFrontierTarget picks a reachable frontier tile and the first step toward it.
func (w *WorldModel) ChooseFrontierTarget(start dungeon.Position) (*dungeon.Position, dungeon.Action, bool) {
	type candidate struct {
		target dungeon.Position
		path   []dungeon.Position
	}
	candidates := make([]candidate, 0)
	for key, tile := range w.memory.knownTiles {
		if tile == dungeon.TileWall {
			continue
		}
		pos := parsePosKey(key)
		if w.IsFrontier(pos) {
			path, ok := w.ShortestKnownPath(start, pos)
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
		return nil, dungeon.Action{}, false
	}
	target := candidates[0].target
	return &target, directionAction(start, candidates[0].path[1]), true
}

func positionOf(player dungeon.PlayerState) dungeon.Position {
	return dungeon.Position{X: player.X, Y: player.Y}
}

func directionAction(from, to dungeon.Position) dungeon.Action {
	switch {
	case to.X == from.X && to.Y == from.Y-1:
		return dungeon.Action{Action: "move", Direction: "up"}
	case to.X == from.X && to.Y == from.Y+1:
		return dungeon.Action{Action: "move", Direction: "down"}
	case to.X == from.X-1 && to.Y == from.Y:
		return dungeon.Action{Action: "move", Direction: "left"}
	default:
		return dungeon.Action{Action: "move", Direction: "right"}
	}
}

func comparePath(a, b []dungeon.Position) int {
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

func parsePosKey(key string) dungeon.Position {
	var pos dungeon.Position
	_, _ = fmt.Sscanf(key, "%d,%d", &pos.X, &pos.Y)
	return pos
}

func posKey(pos dungeon.Position) string {
	return fmt.Sprintf("%d,%d", pos.X, pos.Y)
}

func step(pos dungeon.Position, direction string) (dungeon.Position, bool) {
	switch direction {
	case "up":
		return dungeon.Position{X: pos.X, Y: pos.Y - 1}, true
	case "down":
		return dungeon.Position{X: pos.X, Y: pos.Y + 1}, true
	case "left":
		return dungeon.Position{X: pos.X - 1, Y: pos.Y}, true
	case "right":
		return dungeon.Position{X: pos.X + 1, Y: pos.Y}, true
	default:
		return pos, false
	}
}

func reversePositions(path []dungeon.Position) {
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
}
