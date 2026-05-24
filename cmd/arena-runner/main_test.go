package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"

	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	platformruntime "github.com/yoskeoka/ai-arena/internal/platform/runtime"
)

func TestResolveGameMasterRuntimeResolvesManifestRelativeCommand(t *testing.T) {
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

func TestRunRejectsDebugInputsForGameMasterManifest(t *testing.T) {
	chdirRepoRoot(t)

	err := run([]string{
		"--game-master-manifest", "./testdata/game-master/external-echo/manifest.json",
		"--target-turn", "1",
		"--player", "p1=./testdata/ai/external-echo/echo-ai",
	})
	if err == nil || !strings.Contains(err.Error(), "--game-master-manifest supports fresh run only") {
		t.Fatalf("run error = %v, want fresh-run-only guard", err)
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
