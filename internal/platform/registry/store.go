package registry

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/mod/semver"
)

// InMemoryStore is a registry store backed by an in-memory map.
type InMemoryStore struct {
	records map[RegistryKey][]DescriptorRecord
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
			return fmt.Errorf("registry: duplicate descriptor release %s@%s", record.GameID, record.GameVersion)
		}
	}
	s.records[record.RegistryKey] = append(s.records[record.RegistryKey], copyDescriptorRecord(record))
	return nil
}

// Lookup resolves a descriptor record by registry key.
func (s *InMemoryStore) Lookup(_ context.Context, key RegistryKey) (DescriptorRecord, error) {
	if err := validateRegistryKey(key); err != nil {
		return DescriptorRecord{}, err
	}
	releases, ok := s.records[key]
	if !ok {
		if s.hasGameID(key.GameID) {
			return DescriptorRecord{}, fmt.Errorf("registry: unsupported game version major %d for game %q", key.GameVersionMajor, key.GameID)
		}
		return DescriptorRecord{}, fmt.Errorf("registry: unsupported game %q", key.GameID)
	}
	return copyDescriptorRecord(latestRelease(releases)), nil
}

func (s *InMemoryStore) hasGameID(gameID string) bool {
	for key := range s.records {
		if key.GameID == gameID {
			return true
		}
	}
	return false
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
