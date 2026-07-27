// Package artifactbundle validates the immutable arena-bundle/v1 archive.
package artifactbundle

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"
)

const SchemaVersion = "arena-bundle/v1"

type Manifest struct {
	SchemaVersion string `json:"schema_version"`
	ArtifactKind  string `json:"artifact_kind"`
	GameID        string `json:"game_id"`
	GameVersion   string `json:"game_version"`
	Rulesets      []struct {
		RulesetVersion string `json:"ruleset_version"`
	} `json:"rulesets,omitempty"`
	Runtime struct {
		Kind   string   `json:"kind"`
		Module string   `json:"module"`
		Args   []string `json:"args,omitempty"`
	} `json:"runtime"`
}

type Bundle struct {
	Manifest Manifest
	Digest   string
	Bytes    []byte
}

// ManifestJSON returns the canonical manifest bytes for materialization.
func ManifestJSON(manifest Manifest) []byte { data, _ := json.Marshal(manifest); return data }

// ModuleBytes returns the declared module from a validated bundle.
func ModuleBytes(data []byte, moduleName string) []byte {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil
	}
	for _, file := range zr.File {
		if file.Name == moduleName {
			r, err := file.Open()
			if err != nil {
				return nil
			}
			defer r.Close()
			value, err := io.ReadAll(r)
			if err != nil {
				return nil
			}
			return value
		}
	}
	return nil
}

// Read validates the fixed two-entry archive and returns its immutable digest.
func Read(data []byte) (Bundle, error) {
	if len(data) == 0 {
		return Bundle{}, fmt.Errorf("artifactbundle: empty bundle")
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return Bundle{}, fmt.Errorf("artifactbundle: read ZIP: %w", err)
	}
	if len(zr.File) != 2 {
		return Bundle{}, fmt.Errorf("artifactbundle: exactly manifest.json and one module are required")
	}
	entries := map[string]*zip.File{}
	for _, file := range zr.File {
		if file.FileInfo().IsDir() || file.Mode()&0o170000 != 0 || file.Name != path.Base(file.Name) || strings.Contains(file.Name, "\\") {
			return Bundle{}, fmt.Errorf("artifactbundle: unsafe entry %q", file.Name)
		}
		if _, duplicate := entries[file.Name]; duplicate {
			return Bundle{}, fmt.Errorf("artifactbundle: duplicate entry %q", file.Name)
		}
		entries[file.Name] = file
	}
	manifestFile, ok := entries["manifest.json"]
	if !ok {
		return Bundle{}, fmt.Errorf("artifactbundle: manifest.json is required")
	}
	var manifest Manifest
	if err := readJSON(manifestFile, &manifest); err != nil {
		return Bundle{}, err
	}
	if manifest.SchemaVersion != SchemaVersion || (manifest.ArtifactKind != "game" && manifest.ArtifactKind != "ai") || manifest.Runtime.Kind != "wasm-wasi" || manifest.Runtime.Module == "" {
		return Bundle{}, fmt.Errorf("artifactbundle: unsupported manifest")
	}
	module, ok := entries[manifest.Runtime.Module]
	if !ok || module.Name == "manifest.json" {
		return Bundle{}, fmt.Errorf("artifactbundle: declared module is required")
	}
	r, err := module.Open()
	if err != nil {
		return Bundle{}, err
	}
	defer r.Close()
	magic := make([]byte, 4)
	if _, err := io.ReadFull(r, magic); err != nil || !bytes.Equal(magic, []byte{0, 97, 115, 109}) {
		return Bundle{}, fmt.Errorf("artifactbundle: module is not WebAssembly")
	}
	sum := sha256.Sum256(data)
	return Bundle{Manifest: manifest, Digest: "sha256:" + hex.EncodeToString(sum[:]), Bytes: append([]byte(nil), data...)}, nil
}

func readJSON(file *zip.File, out any) error {
	r, err := file.Open()
	if err != nil {
		return err
	}
	defer r.Close()
	if err := json.NewDecoder(io.LimitReader(r, 1<<20)).Decode(out); err != nil {
		return fmt.Errorf("artifactbundle: parse manifest: %w", err)
	}
	return nil
}
