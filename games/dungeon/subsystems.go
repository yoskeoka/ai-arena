package dungeon

// subsystemSeams fixes the insertion points for future dungeon subsystems
// without widening the current public API.
type subsystemSeams struct {
	actors     actorSubsystem
	items      itemSubsystem
	inventory  inventorySubsystem
	combat     combatSubsystem
	effects    effectSubsystem
	visibility visibilitySubsystem
}

func defaultSubsystemSeams() subsystemSeams {
	return subsystemSeams{}.withDefaults()
}

func (s subsystemSeams) withDefaults() subsystemSeams {
	if s.actors == nil {
		s.actors = noopActorSubsystem{}
	}
	if s.items == nil {
		s.items = noopItemSubsystem{}
	}
	if s.inventory == nil {
		s.inventory = noopInventorySubsystem{}
	}
	if s.combat == nil {
		s.combat = noopCombatSubsystem{}
	}
	if s.effects == nil {
		s.effects = noopEffectSubsystem{}
	}
	if s.visibility == nil {
		s.visibility = defaultVisibilitySubsystem{}
	}
	return s
}

type actorSubsystem interface {
	ResolveInteractions(*turnEngine)
}

type itemSubsystem interface {
	ResolveInteractions(*turnEngine)
}

type inventorySubsystem interface {
	ResolveInteractions(*turnEngine)
}

type combatSubsystem interface {
	ResolveInteractions(*turnEngine)
}

type effectSubsystem interface {
	BeforeTerminalUpdate(*turnEngine)
}

type visibilitySubsystem interface {
	Refresh(*turnEngine)
}

type noopActorSubsystem struct{}

func (noopActorSubsystem) ResolveInteractions(*turnEngine) {}

type noopItemSubsystem struct{}

func (noopItemSubsystem) ResolveInteractions(*turnEngine) {}

type noopInventorySubsystem struct{}

func (noopInventorySubsystem) ResolveInteractions(*turnEngine) {}

type noopCombatSubsystem struct{}

func (noopCombatSubsystem) ResolveInteractions(*turnEngine) {}

type noopEffectSubsystem struct{}

func (noopEffectSubsystem) BeforeTerminalUpdate(*turnEngine) {}

type defaultVisibilitySubsystem struct{}

func (defaultVisibilitySubsystem) Refresh(e *turnEngine) {
	e.match.state.refreshDiscoveries(e.match.ruleset, e.match.layout, e.match.playerOrder)
}
