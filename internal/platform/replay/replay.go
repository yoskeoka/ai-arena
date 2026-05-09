package replay

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
	"github.com/yoskeoka/ai-arena/internal/platform/registry"
)

func LoadRecord(path string) (match.Record, error) {
	// #nosec G304 -- the caller explicitly selects the local debug record input path.
	data, err := os.ReadFile(path)
	if err != nil {
		return match.Record{}, fmt.Errorf("read record input %s: %w", path, err)
	}
	var record match.Record
	if err := json.Unmarshal(data, &record); err != nil {
		return match.Record{}, fmt.Errorf("decode record input %s: %w", path, err)
	}
	return record, nil
}

func LoadSnapshot(path string) (game.Snapshot, error) {
	// #nosec G304 -- the caller explicitly selects the local debug snapshot input path.
	data, err := os.ReadFile(path)
	if err != nil {
		return game.Snapshot{}, fmt.Errorf("read snapshot input %s: %w", path, err)
	}
	var snapshot game.Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return game.Snapshot{}, fmt.Errorf("decode snapshot input %s: %w", path, err)
	}
	return snapshot, nil
}

func LoadHistory(path string) ([]match.Event, error) {
	// #nosec G304 -- the caller explicitly selects the local debug history input path.
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read history input %s: %w", path, err)
	}
	var events []match.Event
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, fmt.Errorf("decode history input %s: %w", path, err)
	}
	return events, nil
}

func SnapshotFromHistory(meta catalog.GameMetadata, players []game.Player, events []match.Event, targetTurn int) (game.Snapshot, error) {
	return SnapshotFromHistoryWithRegistry(registry.Default(), meta, players, events, targetTurn)
}

func SnapshotFromHistoryWithRegistry(reg *registry.Registry, meta catalog.GameMetadata, players []game.Player, events []match.Event, targetTurn int) (game.Snapshot, error) {
	return SnapshotFromHistoryWithBuildSpec(reg, meta, registry.BuildSpec{
		GameVersion: meta.GameVersion,
		Ruleset:     meta.RulesetVersion,
		Players:     append([]game.Player(nil), players...),
	}, events, targetTurn)
}

func SnapshotFromHistoryWithBuildSpec(reg *registry.Registry, meta catalog.GameMetadata, spec registry.BuildSpec, events []match.Event, targetTurn int) (game.Snapshot, error) {
	descriptor, err := reg.LookupVersion(context.Background(), meta.GameID, meta.GameVersion)
	if err != nil {
		return game.Snapshot{}, err
	}
	return descriptor.SnapshotFromHistory(spec, events, targetTurn)
}
