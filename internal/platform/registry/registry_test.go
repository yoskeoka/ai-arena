package registry

import (
	"strings"
	"testing"

	"github.com/yoskeoka/ai-arena/internal/games/echo"
	"github.com/yoskeoka/ai-arena/internal/games/janken"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
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
	if _, err := Lookup("unknown-game", "1.0.0"); err == nil || !strings.Contains(err.Error(), `unsupported game "unknown-game"`) {
		t.Fatalf("Lookup error = %v, want unsupported game", err)
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
