package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"

	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	platformruntime "github.com/yoskeoka/ai-arena/internal/platform/runtime"
)

func TestResolveGameMasterRuntimeResolvesManifestRelativeCommand(t *testing.T) {
	chdirRepoRoot(t)

	cfg, err := resolveGameMasterRuntime("./testdata/game-master/external-echo/manifest.json", catalog.RuntimeManifest{
		Kind:    platformruntime.KindLocalSubprocess,
		Command: []string{"./bin/gamemaster", "--demo"},
	})
	if err != nil {
		t.Fatalf("resolveGameMasterRuntime: %v", err)
	}
	if got, want := cfg.Command[0], mustAbsPath(t, "testdata/game-master/external-echo/bin/gamemaster"); got != want {
		t.Fatalf("command[0] = %q, want %q", got, want)
	}
	if got, want := cfg.Dir, mustAbsPath(t, "testdata/game-master/external-echo"); got != want {
		t.Fatalf("dir = %q, want %q", got, want)
	}
}

func TestResolveGameMasterRuntimeRejectsUnsupportedKind(t *testing.T) {
	chdirRepoRoot(t)

	_, err := resolveGameMasterRuntime("./testdata/game-master/external-echo/manifest.json", catalog.RuntimeManifest{
		Kind: platformruntime.KindWASMWASI,
	})
	if !errors.Is(err, catalog.ErrInvalidMetadata) {
		t.Fatalf("expected ErrInvalidMetadata, got %v", err)
	}
}

func TestRunSupportsExternalGameMasterManifestFreshRun(t *testing.T) {
	chdirRepoRoot(t)

	outputDir := t.TempDir()
	matchID := "external-echo-match"
	err := run([]string{
		"--game-master-manifest", "./testdata/game-master/external-echo/manifest.json",
		"--output-dir", outputDir,
		"--match-id", matchID,
		"--log-output", "none",
		"--player", "p1=./testdata/ai/external-echo/echo-ai",
		"--player", "p2=./testdata/ai/external-echo/echo-ai",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	var summary resultSummary
	if err := readJSONFile(filepath.Join(outputDir, matchID, "result-summary.json"), &summary); err != nil {
		t.Fatalf("read result summary: %v", err)
	}
	if summary.GameID != "external-echo-count" {
		t.Fatalf("game_id = %q, want external-echo-count", summary.GameID)
	}
	if summary.GameVersion != "2.0.0" {
		t.Fatalf("game_version = %q, want 2.0.0", summary.GameVersion)
	}
	if summary.RulesetVersion != "phase2-simultaneous-3turn" {
		t.Fatalf("ruleset_version = %q", summary.RulesetVersion)
	}
	if summary.Status != game.StatusCompleted {
		t.Fatalf("status = %q, want %q", summary.Status, game.StatusCompleted)
	}
}

func TestRunSupportsExternalGameMasterManifestRecordResume(t *testing.T) {
	chdirRepoRoot(t)

	outputDir := t.TempDir()
	initialMatchID := "external-echo-record-source"
	err := run([]string{
		"--game-master-manifest", "./testdata/game-master/external-echo/manifest.json",
		"--output-dir", outputDir,
		"--match-id", initialMatchID,
		"--log-output", "none",
		"--player", "p1=./testdata/ai/external-echo/echo-ai",
		"--player", "p2=./testdata/ai/external-echo/echo-ai",
	})
	if err != nil {
		t.Fatalf("initial run: %v", err)
	}

	resumeMatchID := "external-echo-record-resume"
	err = run([]string{
		"--game-master-manifest", "./testdata/game-master/external-echo/manifest.json",
		"--record-input", filepath.Join(outputDir, initialMatchID, "record.json"),
		"--output-dir", outputDir,
		"--match-id", resumeMatchID,
		"--log-output", "none",
		"--player", "p1=./testdata/ai/external-echo/echo-ai",
		"--player", "p2=./testdata/ai/external-echo/echo-ai",
	})
	if err != nil {
		t.Fatalf("resume run: %v", err)
	}

	var summary resultSummary
	if err := readJSONFile(filepath.Join(outputDir, resumeMatchID, "result-summary.json"), &summary); err != nil {
		t.Fatalf("read result summary: %v", err)
	}
	if summary.Status != game.StatusCompleted {
		t.Fatalf("status = %q, want %q", summary.Status, game.StatusCompleted)
	}
}

func TestRunSupportsExternalGameMasterManifestHistoryReplay(t *testing.T) {
	chdirRepoRoot(t)

	outputDir := t.TempDir()
	initialMatchID := "external-echo-history-source"
	err := run([]string{
		"--game-master-manifest", "./testdata/game-master/external-echo/manifest.json",
		"--output-dir", outputDir,
		"--match-id", initialMatchID,
		"--log-output", "none",
		"--player", "p1=./testdata/ai/external-echo/echo-ai",
		"--player", "p2=./testdata/ai/external-echo/echo-ai",
	})
	if err != nil {
		t.Fatalf("initial run: %v", err)
	}

	replayMatchID := "external-echo-history-replay"
	err = run([]string{
		"--game-master-manifest", "./testdata/game-master/external-echo/manifest.json",
		"--history-input", filepath.Join(outputDir, initialMatchID, "history.json"),
		"--target-turn", "2",
		"--output-dir", outputDir,
		"--match-id", replayMatchID,
		"--log-output", "none",
		"--player", "p1=./testdata/ai/external-echo/echo-ai",
		"--player", "p2=./testdata/ai/external-echo/echo-ai",
	})
	if err != nil {
		t.Fatalf("replay run: %v", err)
	}

	var summary resultSummary
	if err := readJSONFile(filepath.Join(outputDir, replayMatchID, "result-summary.json"), &summary); err != nil {
		t.Fatalf("read result summary: %v", err)
	}
	if summary.Status != game.StatusCompleted {
		t.Fatalf("status = %q, want %q", summary.Status, game.StatusCompleted)
	}
}

func TestRunFailsOnManifestMetadataMismatch(t *testing.T) {
	chdirRepoRoot(t)

	outputDir := t.TempDir()
	matchID := "external-echo-mismatch"
	err := run([]string{
		"--game-master-manifest", "./testdata/game-master/external-echo/manifest-metadata-mismatch.json",
		"--output-dir", outputDir,
		"--match-id", matchID,
		"--log-output", "none",
		"--player", "p1=./testdata/ai/external-echo-declared/echo-ai",
		"--player", "p2=./testdata/ai/external-echo-declared/echo-ai",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	var summary resultSummary
	if err := readJSONFile(filepath.Join(outputDir, matchID, "result-summary.json"), &summary); err != nil {
		t.Fatalf("read result summary: %v", err)
	}
	if summary.Status != game.StatusFailed {
		t.Fatalf("status = %q, want %q", summary.Status, game.StatusFailed)
	}
	if !strings.Contains(summary.Error, "game master metadata incompatible") {
		t.Fatalf("summary error = %q, want metadata incompatible", summary.Error)
	}
}

func TestRunFailsOnManifestCommandPathError(t *testing.T) {
	chdirRepoRoot(t)

	err := run([]string{
		"--game-master-manifest", "./testdata/game-master/external-echo/manifest-missing-binary.json",
		"--output-dir", t.TempDir(),
		"--match-id", "external-echo-missing-binary",
		"--log-output", "none",
		"--player", "p1=./testdata/ai/external-echo/echo-ai",
		"--player", "p2=./testdata/ai/external-echo/echo-ai",
	})
	if err == nil || !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatalf("run error = %v, want missing binary failure", err)
	}
}

func TestRunSupportsGameMasterBundleWithTwoAIBundles(t *testing.T) {
	chdirRepoRoot(t)

	gameBundle := buildTestBundle(t, "./cmd/echo-count-gamemaster", `{"schema_version":"arena-bundle/v1","artifact_kind":"game","game_id":"echo-count","game_version":"2.0.0","rulesets":[{"ruleset_version":"phase2-simultaneous-3turn","player_count":2}],"runtime":{"kind":"wasm-wasi","module":"module.wasm","memory_limit_pages":1024,"args":["module.wasm","--game-id","echo-count","--game-version","2.0.0","--ruleset","phase2-simultaneous-3turn"]}}`)
	aiBundle := buildTestBundle(t, "./testdata/ai/echo/echo-ai", `{"schema_version":"arena-bundle/v1","artifact_kind":"ai","ai_id":"echo-ai","game_id":"echo-count","game_version":"2.0.0","runtime":{"kind":"wasm-wasi","module":"module.wasm","memory_limit_pages":1024}}`)
	before := bundleTempDirs(t)
	outputDir := t.TempDir()
	matchID := "bundle-game-two-ai"
	if err := run([]string{
		"--game-master-bundle", gameBundle,
		"--output-dir", outputDir,
		"--match-id", matchID,
		"--log-output", "none",
		"--player-bundle", "p1=" + aiBundle,
		"--player-bundle", "p2=" + aiBundle,
	}); err != nil {
		t.Fatalf("run: %v", err)
	}
	assertCompletedBundleRun(t, outputDir, matchID, "echo-count")
	assertSameBundleTempDirs(t, before)
}

func TestRunSupportsGameMasterBundleWithLegacyAI(t *testing.T) {
	chdirRepoRoot(t)

	gameBundle := buildTestBundle(t, "./cmd/echo-count-gamemaster", `{"schema_version":"arena-bundle/v1","artifact_kind":"game","game_id":"echo-count","game_version":"2.0.0","rulesets":[{"ruleset_version":"phase2-simultaneous-3turn","player_count":2}],"runtime":{"kind":"wasm-wasi","module":"module.wasm","memory_limit_pages":1024,"args":["module.wasm","--game-id","echo-count","--game-version","2.0.0","--ruleset","phase2-simultaneous-3turn"]}}`)
	outputDir := t.TempDir()
	matchID := "bundle-game-legacy-ai"
	if err := run([]string{
		"--game-master-bundle", gameBundle,
		"--output-dir", outputDir,
		"--match-id", matchID,
		"--log-output", "none",
		"--player", "p1=./testdata/ai/echo/echo-ai",
		"--player", "p2=./testdata/ai/echo/echo-ai",
	}); err != nil {
		t.Fatalf("run: %v", err)
	}
	assertCompletedBundleRun(t, outputDir, matchID, "echo-count")
}

func TestRunSupportsBuiltinGameWithAIBundle(t *testing.T) {
	chdirRepoRoot(t)

	aiBundle := buildTestBundle(t, "./testdata/ai/echo/echo-ai", `{"schema_version":"arena-bundle/v1","artifact_kind":"ai","ai_id":"echo-ai-bundle","game_id":"echo-count","game_version":"2.0.0","runtime":{"kind":"wasm-wasi","module":"module.wasm","memory_limit_pages":1024}}`)
	outputDir := t.TempDir()
	matchID := "builtin-game-ai-bundle"
	if err := run([]string{
		"--game", "echo-count",
		"--game-version", "2.0.0",
		"--ruleset", "phase2-simultaneous-3turn",
		"--output-dir", outputDir,
		"--match-id", matchID,
		"--log-output", "none",
		"--player-bundle", "p1=" + aiBundle,
		"--player-bundle", "p2=" + aiBundle,
	}); err != nil {
		t.Fatalf("run: %v", err)
	}
	assertCompletedBundleRun(t, outputDir, matchID, "echo-count")
}

func TestRunSupportsManifestGameWithMixedAIInputs(t *testing.T) {
	chdirRepoRoot(t)

	aiBundle := buildTestBundle(t, "./testdata/ai/echo/echo-ai", `{"schema_version":"arena-bundle/v1","artifact_kind":"ai","ai_id":"external-echo-ai-bundle","game_id":"external-echo-count","game_version":"2.0.0","runtime":{"kind":"wasm-wasi","module":"module.wasm","memory_limit_pages":1024}}`)
	outputDir := t.TempDir()
	matchID := "manifest-game-mixed-ai"
	if err := run([]string{
		"--game-master-manifest", "./testdata/game-master/external-echo/manifest.json",
		"--output-dir", outputDir,
		"--match-id", matchID,
		"--log-output", "none",
		"--player", "p1=./testdata/ai/external-echo/echo-ai",
		"--player-bundle", "p2=" + aiBundle,
	}); err != nil {
		t.Fatalf("run: %v", err)
	}
	assertCompletedBundleRun(t, outputDir, matchID, "external-echo-count")
}

func TestRunSupportsGameMasterBundleRecordResume(t *testing.T) {
	chdirRepoRoot(t)

	gameBundle := buildTestBundle(t, "./cmd/echo-count-gamemaster", `{"schema_version":"arena-bundle/v1","artifact_kind":"game","game_id":"echo-count","game_version":"2.0.0","rulesets":[{"ruleset_version":"phase2-simultaneous-3turn","player_count":2}],"runtime":{"kind":"wasm-wasi","module":"module.wasm","memory_limit_pages":1024,"args":["module.wasm","--game-id","echo-count","--game-version","2.0.0","--ruleset","phase2-simultaneous-3turn"]}}`)
	aiBundle := buildTestBundle(t, "./testdata/ai/echo/echo-ai", `{"schema_version":"arena-bundle/v1","artifact_kind":"ai","ai_id":"echo-ai","game_id":"echo-count","game_version":"2.0.0","runtime":{"kind":"wasm-wasi","module":"module.wasm","memory_limit_pages":1024}}`)
	outputDir := t.TempDir()
	initialMatchID := "bundle-game-record-source"
	args := []string{
		"--game-master-bundle", gameBundle,
		"--output-dir", outputDir,
		"--match-id", initialMatchID,
		"--log-output", "none",
		"--player-bundle", "p1=" + aiBundle,
		"--player-bundle", "p2=" + aiBundle,
	}
	if err := run(args); err != nil {
		t.Fatalf("initial run: %v", err)
	}
	resumeMatchID := "bundle-game-record-resume"
	if err := run([]string{
		"--game-master-bundle", gameBundle,
		"--record-input", filepath.Join(outputDir, initialMatchID, "record.json"),
		"--output-dir", outputDir,
		"--match-id", resumeMatchID,
		"--log-output", "none",
		"--player-bundle", "p1=" + aiBundle,
		"--player-bundle", "p2=" + aiBundle,
	}); err != nil {
		t.Fatalf("resume run: %v", err)
	}
	assertCompletedBundleRun(t, outputDir, resumeMatchID, "echo-count")
	if err := run([]string{
		"--game-master-bundle", gameBundle,
		"--history-input", filepath.Join(outputDir, initialMatchID, "history.json"),
		"--target-turn", "1",
		"--output-dir", outputDir,
		"--match-id", "bundle-game-history-unsupported",
		"--log-output", "none",
		"--player-bundle", "p1=" + aiBundle,
		"--player-bundle", "p2=" + aiBundle,
	}); err == nil || !strings.Contains(err.Error(), "history replay reconstruction is not supported") {
		t.Fatalf("history replay error = %v, want explicit unsupported error", err)
	}
}

func TestRunRejectsBundleInputConflictsAndMismatches(t *testing.T) {
	chdirRepoRoot(t)

	aiBundle := buildTestBundle(t, "./testdata/ai/echo/echo-ai", `{"schema_version":"arena-bundle/v1","artifact_kind":"ai","ai_id":"echo-ai","game_id":"wrong-game","game_version":"2.0.0","runtime":{"kind":"wasm-wasi","module":"module.wasm","memory_limit_pages":1024}}`)
	if err := run([]string{
		"--game", "echo-count",
		"--game-version", "2.0.0",
		"--ruleset", "phase2-simultaneous-3turn",
		"--output-dir", t.TempDir(),
		"--player-bundle", "p1=" + aiBundle,
	}); err == nil || !strings.Contains(err.Error(), "game_id mismatch") {
		t.Fatalf("run mismatch error = %v, want game_id mismatch", err)
	}

	if err := run([]string{
		"--game-master-manifest", "./testdata/game-master/external-echo/manifest.json",
		"--game-master-bundle", "missing.zip",
		"--output-dir", t.TempDir(),
		"--player", "p1=./testdata/ai/external-echo/echo-ai",
	}); err == nil || !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("run source conflict error = %v, want conflict", err)
	}

	if err := run([]string{
		"--game", "echo-count",
		"--game-version", "2.0.0",
		"--ruleset", "phase2-simultaneous-3turn",
		"--output-dir", t.TempDir(),
		"--player", "p1=./testdata/ai/echo/echo-ai",
		"--player-bundle", "p1=" + aiBundle,
	}); err == nil || !strings.Contains(err.Error(), `duplicate player_id "p1"`) {
		t.Fatalf("run duplicate player error = %v, want duplicate player_id", err)
	}

	invalidBundle := filepath.Join(t.TempDir(), "invalid.zip")
	if err := os.WriteFile(invalidBundle, []byte("not a ZIP"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{
		"--game", "echo-count",
		"--game-version", "2.0.0",
		"--ruleset", "phase2-simultaneous-3turn",
		"--output-dir", t.TempDir(),
		"--player-bundle", "p1=" + invalidBundle,
	}); err == nil || !strings.Contains(err.Error(), "read ZIP") {
		t.Fatalf("run invalid bundle error = %v, want ZIP validation failure", err)
	}
}

func buildTestBundle(t *testing.T, packagePath, manifest string) string {
	t.Helper()
	tmpDir := t.TempDir()
	modulePath := filepath.Join(tmpDir, "module.wasm")
	cacheDir := filepath.Join(os.TempDir(), "ai-arena-runner-test-gocache")
	cmd := exec.CommandContext(context.Background(), "go", "build", "-o", modulePath, packagePath)
	cmd.Dir = filepath.Join(filepath.Dir(testFilePath(t)), "..", "..")
	cmd.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm", "GOCACHE="+cacheDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build WASI bundle module %s: %v\n%s", packagePath, err, output)
	}
	bundlePath := filepath.Join(tmpDir, "bundle.zip")
	file, err := os.Create(bundlePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	manifestEntry, err := writer.CreateHeader(&zip.FileHeader{Name: "manifest.json", Method: zip.Store})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manifestEntry.Write([]byte(manifest)); err != nil {
		t.Fatal(err)
	}
	moduleBytes, err := os.ReadFile(modulePath)
	if err != nil {
		t.Fatal(err)
	}
	moduleEntry, err := writer.CreateHeader(&zip.FileHeader{Name: "module.wasm", Method: zip.Store})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := moduleEntry.Write(moduleBytes); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return bundlePath
}

func assertCompletedBundleRun(t *testing.T, outputDir, matchID, gameID string) {
	t.Helper()
	var summary resultSummary
	if err := readJSONFile(filepath.Join(outputDir, matchID, "result-summary.json"), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.GameID != gameID {
		t.Fatalf("game_id = %q, want %q", summary.GameID, gameID)
	}
	if summary.Status != game.StatusCompleted {
		t.Fatalf("status = %q, want %q (error=%q)", summary.Status, game.StatusCompleted, summary.Error)
	}
	for _, name := range []string{"record.json", "snapshot.json", "exported-snapshot.json", "history.json", "structured-log.ndjson"} {
		if _, err := os.Stat(filepath.Join(outputDir, matchID, name)); err != nil {
			t.Fatalf("artifact %s: %v", name, err)
		}
	}
}

func bundleTempDirs(t *testing.T) map[string]struct{} {
	t.Helper()
	result := make(map[string]struct{})
	for _, pattern := range []string{"ai-arena-game-bundle-*", "ai-arena-ai-bundle-*"} {
		paths, err := filepath.Glob(filepath.Join(os.TempDir(), pattern))
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range paths {
			result[path] = struct{}{}
		}
	}
	return result
}

func assertSameBundleTempDirs(t *testing.T, before map[string]struct{}) {
	t.Helper()
	after := bundleTempDirs(t)
	for path := range after {
		if _, existed := before[path]; !existed {
			t.Errorf("bundle temporary directory was not cleaned up: %s", path)
		}
	}
}

func chdirRepoRoot(t *testing.T) {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(testFilePath(t)), "..", ".."))
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir %s: %v", root, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})
}

func testFilePath(t *testing.T) string {
	t.Helper()

	_, file, _, ok := goruntime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	return file
}

func readJSONFile(path string, dest any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func mustAbsPath(t *testing.T, path string) string {
	t.Helper()

	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("filepath.Abs(%q): %v", path, err)
	}
	return abs
}
