# Janken Game Specification

## Purpose

Janken is the Phase 2 proving game for the platform. It is intentionally small
enough to validate the core platform loop before implementing the more complex
dungeon game.

The game is designed to verify:

- simultaneous-action turn handling
- hidden opponent choices until turn resolution
- timeout behavior
- invalid-action handling
- multi-round ranking based on win rate

## Match Format

- Players: 2 or more
- Turn model: simultaneous action
- Round count: fixed `N`, configured before match start
- Allowed hands: `rock`, `paper`, `scissors`
- Match end: after round `N`

## Round Resolution

Every active player submits one hand per round.

### Outcome rules

- `rock` beats `scissors`
- `scissors` beats `paper`
- `paper` beats `rock`

### Multi-player interpretation

For a round with more than two players:

- if all submitted hands are identical, the round is a draw for all players
- if all three hands appear, the round is a draw for all players
- if exactly two hand types appear, players using the winning hand receive one
  round win and players using the losing hand receive one round loss

No player is eliminated during the match.

## Timeout and Invalid Actions

- Timeout action: treated as `no_action`
- Invalid action: treated as `no_action`
- `no_action` loses to any valid hand present in the round
- if every player submits `no_action`, the round is a draw for all players

This policy makes timeout handling visible in the final ranking without adding a
special penalty subsystem.

## Scoring and Ranking

Each player tracks:

- `wins`
- `losses`
- `draws`
- `timeouts`
- `invalid_actions`

Primary ranking metric:

- win rate = `wins / N`

Tie-breakers in order:

1. fewer losses
2. fewer timeouts
3. fewer invalid actions
4. tied placement

## Visible Information Model

### Initial information

At `init`, every player receives:

- `game`: `janken`
- `player_id`
- `players`: ordered list of player IDs
- `rounds`: total configured round count
- `deadline_ms`

### Per-round information

At each `turn`, every player receives:

- `round`
- `rounds`
- `self_history`
- `public_history`
- `legal_action_hint`

`self_history` contains the player's own past hands and outcomes.

`public_history` contains fully resolved past rounds. It never includes the
current round's pending submissions, so simultaneous hidden choice is preserved.

`legal_action_hint` is always:

```json
["rock", "paper", "scissors"]
```

## Action Schema

AI response `result` for `turn`:

```json
{
  "action": "rock"
}
```

Allowed `action` values:

- `rock`
- `paper`
- `scissors`

Any other value is invalid.

## JSON-RPC Examples

### `init`

Request:

```json
{
  "jsonrpc": "2.0",
  "id": "init",
  "method": "init",
  "params": {
    "match_id": "match-001",
    "player_id": "p1",
    "game": "janken",
    "ruleset_version": "phase1",
    "deadline_ms": 100,
    "state": {
      "players": ["p1", "p2"],
      "rounds": 5
    }
  }
}
```

Response:

```json
{
  "jsonrpc": "2.0",
  "id": "init",
  "result": {
    "ready": true
  }
}
```

### `turn`

Request:

```json
{
  "jsonrpc": "2.0",
  "id": "round-3",
  "method": "turn",
  "params": {
    "turn": 3,
    "visible_state": {
      "round": 3,
      "rounds": 5,
      "self_history": [
        {"round": 1, "action": "rock", "outcome": "win"},
        {"round": 2, "action": "paper", "outcome": "draw"}
      ],
      "public_history": [
        {
          "round": 1,
          "actions": {"p1": "rock", "p2": "scissors"},
          "outcomes": {"p1": "win", "p2": "loss"}
        },
        {
          "round": 2,
          "actions": {"p1": "paper", "p2": "paper"},
          "outcomes": {"p1": "draw", "p2": "draw"}
        }
      ]
    },
    "legal_action_hint": ["rock", "paper", "scissors"],
    "deadline_ms": 100
  }
}
```

Response:

```json
{
  "jsonrpc": "2.0",
  "id": "round-3",
  "result": {
    "action": "scissors"
  }
}
```

### `result`

Request:

```json
{
  "jsonrpc": "2.0",
  "id": "result",
  "method": "result",
  "params": {
    "placement": 1,
    "score": {
      "wins": 3,
      "losses": 1,
      "draws": 1,
      "win_rate": 0.6
    },
    "summary": {
      "players": 2,
      "rounds": 5,
      "tie_breakers_applied": []
    }
  }
}
```

## Spectator-Facing Whole State

The whole-state export for a janken match must include:

- configured round count
- current round number
- all player submissions for resolved rounds
- round outcomes
- cumulative score table
- pending players for the active round

This is sufficient for a future spectator UI showing round-by-round reveals.

## Why This Game Exists

Janken is not meant to be strategically deep enough for the final platform. It
exists because it is the smallest game that still exercises simultaneous hidden
actions, repeated rounds, ranking, and deadline behavior under the shared AI
protocol.
