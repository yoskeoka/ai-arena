// Package botlogic hosts the portable dungeon reference AI decision layer.
package botlogic

import "github.com/yoskeoka/ai-arena/games/dungeon"

// Bot is a stateful dungeon bot composed from shared memory and a pluggable policy.
type Bot struct {
	memory *Memory
	world  *WorldModel
	policy Policy
}

// New returns the balanced reference bot.
func New() *Bot {
	return NewWithPolicy(BalancedPolicy())
}

// NewWithPolicy returns a dungeon bot that reuses the shared memory/world-model layer.
func NewWithPolicy(policy Policy) *Bot {
	memory := NewMemory()
	return &Bot{
		memory: memory,
		world:  NewWorldModel(memory),
		policy: policy,
	}
}

// NewNamed returns a dungeon bot for a named policy variant.
func NewNamed(name string) (*Bot, error) {
	policy, err := PolicyByName(name)
	if err != nil {
		return nil, err
	}
	return NewWithPolicy(policy), nil
}

// Decide chooses the next action from the player's visible state.
func (b *Bot) Decide(state dungeon.VisibleState) dungeon.Action {
	b.memory.Observe(state)
	return b.policy.Decide(state, b.memory, b.world)
}
