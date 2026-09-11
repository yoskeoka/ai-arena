package registry

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/yoskeoka/ai-arena/internal/games/echo"
	"github.com/yoskeoka/ai-arena/internal/games/janken"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/gamemaster"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
)

type failingExternalStore struct{ err error }

func (s failingExternalStore) Lookup(context.Context, RegistryKey) (DescriptorRecord, error) {
	return DescriptorRecord{}, s.err
}

func TestLookupFindsDescriptorByGameIDAndMajorVersion(t *testing.T) {
	descriptor, err := Lookup(janken.GameID, "2.9.4")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if descriptor.GameID != janken.GameID {
		t.Fatalf("descriptor.GameID = %q, want %q", descriptor.GameID, janken.GameID)
	}
	if descriptor.RegistryKey.GameVersionMajor != 2 {
		t.Fatalf("descriptor.RegistryKey.GameVersionMajor = %d, want 2", descriptor.RegistryKey.GameVersionMajor)
	}
	if descriptor.BuildMode != BuildModeInProcess {
		t.Fatalf("descriptor.BuildMode = %q, want %q", descriptor.BuildMode, BuildModeInProcess)
	}
	if descriptor.BuilderID != janken.BuilderIDInProcess {
		t.Fatalf("descriptor.BuilderID = %q, want %q", descriptor.BuilderID, janken.BuilderIDInProcess)
	}
}

func TestLookupRejectsUnknownGame(t *testing.T) {
	if _, err := Lookup("unknown-game", "1.0.0"); err == nil || !strings.Contains(err.Error(), `registry: unsupported game "unknown-game"`) {
		t.Fatalf("Lookup error = %v, want unsupported game", err)
	}
}

func TestLookupRejectsInvalidGameVersion(t *testing.T) {
	if _, err := Lookup(janken.GameID, "not-semver"); err == nil || !strings.Contains(err.Error(), `registry: invalid game version "not-semver": invalid semver`) {
		t.Fatalf("Lookup error = %v, want invalid semver detail", err)
	}
}

func TestLookupRejectsUnsupportedMajorForKnownGame(t *testing.T) {
	if _, err := Lookup(janken.GameID, "3.0.0"); err == nil || !strings.Contains(err.Error(), `registry: unsupported game version major 3 for game "janken"`) {
		t.Fatalf("Lookup error = %v, want unsupported major", err)
	}
}

func TestInMemoryStoreLookupSelectsLatestReleaseWithinMajor(t *testing.T) {
	store, err := NewInMemoryStore(
		DescriptorRecord{RegistryKey: RegistryKey{GameID: "test", GameVersionMajor: 2}, GameID: "test", GameVersion: "2.4.1", ArtifactID: "sha256:old", BuildMode: BuildModeInProcess, BuilderID: "test-builder", BuildConstraints: BuildConstraints{SupportedRulesets: []string{"regular"}}},
		DescriptorRecord{RegistryKey: RegistryKey{GameID: "test", GameVersionMajor: 2}, GameID: "test", GameVersion: "2.10.0", ArtifactID: "sha256:new", BuildMode: BuildModeInProcess, BuilderID: "test-builder", BuildConstraints: BuildConstraints{SupportedRulesets: []string{"regular"}}},
	)
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}
	record, err := store.Lookup(context.Background(), RegistryKey{GameID: "test", GameVersionMajor: 2})
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if record.GameVersion != "2.10.0" || record.ArtifactID != "sha256:new" {
		t.Fatalf("Lookup = %+v, want latest release", record)
	}
}

func TestTieredStorePrefersExternalReleaseWithinMajor(t *testing.T) {
	fallback, err := NewInMemoryStore(DescriptorRecord{
		RegistryKey: RegistryKey{GameID: "tiered-game", GameVersionMajor: 2},
		GameID:      "tiered-game",
		GameVersion: "2.99.0",
		ArtifactID:  "sha256:builtin",
		BuildMode:   BuildModeInProcess,
		BuilderID:   "builtin-builder",
		BuildConstraints: BuildConstraints{
			SupportedRulesets: []string{"regular"},
		},
	})
	if err != nil {
		t.Fatalf("NewInMemoryStore(fallback): %v", err)
	}
	store := newTieredStore(fallback)
	if err := store.Register(DescriptorRecord{
		RegistryKey: RegistryKey{GameID: "tiered-game", GameVersionMajor: 2},
		GameID:      "tiered-game",
		GameVersion: "2.1.0",
		ArtifactID:  "sha256:external",
		BuildMode:   BuildModeInProcess,
		BuilderID:   "external-builder",
		BuildConstraints: BuildConstraints{
			SupportedRulesets: []string{"regular"},
		},
	}); err != nil {
		t.Fatalf("Register(external): %v", err)
	}

	record, err := store.Lookup(context.Background(), RegistryKey{GameID: "tiered-game", GameVersionMajor: 2})
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if record.GameVersion != "2.1.0" || record.ArtifactID != "sha256:external" {
		t.Fatalf("Lookup = %+v, want external release", record)
	}
}

func TestTieredStoreSelectsLatestExternalReleaseAndFallsBackByKey(t *testing.T) {
	fallback, err := NewInMemoryStore(DescriptorRecord{
		RegistryKey: RegistryKey{GameID: "fallback-game", GameVersionMajor: 2},
		GameID:      "fallback-game",
		GameVersion: "2.0.0",
		ArtifactID:  "sha256:fallback",
		BuildMode:   BuildModeInProcess,
		BuilderID:   "fallback-builder",
		BuildConstraints: BuildConstraints{
			SupportedRulesets: []string{"regular"},
		},
	})
	if err != nil {
		t.Fatalf("NewInMemoryStore(fallback): %v", err)
	}
	store := newTieredStore(fallback)
	for _, version := range []string{"2.4.0", "2.10.0"} {
		if err := store.Register(DescriptorRecord{
			RegistryKey: RegistryKey{GameID: "external-game", GameVersionMajor: 2},
			GameID:      "external-game",
			GameVersion: version,
			ArtifactID:  "sha256:external-" + version,
			BuildMode:   BuildModeInProcess,
			BuilderID:   "external-builder",
			BuildConstraints: BuildConstraints{
				SupportedRulesets: []string{"regular"},
			},
		}); err != nil {
			t.Fatalf("Register(%s): %v", version, err)
		}
	}

	external, err := store.Lookup(context.Background(), RegistryKey{GameID: "external-game", GameVersionMajor: 2})
	if err != nil {
		t.Fatalf("Lookup(external): %v", err)
	}
	if external.GameVersion != "2.10.0" {
		t.Fatalf("Lookup(external).GameVersion = %q, want 2.10.0", external.GameVersion)
	}
	fallbackRecord, err := store.Lookup(context.Background(), RegistryKey{GameID: "fallback-game", GameVersionMajor: 2})
	if err != nil {
		t.Fatalf("Lookup(fallback): %v", err)
	}
	if fallbackRecord.ArtifactID != "sha256:fallback" {
		t.Fatalf("Lookup(fallback).ArtifactID = %q, want fallback release", fallbackRecord.ArtifactID)
	}
}

func TestTieredStoreLookupArtifactOnlyResolvesExternalTier(t *testing.T) {
	fallback, err := NewInMemoryStore(DescriptorRecord{
		RegistryKey: RegistryKey{GameID: "fallback-game", GameVersionMajor: 2},
		GameID:      "fallback-game",
		GameVersion: "2.0.0",
		ArtifactID:  "sha256:builtin",
		BuildMode:   BuildModeInProcess,
		BuilderID:   "fallback-builder",
		BuildConstraints: BuildConstraints{
			SupportedRulesets: []string{"regular"},
		},
	})
	if err != nil {
		t.Fatalf("NewInMemoryStore(fallback): %v", err)
	}
	store := newTieredStore(fallback)
	if err := store.Register(DescriptorRecord{
		RegistryKey: RegistryKey{GameID: "external-game", GameVersionMajor: 2},
		GameID:      "external-game",
		GameVersion: "2.1.0",
		ArtifactID:  "sha256:external",
		BuildMode:   BuildModeInProcess,
		BuilderID:   "external-builder",
		BuildConstraints: BuildConstraints{
			SupportedRulesets: []string{"regular"},
		},
	}); err != nil {
		t.Fatalf("Register(external): %v", err)
	}

	if record, err := store.LookupArtifact(context.Background(), "sha256:external"); err != nil || record.ArtifactID != "sha256:external" {
		t.Fatalf("LookupArtifact(external) = %+v, %v", record, err)
	}
	if _, err := store.LookupArtifact(context.Background(), "sha256:builtin"); err == nil || !strings.Contains(err.Error(), `registry: unsupported artifact "sha256:builtin"`) {
		t.Fatalf("LookupArtifact(builtin) error = %v, want external-tier rejection", err)
	}
}

func TestExternalPrimaryStoreDoesNotFallbackOnPrimaryFailure(t *testing.T) {
	fallback, err := NewInMemoryStore(DescriptorRecord{RegistryKey: RegistryKey{GameID: "fallback", GameVersionMajor: 1}, GameID: "fallback", BuildMode: BuildModeInProcess, BuilderID: "fallback", BuildConstraints: BuildConstraints{SupportedRulesets: []string{"regular"}}})
	if err != nil {
		t.Fatal(err)
	}
	want := errors.New("database unavailable")
	store, err := NewExternalPrimaryStore(failingExternalStore{err: want}, fallback)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Lookup(context.Background(), RegistryKey{GameID: "fallback", GameVersionMajor: 1})
	if !errors.Is(err, want) {
		t.Fatalf("Lookup error = %v, want primary failure", err)
	}
}

func TestDescriptorBuildSessionReturnsRulesetError(t *testing.T) {
	descriptor, err := Lookup(echo.GameID, echo.GameVersion)
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	_, err = descriptor.BuildSession(BuildSpec{
		GameVersion: echo.GameVersion,
		Ruleset:     "missing-ruleset",
		Players:     []game.Player{{PlayerID: "p1"}},
	})
	if err == nil || !strings.Contains(err.Error(), `unsupported ruleset "missing-ruleset"`) {
		t.Fatalf("BuildSession error = %v, want unsupported ruleset", err)
	}
}

func TestInMemoryStoreRejectsMissingBuildMode(t *testing.T) {
	_, err := NewInMemoryStore(DescriptorRecord{
		RegistryKey: RegistryKey{GameID: "test", GameVersionMajor: 1},
		GameID:      "test",
		BuilderID:   "test/in-process",
		BuildConstraints: BuildConstraints{
			SupportedRulesets: []string{"regular"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "registry: BuildMode is required") {
		t.Fatalf("NewInMemoryStore error = %v, want BuildMode required", err)
	}
}

func TestLookupRejectsUnknownBuilderID(t *testing.T) {
	registry := newTestRegistry(t,
		[]DescriptorRecord{{
			RegistryKey: RegistryKey{GameID: "test", GameVersionMajor: 1},
			GameID:      "test",
			BuildMode:   BuildModeInProcess,
			BuilderID:   "missing-builder",
			BuildConstraints: BuildConstraints{
				SupportedRulesets: []string{"regular"},
			},
		}},
		map[string]DescriptorBuilder{},
	)

	_, err := registry.Lookup(context.Background(), RegistryKey{GameID: "test", GameVersionMajor: 1})
	if err == nil || !strings.Contains(err.Error(), `registry: unknown builder_id "missing-builder"`) {
		t.Fatalf("Lookup error = %v, want unknown builder_id", err)
	}
}

func TestLookupRejectsIncompatibleBuildMetadata(t *testing.T) {
	registry := newTestRegistry(t,
		[]DescriptorRecord{{
			RegistryKey: RegistryKey{GameID: "test", GameVersionMajor: 1},
			GameID:      "test",
			BuildMode:   BuildModeLocalSubprocess,
			BuilderID:   "test/in-process",
			BuildConstraints: BuildConstraints{
				SupportedRulesets: []string{"regular"},
			},
		}},
		map[string]DescriptorBuilder{
			"test/in-process": stubDescriptorBuilder(BuildModeInProcess, []string{"regular"}),
		},
	)

	_, err := registry.Lookup(context.Background(), RegistryKey{GameID: "test", GameVersionMajor: 1})
	if err == nil || !strings.Contains(err.Error(), `registry: incompatible build metadata for builder_id "test/in-process"`) {
		t.Fatalf("Lookup error = %v, want incompatible build metadata", err)
	}
}

func TestLookupEchoSubprocessRegistersAsSeparateGame(t *testing.T) {
	descriptor, err := Lookup(echo.SubprocessGameID, echo.GameVersion)
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if descriptor.GameID != echo.SubprocessGameID {
		t.Fatalf("descriptor.GameID = %q, want %q", descriptor.GameID, echo.SubprocessGameID)
	}
	if descriptor.BuildMode != BuildModeLocalSubprocess {
		t.Fatalf("descriptor.BuildMode = %q, want %q", descriptor.BuildMode, BuildModeLocalSubprocess)
	}
	if descriptor.BuilderID != echo.BuilderIDLocalSubprocess {
		t.Fatalf("descriptor.BuilderID = %q, want %q", descriptor.BuilderID, echo.BuilderIDLocalSubprocess)
	}
}

func TestEchoSubprocessSnapshotUsesSubprocessGameID(t *testing.T) {
	descriptor, err := Lookup(echo.GameID, echo.GameVersion)
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	snapshot, err := descriptor.SnapshotFromHistory(BuildSpec{
		GameVersion: echo.GameVersion,
		Ruleset:     echo.RulesetSimultaneous2Turn,
		Players:     []game.Player{{PlayerID: "p1"}},
	}, nil, 0)
	if err != nil {
		t.Fatalf("SnapshotFromHistory: %v", err)
	}
	if snapshot.GameID != echo.GameID {
		t.Fatalf("snapshot.GameID = %q, want %q", snapshot.GameID, echo.GameID)
	}

	descriptor, err = Lookup(echo.SubprocessGameID, echo.GameVersion)
	if err != nil {
		t.Fatalf("Lookup subprocess: %v", err)
	}
	snapshot, err = descriptor.SnapshotFromHistory(BuildSpec{
		GameVersion: echo.GameVersion,
		Ruleset:     echo.RulesetSimultaneous2Turn,
		Players:     []game.Player{{PlayerID: "p1"}},
	}, nil, 0)
	if err != nil {
		t.Fatalf("SnapshotFromHistory subprocess: %v", err)
	}
	if snapshot.GameID != echo.SubprocessGameID {
		t.Fatalf("snapshot.GameID = %q, want %q", snapshot.GameID, echo.SubprocessGameID)
	}
}

func newTestRegistry(t *testing.T, records []DescriptorRecord, builders map[string]DescriptorBuilder) *Registry {
	t.Helper()

	store, err := NewInMemoryStore(records...)
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}
	resolver, err := NewStaticResolver(builders)
	if err != nil {
		t.Fatalf("NewStaticResolver: %v", err)
	}
	registry, err := New(store, resolver)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return registry
}

func stubDescriptorBuilder(mode BuildMode, rulesets []string) DescriptorBuilder {
	return DescriptorBuilder{
		BuildMode: mode,
		BuildConstraints: BuildConstraints{
			SupportedRulesets: append([]string(nil), rulesets...),
		},
		BuildSession: func(BuildSpec) (gamemaster.Session, error) {
			return nil, nil
		},
		BuildSessionFromSnapshot: func(BuildSpec, game.Snapshot) (gamemaster.Session, error) {
			return nil, nil
		},
		SnapshotFromHistory: func(BuildSpec, []match.Event, int) (game.Snapshot, error) {
			return game.Snapshot{}, nil
		},
	}
}
