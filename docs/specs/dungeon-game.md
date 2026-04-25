# Dungeon Game Concept Specification

## Purpose

This document captures the Phase 1 concept for the dungeon game that will
become the platform's flagship ruleset. It defines the design direction and the
platform requirements implied by that direction, but it intentionally stops
short of Phase 3 implementation-level detail.

## Design Goals

- reward fast algorithmic decision-making rather than prompt-only LLM usage
- create information asymmetry through limited vision
- produce spectator-friendly state changes on a grid
- support both indirect competition and direct player interaction
- leave room for staged complexity growth across future phases

## High-Level Match Structure

- Map: tile-based dungeon grid
- Players: multiple concurrent explorers
- Turn model: simultaneous action by default
- Match length: fixed turn budget or earlier termination when the dungeon goal is
  resolved
- Outcome: ranked by final score

The exact player count, map dimensions, and turn budget are deferred to Phase 3.

## Core Loop

Each turn:

1. the game master computes each player's visible area
2. the platform sends that visible state to the player AI
3. every player submits one action before the deadline
4. the game master resolves movement, interactions, combat, and pickups
5. the game master publishes the updated whole-state snapshot

This keeps the transport model aligned with janken while introducing richer
state and partial information.

## Information Model

### Limited vision

Each player can observe only a bounded area around its current position.

Phase 1 assumption:

- a player's visible area is a local neighborhood with configurable radius `N`
- tiles outside that area are omitted from `visible_state`
- remembering old observations is the AI's responsibility, not the platform's

This is a deliberate design choice to test persistent internal state inside the
AI process.

### Whole-state export

The platform still requires a complete hidden-information snapshot for operators
and spectators. The game therefore needs two separate representations:

- `visible_state`: per-player filtered state for AI decisions
- `full_state`: authoritative dungeon state for replay and viewing

## Proposed Game Elements

### Navigation

- grid movement between neighboring walkable tiles
- walls and obstacles that shape local pathfinding
- exploration pressure from finite turn count

### Resources

- collectible items with positive score value
- scarce objectives that create competition between players
- optional boss reward or treasure-room reward that is worth contesting

### Player interaction

- collision over the same target tile
- contest over the same item
- optional direct attack or sabotage action
- optional temporary alliances in later phases

Phase 1 leaves the exact combat/interaction rules open, but the game must allow
meaningful player-to-player influence beyond isolated racing.

## Action Categories

The Phase 3 detailed spec is expected to define concrete actions from a set like:

- `move`
- `wait`
- `pickup`
- `use`
- `attack`
- `interact`

The exact schema is intentionally deferred, but the platform must support
game-specific action objects more complex than janken's single string action.

## Scoring Direction

Final ranking should derive from score rather than survival alone.

Likely score sources:

- treasure collected
- rare objective captures
- combat rewards
- extraction/escape bonus

Likely penalties:

- defeat or incapacitation
- wasted turns under hazardous conditions

This scoring model supports both aggression and efficient exploration instead of
collapsing the game into pure elimination.

## Spectator Value

The dungeon concept is chosen partly because it creates a readable visual match:

- map reveal over time
- path divergence and convergence
- contested objectives
- surprise encounters caused by limited vision

This makes it a better flagship spectator game than an abstract hidden-state
game with weak spatial representation.

## Platform Implications

The dungeon game requires the platform to support:

- per-player filtered `visible_state`
- long-lived AI memory across many turns
- simultaneous action resolution with collisions/conflicts
- whole-state snapshots for replay
- ranking by composite score rather than binary win/loss

These requirements are why the Phase 1 platform spec is broader than what
janken alone would demand.

## Deferred to Phase 3

Phase 3 must define:

- concrete tile types and map generation rules
- exact vision geometry
- action schema and validation rules
- combat/interference rules
- item taxonomy
- termination conditions
- scoring formula and tie-breakers
- canonical JSON-RPC payload examples
