package service

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/yoskeoka/ai-arena/internal/platform/contract"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/gamemaster"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
	"github.com/yoskeoka/ai-arena/internal/platform/registry"
)

func TestDefaultAdmissionValidatorUsesExactGameArtifact(t *testing.T) {
	reg := newAdmissionTestRegistry(t, registry.DescriptorRecord{
		RegistryKey: registry.RegistryKey{GameID: "admitted-game", GameVersionMajor: 2},
		GameID:      "admitted-game",
		GameVersion: "2.1.0",
		ArtifactID:  "sha256:game",
		BuildMode:   registry.BuildModeInProcess,
		BuilderID:   "test-builder",
		BuildConstraints: registry.BuildConstraints{
			SupportedRulesets: []string{"regular"},
		},
	})
	dryRun := &recordingDryRunChecker{}
	validator, err := NewDefaultAdmissionValidator(reg, dryRun)
	if err != nil {
		t.Fatalf("NewDefaultAdmissionValidator: %v", err)
	}

	if err := validator.Validate(context.Background(), admissionTestSubmission("sha256:game")); err != nil {
		t.Fatalf("Validate(valid): %v", err)
	}
	if dryRun.calls != 1 {
		t.Fatalf("dry-run calls = %d, want 1", dryRun.calls)
	}
}

func TestDefaultAdmissionValidatorRejectsArtifactMismatchBeforeQueue(t *testing.T) {
	reg := newAdmissionTestRegistry(t, registry.DescriptorRecord{
		RegistryKey: registry.RegistryKey{GameID: "admitted-game", GameVersionMajor: 2},
		GameID:      "admitted-game",
		GameVersion: "2.1.0",
		ArtifactID:  "sha256:game",
		BuildMode:   registry.BuildModeInProcess,
		BuilderID:   "test-builder",
		BuildConstraints: registry.BuildConstraints{
			SupportedRulesets: []string{"regular"},
		},
	})
	dryRun := &recordingDryRunChecker{}
	validator, err := NewDefaultAdmissionValidator(reg, dryRun)
	if err != nil {
		t.Fatalf("NewDefaultAdmissionValidator: %v", err)
	}
	queue := NewInMemoryQueueStore()
	commands, err := NewCommandService(queue, validator)
	if err != nil {
		t.Fatalf("NewCommandService: %v", err)
	}

	cases := []struct {
		name    string
		mutate  func(*MatchSubmission)
		message string
	}{
		{
			name:    "unknown artifact does not fall back to version lookup",
			mutate:  func(submission *MatchSubmission) { submission.GameArtifactID = "sha256:missing" },
			message: `registry: unsupported artifact "sha256:missing"`,
		},
		{
			name:    "game id mismatch",
			mutate:  func(submission *MatchSubmission) { submission.Game.GameID = "other-game" },
			message: "does not match game",
		},
		{
			name:    "exact version mismatch",
			mutate:  func(submission *MatchSubmission) { submission.Game.GameVersion = "2.2.0" },
			message: "does not match game",
		},
		{
			name:    "ruleset mismatch",
			mutate:  func(submission *MatchSubmission) { submission.Game.RulesetVersion = "blitz" },
			message: "is not supported",
		},
	}
	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			submission := admissionTestSubmission("sha256:game")
			submission.RunID = fmt.Sprintf("run-rejected-%d", index)
			tc.mutate(&submission)
			if _, err := commands.Submit(context.Background(), submission); err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("Submit() error = %v, want message containing %q", err, tc.message)
			}
			records, err := queue.List(context.Background())
			if err != nil {
				t.Fatalf("queue.List(): %v", err)
			}
			if len(records) != 0 {
				t.Fatalf("queue records = %+v, want no record", records)
			}
		})
	}
	if dryRun.calls != 0 {
		t.Fatalf("dry-run calls = %d, want 0 for rejected submissions", dryRun.calls)
	}
}

type recordingDryRunChecker struct{ calls int }

func (c *recordingDryRunChecker) Check(context.Context, MatchSubmission) error {
	c.calls++
	return nil
}

func admissionTestSubmission(artifactID string) MatchSubmission {
	return MatchSubmission{
		RunID:          "run-admission",
		MatchID:        "match-admission",
		GameArtifactID: artifactID,
		Game: contract.GameMetadata{
			GameID:         "admitted-game",
			GameVersion:    "2.1.0",
			RulesetVersion: "regular",
		},
		Players:      []SubmittedPlayer{{PlayerID: "p1", ArtifactRef: "local-ai"}},
		OutputDir:    "matches",
		AttemptCount: 1,
		RunKind:      RunKindInitial,
	}
}

func newAdmissionTestRegistry(t *testing.T, records ...registry.DescriptorRecord) *registry.Registry {
	t.Helper()
	store, err := registry.NewInMemoryStore(records...)
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}
	resolver, err := registry.NewStaticResolver(map[string]registry.DescriptorBuilder{
		"test-builder": {
			BuildMode: registry.BuildModeInProcess,
			BuildConstraints: registry.BuildConstraints{
				SupportedRulesets: []string{"regular"},
			},
			BuildSession: func(registry.BuildSpec) (gamemaster.Session, error) { return nil, nil },
			BuildSessionFromSnapshot: func(registry.BuildSpec, game.Snapshot) (gamemaster.Session, error) {
				return nil, nil
			},
			SnapshotFromHistory: func(registry.BuildSpec, []match.Event, int) (game.Snapshot, error) {
				return game.Snapshot{}, nil
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
	return reg
}
