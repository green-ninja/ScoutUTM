package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Store manages loading, saving, and atomic updates to the config file.
type Store struct {
	configPath     string
	pendingPath    string
	backupPath     string
	preMigratePath string
	mu             sync.RWMutex
	current        *Config
}

// NewStore creates a Store for the given config file path.
func NewStore(path string) *Store {
	return &Store{
		configPath:     path,
		pendingPath:    path + ".pending",
		backupPath:     path + ".bak",
		preMigratePath: path + ".pre-migrate",
	}
}

// Load reads and parses the config file. If the file doesn't exist and
// createDefault is true, a default config is written first.
func (s *Store) Load(createDefault bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.configPath)
	if os.IsNotExist(err) && createDefault {
		cfg := DefaultConfig()
		if writeErr := s.writeConfigLocked(s.configPath, cfg); writeErr != nil {
			return fmt.Errorf("creating default config: %w", writeErr)
		}
		s.current = cfg
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}
	s.current = &cfg
	return nil
}

// Current returns a copy of the current config.
func (s *Store) Current() *Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.current == nil {
		return nil
	}
	cp := *s.current
	return &cp
}

// Save writes the current config atomically.
func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		return fmt.Errorf("no config loaded")
	}
	return s.writeConfigLocked(s.configPath, s.current)
}

// StagePending writes a proposed config to the pending file.
func (s *Store) StagePending(cfg *Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.writeConfigLocked(s.pendingPath, cfg)
}

// CommitPending promotes the pending config to live and backs up the old one.
func (s *Store) CommitPending() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Read the pending config.
	data, err := os.ReadFile(s.pendingPath)
	if err != nil {
		return fmt.Errorf("reading pending config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parsing pending config: %w", err)
	}

	// Backup current config (best-effort).
	if s.current != nil {
		_ = s.writeConfigLocked(s.backupPath, s.current)
	}

	// Atomic rename pending → live.
	if err := os.Rename(s.pendingPath, s.configPath); err != nil {
		return fmt.Errorf("committing pending config: %w", err)
	}

	s.current = &cfg
	return nil
}

// RollbackPending removes the pending file without applying it.
func (s *Store) RollbackPending() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := os.Remove(s.pendingPath)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// RecoverIfNeeded cleans up a leftover pending file from a crashed apply.
func (s *Store) RecoverIfNeeded() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := os.Stat(s.pendingPath); err == nil {
		// Pending file exists — previous apply didn't complete. Remove it.
		return os.Remove(s.pendingPath)
	}
	return nil
}

// SetCurrent replaces the in-memory config (used by migration).
func (s *Store) SetCurrent(cfg *Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current = cfg
}

// ConfigPath returns the main config file path.
func (s *Store) ConfigPath() string {
	return s.configPath
}

// PreMigratePath returns the pre-migration backup path.
func (s *Store) PreMigratePath() string {
	return s.preMigratePath
}

// writeConfigLocked writes a config to path atomically (write tmp → fsync → rename).
// Caller must hold s.mu.
func (s *Store) writeConfigLocked(path string, cfg *Config) error {
	return atomicWriteFile(path, cfg)
}

// atomicWriteFile marshals cfg to JSON and writes it atomically.
func atomicWriteFile(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("syncing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("closing temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("renaming temp file: %w", err)
	}
	return nil
}
