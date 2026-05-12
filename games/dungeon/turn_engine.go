package dungeon

type turnEngine struct {
	match *Match
	frame turnFrame
}

type turnFrame struct {
	activePlayers  []string
	actions        map[string]Action
	nextPositions  map[string]Position
	chestClaimants map[string][]string
}

func newTurnEngine(match *Match, actions map[string]Action) *turnEngine {
	actionCopy := make(map[string]Action, len(actions))
	for playerID, action := range actions {
		actionCopy[playerID] = action
	}
	return &turnEngine{
		match: match,
		frame: turnFrame{
			activePlayers:  match.PendingPlayerIDs(),
			actions:        actionCopy,
			nextPositions:  make(map[string]Position),
			chestClaimants: make(map[string][]string),
		},
	}
}

func (e *turnEngine) run() {
	e.normalizeActions()
	e.resolveMovement()
	e.resolveInteractions()
	e.updateTerminalAndScores()
	e.refreshVisibility()
}

func (e *turnEngine) normalizeActions() {
	for _, playerID := range e.frame.activePlayers {
		action := e.frame.actions[playerID]
		if action.Action == "" {
			action = Action{Action: "wait"}
		}
		if !e.match.CanApply(playerID, action) {
			action = Action{Action: "wait"}
		}
		e.frame.actions[playerID] = action
	}
}

func (e *turnEngine) resolveMovement() {
	for _, playerID := range e.frame.activePlayers {
		current := e.match.state.playerStates[playerID].position()
		e.frame.nextPositions[playerID] = e.match.resolvePosition(current, e.frame.actions[playerID])
	}
	for _, playerID := range e.frame.activePlayers {
		player := e.match.state.playerStates[playerID]
		target := e.frame.nextPositions[playerID]
		player.X = target.X
		player.Y = target.Y
		e.match.state.playerStates[playerID] = player
	}
}

func (e *turnEngine) resolveInteractions() {
	for chestID, chest := range chestsCopy(e.match.state.uncollectedChests) {
		claimants := e.claimantsAt(Position{X: chest.X, Y: chest.Y})
		if len(claimants) == 0 {
			continue
		}
		e.frame.chestClaimants[chestID] = append([]string(nil), claimants...)
		share := chest.Points / len(claimants)
		for _, playerID := range claimants {
			player := e.match.state.playerStates[playerID]
			player.ChestPoints += share
			player.Score += share
			e.match.state.playerStates[playerID] = player
		}
		delete(e.match.state.uncollectedChests, chestID)
		for _, known := range e.match.state.discoveredChests {
			delete(known, chestID)
		}
	}
}

func (e *turnEngine) updateTerminalAndScores() {
	// Future effect ticks stay inside this phase until a later plan expands the
	// deterministic phase contract beyond the current 5-step pipeline.
	for _, playerID := range e.frame.activePlayers {
		player := e.match.state.playerStates[playerID]
		if player.FinishedTurn == nil && player.X == e.match.layout.Goal.X && player.Y == e.match.layout.Goal.Y {
			finishedTurn := e.match.state.turn + 1
			player.FinishedTurn = &finishedTurn
			e.match.state.playerStates[playerID] = player
		}
	}
	e.match.state.turn++
	e.match.state.applyGoalBonuses(e.match.ruleset, e.match.playerOrder)
}

func (e *turnEngine) refreshVisibility() {
	e.match.state.refreshDiscoveries(e.match.ruleset, e.match.layout, e.match.playerOrder)
}

func (e *turnEngine) claimantsAt(pos Position) []string {
	claimants := make([]string, 0, len(e.frame.activePlayers))
	for _, playerID := range e.frame.activePlayers {
		player := e.match.state.playerStates[playerID]
		if player.X == pos.X && player.Y == pos.Y {
			claimants = append(claimants, playerID)
		}
	}
	return claimants
}
