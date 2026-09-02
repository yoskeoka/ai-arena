package service

import (
	"context"
	"fmt"

	"github.com/yoskeoka/ai-arena/artifactbundle"
	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/registry"
)

// ArtifactAdmissionService validates and persists uploaded immutable bundles.
type ArtifactAdmissionService struct {
	bundles  BundleStore
	registry *registry.Registry
}

// NewArtifactAdmissionService constructs an admission service backed by a bundle store and writable registry.
func NewArtifactAdmissionService(bundles BundleStore, reg *registry.Registry) (*ArtifactAdmissionService, error) {
	if bundles == nil || reg == nil {
		return nil, fmt.Errorf("service: bundle store and registry are required")
	}
	return &ArtifactAdmissionService{bundles: bundles, registry: reg}, nil
}

// RegisterGameBundle admits one official WASI game bundle and returns its release record.
func (s *ArtifactAdmissionService) RegisterGameBundle(ctx context.Context, data []byte) (registry.DescriptorRecord, error) {
	bundle, err := artifactbundle.Read(data)
	if err != nil {
		return registry.DescriptorRecord{}, err
	}
	if bundle.Manifest.ArtifactKind != "game" {
		return registry.DescriptorRecord{}, fmt.Errorf("service: artifact kind %q is not game", bundle.Manifest.ArtifactKind)
	}
	major, err := catalog.MajorVersion(bundle.Manifest.GameVersion)
	if err != nil {
		return registry.DescriptorRecord{}, err
	}
	rulesets := make([]string, 0, len(bundle.Manifest.Rulesets))
	for _, item := range bundle.Manifest.Rulesets {
		if item.RulesetVersion != "" {
			rulesets = append(rulesets, item.RulesetVersion)
		}
	}
	if len(rulesets) == 0 {
		return registry.DescriptorRecord{}, fmt.Errorf("service: game bundle must declare at least one ruleset")
	}
	if err := s.bundles.Put(ctx, bundle); err != nil {
		return registry.DescriptorRecord{}, err
	}
	record := registry.DescriptorRecord{RegistryKey: registry.RegistryKey{GameID: bundle.Manifest.GameID, GameVersionMajor: major}, GameID: bundle.Manifest.GameID, GameVersion: bundle.Manifest.GameVersion, ArtifactID: bundle.Digest, BuildMode: registry.BuildModeWASMWASI, BuilderID: "artifact/" + bundle.Digest, RuntimeArgs: append([]string(nil), bundle.Manifest.Runtime.Args...), MemoryLimitPages: bundle.Manifest.Runtime.MemoryLimitPages, BuildConstraints: registry.BuildConstraints{SupportedRulesets: rulesets}}
	if err := s.registry.Register(ctx, record); err != nil {
		return registry.DescriptorRecord{}, err
	}
	return record, nil
}
