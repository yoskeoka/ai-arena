package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/runtime"
)

var preparedFixtureCache = struct {
	mu      sync.Mutex
	root    string
	entries map[string]*preparedFixtureResult
}{
	entries: make(map[string]*preparedFixtureResult),
}

type preparedFixtureResult struct {
	once      sync.Once
	entryPath string
	err       error
}

func TestMain(m *testing.M) {
	code := m.Run()
	cleanupPreparedFixtures()
	os.Exit(code)
}

func cleanupPreparedFixtures() {
	preparedFixtureCache.mu.Lock()
	root := preparedFixtureCache.root
	preparedFixtureCache.root = ""
	preparedFixtureCache.mu.Unlock()
	if root != "" {
		_ = os.RemoveAll(root)
	}
}

func newArenaRunnerCommand(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()

	fullArgs := append([]string{"run", "./cmd/arena-runner"}, prepareArenaPlayerArgs(t, args)...)
	cmd := exec.CommandContext(newTestContext(t), "go", fullArgs...)
	cmd.Dir = repoRoot(t)
	return cmd
}

func prepareArenaPlayerArgs(t *testing.T, args []string) []string {
	t.Helper()

	prepared := append([]string(nil), args...)
	repo := repoRoot(t)
	for i := 0; i < len(prepared)-1; i++ {
		if prepared[i] != "--player" {
			continue
		}
		playerID, entryPath, ok := strings.Cut(prepared[i+1], "=")
		if !ok {
			continue
		}
		resolved, err := preparedFixtureEntry(t, repo, entryPath)
		if err != nil {
			t.Fatalf("prepare fixture %s: %v", entryPath, err)
		}
		prepared[i+1] = playerID + "=" + resolved
	}
	return prepared
}

func preparedFixtureEntry(t *testing.T, repoRoot, entryPath string) (string, error) {
	t.Helper()

	if !strings.HasPrefix(entryPath, "./testdata/ai/") {
		return entryPath, nil
	}

	preparedFixtureCache.mu.Lock()
	result := preparedFixtureCache.entries[entryPath]
	if result == nil {
		result = &preparedFixtureResult{}
		preparedFixtureCache.entries[entryPath] = result
	}
	preparedFixtureCache.mu.Unlock()

	result.once.Do(func() {
		result.entryPath, result.err = buildPreparedFixture(t, repoRoot, entryPath)
	})
	return result.entryPath, result.err
}

func buildPreparedFixture(t *testing.T, repoRoot, entryPath string) (string, error) {
	t.Helper()

	entryAbs := filepath.Join(repoRoot, filepath.Clean(entryPath))
	manifestPath := entryAbs + ".arena.json"
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return entryPath, nil
		}
		return "", fmt.Errorf("read sidecar manifest: %w", err)
	}

	var manifest catalog.SidecarManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return "", fmt.Errorf("decode sidecar manifest: %w", err)
	}

	preparedRoot, err := preparedFixtureRoot(repoRoot)
	if err != nil {
		return "", err
	}
	relativeKey := strings.TrimPrefix(filepath.Clean(entryPath), "./")
	preparedDir := filepath.Join(preparedRoot, relativeKey)
	if err := os.MkdirAll(preparedDir, 0o755); err != nil {
		return "", fmt.Errorf("create prepared fixture directory: %w", err)
	}
	preparedEntry := filepath.Join(preparedDir, filepath.Base(entryAbs))

	switch manifest.Runtime.Kind {
	case "", runtime.KindLocalSubprocess:
		command, err := buildPreparedLocalSubprocess(t, repoRoot, manifest.Runtime.Command, preparedEntry)
		if err != nil {
			return "", err
		}
		manifest.Runtime.Kind = runtime.KindLocalSubprocess
		manifest.Runtime.Command = command
		manifest.Runtime.Module = ""
		manifest.Runtime.Args = nil
		manifest.Runtime.MemoryLimitPages = 0
	case runtime.KindWASMWASI:
		modulePath, args, err := buildPreparedWASMFixture(t, repoRoot, entryPath, entryAbs, manifest.Runtime)
		if err != nil {
			return "", err
		}
		manifest.Runtime.Module = modulePath
		manifest.Runtime.Args = args
	default:
		return "", fmt.Errorf("unsupported runtime kind %q", manifest.Runtime.Kind)
	}

	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal prepared sidecar manifest: %w", err)
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(preparedEntry+".arena.json", encoded, 0o644); err != nil {
		return "", fmt.Errorf("write prepared sidecar manifest: %w", err)
	}
	return repoRelativePath(repoRoot, preparedEntry)
}

func preparedFixtureRoot(repoRoot string) (string, error) {
	preparedFixtureCache.mu.Lock()
	defer preparedFixtureCache.mu.Unlock()

	if preparedFixtureCache.root != "" {
		return preparedFixtureCache.root, nil
	}
	base := filepath.Join(repoRoot, ".tmp")
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", fmt.Errorf("create fixture temp base: %w", err)
	}
	root, err := os.MkdirTemp(base, "ai-arena-e2e-fixtures-")
	if err != nil {
		return "", fmt.Errorf("create prepared fixture root: %w", err)
	}
	preparedFixtureCache.root = root
	return root, nil
}

func buildPreparedLocalSubprocess(t *testing.T, repoRoot string, command []string, outputPath string) ([]string, error) {
	t.Helper()

	pkg, args, ok := parseGoRunCommand(command)
	if !ok {
		return append([]string(nil), command...), nil
	}
	if err := buildGoBinary(newTestContext(t), repoRoot, pkg, outputPath); err != nil {
		return nil, err
	}
	relativeOutput, err := repoRelativePath(repoRoot, outputPath)
	if err != nil {
		return nil, err
	}
	preparedCommand := []string{relativeOutput}
	preparedCommand = append(preparedCommand, args...)
	return preparedCommand, nil
}

func buildPreparedWASMFixture(t *testing.T, repoRoot, entryPath, entryAbs string, manifest catalog.RuntimeManifest) (string, []string, error) {
	t.Helper()

	outputPath := filepath.Join(filepath.Dir(entryAbs), filepath.Base(manifest.Module))
	relativePrepared := strings.TrimPrefix(outputPath, filepath.Dir(entryAbs)+string(filepath.Separator))
	preparedOutput := filepath.Join(filepath.Join(mustPreparedFixtureRoot(t, repoRoot), strings.TrimPrefix(filepath.Clean(entryPath), "./")), relativePrepared)
	if err := os.MkdirAll(filepath.Dir(preparedOutput), 0o755); err != nil {
		return "", nil, fmt.Errorf("create prepared wasm directory: %w", err)
	}

	if _, err := os.Stat(filepath.Join(entryAbs, "Cargo.toml")); err == nil {
		manifestPath := filepath.Join(entryAbs, "Cargo.toml")
		if err := buildRustWASM(newTestContext(t), repoRoot, manifestPath, preparedOutput); err != nil {
			return "", nil, err
		}
	} else {
		if err := buildGoWASM(newTestContext(t), repoRoot, entryPath, preparedOutput); err != nil {
			return "", nil, err
		}
	}

	moduleRef := "./" + filepath.Base(preparedOutput)
	args := append([]string(nil), manifest.Args...)
	for i := range args {
		if args[i] == manifest.Module {
			args[i] = moduleRef
		}
	}
	return moduleRef, args, nil
}

func mustPreparedFixtureRoot(t *testing.T, repoRoot string) string {
	t.Helper()

	root, err := preparedFixtureRoot(repoRoot)
	if err != nil {
		t.Fatalf("prepare fixture root: %v", err)
	}
	return root
}

func parseGoRunCommand(command []string) (string, []string, bool) {
	if len(command) < 3 || command[0] != "go" || command[1] != "run" {
		return "", nil, false
	}
	return command[2], append([]string(nil), command[3:]...), true
}

func buildGoBinary(ctx context.Context, dir, pkg, outputPath string) error {
	cmd := exec.CommandContext(ctx, "go", "build", "-o", outputPath, pkg)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build binary %s: %w\n%s", pkg, err, output)
	}
	return nil
}

func repoRelativePath(repoRoot, path string) (string, error) {
	rel, err := filepath.Rel(repoRoot, path)
	if err != nil {
		return "", fmt.Errorf("rel path: %w", err)
	}
	return "." + string(filepath.Separator) + filepath.ToSlash(rel), nil
}
