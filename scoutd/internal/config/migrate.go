package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// MigrateFunc transforms a config from version N to N+1.
type MigrateFunc func(data []byte) ([]byte, error)

// migrations maps source version → migration function.
// Empty for v1 since there's nothing to migrate from.
var migrations = map[int]MigrateFunc{}

// MigrateIfNeeded checks the config version and runs sequential migrations
// if the config is older than CurrentSchemaVersion. If the config is newer
// (downgrade/rollback scenario), it returns an error so the caller can
// restore the pre-migrate backup.
func (s *Store) MigrateIfNeeded() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current == nil {
		return fmt.Errorf("no config loaded")
	}

	if s.current.Version == CurrentSchemaVersion {
		return nil // Nothing to do.
	}

	if s.current.Version > CurrentSchemaVersion {
		return fmt.Errorf("config version %d is newer than supported %d (rollback needed)", s.current.Version, CurrentSchemaVersion)
	}

	// Save a pre-migrate backup.
	if err := s.writeConfigLocked(s.preMigratePath, s.current); err != nil {
		return fmt.Errorf("saving pre-migrate backup: %w", err)
	}

	// Run migrations sequentially.
	data, err := json.MarshalIndent(s.current, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config for migration: %w", err)
	}

	for v := s.current.Version; v < CurrentSchemaVersion; v++ {
		fn, ok := migrations[v]
		if !ok {
			return fmt.Errorf("no migration from version %d to %d", v, v+1)
		}
		data, err = fn(data)
		if err != nil {
			return fmt.Errorf("migration v%d→v%d failed: %w", v, v+1, err)
		}
	}

	var migrated Config
	if err := json.Unmarshal(data, &migrated); err != nil {
		return fmt.Errorf("parsing migrated config: %w", err)
	}

	s.current = &migrated
	return s.writeConfigLocked(s.configPath, s.current)
}

// RestorePreMigrate restores the pre-migration backup.
func (s *Store) RestorePreMigrate() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.preMigratePath)
	if err != nil {
		return fmt.Errorf("reading pre-migrate backup: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parsing pre-migrate backup: %w", err)
	}

	if err := atomicWriteFile(s.configPath, &cfg); err != nil {
		return fmt.Errorf("restoring config: %w", err)
	}

	s.current = &cfg
	return nil
}
