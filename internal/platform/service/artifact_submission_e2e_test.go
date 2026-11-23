package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/yoskeoka/ai-arena/artifactbundle"
	"github.com/yoskeoka/ai-arena/gamemaster"
	"github.com/yoskeoka/ai-arena/internal/platform/contract"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
	"github.com/yoskeoka/ai-arena/internal/platform/registry"
	"github.com/yoskeoka/ai-arena/internal/platform/replay"
)

func TestArtifactSubmissionUploadToWASIStartAcrossBundleStores(t *testing.T) {
	gameBytes := buildWASIBundle(t, "./cmd/janken-gamemaster", `{"schema_version":"arena-bundle/v1","artifact_kind":"game","game_id":"janken","game_version":"2.1.0","rulesets":[{"ruleset_version":"regular","player_count":2,"max_active_bots_per_owner":2}],"runtime":{"kind":"wasm-wasi","module":"module.wasm","memory_limit_pages":1024}}`)
	aiBytes := buildWASIBundle(t, "./testdata/ai/janken/janken-go-wasm-ai", `{"schema_version":"arena-bundle/v1","artifact_kind":"ai","ai_id":"janken-go-wasm-ai","game_id":"janken","game_version":"2.1.0","runtime":{"kind":"wasm-wasi","module":"module.wasm","memory_limit_pages":1024}}`)

	for _, backend := range []struct {
		name  string
		store func(*testing.T) BundleStore
	}{
		{name: "filesystem", store: newFilesystemTestBundleStore},
		{name: "s3-compatible", store: newS3TestBundleStore},
	} {
		t.Run(backend.name, func(t *testing.T) {
			runArtifactSubmissionProof(t, backend.store(t), gameBytes, aiBytes)
		})
	}
}

func newFilesystemTestBundleStore(t *testing.T) BundleStore {
	t.Helper()
	store, err := NewFilesystemBundleStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func newS3TestBundleStore(t *testing.T) BundleStore {
	t.Helper()
	artifactStore, shutdown := newTestS3ArtifactStore(t)
	t.Cleanup(shutdown)
	store, err := NewS3BundleStore(artifactStore)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func runArtifactSubmissionProof(t *testing.T, bundles BundleStore, gameBytes, aiBytes []byte) {
	t.Helper()
	ctx := context.Background()
	gameBundle, err := artifactbundle.Read(gameBytes)
	if err != nil {
		t.Fatal(err)
	}
	aiBundle, err := artifactbundle.Read(aiBytes)
	if err != nil {
		t.Fatal(err)
	}

	reg, err := registry.NewWASIOverlay(bundles)
	if err != nil {
		t.Fatal(err)
	}
	admission, err := NewArtifactAdmissionService(bundles, reg)
	if err != nil {
		t.Fatal(err)
	}
	gameRelease, err := admission.RegisterGameBundle(ctx, gameBytes)
	if err != nil {
		t.Fatalf("RegisterGameBundle() error = %v", err)
	}
	if gameRelease.ArtifactID != gameBundle.Digest {
		t.Fatalf("game artifact_id = %q, want %q", gameRelease.ArtifactID, gameBundle.Digest)
	}
	if gameRelease.MemoryLimitPages != gameBundle.Manifest.Runtime.MemoryLimitPages {
		t.Fatalf("game memory_limit_pages = %d, want %d", gameRelease.MemoryLimitPages, gameBundle.Manifest.Runtime.MemoryLimitPages)
	}

	general, err := NewGeneralSubmissionService(repoRoot(t), reg, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	general.WithBundleStore(bundles)
	game, err := general.RegisterGame(ctx, GameRegistrationRequest{ArtifactID: gameBundle.Digest, RulesetVersion: "regular"})
	if err != nil {
		t.Fatalf("RegisterGame() error = %v", err)
	}
	if game.ArtifactID != gameBundle.Digest || game.Game.GameID != gameBundle.Manifest.GameID || game.Game.GameVersion != gameBundle.Manifest.GameVersion {
		t.Fatalf("registered game = %+v, want selected artifact manifest identity", game)
	}
	ai1, err := general.RegisterAIBundle(ctx, AISubmissionRequest{AISubmissionID: "ai-1", GameRegistrationID: game.RegistrationID, DisplayName: "Echo 1"}, aiBytes)
	if err != nil {
		t.Fatalf("RegisterAIBundle(ai1) error = %v", err)
	}
	ai2, err := general.RegisterAIBundle(ctx, AISubmissionRequest{AISubmissionID: "ai-2", GameRegistrationID: game.RegistrationID, DisplayName: "Echo 2"}, aiBytes)
	if err != nil {
		t.Fatalf("RegisterAIBundle(ai2) error = %v", err)
	}
	if ai1.ArtifactID != aiBundle.Digest || ai2.ArtifactID != aiBundle.Digest {
		t.Fatalf("AI artifact IDs = %q, %q; want %q", ai1.ArtifactID, ai2.ArtifactID, aiBundle.Digest)
	}
	for _, bundle := range []artifactbundle.Bundle{gameBundle, aiBundle} {
		stored, readErr := bundles.Read(ctx, bundle.Digest)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if !bytes.Equal(stored, bundle.Bytes) {
			t.Fatalf("stored bytes for %s differ from uploaded bytes", bundle.Digest)
		}
	}

	queue := NewInMemoryQueueStore()
	dryRun, err := NewLocalDryRunChecker(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	dryRun.WithBundleStore(bundles)
	validator, err := NewDefaultAdmissionValidator(reg, dryRun)
	if err != nil {
		t.Fatal(err)
	}
	commands, err := NewCommandService(queue, validator)
	if err != nil {
		t.Fatal(err)
	}
	requests, err := NewMatchRequestService(general, commands, queue, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, queued, err := requests.Create(ctx, MatchRequestCreateRequest{
		RequestID:          "request-1",
		GameRegistrationID: game.RegistrationID,
		MatchID:            "artifact-submission-match",
		OutputDir:          t.TempDir(),
		Participants: []MatchRequestParticipant{
			{PlayerID: "p1", AISubmissionID: ai1.AISubmissionID},
			{PlayerID: "p2", AISubmissionID: ai2.AISubmissionID},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if queued.Submission.GameArtifactID != gameBundle.Digest {
		t.Fatalf("queued game artifact_id = %q, want %q", queued.Submission.GameArtifactID, gameBundle.Digest)
	}

	tmpDir := t.TempDir()
	t.Setenv("TMPDIR", tmpDir)
	before := bundleMaterializationDirs(t, tmpDir)
	invoker, err := NewLocalRunnerInvoker(repoRoot(t), reg, 0)
	if err != nil {
		t.Fatal(err)
	}
	invoker.WithBundleStore(bundles)
	worker, err := NewWorker(queue, invoker, LocalTerminalPersister{})
	if err != nil {
		t.Fatal(err)
	}
	completed, err := worker.ProcessNext(ctx, "worker-1")
	if err != nil {
		t.Fatalf("ProcessNext() error = %v", err)
	}
	if completed.State != StateCompleted || completed.Terminal == nil || completed.Terminal.MatchStatus != contract.StatusCompleted {
		t.Fatalf("completed record = %+v, want completed WASI run", completed)
	}
	if completed.Submission.GameArtifactID != gameBundle.Digest {
		t.Fatalf("completed game artifact_id = %q, want %q", completed.Submission.GameArtifactID, gameBundle.Digest)
	}
	if completed.Submission.Players[0].ArtifactID != aiBundle.Digest || completed.Submission.Players[1].ArtifactID != aiBundle.Digest {
		t.Fatalf("queued AI digests = %+v, want %q", completed.Submission.Players, aiBundle.Digest)
	}
	assertBundleRunRecord(t, completed.Terminal.RecordPath, aiBundle.Digest)
	if after := bundleMaterializationDirs(t, tmpDir); !samePaths(before, after) {
		t.Fatalf("materialization directories leaked: before=%v after=%v", before, after)
	}
}

func buildWASIBundle(t *testing.T, packagePath, manifest string) []byte {
	t.Helper()
	modulePath := filepath.Join(t.TempDir(), "module.wasm")
	cmd := exec.CommandContext(context.Background(), "go", "build", "-o", modulePath, packagePath)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm", "GOCACHE="+filepath.Join(t.TempDir(), "go-build"))
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build WASI module %s: %v\n%s", packagePath, err, output)
	}
	module, err := os.ReadFile(modulePath)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	for _, entry := range []struct {
		name string
		data []byte
	}{{name: "manifest.json", data: []byte(manifest)}, {name: "module.wasm", data: module}} {
		file, createErr := writer.CreateHeader(&zip.FileHeader{Name: entry.name, Method: zip.Store})
		if createErr != nil {
			t.Fatal(createErr)
		}
		if _, writeErr := file.Write(entry.data); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func assertBundleRunRecord(t *testing.T, recordPath, aiDigest string) {
	t.Helper()
	data, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	var record match.Record
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	if record.Status != contract.StatusCompleted {
		t.Fatalf("record status = %q, want completed", record.Status)
	}
	if len(record.Players) != 2 {
		t.Fatalf("record players = %+v, want two", record.Players)
	}
	for _, player := range record.Players {
		if player.AIID != aiDigest {
			t.Fatalf("player %q AIID = %q, want %q", player.PlayerID, player.AIID, aiDigest)
		}
	}
}

func bundleMaterializationDirs(t *testing.T, tmpDir string) map[string]struct{} {
	t.Helper()
	paths := make(map[string]struct{})
	for _, pattern := range []string{"ai-arena-game-*", "ai-arena-ai-*"} {
		matches, err := filepath.Glob(filepath.Join(tmpDir, pattern))
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range matches {
			paths[path] = struct{}{}
		}
	}
	return paths
}

func samePaths(left, right map[string]struct{}) bool {
	if len(left) != len(right) {
		return false
	}
	for path := range left {
		if _, ok := right[path]; !ok {
			return false
		}
	}
	return true
}

func TestArtifactBundleRulesetSelection(t *testing.T) {
	ctx := context.Background()
	t.Setenv(gamemaster.WASIRulesetVersionEnv, "host-only")
	gameBytes := buildWASIBundle(t, "./cmd/echo-count-gamemaster", `{"schema_version":"arena-bundle/v1","artifact_kind":"game","game_id":"echo-count","game_version":"2.0.0","rulesets":[{"ruleset_version":"phase2-simultaneous-3turn","player_count":2,"max_active_bots_per_owner":2},{"ruleset_version":"phase2-simultaneous-2turn","player_count":2,"max_active_bots_per_owner":2}],"runtime":{"kind":"wasm-wasi","module":"module.wasm","memory_limit_pages":1024,"args":["module.wasm","--game-id","echo-count","--game-version","2.0.0"]}}`)
	aiBytes := buildWASIBundle(t, "./testdata/ai/echo/echo-ai", `{"schema_version":"arena-bundle/v1","artifact_kind":"ai","ai_id":"echo-ai","game_id":"echo-count","game_version":"2.0.0","runtime":{"kind":"wasm-wasi","module":"module.wasm","memory_limit_pages":1024}}`)
	bundles := newFilesystemTestBundleStore(t)
	reg, err := registry.NewWASIOverlay(bundles)
	if err != nil {
		t.Fatal(err)
	}
	admission, err := NewArtifactAdmissionService(bundles, reg)
	if err != nil {
		t.Fatal(err)
	}
	release, err := admission.RegisterGameBundle(ctx, gameBytes)
	if err != nil {
		t.Fatal(err)
	}
	root := repoRoot(t)
	general, err := NewGeneralSubmissionService(root, reg, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	general.WithBundleStore(bundles)
	queue := NewInMemoryQueueStore()
	dryRun, err := NewLocalDryRunChecker(root)
	if err != nil {
		t.Fatal(err)
	}
	validator, err := NewDefaultAdmissionValidator(reg, dryRun.WithBundleStore(bundles))
	if err != nil {
		t.Fatal(err)
	}
	commands, err := NewCommandService(queue, validator)
	if err != nil {
		t.Fatal(err)
	}
	requests, err := NewMatchRequestService(general, commands, queue, nil)
	if err != nil {
		t.Fatal(err)
	}
	followups, err := NewRunCommandService(commands, queue, nil)
	if err != nil {
		t.Fatal(err)
	}
	var queued []QueueRecord
	for _, turns := range []int{2, 3} {
		ruleset := fmt.Sprintf("phase2-simultaneous-%dturn", turns)
		registration, err := general.RegisterGame(ctx, GameRegistrationRequest{ArtifactID: release.ArtifactID, RulesetVersion: ruleset})
		if err != nil {
			t.Fatal(err)
		}
		var participants []MatchRequestParticipant
		for _, player := range []string{"p1", "p2"} {
			ai, err := general.RegisterAIBundle(ctx, AISubmissionRequest{GameRegistrationID: registration.RegistrationID}, aiBytes)
			if err != nil {
				t.Fatal(err)
			}
			participants = append(participants, MatchRequestParticipant{PlayerID: player, AISubmissionID: ai.AISubmissionID})
		}
		_, run, err := requests.Create(ctx, MatchRequestCreateRequest{
			RequestID: ruleset, GameRegistrationID: registration.RegistrationID, MatchID: ruleset,
			OutputDir: t.TempDir(), Participants: participants,
		})
		if err != nil {
			t.Fatal(err)
		}
		if run.Submission.Game.RulesetVersion != ruleset || run.Submission.GameArtifactID != release.ArtifactID {
			t.Fatalf("queued run identity = %+v", run.Submission)
		}
		queued = append(queued, run)
	}

	invoker, err := NewLocalRunnerInvoker(root, reg, 0)
	if err != nil {
		t.Fatal(err)
	}
	worker, err := NewWorker(queue, invoker.WithBundleStore(bundles), LocalTerminalPersister{})
	if err != nil {
		t.Fatal(err)
	}
	var workers sync.WaitGroup
	for i := range queued {
		workers.Go(func() {
			if _, err := worker.ProcessNext(ctx, fmt.Sprintf("worker-%d", i)); err != nil {
				t.Errorf("ProcessNext: %v", err)
			}
		})
	}
	workers.Wait()

	for i, original := range queued {
		completed, err := queue.Get(ctx, original.Submission.RunID)
		if err != nil {
			t.Fatal(err)
		}
		assertRulesetBundleResult(t, completed, release.ArtifactID, i+2)

		rerun, err := followups.Rerun(ctx, original.Submission.RunID)
		if err != nil {
			t.Fatal(err)
		}
		if rerun.Submission.Game != original.Submission.Game || rerun.Submission.GameArtifactID != release.ArtifactID {
			t.Fatalf("rerun identity = %+v", rerun.Submission)
		}
		repeated, err := worker.ProcessNext(ctx, "rerun-worker")
		if err != nil {
			t.Fatal(err)
		}
		assertRulesetBundleResult(t, repeated, release.ArtifactID, i+2)
	}

	original := queued[0]
	if _, err := followups.Rerun(ctx, original.Submission.RunID); err != nil {
		t.Fatal(err)
	}
	startFailure := fmt.Errorf("simulated worker failure before game start")
	failingWorker, err := NewWorker(queue, stubRunnerInvoker{err: startFailure}, LocalTerminalPersister{})
	if err != nil {
		t.Fatal(err)
	}
	failed, err := failingWorker.ProcessNext(ctx, "failing-worker")
	if err != startFailure || failed.State != StateFailed {
		t.Fatalf("failed run state = %q, error = %v", failed.State, err)
	}
	retry, err := followups.Retry(ctx, failed.Submission.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if retry.Submission.Game != original.Submission.Game || retry.Submission.GameArtifactID != release.ArtifactID {
		t.Fatalf("retry identity = %+v", retry.Submission)
	}
	retried, err := worker.ProcessNext(ctx, "retry-worker")
	if err != nil {
		t.Fatal(err)
	}
	assertRulesetBundleResult(t, retried, release.ArtifactID, 2)
}

func assertRulesetBundleResult(t *testing.T, run QueueRecord, digest string, turns int) {
	t.Helper()
	if run.State != StateCompleted || run.Terminal == nil {
		t.Fatalf("run did not complete: %+v", run)
	}
	record, err := replay.LoadRecord(run.Terminal.RecordPath)
	if err != nil {
		t.Fatal(err)
	}
	if record.GameArtifactID != digest || record.Game.RulesetVersion != run.Submission.Game.RulesetVersion ||
		record.Snapshot.RulesetVersion != record.Game.RulesetVersion || record.Snapshot.Turn != turns {
		t.Fatalf("record digest, ruleset, or turn count = %q, %q, %q, %d", record.GameArtifactID, record.Game.RulesetVersion, record.Snapshot.RulesetVersion, record.Snapshot.Turn)
	}
}
