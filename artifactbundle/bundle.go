// Package artifactbundle validates the immutable arena-bundle/v1 archive.
package artifactbundle

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/tetratelabs/wazero"
)

const (
	SchemaVersion             = "arena-bundle/v1"
	MaxBundleBytes      int64 = 64 << 20
	MaxManifestBytes    int64 = 1 << 20
	MaxModuleBytes      int64 = 32 << 20
	maxCompressionRatio int64 = 100
)

type Manifest struct {
	SchemaVersion string `json:"schema_version"`
	ArtifactKind  string `json:"artifact_kind"`
	GameID        string `json:"game_id"`
	GameVersion   string `json:"game_version"`
	Rulesets      []struct {
		RulesetVersion        string `json:"ruleset_version"`
		PlayerCount           int    `json:"player_count,omitempty"`
		MaxActiveBotsPerOwner int    `json:"max_active_bots_per_owner,omitempty"`
	} `json:"rulesets,omitempty"`
	Runtime struct {
		Kind             string   `json:"kind"`
		Module           string   `json:"module"`
		Args             []string `json:"args,omitempty"`
		MemoryLimitPages uint32   `json:"memory_limit_pages,omitempty"`
		TimeoutMS        uint32   `json:"timeout_ms,omitempty"`
	} `json:"runtime"`
}

type Bundle struct {
	Manifest Manifest
	Digest   string
	Bytes    []byte
}

func ManifestJSON(manifest Manifest) []byte { data, _ := json.Marshal(manifest); return data }

func ModuleBytes(data []byte, moduleName string) []byte {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil
	}
	for _, file := range zr.File {
		if file.Name == moduleName {
			value, err := readEntry(file, MaxModuleBytes)
			if err != nil {
				return nil
			}
			return value
		}
	}
	return nil
}

// Read applies archive, manifest, and WASI module policy before returning bytes.
func Read(data []byte) (Bundle, error) {
	if len(data) == 0 {
		return Bundle{}, fmt.Errorf("artifactbundle: empty bundle")
	}
	if int64(len(data)) > MaxBundleBytes {
		return Bundle{}, fmt.Errorf("artifactbundle: bundle exceeds %d bytes", MaxBundleBytes)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return Bundle{}, fmt.Errorf("artifactbundle: read ZIP: %w", err)
	}
	if len(zr.File) != 2 {
		return Bundle{}, fmt.Errorf("artifactbundle: exactly manifest.json and one module are required")
	}
	entries := map[string]*zip.File{}
	folded := map[string]struct{}{}
	for _, file := range zr.File {
		if file.FileInfo().IsDir() || file.Mode()&0o170000 != 0 || file.Name != path.Base(file.Name) || strings.Contains(file.Name, "\\") {
			return Bundle{}, fmt.Errorf("artifactbundle: unsafe entry %q", file.Name)
		}
		if _, duplicate := entries[file.Name]; duplicate {
			return Bundle{}, fmt.Errorf("artifactbundle: duplicate entry %q", file.Name)
		}
		foldedName := strings.ToLower(file.Name)
		if _, duplicate := folded[foldedName]; duplicate {
			return Bundle{}, fmt.Errorf("artifactbundle: case-insensitive duplicate entry %q", file.Name)
		}
		folded[foldedName] = struct{}{}
		if file.UncompressedSize64 > uint64(MaxModuleBytes) || (file.CompressedSize64 > 0 && file.UncompressedSize64 > file.CompressedSize64*uint64(maxCompressionRatio)) {
			return Bundle{}, fmt.Errorf("artifactbundle: entry %q exceeds resource limits", file.Name)
		}
		entries[file.Name] = file
	}
	manifestFile, ok := entries["manifest.json"]
	if !ok {
		return Bundle{}, fmt.Errorf("artifactbundle: manifest.json is required")
	}
	manifestBytes, err := readEntry(manifestFile, MaxManifestBytes)
	if err != nil {
		return Bundle{}, fmt.Errorf("artifactbundle: read manifest: %w", err)
	}
	var manifest Manifest
	dec := json.NewDecoder(bytes.NewReader(manifestBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&manifest); err != nil {
		return Bundle{}, fmt.Errorf("artifactbundle: parse manifest: %w", err)
	}
	if err := validateManifest(manifest); err != nil {
		return Bundle{}, err
	}
	module, ok := entries[manifest.Runtime.Module]
	if !ok || module.Name == "manifest.json" {
		return Bundle{}, fmt.Errorf("artifactbundle: declared module is required")
	}
	moduleBytes, err := readEntry(module, MaxModuleBytes)
	if err != nil {
		return Bundle{}, fmt.Errorf("artifactbundle: read module: %w", err)
	}
	if len(moduleBytes) < 8 || !bytes.Equal(moduleBytes[:4], []byte{0, 97, 115, 109}) || !bytes.Equal(moduleBytes[4:8], []byte{1, 0, 0, 0}) {
		return Bundle{}, fmt.Errorf("artifactbundle: module is not WebAssembly v1")
	}
	if err := validateWASM(moduleBytes); err != nil {
		return Bundle{}, err
	}
	sum := sha256.Sum256(data)
	return Bundle{Manifest: manifest, Digest: "sha256:" + hex.EncodeToString(sum[:]), Bytes: append([]byte(nil), data...)}, nil
}

func validateManifest(m Manifest) error {
	if m.SchemaVersion != SchemaVersion || (m.ArtifactKind != "game" && m.ArtifactKind != "ai") || m.GameID == "" || m.GameVersion == "" || m.Runtime.Kind != "wasm-wasi" || m.Runtime.Module == "" || m.Runtime.Module != path.Base(m.Runtime.Module) || strings.Contains(m.Runtime.Module, "\\") {
		return fmt.Errorf("artifactbundle: unsupported manifest")
	}
	if m.Runtime.MemoryLimitPages > 1024 || m.Runtime.TimeoutMS > 600000 {
		return fmt.Errorf("artifactbundle: runtime budget exceeds policy")
	}
	for _, ruleset := range m.Rulesets {
		if ruleset.RulesetVersion == "" || ruleset.PlayerCount < 0 || ruleset.MaxActiveBotsPerOwner < 0 {
			return fmt.Errorf("artifactbundle: invalid ruleset")
		}
	}
	return nil
}

func validateWASM(module []byte) error {
	runtime := wazero.NewRuntime(context.Background())
	defer runtime.Close(context.Background())
	compiled, err := runtime.CompileModule(context.Background(), module)
	if err != nil {
		return fmt.Errorf("artifactbundle: compile module: %w", err)
	}
	defer compiled.Close(context.Background())
	for _, fn := range compiled.ImportedFunctions() {
		moduleName, _, _ := fn.Import()
		if moduleName != "wasi_snapshot_preview1" {
			return fmt.Errorf("artifactbundle: unsupported WASM import module %q", moduleName)
		}
	}
	for _, memory := range compiled.ImportedMemories() {
		moduleName, _, _ := memory.Import()
		if moduleName != "wasi_snapshot_preview1" {
			return fmt.Errorf("artifactbundle: unsupported WASM import module %q", moduleName)
		}
	}
	return nil
}

func readEntry(file *zip.File, limit int64) ([]byte, error) {
	r, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	value, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(value)) > limit {
		return nil, fmt.Errorf("entry exceeds %d bytes", limit)
	}
	return value, nil
}
