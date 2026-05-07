package registry

import (
	"strings"
	"testing"

	"github.com/yoskeoka/ai-arena/internal/games/echo"
	"github.com/yoskeoka/ai-arena/internal/games/janken"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
)

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

func TestDescriptorBuildFreshReturnsRulesetError(t *testing.T) {
	descriptor, err := Lookup(echo.GameID, echo.GameVersion)
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	_, err = descriptor.BuildFresh(BuildSpec{
		GameVersion: echo.GameVersion,
		Ruleset:     "missing-ruleset",
		Players:     []game.Player{{PlayerID: "p1"}},
	})
	if err == nil || !strings.Contains(err.Error(), `unsupported ruleset "missing-ruleset"`) {
		t.Fatalf("BuildFresh error = %v, want unsupported ruleset", err)
	}
}

func TestRegisterRejectsMissingBuildMode(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	err = r.Register(GameDescriptor{
		RegistryKey: RegistryKey{GameID: "test", GameVersionMajor: 1},
		GameID:      "test",
		BuildFresh: func(BuildSpec) (game.Master, error) {
			return nil, nil
		},
		BuildFromSnapshot: func(BuildSpec, game.Snapshot) (game.Master, error) {
			return nil, nil
		},
		SnapshotFromHistory: func(BuildSpec, []match.Event, int) (game.Snapshot, error) {
			return game.Snapshot{}, nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "registry: BuildMode is required") {
		t.Fatalf("Register error = %v, want BuildMode required", err)
	}
}
