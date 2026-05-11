package dungeon

import (
	cryptorand "crypto/rand"
	"encoding/hex"
	"fmt"
	randv2 "math/rand/v2"
	"sort"
	"strings"
	"sync"
)

func buildLayout(ruleset, seed string) (GeneratedLayout, error) {
	switch ruleset {
	case RulesetFixedMapV1:
		return fixedMapLayout(), nil
	case RulesetSeededMazeV1:
		return seededMazeLayout(seed)
	default:
		return GeneratedLayout{}, fmt.Errorf("dungeon: unsupported ruleset %q", ruleset)
	}
}

func fixedMapLayout() GeneratedLayout {
	return GeneratedLayout{
		Tiles: []string{
			"#########",
			"#...#...#",
			"#...#...#",
			"#.......#",
			"#.#.#.#.#",
			"#...#...#",
			"#.......#",
			"#.......#",
			"#########",
		},
		SpawnPoints: []Position{{X: 1, Y: 1}, {X: 7, Y: 1}, {X: 1, Y: 7}, {X: 7, Y: 7}},
		Goal:        Position{X: 6, Y: 6},
		InitialChests: []ChestState{
			{X: 2, Y: 3, Points: 12},
			{X: 6, Y: 3, Points: 12},
			{X: 2, Y: 6, Points: 12},
		},
	}
}

func seededMazeLayout(seed string) (GeneratedLayout, error) {
	tiles := generatePerfectMaze9x9(seed)
	walkable := walkablePositions(tiles)
	if len(walkable) < 8 {
		return GeneratedLayout{}, fmt.Errorf("dungeon: generated maze has insufficient walkable tiles")
	}
	rng, err := newSeededRand(seed)
	if err != nil {
		return GeneratedLayout{}, err
	}
	goal := walkable[rng.IntN(len(walkable))]
	start := farthestPosition(tiles, goal)
	spawns, err := nearestUniquePositions(tiles, start, 4, map[string]struct{}{posKey(goal): {}})
	if err != nil {
		return GeneratedLayout{}, err
	}
	chestPositions, err := selectChestPositions(tiles, walkable, start, goal, spawns, 3, rng)
	if err != nil {
		return GeneratedLayout{}, err
	}
	chestScores := append([]int(nil), seededChestPoints...)
	rng.Shuffle(len(chestScores), func(i, j int) {
		chestScores[i], chestScores[j] = chestScores[j], chestScores[i]
	})
	initialChests := make([]ChestState, 0, len(chestPositions))
	for i, pos := range chestPositions {
		initialChests = append(initialChests, ChestState{X: pos.X, Y: pos.Y, Points: chestScores[i]})
	}
	return GeneratedLayout{
		Tiles:         tiles,
		SpawnPoints:   spawns,
		Goal:          goal,
		InitialChests: initialChests,
	}, nil
}

func generatePerfectMaze9x9(seed string) []string {
	const size = 9
	grid := make([][]byte, size)
	for y := range grid {
		grid[y] = make([]byte, size)
		for x := range grid[y] {
			grid[y][x] = '#'
		}
	}
	type cell struct{ x, y int }
	cells := []cell{}
	for y := 1; y < size; y += 2 {
		for x := 1; x < size; x += 2 {
			cells = append(cells, cell{x: x, y: y})
		}
	}
	rng, err := newSeededRand(seed)
	if err != nil {
		panic(err)
	}
	start := cells[rng.IntN(len(cells))]
	stack := []cell{start}
	visited := map[cell]struct{}{start: {}}
	grid[start.y][start.x] = '.'
	dirs := []cell{{x: 0, y: -2}, {x: 2, y: 0}, {x: 0, y: 2}, {x: -2, y: 0}}
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		order := append([]cell(nil), dirs...)
		rng.Shuffle(len(order), func(i, j int) {
			order[i], order[j] = order[j], order[i]
		})
		advanced := false
		for _, dir := range order {
			next := cell{x: current.x + dir.x, y: current.y + dir.y}
			if next.x <= 0 || next.x >= size-1 || next.y <= 0 || next.y >= size-1 {
				continue
			}
			if _, ok := visited[next]; ok {
				continue
			}
			grid[current.y+dir.y/2][current.x+dir.x/2] = '.'
			grid[next.y][next.x] = '.'
			visited[next] = struct{}{}
			stack = append(stack, next)
			advanced = true
			break
		}
		if !advanced {
			stack = stack[:len(stack)-1]
		}
	}
	rows := make([]string, size)
	for i := range grid {
		rows[i] = string(grid[i])
	}
	return rows
}

func nearestUniquePositions(layout []string, from Position, count int, exclude map[string]struct{}) ([]Position, error) {
	candidates := make([]positionDistance, 0)
	for _, pos := range walkablePositions(layout) {
		if _, skip := exclude[posKey(pos)]; skip {
			continue
		}
		path, ok := shortestPath(layout, from, pos)
		if !ok {
			continue
		}
		candidates = append(candidates, positionDistance{Position: pos, Distance: len(path) - 1})
	}
	sortPositionDistances(candidates, false)
	if len(candidates) < count {
		return nil, fmt.Errorf("dungeon: insufficient spawn positions")
	}
	out := make([]Position, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, candidates[i].Position)
	}
	return out, nil
}

func selectChestPositions(layout []string, walkable []Position, start, goal Position, spawns []Position, count int, rng *deterministicRand) ([]Position, error) {
	excluded := make(map[string]struct{}, len(spawns)+1)
	excluded[posKey(goal)] = struct{}{}
	for _, spawn := range spawns {
		excluded[posKey(spawn)] = struct{}{}
	}
	candidates := make([]Position, 0, len(walkable))
	for _, pos := range walkable {
		if _, skip := excluded[posKey(pos)]; skip {
			continue
		}
		candidates = append(candidates, pos)
	}
	rng.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	selected := make([]Position, 0, count)
	seen := map[string]struct{}{}
	for _, pos := range candidates {
		if distanceBetween(layout, start, pos) <= 1 || distanceBetween(layout, goal, pos) <= 1 {
			continue
		}
		key := posKey(pos)
		if _, exists := seen[key]; exists {
			continue
		}
		selected = append(selected, pos)
		seen[key] = struct{}{}
		if len(selected) == count {
			return selected, nil
		}
	}
	for _, pos := range candidates {
		key := posKey(pos)
		if _, exists := seen[key]; exists {
			continue
		}
		selected = append(selected, pos)
		seen[key] = struct{}{}
		if len(selected) == count {
			return selected, nil
		}
	}
	return nil, fmt.Errorf("dungeon: insufficient chest positions")
}

func farthestPosition(layout []string, from Position) Position {
	best := from
	bestDistance := -1
	for _, pos := range walkablePositions(layout) {
		path, ok := shortestPath(layout, from, pos)
		if !ok {
			continue
		}
		distance := len(path) - 1
		if distance > bestDistance || (distance == bestDistance && comparePosition(pos, best) < 0) {
			best = pos
			bestDistance = distance
		}
	}
	return best
}

type positionDistance struct {
	Position
	Distance int
}

func sortPositionDistances(values []positionDistance, reverseDistance bool) {
	sort.Slice(values, func(i, j int) bool {
		if values[i].Distance != values[j].Distance {
			if reverseDistance {
				return values[i].Distance > values[j].Distance
			}
			return values[i].Distance < values[j].Distance
		}
		return comparePosition(values[i].Position, values[j].Position) < 0
	})
}

func walkablePositions(layout []string) []Position {
	positions := make([]Position, 0)
	for y, row := range layout {
		for x := 0; x < len(row); x++ {
			if row[x] != '#' {
				positions = append(positions, Position{X: x, Y: y})
			}
		}
	}
	return positions
}

type deterministicRand struct {
	mu  sync.Mutex
	rng *randv2.Rand
}

func newSeededRand(seed string) (*deterministicRand, error) {
	material, err := decodeSeedMaterial(seed)
	if err != nil {
		return nil, err
	}
	var seed32 [32]byte
	copy(seed32[:], material)
	// #nosec G404 -- deterministic gameplay generation requires a reproducible PRNG, and ChaCha8 gives a stable stream
	// while being less trivially guessable than simpler generators. This source is wrapped so multi-site callers can
	// serialize access without introducing additional independent RNG state.
	return &deterministicRand{rng: randv2.New(randv2.NewChaCha8(seed32))}, nil
}

func (r *deterministicRand) IntN(n int) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.rng.IntN(n)
}

func (r *deterministicRand) Shuffle(n int, swap func(i, j int)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rng.Shuffle(n, swap)
}

func normalizeSeed(seed string) (string, error) {
	seed = strings.TrimSpace(seed)
	if seed == "" {
		return "", fmt.Errorf("rng_seed is required")
	}
	decoded, err := decodeSeedMaterial(seed)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(decoded), nil
}

func normalizeSeedOrGenerate(seed string) (string, error) {
	seed = strings.TrimSpace(seed)
	if seed == "" {
		return GenerateSeedHex()
	}
	return normalizeSeed(seed)
}

func decodeSeedMaterial(seed string) ([]byte, error) {
	seed = strings.TrimSpace(seed)
	if seed == "" {
		return nil, fmt.Errorf("rng_seed is required")
	}
	if len(seed) != 64 {
		return nil, fmt.Errorf("rng_seed must be 64 hex characters")
	}
	decoded, err := hex.DecodeString(seed)
	if err != nil {
		return nil, fmt.Errorf("rng_seed must be lowercase/uppercase hex: %w", err)
	}
	if len(decoded) != 32 {
		return nil, fmt.Errorf("rng_seed must decode to 32 bytes")
	}
	return decoded, nil
}

func GenerateSeedHex() (string, error) {
	buf := make([]byte, 32)
	if _, err := cryptorand.Read(buf); err != nil {
		return "", fmt.Errorf("generate rng_seed: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func cloneRuleset(r Ruleset) Ruleset {
	r.GoalBonuses = append([]int(nil), r.GoalBonuses...)
	return r
}

func cloneLayout(layout GeneratedLayout) GeneratedLayout {
	layout.Tiles = append([]string(nil), layout.Tiles...)
	layout.SpawnPoints = append([]Position(nil), layout.SpawnPoints...)
	layout.InitialChests = append([]ChestState(nil), layout.InitialChests...)
	return layout
}

func step(pos Position, direction string) (Position, bool) {
	switch direction {
	case "up":
		return Position{X: pos.X, Y: pos.Y - 1}, true
	case "down":
		return Position{X: pos.X, Y: pos.Y + 1}, true
	case "left":
		return Position{X: pos.X - 1, Y: pos.Y}, true
	case "right":
		return Position{X: pos.X + 1, Y: pos.Y}, true
	default:
		return Position{}, false
	}
}

func manhattan(a, b Position) int {
	dx := a.X - b.X
	if dx < 0 {
		dx = -dx
	}
	dy := a.Y - b.Y
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}

func shortestPath(layout []string, from, to Position) ([]Position, bool) {
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
			next, ok := step(current, direction)
			if !ok || next.Y < 0 || next.Y >= len(layout) || next.X < 0 || next.X >= len(layout[next.Y]) {
				continue
			}
			if layout[next.Y][next.X] == '#' {
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

func reversePositions(path []Position) {
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
}

func comparePosition(a, b Position) int {
	if a.Y != b.Y {
		return a.Y - b.Y
	}
	return a.X - b.X
}

func distanceBetween(layout []string, from, to Position) int {
	path, ok := shortestPath(layout, from, to)
	if !ok {
		return 1 << 30
	}
	return len(path) - 1
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalPositions(a, b []Position) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalChests(a, b []ChestState) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
