package service

import (
	"context"
	"os"
	"testing"

	"github.com/yoskeoka/ai-arena/artifactbundle"
)

func TestLocalDryRunCheckerAcceptsStoredAIArtifact(t *testing.T) {
	store, err := NewFilesystemBundleStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := artifactbundle.Read(aiBundle(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put(context.Background(), bundle); err != nil {
		t.Fatal(err)
	}
	dryRun, err := NewLocalDryRunChecker(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	dryRun.WithBundleStore(store)
	validator, err := NewDefaultAdmissionValidator(nil, dryRun)
	if err != nil {
		t.Fatal(err)
	}
	submission := testSubmission("")
	submission.Players[0].ArtifactID = bundle.Digest
	if err := validator.Validate(context.Background(), submission); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestMaterializedPlayerSessionRemovesDirectoryAndUsesDigestIdentity(t *testing.T) {
	store, err := NewFilesystemBundleStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := artifactbundle.Read(aiBundle(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put(context.Background(), bundle); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", t.TempDir())
	invoker, err := NewLocalRunnerInvoker(repoRoot(t), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	invoker.WithBundleStore(store)
	players, sessions, err := invoker.loadPlayersAndSessions(context.Background(), MatchSubmission{
		Game:    testSubmission("").Game,
		Players: []SubmittedPlayer{{PlayerID: "p1", ArtifactID: bundle.Digest}},
	})
	if err != nil {
		t.Fatalf("loadPlayersAndSessions() error = %v", err)
	}
	if players[0].AIID != bundle.Digest {
		t.Fatalf("players[0].AIID = %q, want %q", players[0].AIID, bundle.Digest)
	}
	entries, err := os.ReadDir(os.Getenv("TMPDIR"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary dirs before close = %v, %v; want one", entries, err)
	}
	closeSessions(sessions)
	entries, err = os.ReadDir(os.Getenv("TMPDIR"))
	if err != nil || len(entries) != 0 {
		t.Fatalf("temporary dirs after close = %v, %v; want none", entries, err)
	}
}
