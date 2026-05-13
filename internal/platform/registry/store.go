package registry

import (
	"context"
	"fmt"
)

// InMemoryStore is a registry store backed by an in-memory map.
type InMemoryStore struct {
	records map[RegistryKey]DescriptorRecord
}

// NewInMemoryStore constructs a store preloaded with descriptor records.
func NewInMemoryStore(records ...DescriptorRecord) (*InMemoryStore, error) {
	store := &InMemoryStore{records: make(map[RegistryKey]DescriptorRecord, len(records))}
	for _, record := range records {
		if err := store.Register(record); err != nil {
			return nil, err
		}
	}
	return store, nil
}

// Register inserts one descriptor record into the store.
func (s *InMemoryStore) Register(record DescriptorRecord) error {
	if err := validateDescriptorRecord(record); err != nil {
		return err
	}
	if _, exists := s.records[record.RegistryKey]; exists {
		return fmt.Errorf("registry: duplicate descriptor for %s@%d", record.GameID, record.RegistryKey.GameVersionMajor)
	}
	s.records[record.RegistryKey] = copyDescriptorRecord(record)
	return nil
}

// Lookup resolves a descriptor record by registry key.
func (s *InMemoryStore) Lookup(_ context.Context, key RegistryKey) (DescriptorRecord, error) {
	if err := validateRegistryKey(key); err != nil {
		return DescriptorRecord{}, err
	}
	record, ok := s.records[key]
	if !ok {
		if s.hasGameID(key.GameID) {
			return DescriptorRecord{}, fmt.Errorf("registry: unsupported game version major %d for game %q", key.GameVersionMajor, key.GameID)
		}
		return DescriptorRecord{}, fmt.Errorf("registry: unsupported game %q", key.GameID)
	}
	return copyDescriptorRecord(record), nil
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
	return record
}
