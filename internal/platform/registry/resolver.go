package registry

import (
	"context"
	"fmt"

	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/gamemaster"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
)

type DescriptorBuilder struct {
	BuildMode                BuildMode
	BuildConstraints         BuildConstraints
	BuildSession             func(BuildSpec) (gamemaster.Session, error)
	BuildSessionFromSnapshot func(BuildSpec, game.Snapshot) (gamemaster.Session, error)
	SnapshotFromHistory      func(BuildSpec, []match.Event, int) (game.Snapshot, error)
}

type StaticResolver struct {
	builders map[string]DescriptorBuilder
}

func NewStaticResolver(builders map[string]DescriptorBuilder) (*StaticResolver, error) {
	resolver := &StaticResolver{builders: make(map[string]DescriptorBuilder, len(builders))}
	for builderID, builder := range builders {
		if err := resolver.Register(builderID, builder); err != nil {
			return nil, err
		}
	}
	return resolver, nil
}

func (r *StaticResolver) Register(builderID string, builder DescriptorBuilder) error {
	if builderID == "" {
		return fmt.Errorf("registry: builder_id is required")
	}
	if err := validateDescriptorBuilder(builder); err != nil {
		return err
	}
	if _, exists := r.builders[builderID]; exists {
		return fmt.Errorf("registry: duplicate builder_id %q", builderID)
	}
	r.builders[builderID] = copyDescriptorBuilder(builder)
	return nil
}

func (r *StaticResolver) Resolve(_ context.Context, record DescriptorRecord) (GameDescriptor, error) {
	if err := validateDescriptorRecord(record); err != nil {
		return GameDescriptor{}, err
	}
	builder, ok := r.builders[record.BuilderID]
	if !ok {
		return GameDescriptor{}, fmt.Errorf("registry: unknown builder_id %q", record.BuilderID)
	}
	if builder.BuildMode != record.BuildMode {
		return GameDescriptor{}, fmt.Errorf(
			"registry: incompatible build metadata for builder_id %q: build_mode %q does not match record %q",
			record.BuilderID,
			builder.BuildMode,
			record.BuildMode,
		)
	}
	if !sameRulesets(builder.BuildConstraints.SupportedRulesets, record.BuildConstraints.SupportedRulesets) {
		return GameDescriptor{}, fmt.Errorf(
			"registry: incompatible build metadata for builder_id %q: supported rulesets do not match record",
			record.BuilderID,
		)
	}
	return GameDescriptor{
		RegistryKey:              record.RegistryKey,
		GameID:                   record.GameID,
		BuilderID:                record.BuilderID,
		BuildMode:                builder.BuildMode,
		BuildConstraints:         copyBuildConstraints(builder.BuildConstraints),
		BuildSession:             builder.BuildSession,
		BuildSessionFromSnapshot: builder.BuildSessionFromSnapshot,
		SnapshotFromHistory:      builder.SnapshotFromHistory,
	}, nil
}

func validateDescriptorBuilder(builder DescriptorBuilder) error {
	if err := validateBuildMode(builder.BuildMode); err != nil {
		return err
	}
	if err := validateBuildConstraints(builder.BuildConstraints); err != nil {
		return err
	}
	if builder.BuildSession == nil {
		return fmt.Errorf("registry: BuildSession is required")
	}
	if builder.BuildSessionFromSnapshot == nil {
		return fmt.Errorf("registry: BuildSessionFromSnapshot is required")
	}
	if builder.SnapshotFromHistory == nil {
		return fmt.Errorf("registry: SnapshotFromHistory is required")
	}
	return nil
}

func copyDescriptorBuilder(builder DescriptorBuilder) DescriptorBuilder {
	builder.BuildConstraints = copyBuildConstraints(builder.BuildConstraints)
	return builder
}
