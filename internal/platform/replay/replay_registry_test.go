package replay

import (
	"testing"

	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/gamemaster"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
	"github.com/yoskeoka/ai-arena/internal/platform/registry"
)

func TestSnapshotFromHistoryWithRegistryUsesInjectedRegistry(t *testing.T) {
	store, err := registry.NewInMemoryStore(registry.DescriptorRecord{
		RegistryKey: registry.RegistryKey{GameID: "test", GameVersionMajor: 1},
		GameID:      "test",
		BuildMode:   registry.BuildModeInProcess,
		BuilderID:   "test/in-process",
		BuildConstraints: registry.BuildConstraints{
			SupportedRulesets: []string{"regular"},
		},
	})
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}
	resolver, err := registry.NewStaticResolver(map[string]registry.DescriptorBuilder{
		"test/in-process": {
			BuildMode: registry.BuildModeInProcess,
			BuildConstraints: registry.BuildConstraints{
				SupportedRulesets: []string{"regular"},
			},
			BuildSession: func(registry.BuildSpec) (gamemaster.Session, error) {
				return nil, nil
			},
			BuildSessionFromSnapshot: func(registry.BuildSpec, game.Snapshot) (gamemaster.Session, error) {
				return nil, nil
			},
			SnapshotFromHistory: func(spec registry.BuildSpec, _ []match.Event, targetTurn int) (game.Snapshot, error) {
				return game.Snapshot{
					GameID:         "test",
					GameVersion:    spec.GameVersion,
					RulesetVersion: spec.Ruleset,
					Turn:           targetTurn,
				}, nil
			},
		},
	})
	if err != nil {
		t.Fatalf("NewStaticResolver: %v", err)
	}
	reg, err := registry.New(store, resolver)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	snapshot, err := SnapshotFromHistoryWithRegistry(reg, catalog.GameMetadata{
		GameID:         "test",
		GameVersion:    "1.2.3",
		RulesetVersion: "regular",
	}, []game.Player{{PlayerID: "p1"}}, nil, 4)
	if err != nil {
		t.Fatalf("SnapshotFromHistoryWithRegistry: %v", err)
	}
	if snapshot.Turn != 4 {
		t.Fatalf("snapshot.Turn = %d, want 4", snapshot.Turn)
	}
	if snapshot.GameID != "test" {
		t.Fatalf("snapshot.GameID = %q, want test", snapshot.GameID)
	}
}
