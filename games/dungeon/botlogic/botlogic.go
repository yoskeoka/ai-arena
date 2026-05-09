// Package botlogic hosts the portable dungeon reference AI decision layer.
package botlogic

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/yoskeoka/ai-arena/games/dungeon"
)

const jsonRPCVersion = "2.0"

// Bot is a deterministic baseline bot for the dungeon visible-state API.
type Bot struct {
	knownTiles    map[string]string
	knownGoal     *dungeon.Position
	knownChests   map[string]dungeon.ChestState
	exploreTarget *dungeon.Position
}

type chestCandidate struct {
	path    []dungeon.Position
	score   int
	target  dungeon.Position
	points  int
	dist    int
	goalDet int
}

// New returns a stateful dungeon baseline bot.
func New() *Bot {
	return &Bot{
		knownTiles:  make(map[string]string),
		knownChests: make(map[string]dungeon.ChestState),
	}
}

// Run serves the dungeon reference bot over NDJSON JSON-RPC.
func Run(r io.Reader, w io.Writer) error {
	dec := newDecoder(r)
	enc := newEncoder(w)
	bot := New()

	for {
		req, err := dec.decodeRequest()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		switch req.Method {
		case "init":
			resp, err := newResponse(req.ID, map[string]any{"ready": true})
			if err != nil {
				return err
			}
			if err := enc.encode(resp); err != nil {
				return err
			}
		case "turn":
			var payload struct {
				VisibleState dungeon.VisibleState `json:"visible_state"`
			}
			if err := json.Unmarshal(req.Params, &payload); err != nil {
				return err
			}
			action := bot.Decide(payload.VisibleState)
			resp, err := newResponse(req.ID, action)
			if err != nil {
				return err
			}
			if err := enc.encode(resp); err != nil {
				return err
			}
		case "game_over":
			resp, err := newResponse(req.ID, map[string]any{"ack": true})
			if err != nil {
				return err
			}
			if err := enc.encode(resp); err != nil {
				return err
			}
			return nil
		}
	}
}

// Decide chooses the next action from the player's visible state.
func (b *Bot) Decide(state dungeon.VisibleState) dungeon.Action {
	b.observe(state)
	start := positionOf(state.Self)
	goalPath, hasGoalPath := b.goalPath(start)
	if hasGoalPath && b.mustFinishSoon(state, goalPath) {
		b.exploreTarget = nil
		return directionAction(start, goalPath[1])
	}
	if step, ok := b.chooseChestAction(state, start, goalPath, hasGoalPath); ok {
		b.exploreTarget = nil
		return step
	}
	if hasGoalPath {
		b.exploreTarget = nil
		return directionAction(start, goalPath[1])
	}
	if step, ok := b.stepTowardExploreTarget(start); ok {
		return step
	}
	if step, ok := b.chooseFrontierTarget(start); ok {
		return step
	}
	return dungeon.Action{Action: "wait"}
}

func (b *Bot) observe(state dungeon.VisibleState) {
	for _, tile := range state.VisibleTiles {
		pos := dungeon.Position{X: tile.X, Y: tile.Y}
		b.knownTiles[posKey(pos)] = tile.Tile
		if tile.Tile == dungeon.TileGoal {
			goal := pos
			b.knownGoal = &goal
		}
		if tile.Tile == dungeon.TileChest {
			b.knownChests[posKey(pos)] = dungeon.ChestState{X: pos.X, Y: pos.Y}
		}
	}
	if state.KnownGoal != nil {
		goal := *state.KnownGoal
		b.knownGoal = &goal
	}
	currentKnown := make(map[string]struct{}, len(state.KnownChests))
	for _, chest := range state.KnownChests {
		key := posKey(dungeon.Position{X: chest.X, Y: chest.Y})
		currentKnown[key] = struct{}{}
		b.knownChests[key] = chest
	}
	for key := range b.knownChests {
		if _, ok := currentKnown[key]; !ok {
			delete(b.knownChests, key)
		}
	}
}

func (b *Bot) chooseChestAction(
	state dungeon.VisibleState,
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
	for _, chest := range sortedChests(b.knownChests) {
		target := dungeon.Position{X: chest.X, Y: chest.Y}
		path, ok := b.shortestKnownPath(start, target)
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
			chestGoalPath, ok := b.shortestKnownPath(target, *b.knownGoal)
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

func (b *Bot) goalPath(start dungeon.Position) ([]dungeon.Position, bool) {
	if b.knownGoal == nil {
		return nil, false
	}
	path, ok := b.shortestKnownPath(start, *b.knownGoal)
	if !ok || len(path) < 2 {
		return nil, false
	}
	return path, true
}

func (b *Bot) mustFinishSoon(state dungeon.VisibleState, goalPath []dungeon.Position) bool {
	goalDist := len(goalPath) - 1
	if state.RemainingTurns <= goalDist+1 {
		return true
	}
	if goalDist <= 2 && state.Self.ChestPoints > 0 {
		return true
	}
	return false
}

func (b *Bot) stepTowardExploreTarget(start dungeon.Position) (dungeon.Action, bool) {
	if b.exploreTarget == nil || !b.isFrontier(*b.exploreTarget) {
		b.exploreTarget = nil
		return dungeon.Action{}, false
	}
	if step, ok := b.stepTowardAny(start, []dungeon.Position{*b.exploreTarget}); ok {
		return step, true
	}
	b.exploreTarget = nil
	return dungeon.Action{}, false
}

func (b *Bot) chooseFrontierTarget(start dungeon.Position) (dungeon.Action, bool) {
	type candidate struct {
		target dungeon.Position
		path   []dungeon.Position
	}
	candidates := make([]candidate, 0)
	for key, tile := range b.knownTiles {
		if tile == dungeon.TileWall {
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
		return dungeon.Action{}, false
	}
	target := candidates[0].target
	b.exploreTarget = &target
	return directionAction(start, candidates[0].path[1]), true
}

func (b *Bot) stepTowardAny(start dungeon.Position, targets []dungeon.Position) (dungeon.Action, bool) {
	bestPath := []dungeon.Position(nil)
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
		return dungeon.Action{}, false
	}
	return directionAction(start, bestPath[1]), true
}

func (b *Bot) shortestKnownPath(from, to dungeon.Position) ([]dungeon.Position, bool) {
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
			tile, ok := b.knownTiles[posKey(next)]
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

func (b *Bot) isFrontier(pos dungeon.Position) bool {
	directions := []string{"up", "down", "left", "right"}
	for _, direction := range directions {
		next, _ := step(pos, direction)
		if _, ok := b.knownTiles[posKey(next)]; !ok {
			return true
		}
	}
	return false
}

func sortedChests(values map[string]dungeon.ChestState) []dungeon.ChestState {
	chests := make([]dungeon.ChestState, 0, len(values))
	for _, chest := range values {
		chests = append(chests, chest)
	}
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

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
}

type encoder struct {
	enc *json.Encoder
}

type decoder struct {
	scanner *bufio.Scanner
}

func newEncoder(w io.Writer) *encoder {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return &encoder{enc: enc}
}

func (e *encoder) encode(v any) error {
	return e.enc.Encode(v)
}

func newDecoder(r io.Reader) *decoder {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	return &decoder{scanner: scanner}
}

func (d *decoder) decodeRequest() (request, error) {
	if !d.scanner.Scan() {
		if err := d.scanner.Err(); err != nil {
			return request{}, err
		}
		return request{}, io.EOF
	}

	var req request
	if err := json.Unmarshal(d.scanner.Bytes(), &req); err != nil {
		return request{}, err
	}
	if req.JSONRPC != jsonRPCVersion {
		return request{}, fmt.Errorf("unsupported jsonrpc version %q", req.JSONRPC)
	}
	if req.Method == "" {
		return request{}, fmt.Errorf("request method is required")
	}
	return req, nil
}

func newResponse(id string, result any) (response, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return response{}, err
	}
	return response{
		JSONRPC: jsonRPCVersion,
		ID:      id,
		Result:  raw,
	}, nil
}
