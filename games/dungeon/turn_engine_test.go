package dungeon

import (
	"reflect"
	"testing"
)

func TestTurnEngineNormalizesActionsBeforeMovement(t *testing.T) {
	match := restoreFixedMapScenario(t, []string{"p1", "p2"}, fixedMapState(
		mustNewFixedMapMatch(t, []string{"p1", "p2"}),
		FullState{
			Players: []PlayerState{
				{PlayerID: "p1", X: 1, Y: 1},
				{PlayerID: "p2", X: 7, Y: 1},
			},
		},
	))

	engine := newTurnEngine(match, map[string]Action{
		"p1": {Action: "move", Direction: "left"},
	})
	engine.normalizeActions()

	if got := engine.frame.actions["p1"]; got != (Action{Action: "wait"}) {
		t.Fatalf("normalized p1 action = %+v, want wait", got)
	}
	if got := engine.frame.actions["p2"]; got != (Action{Action: "wait"}) {
		t.Fatalf("missing p2 action = %+v, want wait", got)
	}
}

func TestTurnEngineTracksMovementAndChestClaimantsByPhase(t *testing.T) {
	match := restoreFixedMapScenario(t, []string{"p1", "p2", "p3"}, fixedMapState(
		mustNewFixedMapMatch(t, []string{"p1", "p2", "p3"}),
		FullState{
			Turn: 5,
			Players: []PlayerState{
				{PlayerID: "p1", X: 1, Y: 6},
				{PlayerID: "p2", X: 3, Y: 6},
				{PlayerID: "p3", X: 5, Y: 6},
			},
			UncollectedChests: []ChestState{
				{X: 2, Y: 6, Points: 12},
			},
		},
	))

	engine := newTurnEngine(match, map[string]Action{
		"p1": {Action: "move", Direction: "right"},
		"p2": {Action: "move", Direction: "left"},
		"p3": {Action: "wait"},
	})

	engine.normalizeActions()
	engine.resolveMovement()

	if got := engine.frame.nextPositions["p1"]; got != (Position{X: 2, Y: 6}) {
		t.Fatalf("p1 next position = %+v, want (2,6)", got)
	}
	if got := engine.frame.nextPositions["p2"]; got != (Position{X: 2, Y: 6}) {
		t.Fatalf("p2 next position = %+v, want (2,6)", got)
	}

	engine.resolveInteractions()

	claimants := engine.frame.chestClaimants["2,6"]
	if len(claimants) != 2 || claimants[0] != "p1" || claimants[1] != "p2" {
		t.Fatalf("claimants = %+v, want [p1 p2]", claimants)
	}
	assertPlayerScore(t, match, "p1", 6, 0, 6)
	assertPlayerScore(t, match, "p2", 6, 0, 6)
	assertPlayerScore(t, match, "p3", 0, 0, 0)
}

func TestTurnEngineRunsSubsystemSeamsInFixedOrder(t *testing.T) {
	match := restoreFixedMapScenario(t, []string{"p1", "p2"}, fixedMapState(
		mustNewFixedMapMatch(t, []string{"p1", "p2"}),
		FullState{
			Turn: 5,
			Players: []PlayerState{
				{PlayerID: "p1", X: 1, Y: 6},
				{PlayerID: "p2", X: 7, Y: 1},
			},
			UncollectedChests: []ChestState{
				{X: 2, Y: 6, Points: 12},
			},
		},
	))

	trace := make([]string, 0, 6)
	match.seams = subsystemSeams{
		actors: traceInteractionSubsystem{
			label: "actors",
			trace: &trace,
		},
		items: traceInteractionSubsystem{
			label: "items",
			trace: &trace,
			t:     t,
			check: func(t *testing.T, e *turnEngine) {
				if got := len(e.match.UncollectedChests()); got != 0 {
					t.Fatalf("items hook saw %d remaining chests, want built-in chest resolution first", got)
				}
				assertPlayerScore(t, e.match, "p1", 12, 0, 12)
			},
		},
		inventory: traceInteractionSubsystem{
			label: "inventory",
			trace: &trace,
		},
		combat: traceInteractionSubsystem{
			label: "combat",
			trace: &trace,
		},
		effects: traceEffectSubsystem{
			trace: &trace,
		},
		visibility: traceVisibilitySubsystem{
			trace: &trace,
		},
	}.withDefaults()

	if err := match.Apply(map[string]Action{
		"p1": {Action: "move", Direction: "right"},
		"p2": {Action: "wait"},
	}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	want := []string{"actors", "items", "inventory", "combat", "effects", "visibility"}
	if !reflect.DeepEqual(trace, want) {
		t.Fatalf("subsystem hook order = %v, want %v", trace, want)
	}
}

type traceInteractionSubsystem struct {
	label string
	trace *[]string
	t     *testing.T
	check func(t *testing.T, e *turnEngine)
}

func (s traceInteractionSubsystem) ResolveInteractions(e *turnEngine) {
	*s.trace = append(*s.trace, s.label)
	if s.check != nil {
		s.check(s.t, e)
	}
}

type traceEffectSubsystem struct {
	trace *[]string
}

func (s traceEffectSubsystem) BeforeTerminalUpdate(*turnEngine) {
	*s.trace = append(*s.trace, "effects")
}

type traceVisibilitySubsystem struct {
	trace *[]string
}

func (s traceVisibilitySubsystem) Refresh(e *turnEngine) {
	*s.trace = append(*s.trace, "visibility")
	defaultVisibilitySubsystem{}.Refresh(e)
}
