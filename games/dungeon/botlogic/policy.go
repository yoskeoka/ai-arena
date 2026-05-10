package botlogic

import (
	"fmt"
	"sort"

	"github.com/yoskeoka/ai-arena/games/dungeon"
)

const (
	PolicyBalanced = "balanced"
	PolicyGoalRush = "goal-rush"
)

// Policy decides an action from the current visible state plus shared world knowledge.
type Policy interface {
	Name() string
	Decide(state dungeon.VisibleState, memory *Memory, world *WorldModel) dungeon.Action
}

type balancedPolicy struct{}

type goalRushPolicy struct{}

type chestCandidate struct {
	path    []dungeon.Position
	score   int
	target  dungeon.Position
	points  int
	dist    int
	goalDet int
}

// BalancedPolicy returns the default policy variant.
func BalancedPolicy() Policy {
	return balancedPolicy{}
}

// GoalRushPolicy returns a policy variant that prefers fast finishes once the goal is known.
func GoalRushPolicy() Policy {
	return goalRushPolicy{}
}

// PolicyByName resolves a named policy variant.
func PolicyByName(name string) (Policy, error) {
	switch name {
	case "", PolicyBalanced:
		return BalancedPolicy(), nil
	case PolicyGoalRush:
		return GoalRushPolicy(), nil
	default:
		return nil, fmt.Errorf("unknown dungeon policy %q", name)
	}
}

func (balancedPolicy) Name() string {
	return PolicyBalanced
}

func (goalRushPolicy) Name() string {
	return PolicyGoalRush
}

func (balancedPolicy) Decide(state dungeon.VisibleState, memory *Memory, world *WorldModel) dungeon.Action {
	start := positionOf(state.Self)
	goalPath, hasGoalPath := world.GoalPath(start)
	if hasGoalPath && mustFinishSoon(state, goalPath) {
		memory.SetExploreTarget(nil)
		return directionAction(start, goalPath[1])
	}
	if step, ok := chooseChestAction(state, memory, world, start, goalPath, hasGoalPath); ok {
		memory.SetExploreTarget(nil)
		return step
	}
	if hasGoalPath {
		memory.SetExploreTarget(nil)
		return directionAction(start, goalPath[1])
	}
	if step, ok := stepTowardExploreTarget(memory, world, start); ok {
		return step
	}
	if target, step, ok := world.ChooseFrontierTarget(start); ok {
		memory.SetExploreTarget(target)
		return step
	}
	return dungeon.Action{Action: "wait"}
}

func (goalRushPolicy) Decide(state dungeon.VisibleState, memory *Memory, world *WorldModel) dungeon.Action {
	start := positionOf(state.Self)
	goalPath, hasGoalPath := world.GoalPath(start)
	if hasGoalPath {
		memory.SetExploreTarget(nil)
		if chestStep, ok := chooseChestAction(state, memory, world, start, goalPath, hasGoalPath); ok {
			if shouldTakeGoalRushDetour(state, memory, world, start, goalPath) {
				return chestStep
			}
		}
		return directionAction(start, goalPath[1])
	}
	if step, ok := stepTowardExploreTarget(memory, world, start); ok {
		return step
	}
	if target, step, ok := world.ChooseFrontierTarget(start); ok {
		memory.SetExploreTarget(target)
		return step
	}
	return dungeon.Action{Action: "wait"}
}

func mustFinishSoon(state dungeon.VisibleState, goalPath []dungeon.Position) bool {
	goalDist := len(goalPath) - 1
	if state.RemainingTurns <= goalDist+1 {
		return true
	}
	if goalDist <= 2 && state.Self.ChestPoints > 0 {
		return true
	}
	return false
}

func shouldTakeGoalRushDetour(
	state dungeon.VisibleState,
	memory *Memory,
	world *WorldModel,
	start dungeon.Position,
	goalPath []dungeon.Position,
) bool {
	if len(goalPath) < 2 {
		return false
	}
	goalDist := len(goalPath) - 1
	if goalDist <= 3 || state.RemainingTurns <= goalDist+2 {
		return false
	}
	for _, chest := range memory.KnownChests() {
		target := dungeon.Position{X: chest.X, Y: chest.Y}
		path, ok := world.ShortestKnownPath(start, target)
		if !ok || len(path) < 2 {
			continue
		}
		chestGoalPath, ok := world.ShortestKnownPath(target, goalPath[len(goalPath)-1])
		if !ok || len(chestGoalPath) < 2 {
			continue
		}
		total := len(path) + len(chestGoalPath) - 2
		if total > state.RemainingTurns {
			continue
		}
		detour := total - goalDist
		if chest.Points >= 18 && detour <= 1 {
			return true
		}
	}
	return false
}

func stepTowardExploreTarget(memory *Memory, world *WorldModel, start dungeon.Position) (dungeon.Action, bool) {
	target := memory.ExploreTarget()
	if target == nil || !world.IsFrontier(*target) {
		memory.SetExploreTarget(nil)
		return dungeon.Action{}, false
	}
	if step, ok := world.StepTowardAny(start, []dungeon.Position{*target}); ok {
		return step, true
	}
	memory.SetExploreTarget(nil)
	return dungeon.Action{}, false
}

func chooseChestAction(
	state dungeon.VisibleState,
	memory *Memory,
	world *WorldModel,
	start dungeon.Position,
	goalPath []dungeon.Position,
	hasGoalPath bool,
) (dungeon.Action, bool) {
	goalUtility := -1 << 30
	goalDist := 0
	if hasGoalPath {
		goalDist = len(goalPath) - 1
		goalUtility = 32 - goalDist*6
		if goalDist <= 2 {
			goalUtility += 12
		}
		if state.RemainingTurns <= goalDist+2 {
			goalUtility += 64
		}
		if state.Self.ChestPoints > 0 {
			goalUtility += 6
		}
	}

	leaderGap := scoreGapToLeader(state)
	var best *chestCandidate
	for _, chest := range memory.KnownChests() {
		target := dungeon.Position{X: chest.X, Y: chest.Y}
		path, ok := world.ShortestKnownPath(start, target)
		if !ok || len(path) < 2 {
			continue
		}
		dist := len(path) - 1
		if dist > state.RemainingTurns {
			continue
		}

		score := chest.Points*4 - dist*5
		goalDetour := 0
		if hasGoalPath {
			chestGoalPath, ok := world.ShortestKnownPath(target, goalPath[len(goalPath)-1])
			if !ok || len(chestGoalPath) < 2 {
				score -= 18
			} else {
				chestGoalDist := len(chestGoalPath) - 1
				if dist+chestGoalDist > state.RemainingTurns {
					continue
				}
				goalDetour = max(0, dist+chestGoalDist-goalDist)
				score -= goalDetour * 4
				if goalDist <= 3 {
					score -= 8
				}
			}
		}
		if chest.Points >= 18 {
			score += 8
		}
		if leaderGap > 0 {
			if chest.Points > leaderGap {
				score += 10
			} else {
				score -= 3
			}
		}

		option := &chestCandidate{
			path:    path,
			score:   score,
			target:  target,
			points:  chest.Points,
			dist:    dist,
			goalDet: goalDetour,
		}
		if best == nil || compareCandidate(*option, *best) > 0 {
			best = option
		}
	}
	if best == nil {
		return dungeon.Action{}, false
	}
	if hasGoalPath && best.score <= goalUtility {
		return dungeon.Action{}, false
	}
	if best.score < 16 {
		return dungeon.Action{}, false
	}
	return directionAction(start, best.path[1]), true
}

func compareCandidate(a, b chestCandidate) int {
	switch {
	case a.score != b.score:
		return a.score - b.score
	case a.points != b.points:
		return a.points - b.points
	case a.goalDet != b.goalDet:
		return b.goalDet - a.goalDet
	case a.dist != b.dist:
		return b.dist - a.dist
	default:
		return -comparePath(a.path, b.path)
	}
}

func sortedChests(values []dungeon.ChestState) []dungeon.ChestState {
	chests := append([]dungeon.ChestState(nil), values...)
	sort.Slice(chests, func(i, j int) bool {
		if chests[i].Points != chests[j].Points {
			return chests[i].Points > chests[j].Points
		}
		if chests[i].Y != chests[j].Y {
			return chests[i].Y < chests[j].Y
		}
		return chests[i].X < chests[j].X
	})
	return chests
}

func sortedChestMap(values map[string]dungeon.ChestState) []dungeon.ChestState {
	chests := make([]dungeon.ChestState, 0, len(values))
	for _, chest := range values {
		chests = append(chests, chest)
	}
	return sortedChests(chests)
}

func scoreGapToLeader(state dungeon.VisibleState) int {
	leader := state.Self.Score
	for _, player := range state.Scores {
		if player.PlayerID == state.Self.PlayerID {
			continue
		}
		if player.Score > leader {
			leader = player.Score
		}
	}
	return max(0, leader-state.Self.Score)
}
