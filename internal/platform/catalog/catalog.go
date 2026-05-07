package catalog

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/yoskeoka/ai-arena/internal/platform/contract"
	"github.com/yoskeoka/ai-arena/internal/platform/runtime"
)

var (
	ErrInvalidMetadata      = errors.New("catalog: invalid metadata")
	ErrIncompatibleMetadata = errors.New("catalog: incompatible metadata")
)

type GameMetadata = contract.GameMetadata

type SidecarManifest struct {
	AIID     string           `json:"ai_id"`
	Protocol ProtocolManifest `json:"protocol"`
	Runtime  RuntimeManifest  `json:"runtime"`
}

type ProtocolManifest struct {
	Transport      string `json:"transport"`
	GameID         string `json:"game_id"`
	GameVersion    string `json:"game_version"`
	RulesetVersion string `json:"ruleset_version"`
}

type RuntimeManifest struct {
	Kind             runtime.Kind `json:"kind"`
	Command          []string     `json:"command"`
	Module           string       `json:"module"`
	Args             []string     `json:"args,omitempty"`
	MemoryLimitPages uint32       `json:"memory_limit_pages,omitempty"`
}

func ValidateMetadata(meta GameMetadata) error {
	if meta.GameID == "" {
		return fmt.Errorf("%w: game_id is required", ErrInvalidMetadata)
	}
	if meta.GameVersion == "" {
		return fmt.Errorf("%w: game_version is required", ErrInvalidMetadata)
	}
	if meta.RulesetVersion == "" {
		return fmt.Errorf("%w: ruleset_version is required", ErrInvalidMetadata)
	}
	if _, err := majorVersion(meta.GameVersion); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidMetadata, err)
	}
	return nil
}

func MajorVersion(version string) (int, error) {
	return majorVersion(version)
}

func Compatible(expected, actual GameMetadata) error {
	if err := ValidateMetadata(expected); err != nil {
		return err
	}
	if err := ValidateMetadata(actual); err != nil {
		return err
	}
	if expected.GameID != actual.GameID {
		return fmt.Errorf("%w: game_id mismatch", ErrIncompatibleMetadata)
	}
	expectedMajor, _ := majorVersion(expected.GameVersion)
	actualMajor, _ := majorVersion(actual.GameVersion)
	if expectedMajor != actualMajor {
		return fmt.Errorf("%w: game_version major mismatch", ErrIncompatibleMetadata)
	}
	if expected.RulesetVersion != actual.RulesetVersion {
		return fmt.Errorf("%w: ruleset_version mismatch", ErrIncompatibleMetadata)
	}
	return nil
}

func majorVersion(version string) (int, error) {
	parts := strings.Split(version, ".")
	if len(parts) == 0 || parts[0] == "" {
		return 0, errors.New("invalid semver")
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, errors.New("invalid semver")
	}
	return major, nil
}

func ResolveRuntime(entryPath string, manifest RuntimeManifest) (runtime.Config, error) {
	switch manifest.Kind {
	case "", runtime.KindLocalSubprocess:
		if len(manifest.Command) == 0 {
			return runtime.Config{}, fmt.Errorf("%w: runtime.command is required", ErrInvalidMetadata)
		}
		return runtime.Config{
			Kind:    runtime.KindLocalSubprocess,
			Command: append([]string(nil), manifest.Command...),
		}, nil
	case runtime.KindWASMWASI:
		if manifest.Module == "" {
			return runtime.Config{}, fmt.Errorf("%w: runtime.module is required", ErrInvalidMetadata)
		}
		return runtime.Config{
			Kind:             runtime.KindWASMWASI,
			ModulePath:       resolveEntryRelative(entryPath, manifest.Module),
			Args:             append([]string(nil), manifest.Args...),
			MemoryLimitPages: manifest.MemoryLimitPages,
		}, nil
	default:
		return runtime.Config{}, fmt.Errorf("%w: unsupported runtime kind %q", ErrInvalidMetadata, manifest.Kind)
	}
}

func FallbackRuntime(entryPath string) runtime.Config {
	return runtime.Config{
		Kind:    runtime.KindLocalSubprocess,
		Command: []string{entryPath},
	}
}

func resolveEntryRelative(entryPath, runtimePath string) string {
	if filepath.IsAbs(runtimePath) {
		return runtimePath
	}
	return filepath.Join(filepath.Dir(entryPath), runtimePath)
}
