package service

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/registry"
	"github.com/yoskeoka/ai-arena/internal/platform/runtime"
)

// DefaultAdmissionValidator checks submission structure, registry compatibility, and dry-run startability.
type DefaultAdmissionValidator struct {
	registry *registry.Registry
	dryRun   DryRunChecker
}

// NewDefaultAdmissionValidator constructs the initial submission admission validator.
func NewDefaultAdmissionValidator(reg *registry.Registry, dryRun DryRunChecker) (*DefaultAdmissionValidator, error) {
	if dryRun == nil {
		return nil, fmt.Errorf("service: dry-run checker is required")
	}
	if reg == nil {
		reg = registry.Default()
	}
	return &DefaultAdmissionValidator{
		registry: reg,
		dryRun:   dryRun,
	}, nil
}

// Validate enforces the initial service skeleton admission contract.
func (v *DefaultAdmissionValidator) Validate(ctx context.Context, submission MatchSubmission) error {
	if err := ValidateSubmission(submission); err != nil {
		return err
	}

	descriptor, err := v.registry.LookupVersion(ctx, submission.Game.GameID, submission.Game.GameVersion)
	if err != nil {
		return err
	}
	if !slicesContain(descriptor.BuildConstraints.SupportedRulesets, submission.Game.RulesetVersion) {
		return fmt.Errorf(
			"service: ruleset %q is not supported for game %q version %q",
			submission.Game.RulesetVersion,
			submission.Game.GameID,
			submission.Game.GameVersion,
		)
	}
	return v.dryRun.Check(ctx, submission)
}

// LocalDryRunChecker resolves local artifact locators and verifies minimal runtime startability.
type LocalDryRunChecker struct {
	baseDir string
}

// NewLocalDryRunChecker constructs the initial local file-backed dry-run validator.
func NewLocalDryRunChecker(baseDir string) (*LocalDryRunChecker, error) {
	if strings.TrimSpace(baseDir) == "" {
		return nil, fmt.Errorf("service: base_dir is required")
	}
	return &LocalDryRunChecker{baseDir: baseDir}, nil
}

// Check validates local artifact locator resolution and runtime entrypoint existence.
func (c *LocalDryRunChecker) Check(_ context.Context, submission MatchSubmission) error {
	if looksLikeURI(submission.OutputDir) {
		return fmt.Errorf("service: output_dir must be a local path")
	}

	matchMeta := catalog.GameMetadata{
		GameID:         submission.Game.GameID,
		GameVersion:    submission.Game.GameVersion,
		RulesetVersion: submission.Game.RulesetVersion,
	}
	for _, player := range submission.Players {
		entryPath, err := resolveLocalArtifactRef(c.baseDir, player.ArtifactRef)
		if err != nil {
			return fmt.Errorf("service: %s artifact_ref invalid: %w", player.PlayerID, err)
		}
		loaded, err := catalog.LoadEntry(matchMeta, entryPath)
		if err != nil {
			return fmt.Errorf("service: %s admission failed: %w", player.PlayerID, err)
		}
		if err := ensureRuntimeStartable(c.baseDir, loaded.Runtime); err != nil {
			return fmt.Errorf("service: %s runtime entrypoint invalid: %w", player.PlayerID, err)
		}
	}
	return nil
}

func resolveLocalArtifactRef(baseDir, artifactRef string) (string, error) {
	if strings.TrimSpace(artifactRef) == "" {
		return "", fmt.Errorf("artifact_ref is required")
	}
	if parsed, err := url.Parse(artifactRef); err == nil && parsed.Scheme != "" {
		if parsed.Scheme != "file" {
			return "", fmt.Errorf("unsupported artifact_ref scheme %q", parsed.Scheme)
		}
		if parsed.Host != "" && parsed.Host != "localhost" {
			return "", fmt.Errorf("unsupported file host %q", parsed.Host)
		}
		return filepath.Clean(parsed.Path), nil
	}
	if filepath.IsAbs(artifactRef) {
		return filepath.Clean(artifactRef), nil
	}
	return filepath.Join(baseDir, filepath.Clean(artifactRef)), nil
}

func ensureRuntimeStartable(baseDir string, cfg runtime.Config) error {
	switch cfg.Kind {
	case "", runtime.KindLocalSubprocess:
		return ensureCommandStartable(baseDir, cfg.Command)
	case runtime.KindWASMWASI:
		return ensurePathExists(cfg.ModulePath)
	default:
		return fmt.Errorf("unsupported runtime kind %q", cfg.Kind)
	}
}

func ensureCommandStartable(baseDir string, command []string) error {
	if len(command) == 0 {
		return fmt.Errorf("runtime.command is required")
	}
	if len(command) >= 3 && command[0] == "go" && command[1] == "run" {
		target, err := goRunTarget(command)
		if err != nil {
			return err
		}
		return ensureCommandTargetExists(baseDir, target)
	}
	if !looksLikePath(command[0]) {
		return nil
	}
	return ensureCommandTargetExists(baseDir, command[0])
}

func goRunTarget(command []string) (string, error) {
	const separator = "--"

	for i := 2; i < len(command); i++ {
		arg := command[i]
		if arg == separator {
			if i+1 >= len(command) {
				return "", fmt.Errorf("go run command is missing a package or file target")
			}
			return command[i+1], nil
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			return arg, nil
		}
		if strings.Contains(arg, "=") || isGoRunBooleanFlag(arg) {
			continue
		}
		if isGoRunValueFlag(arg) {
			i++
			if i >= len(command) {
				return "", fmt.Errorf("go run flag %q is missing a value", arg)
			}
			continue
		}
	}
	return "", fmt.Errorf("go run command is missing a package or file target")
}

func ensureCommandTargetExists(baseDir, target string) error {
	resolved := target
	if !filepath.IsAbs(target) {
		resolved = filepath.Join(baseDir, filepath.Clean(target))
	}
	return ensurePathExists(resolved)
}

func ensurePathExists(path string) error {
	// #nosec G703 -- admission only probes operator-selected local paths after prior path/URI validation.
	if _, err := os.Stat(path); err != nil {
		return err
	}
	return nil
}

func looksLikePath(value string) bool {
	return filepath.IsAbs(value) ||
		strings.HasPrefix(value, ".") ||
		strings.Contains(value, string(filepath.Separator))
}

func looksLikeURI(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme != ""
}

func slicesContain(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func isGoRunBooleanFlag(flag string) bool {
	switch flag {
	case "-asan", "-cover", "-modcacherw", "-msan", "-n", "-race", "-trimpath", "-v", "-work", "-x":
		return true
	default:
		return false
	}
}

func isGoRunValueFlag(flag string) bool {
	switch flag {
	case "-C", "-buildmode", "-buildvcs", "-compiler", "-exec", "-mod", "-modfile", "-overlay", "-pgo", "-pkgdir", "-tags", "-toolexec":
		return true
	default:
		return false
	}
}
