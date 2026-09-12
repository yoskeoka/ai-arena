package service

import "github.com/yoskeoka/ai-arena/internal/platform/contract"

// These fixtures keep unrelated HTTP tests focused while the retired endpoint is
// asserted separately. They are test-only and are not wired into production.
const SourcePreset RegistrationSource = "preset"

type PresetMatchRequest struct {
	PresetID string `json:"preset_id"`
}

type MatchPresetDefinition struct {
	PresetID  string
	Game      contract.GameMetadata
	Players   []SubmittedPlayer
	OutputDir string
}

type StaticPresetCatalog struct{}

func NewStaticPresetCatalog([]MatchPresetDefinition) (*StaticPresetCatalog, error) {
	return &StaticPresetCatalog{}, nil
}
