// Package dungeon implements the fixed-map dungeon game domain.
//
// The package is kept in this monorepo for convenience, but it is intended to
// remain movable to its own repository. Keep dungeon-specific code free of
// ai-arena internal dependencies so the package and its sidecars can move
// without bringing platform internals with them.
//
// Deterministic replay is a hard requirement. Randomness-consuming game-master
// logic should stay sequential, avoid runtime-dependent ordering such as map
// iteration, and consume randomness from a single wrapped source.
package dungeon
