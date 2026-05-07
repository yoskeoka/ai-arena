package catalog

import (
	"errors"
	"testing"

	"github.com/yoskeoka/ai-arena/internal/platform/runtime"
)

func TestValidateMetadata(t *testing.T) {
	meta := GameMetadata{
		GameID:         "janken",
		GameVersion:    "2.1.0",
		RulesetVersion: "phase2",
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
	}
	actual := GameMetadata{
		GameID:         "janken",
		GameVersion:    "2.9.4",
		RulesetVersion: "phase2",
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

func TestResolveRuntimeSupportsSubprocessAndWASM(t *testing.T) {
	subprocess, err := ResolveRuntime("./testdata/ai/bot", RuntimeManifest{
		Kind:    runtime.KindLocalSubprocess,
		Command: []string{"./bot"},
	})
	if err != nil {
		t.Fatalf("ResolveRuntime subprocess: %v", err)
	}
	if subprocess.Kind != runtime.KindLocalSubprocess || len(subprocess.Command) != 1 || subprocess.Command[0] != "./bot" {
		t.Fatalf("subprocess runtime = %+v", subprocess)
	}

	wasmCfg, err := ResolveRuntime("./testdata/ai/bot", RuntimeManifest{
		Kind:             runtime.KindWASMWASI,
		Module:           "bot.wasm",
		Args:             []string{"bot.wasm", "--demo"},
		MemoryLimitPages: 32,
	})
	if err != nil {
		t.Fatalf("ResolveRuntime wasm: %v", err)
	}
	if wasmCfg.Kind != runtime.KindWASMWASI {
		t.Fatalf("wasm kind = %q, want %q", wasmCfg.Kind, runtime.KindWASMWASI)
	}
	if wasmCfg.ModulePath != "testdata/ai/bot.wasm" {
		t.Fatalf("wasm module path = %q, want testdata/ai/bot.wasm", wasmCfg.ModulePath)
	}
	if wasmCfg.MemoryLimitPages != 32 {
		t.Fatalf("wasm memory limit = %d, want 32", wasmCfg.MemoryLimitPages)
	}
}

func TestResolveRuntimeRejectsInvalidManifest(t *testing.T) {
	if _, err := ResolveRuntime("./testdata/ai/bot", RuntimeManifest{
		Kind: runtime.KindWASMWASI,
	}); !errors.Is(err, ErrInvalidMetadata) {
		t.Fatalf("expected ErrInvalidMetadata for missing wasm module, got %v", err)
	}

	if _, err := ResolveRuntime("./testdata/ai/bot", RuntimeManifest{
		Kind: runtime.Kind("custom"),
	}); !errors.Is(err, ErrInvalidMetadata) {
		t.Fatalf("expected ErrInvalidMetadata for unsupported kind, got %v", err)
	}
}
