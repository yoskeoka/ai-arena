package contract

import "github.com/yoskeoka/ai-arena/gamemaster"

// Placement records one player's final placement.
type Placement = gamemaster.Placement

// MatchResult is the persisted placement summary for a match.
type MatchResult = gamemaster.MatchResult

// PlayerSnapshot stores the player-scoped portion of a match snapshot.
type PlayerSnapshot = gamemaster.PlayerSnapshot

// Snapshot stores the full internal match state used for replay and resume.
type Snapshot = gamemaster.Snapshot

// ExportedPlayerSnapshot stores the public per-player snapshot fields.
type ExportedPlayerSnapshot = gamemaster.ExportedPlayerSnapshot

// ExportedSnapshot stores the public snapshot fields safe to expose externally.
type ExportedSnapshot = gamemaster.ExportedSnapshot
