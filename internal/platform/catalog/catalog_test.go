package catalog

import (
	"errors"
	"testing"
)

func TestValidateMetadata(t *testing.T) {
	meta := GameMetadata{
		GameID:         "janken",
		GameVersion:    "2.1.0",
		RulesetVersion: "phase2",
		TurnMode:       "simultaneous",
	}
	if err := ValidateMetadata(meta); err != nil {
		t.Fatalf("ValidateMetadata: %v", err)
	}
}

func TestCompatibleUsesGameVersionMajorAndRuleset(t *testing.T) {
	expected := GameMetadata{
		GameID:         "janken",
		GameVersion:    "2.1.0",
		RulesetVersion: "phase2",
		TurnMode:       "simultaneous",
	}
	actual := GameMetadata{
		GameID:         "janken",
		GameVersion:    "2.9.4",
		RulesetVersion: "phase2",
		TurnMode:       "simultaneous",
	}
	if err := Compatible(expected, actual); err != nil {
		t.Fatalf("Compatible: %v", err)
	}

	actual.GameVersion = "3.0.0"
	if err := Compatible(expected, actual); !errors.Is(err, ErrIncompatibleMetadata) {
		t.Fatalf("expected ErrIncompatibleMetadata for major mismatch, got %v", err)
	}

	actual.GameVersion = "2.0.0"
	actual.RulesetVersion = "phase3"
	if err := Compatible(expected, actual); !errors.Is(err, ErrIncompatibleMetadata) {
		t.Fatalf("expected ErrIncompatibleMetadata for ruleset mismatch, got %v", err)
	}
}
