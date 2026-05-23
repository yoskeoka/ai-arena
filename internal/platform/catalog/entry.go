package catalog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yoskeoka/ai-arena/internal/platform/runtime"
)

// LoadedEntry captures one validated AI runtime configuration.
type LoadedEntry struct {
	Runtime runtime.Config
	AIID    string
}

// LoadEntry resolves one AI entry path plus optional sidecar manifest.
func LoadEntry(matchMeta GameMetadata, entryPath string) (LoadedEntry, error) {
	sidecarPath := entryPath + ".arena.json"
	if _, err := os.Stat(sidecarPath); err == nil {
		var manifest SidecarManifest
		if err := readJSON(sidecarPath, &manifest); err != nil {
			return LoadedEntry{}, err
		}
		aiMeta := GameMetadata{
			GameID:         manifest.Protocol.GameID,
			GameVersion:    manifest.Protocol.GameVersion,
			RulesetVersion: manifest.Protocol.RulesetVersion,
		}
		if err := Compatible(matchMeta, aiMeta); err != nil {
			return LoadedEntry{}, fmt.Errorf("%s metadata incompatible: %w", filepath.Base(entryPath), err)
		}
		if manifest.Protocol.Transport != "" && manifest.Protocol.Transport != "stdio-jsonrpc-ndjson" {
			return LoadedEntry{}, fmt.Errorf("%s transport %q is unsupported", filepath.Base(entryPath), manifest.Protocol.Transport)
		}
		runtimeCfg, err := ResolveRuntime(entryPath, manifest.Runtime)
		if err != nil {
			return LoadedEntry{}, fmt.Errorf("%s runtime invalid: %w", filepath.Base(entryPath), err)
		}
		return LoadedEntry{Runtime: runtimeCfg, AIID: manifest.AIID}, nil
	} else if err != nil && !os.IsNotExist(err) {
		return LoadedEntry{}, err
	}

	if err := ValidateMetadata(matchMeta); err != nil {
		return LoadedEntry{}, err
	}
	return LoadedEntry{Runtime: FallbackRuntime(entryPath), AIID: filepath.Base(entryPath)}, nil
}

func readJSON(path string, dest any) error {
	// #nosec G304 -- callers explicitly choose local entry and manifest paths during validation.
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}
