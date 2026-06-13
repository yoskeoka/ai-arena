package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/yoskeoka/ai-arena/internal/platform/contract"
)

var (
	// ErrPresetNotFound reports that the requested preset id is unknown.
	ErrPresetNotFound = errors.New("service: preset not found")
)

// PresetMatchRequest is the operator-facing enqueue request for one preset match.
type PresetMatchRequest struct {
	PresetID  string `json:"preset_id"`
	RunID     string `json:"run_id,omitempty"`
	MatchID   string `json:"match_id,omitempty"`
	OutputDir string `json:"output_dir,omitempty"`
}

// MatchPresetDefinition is one server-known preset match template.
type MatchPresetDefinition struct {
	PresetID  string                `json:"preset_id"`
	Game      contract.GameMetadata `json:"game"`
	Players   []SubmittedPlayer     `json:"players"`
	OutputDir string                `json:"output_dir"`
}

// MatchPresetConfig is the file-backed preset catalog shape.
type MatchPresetConfig struct {
	Presets []MatchPresetDefinition `json:"presets"`
}

// PresetCatalog resolves operator-facing preset requests into concrete submissions.
type PresetCatalog interface {
	Build(context.Context, PresetMatchRequest) (MatchSubmission, error)
}

// StaticPresetCatalog resolves requests from an in-memory preset map.
type StaticPresetCatalog struct {
	presets      map[string]MatchPresetDefinition
	newRunIDFn   func() string
	newMatchIDFn func() string
}

// NewStaticPresetCatalog constructs a catalog from validated preset definitions.
func NewStaticPresetCatalog(definitions []MatchPresetDefinition) (*StaticPresetCatalog, error) {
	if len(definitions) == 0 {
		return nil, fmt.Errorf("service: at least one preset is required")
	}

	presets := make(map[string]MatchPresetDefinition, len(definitions))
	for _, definition := range definitions {
		if strings.TrimSpace(definition.PresetID) == "" {
			return nil, fmt.Errorf("service: preset_id is required")
		}
		if _, exists := presets[definition.PresetID]; exists {
			return nil, fmt.Errorf("service: duplicate preset_id %q", definition.PresetID)
		}
		if strings.TrimSpace(definition.OutputDir) == "" {
			return nil, fmt.Errorf("service: preset %q output_dir is required", definition.PresetID)
		}
		submission := MatchSubmission{
			RunID:        "preset-validation-run",
			MatchID:      "preset-validation-match",
			Game:         definition.Game,
			Players:      append([]SubmittedPlayer(nil), definition.Players...),
			OutputDir:    definition.OutputDir,
			AttemptCount: 1,
			RunKind:      RunKindInitial,
		}
		if err := ValidateSubmission(submission); err != nil {
			return nil, fmt.Errorf("service: preset %q invalid: %w", definition.PresetID, err)
		}
		presets[definition.PresetID] = MatchPresetDefinition{
			PresetID:  definition.PresetID,
			Game:      definition.Game,
			Players:   append([]SubmittedPlayer(nil), definition.Players...),
			OutputDir: definition.OutputDir,
		}
	}

	return &StaticPresetCatalog{
		presets:      presets,
		newRunIDFn:   func() string { return "run-" + uuid.NewString() },
		newMatchIDFn: func() string { return "match-" + uuid.NewString() },
	}, nil
}

// LoadPresetCatalog reads preset definitions from one JSON file.
func LoadPresetCatalog(path string) (*StaticPresetCatalog, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("service: preset config path is required")
	}
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("service: read preset config %s: %w", path, err)
	}
	var config MatchPresetConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("service: decode preset config %s: %w", path, err)
	}
	return NewStaticPresetCatalog(config.Presets)
}

// Build resolves one request into a concrete match submission.
func (c *StaticPresetCatalog) Build(_ context.Context, req PresetMatchRequest) (MatchSubmission, error) {
	if c == nil {
		return MatchSubmission{}, fmt.Errorf("service: preset catalog is required")
	}
	presetID := strings.TrimSpace(req.PresetID)
	if presetID == "" {
		return MatchSubmission{}, fmt.Errorf("service: preset_id is required")
	}
	definition, ok := c.presets[presetID]
	if !ok {
		return MatchSubmission{}, ErrPresetNotFound
	}

	runID := strings.TrimSpace(req.RunID)
	if runID == "" {
		runID = c.newRunIDFn()
	}
	matchID := strings.TrimSpace(req.MatchID)
	if matchID == "" {
		matchID = c.newMatchIDFn()
	}
	outputDir := strings.TrimSpace(req.OutputDir)
	if outputDir == "" {
		outputDir = definition.OutputDir
	}

	return MatchSubmission{
		RunID:        runID,
		MatchID:      matchID,
		Game:         definition.Game,
		Players:      append([]SubmittedPlayer(nil), definition.Players...),
		OutputDir:    outputDir,
		AttemptCount: 1,
		RunKind:      RunKindInitial,
	}, nil
}
