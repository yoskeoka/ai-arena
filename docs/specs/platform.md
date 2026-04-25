# Platform Specification

## Purpose

This document defines the Phase 1 specification for the AI Arena platform layer.
The platform layer is game-agnostic and is responsible for AI execution, turn
coordination, result recording, and spectator-facing state retrieval.

## Scope

The platform layer must support:

- registering multiple games behind a common execution model
- running long-lived AI processes for a single match
- exchanging actions over stdin/stdout with JSON-RPC 2.0 envelopes
- handling simultaneous-action and turn-order games
- recording match results, rankings, and AI stderr logs
- exposing whole-match state for operators and future spectator clients

This document does not define the concrete HTTP API surface or persistence
schema. Those are deferred to later implementation phases.

## Core Model

### Platform responsibilities

The platform is responsible for:

- accepting a game definition and match configuration
- starting one AI runtime instance per participating player
- delivering game-specific visible state to each AI
- collecting actions within the configured deadline
- applying timeout and invalid-action policies
- delegating state transition logic to the selected game master
- persisting match results and AI stderr logs
- exposing full game state to future observer and admin interfaces

### Non-responsibilities

The platform is not responsible for:

- defining game-specific action schemas beyond transport requirements
- providing AI source build pipelines
- letting AIs communicate externally during a match

## AI Runtime Model

### Submission format

- AI submissions are WASM binaries targeting WASI.
- The platform does not compile user source code.
- A submission is versioned per player and selected at match creation time.

### Runtime constraints

- Each AI is started once at match start and remains alive until the match ends.
- AI processes may keep in-memory state across turns in the same match.
- Runtime memory limits are enforced via WASM linear memory limits.
- Turn deadlines are enforced by the platform with timeout-aware execution
  control.
- Network access is unavailable by design.
- Filesystem access is denied by default unless a future spec explicitly allows
  a bounded scratch area.

### Standard streams

- `stdin`: platform-to-AI JSON-RPC requests
- `stdout`: AI-to-platform JSON-RPC responses only
- `stderr`: free-form AI logs captured verbatim by the platform

Any non-JSON-RPC payload written to `stdout` is treated as protocol violation.

## Transport Protocol

### Envelope

All platform-to-AI requests and AI-to-platform responses use JSON-RPC 2.0.

Platform request shape:

```json
{
  "jsonrpc": "2.0",
  "id": "turn-7-player-2",
  "method": "turn",
  "params": {}
}
```

AI response shape:

```json
{
  "jsonrpc": "2.0",
  "id": "turn-7-player-2",
  "result": {}
}
```

### Request lifecycle

1. The platform sends a request with a unique `id`.
2. The AI must respond with the same `id` before the deadline.
3. A missing response by the deadline is treated as timeout.
4. A malformed response or mismatched `id` is treated as protocol error.

The platform sends requests sequentially per AI process. A game may still use
simultaneous turns by sending each player its own request for the same turn
window and waiting for all responses or deadlines before resolving the turn.

### Standard methods

The platform reserves these request methods:

- `init`: sent once at match start
- `turn`: sent once per decision point visible to that AI
- `result`: sent once at match end

Games may extend `params` but may not redefine these method names.

### `init`

Purpose:

- inform the AI of static match metadata
- provide the game identifier and game-specific setup payload

Required `params` fields:

- `match_id`
- `player_id`
- `game`
- `ruleset_version`
- `deadline_ms`
- `state`

The `state` field contains the game-specific initial visible state for that AI.

### `turn`

Purpose:

- request one decision from the AI for the current turn or action window

Required `params` fields:

- `turn`
- `visible_state`
- `legal_action_hint`
- `deadline_ms`

`legal_action_hint` is optional for games that do not precompute legal actions,
but Phase 1 game specs should include it where practical because it simplifies
AI prototyping and spectator debugging.

### `result`

Purpose:

- notify the AI that the match is complete
- provide final ranking and game-specific outcome data

Required `params` fields:

- `placement`
- `score`
- `summary`

No further requests are sent after `result`.

## Error Handling

### Timeout

If the AI fails to answer before the deadline:

- the platform records a timeout event
- the AI action for that window becomes `no_action`
- the game master resolves `no_action` according to game rules

### Malformed protocol output

Malformed JSON, invalid JSON-RPC envelopes, or responses with wrong `id` are
protocol violations. The platform records the error and treats the turn as
`no_action`.

Repeated protocol violations may trigger early match termination in future
phases, but Phase 1 specs assume the match continues unless the game master
cannot proceed.

### Invalid in-game action

If the AI returns syntactically valid JSON-RPC with a semantically invalid
action:

- the platform records the invalid action
- the game master applies the game's invalid-action policy

Each game spec must state whether invalid actions become `no_action`, are
rejected with fallback behavior, or immediately lose the round.

## Game Integration Contract

Each registered game must provide:

- a human-readable game specification
- a machine-readable action and visible-state schema definition
- a game master implementation
- a full-state export model for operators and spectators

### Game master responsibilities

The game master must implement logic for:

- creating initial match state
- producing each player's visible state
- validating actions
- applying actions to full state
- resolving simultaneous submissions when required
- determining round and match termination
- computing score and placement
- exporting full-state snapshots for observation

### Full-state export

The platform expects each game to expose a whole-state representation that is
safe for observer/admin consumption. This representation may contain hidden
information not visible to individual AIs.

The spectator API in later phases will build on this whole-state export instead
of reading internal engine memory directly.

## Turn Models

### Simultaneous-action games

Flow:

1. The platform sends a `turn` request to every active player.
2. The platform waits until every player responds or times out.
3. The game master resolves the turn using the collected action set.
4. The platform advances to the next turn.

### Sequential-turn games

Flow:

1. The platform sends a `turn` request only to the active player.
2. The game master applies the returned action or timeout fallback.
3. The platform advances the active player pointer.

The platform must support both models without changing the AI transport.

## Ranking

Each game defines its own scoring and placement rule. The platform only assumes
that a match produces:

- a per-player score value
- a final placement order or tie group
- an outcome summary suitable for logs and future ranking systems

Phase 1 does not standardize cross-game ladder aggregation.

## Logging and Observability

- AI `stderr` is stored with player, match, and timestamp metadata.
- Match execution records include timeouts, invalid actions, and protocol
  violations.
- Whole-state snapshots are retained at least at turn boundaries.

This data is required for:

- AI debugging
- spectator playback
- operator incident analysis

## Deferred Items

The following are intentionally deferred beyond Phase 1:

- concrete spectator API transport
- matchmaking and tournament orchestration
- persistent rating systems
- plugin packaging/discovery for third-party games
- hot-swapping AI versions mid-match
