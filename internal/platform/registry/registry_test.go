package registry

import (
	"strings"
	"testing"

	"github.com/yoskeoka/ai-arena/internal/games/echo"
	"github.com/yoskeoka/ai-arena/internal/games/janken"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/gamemaster"
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

func TestRegisterRejectsMissingBuildMode(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	err = r.Register(GameDescriptor{
		RegistryKey: RegistryKey{GameID: "test", GameVersionMajor: 1},
		GameID:      "test",
		BuildSession: func(BuildSpec) (gamemaster.Session, error) {
			return nil, nil
		},
		BuildSessionFromSnapshot: func(BuildSpec, game.Snapshot) (gamemaster.Session, error) {
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
