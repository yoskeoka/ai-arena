package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yoskeoka/ai-arena/internal/games/janken"
	"github.com/yoskeoka/ai-arena/internal/platform/contract"
)

func TestArenaRunnerJankenGoWASMMixedRuntimePath(t *testing.T) {
	requireWASME2E(t)

	result := runArena(t,
		"--game", janken.GameID,
		"--game-version", janken.GameVersion,
		"--ruleset", janken.RulesetRegular,
		"--match-id", "janken-go-wasm-happy",
		"--player", "p1=./testdata/ai/janken/janken-go-wasm-ai",
		"--player", "p2=./testdata/ai/janken/janken-rock-ai-wasm",
	)

	if result.Record.Status != contract.StatusCompleted {
		t.Fatalf("status = %q, want completed", result.Record.Status)
	}
	if result.Record.Result.Placements[0].PlayerID != "p1" {
		t.Fatalf("winner = %q, want p1", result.Record.Result.Placements[0].PlayerID)
	}
	if result.Record.Snapshot.PerPlayer["p1"].StderrBytes == 0 {
		t.Fatal("expected stderr bytes for Go-WASM player")
	}
	if _, err := os.Stat(filepath.Join(result.MatchDir, "history.json")); err != nil {
		t.Fatalf("history.json missing: %v", err)
	}
}

func TestArenaRunnerJankenGoWASMMissingModuleFails(t *testing.T) {
	requireWASME2E(t)
	entryPath := filepath.Join(t.TempDir(), "missing-go-wasm-ai")
	manifestPath := entryPath + ".arena.json"
	manifest := fmt.Sprintf(`{
  "ai_id": "missing-go-wasm-ai",
  "protocol": {
    "transport": "stdio-jsonrpc-ndjson",
    "game_id": %q,
    "game_version": %q,
    "ruleset_version": %q
  },
  "runtime": {
    "kind": "wasm-wasi",
    "module": "./missing-go-wasm-ai.wasm"
  }
}`, janken.GameID, janken.GameVersion, janken.RulesetRegular)
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	cmd := newArenaRunnerCommand(t,
		"--game", janken.GameID,
		"--game-version", janken.GameVersion,
		"--ruleset", janken.RulesetRegular,
		"--match-id", "janken-go-wasm-missing-module",
		"--output-dir", t.TempDir(),
		"--player", "p1="+entryPath,
		"--player", "p2=./testdata/ai/janken/janken-rock-ai",
	)
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected missing module error")
	}
	if !strings.Contains(string(output), "openat") && !strings.Contains(string(output), "no such file") {
		t.Fatalf("output = %s, want missing wasm module read failure", output)
	}
}

func TestBuildGoWASMReportsBuildFailure(t *testing.T) {
	requireWASME2E(t)
	outputPath := filepath.Join(t.TempDir(), "missing.wasm")
	err := buildGoWASM(newTestContext(t), repoRoot(t), "./testdata/ai/janken/does-not-exist", outputPath)
	if err == nil {
		t.Fatal("expected build failure for missing package")
	}
	if _, statErr := os.Stat(outputPath); !os.IsNotExist(statErr) {
		t.Fatalf("output path = %q, want missing artifact", outputPath)
	}
}

func TestArenaRunnerJankenRustWASMEvaluationPath(t *testing.T) {
	requireWASME2E(t)
	if os.Getenv("AI_ARENA_EXPERIMENT_RUST_WASM") != "1" {
		t.Skip("set AI_ARENA_EXPERIMENT_RUST_WASM=1 to enable Rust-WASM evaluation")
	}

	result := runArena(t,
		"--game", janken.GameID,
		"--game-version", janken.GameVersion,
		"--ruleset", janken.RulesetRegular,
		"--match-id", "janken-rust-wasm-eval",
		"--player", "p1=./testdata/ai/janken/janken-rust-wasm-ai",
		"--player", "p2=./testdata/ai/janken/janken-rock-ai-wasm",
	)

	if result.Record.Status != contract.StatusCompleted {
		t.Fatalf("status = %q, want completed", result.Record.Status)
	}
	if result.Record.Result.Placements[0].PlayerID != "p1" {
		t.Fatalf("winner = %q, want p1", result.Record.Result.Placements[0].PlayerID)
	}
	if result.Record.Snapshot.PerPlayer["p1"].StderrBytes == 0 {
		t.Fatal("expected stderr bytes for Rust-WASM player")
	}
	if _, err := os.Stat(filepath.Join(result.MatchDir, "history.json")); err != nil {
		t.Fatalf("history.json missing: %v", err)
	}
}

func requireWASME2E(t *testing.T) {
	t.Helper()

	if os.Getenv("AI_ARENA_WASM_E2E") != "1" {
		t.Skip("set AI_ARENA_WASM_E2E=1 to enable WASM verification tests")
	}
}

func buildGoWASM(ctx context.Context, dir, pkg, outputPath string) error {
	cmd := exec.CommandContext(ctx, "go", "build", "-o", outputPath, pkg)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build wasm %s: %w\n%s", pkg, err, output)
	}
	return nil
}

func buildRustWASM(ctx context.Context, dir, manifestPath, outputPath string) error {
	if _, err := exec.LookPath("cargo"); err != nil {
		return fmt.Errorf("cargo not found: %w", err)
	}
	if _, err := exec.LookPath("rustup"); err != nil {
		return fmt.Errorf("rustup not found: %w", err)
	}

	targetCheck := exec.CommandContext(ctx, "rustup", "target", "list", "--installed")
	targetCheck.Dir = dir
	installedTargets, err := targetCheck.Output()
	if err != nil {
		return fmt.Errorf("list installed rust targets: %w", err)
	}
	if !strings.Contains(string(installedTargets), "wasm32-wasip1") {
		return fmt.Errorf("missing rust target wasm32-wasip1; run `rustup target add wasm32-wasip1`")
	}

	targetDir, cleanup, err := rustWASMTargetDir(outputPath)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}
	cmd := exec.CommandContext(ctx, "cargo", "build",
		"--manifest-path", manifestPath,
		"--target", "wasm32-wasip1",
		"--release",
		"--target-dir", targetDir,
	)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build rust wasm %s: %w\n%s", manifestPath, err, output)
	}

	builtArtifact := filepath.Join(targetDir, "wasm32-wasip1", "release", "janken-rust-wasm-ai.wasm")
	wasmBytes, err := os.ReadFile(builtArtifact)
	if err != nil {
		return fmt.Errorf("read built rust wasm artifact: %w", err)
	}
	if err := os.WriteFile(outputPath, wasmBytes, 0o644); err != nil {
		return fmt.Errorf("write rust wasm fixture: %w", err)
	}
	return nil
}

func rustWASMTargetDir(outputPath string) (string, func(), error) {
	if targetDir := os.Getenv("CARGO_TARGET_DIR"); targetDir != "" {
		return targetDir, nil, nil
	}

	targetDir, err := os.MkdirTemp(filepath.Dir(outputPath), "rust-target-")
	if err != nil {
		return "", nil, fmt.Errorf("create rust wasm target dir: %w", err)
	}
	return targetDir, func() { _ = os.RemoveAll(targetDir) }, nil
}
