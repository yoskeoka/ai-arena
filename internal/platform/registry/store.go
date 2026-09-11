package registry

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/mod/semver"
)

// InMemoryStore is a registry store backed by an in-memory map.
type InMemoryStore struct {
	records  map[RegistryKey][]DescriptorRecord
	fallback *InMemoryStore
}

// NewInMemoryStore constructs a store preloaded with descriptor records.
func NewInMemoryStore(records ...DescriptorRecord) (*InMemoryStore, error) {
	store := &InMemoryStore{records: make(map[RegistryKey][]DescriptorRecord, len(records))}
	for _, record := range records {
		if err := store.Register(record); err != nil {
			return nil, err
		}
	}
	return store, nil
}

// Register inserts one admitted release. Lookup selects the greatest exact
// semantic version within the record's stable game-id/major key.
func (s *InMemoryStore) Register(record DescriptorRecord) error {
	if err := validateDescriptorRecord(record); err != nil {
		return err
	}
	for _, existing := range s.records[record.RegistryKey] {
		if existing.GameVersion == record.GameVersion {
			if existing.ArtifactID == record.ArtifactID && existing.BuilderID == record.BuilderID {
				return nil
			}
			return fmt.Errorf("registry: duplicate descriptor release %s@%s", record.GameID, record.GameVersion)
		}
	}
	s.records[record.RegistryKey] = append(s.records[record.RegistryKey], copyDescriptorRecord(record))
	return nil
}

// Lookup resolves a descriptor record by registry key.
func (s *InMemoryStore) Lookup(ctx context.Context, key RegistryKey) (DescriptorRecord, error) {
	if err := validateRegistryKey(key); err != nil {
		return DescriptorRecord{}, err
	}
	releases, ok := s.records[key]
	if ok {
		return copyDescriptorRecord(latestRelease(releases)), nil
	}
	if s.fallback != nil {
		if record, err := s.fallback.Lookup(ctx, key); err == nil {
			return record, nil
		} else if !errors.Is(err, ErrRecordNotFound) {
			return DescriptorRecord{}, err
		}
	}
	if s.hasGameID(key.GameID) {
		return DescriptorRecord{}, fmt.Errorf("registry: unsupported game version major %d for game %q", key.GameVersionMajor, key.GameID)
	}
	return DescriptorRecord{}, fmt.Errorf("registry: unsupported game %q: %w", key.GameID, ErrRecordNotFound)
}

// LookupArtifact finds an exact admitted release by its immutable digest.
func (s *InMemoryStore) LookupArtifact(_ context.Context, artifactID string) (DescriptorRecord, error) {
	for _, releases := range s.records {
		for _, record := range releases {
			if record.ArtifactID == artifactID {
				return copyDescriptorRecord(record), nil
			}
		}
	}
	return DescriptorRecord{}, fmt.Errorf("registry: unsupported artifact %q: %w", artifactID, ErrRecordNotFound)
}

// ExternalPrimaryStore composes a durable admitted tier with immutable built-ins.
// Only an explicit not-found result may fall through to the built-in tier.
type ExternalPrimaryStore struct {
	primary  RegistryStore
	fallback RegistryStore
}

func NewExternalPrimaryStore(primary, fallback RegistryStore) (*ExternalPrimaryStore, error) {
	if primary == nil || fallback == nil {
		return nil, fmt.Errorf("registry: primary and fallback stores are required")
	}
	return &ExternalPrimaryStore{primary: primary, fallback: fallback}, nil
}

func (s *ExternalPrimaryStore) Lookup(ctx context.Context, key RegistryKey) (DescriptorRecord, error) {
	record, err := s.primary.Lookup(ctx, key)
	if err == nil || !errors.Is(err, ErrRecordNotFound) {
		return record, err
	}
	return s.fallback.Lookup(ctx, key)
}

func (s *ExternalPrimaryStore) LookupArtifact(ctx context.Context, artifactID string) (DescriptorRecord, error) {
	lookup, ok := s.primary.(ArtifactRecordLookup)
	if !ok {
		return DescriptorRecord{}, fmt.Errorf("registry: configured primary cannot look up artifact identity")
	}
	return lookup.LookupArtifact(ctx, artifactID)
}

func (s *ExternalPrimaryStore) Register(ctx context.Context, record DescriptorRecord) error {
	registrar, ok := s.primary.(RecordRegistrar)
	if !ok {
		return fmt.Errorf("registry: configured primary is read-only")
	}
	return registrar.Register(ctx, record)
}

func (s *InMemoryStore) hasGameID(gameID string) bool {
	for key := range s.records {
		if key.GameID == gameID {
			return true
		}
	}
	return s.fallback != nil && s.fallback.hasGameID(gameID)
}

// newTieredStore creates a writable primary store with a read-only fallback tier.
func newTieredStore(fallback *InMemoryStore) *InMemoryStore {
	return &InMemoryStore{
		records:  make(map[RegistryKey][]DescriptorRecord),
		fallback: fallback,
	}
}

func cloneInMemoryStore(source *InMemoryStore) (*InMemoryStore, error) {
	clone, err := NewInMemoryStore()
	if err != nil {
		return nil, err
	}
	for _, releases := range source.records {
		for _, record := range releases {
			if err := clone.Register(record); err != nil {
				return nil, err
			}
		}
	}
	return clone, nil
}

func validateDescriptorRecord(record DescriptorRecord) error {
	if record.GameID == "" {
		return fmt.Errorf("registry: game_id is required")
	}
	if err := validateRegistryKey(record.RegistryKey); err != nil {
		return err
	}
	if record.RegistryKey.GameID != record.GameID {
		return fmt.Errorf("registry: descriptor game_id %q does not match key %q", record.GameID, record.RegistryKey.GameID)
	}
	if record.GameVersion != "" {
		version := "v" + strings.TrimPrefix(record.GameVersion, "v")
		if !semver.IsValid(version) {
			return fmt.Errorf("registry: invalid game_version %q", record.GameVersion)
		}
		if semver.Major(version) != fmt.Sprintf("v%d", record.RegistryKey.GameVersionMajor) {
			return fmt.Errorf("registry: game_version %q does not match key major %d", record.GameVersion, record.RegistryKey.GameVersionMajor)
		}
	}
	if err := validateBuildMode(record.BuildMode); err != nil {
		return err
	}
	if record.BuilderID == "" {
		return fmt.Errorf("registry: builder_id is required")
	}
	if err := validateBuildConstraints(record.BuildConstraints); err != nil {
		return err
	}
	return nil
}

func copyDescriptorRecord(record DescriptorRecord) DescriptorRecord {
	record.BuildConstraints = copyBuildConstraints(record.BuildConstraints)
	record.RuntimeArgs = append([]string(nil), record.RuntimeArgs...)
	return record
}

func latestRelease(releases []DescriptorRecord) DescriptorRecord {
	latest := releases[0]
	for _, candidate := range releases[1:] {
		if semver.Compare(releaseVersion(candidate), releaseVersion(latest)) > 0 {
			latest = candidate
		}
	}
	return latest
}

func releaseVersion(record DescriptorRecord) string {
	if record.GameVersion == "" {
		return "v0.0.0"
	}
	return "v" + strings.TrimPrefix(record.GameVersion, "v")
}
