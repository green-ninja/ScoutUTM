package config

import (
	"testing"
)

func TestMigrateIfNeeded_CurrentVersion(t *testing.T) {
	path := tempConfigPath(t)
	s := NewStore(path)
	s.Load(true)

	if err := s.MigrateIfNeeded(); err != nil {
		t.Fatalf("MigrateIfNeeded on current version should be no-op: %v", err)
	}
	if s.Current().Version != CurrentSchemaVersion {
		t.Errorf("version = %d, want %d", s.Current().Version, CurrentSchemaVersion)
	}
}

func TestMigrateIfNeeded_FutureVersion(t *testing.T) {
	path := tempConfigPath(t)
	s := NewStore(path)
	s.Load(true)

	// Simulate a config from a newer firmware.
	s.SetCurrent(&Config{Version: CurrentSchemaVersion + 1})

	err := s.MigrateIfNeeded()
	if err == nil {
		t.Fatal("MigrateIfNeeded should fail for future version")
	}
}

func TestRestorePreMigrate(t *testing.T) {
	path := tempConfigPath(t)
	s := NewStore(path)
	s.Load(true)

	// Write a pre-migrate backup with different hostname.
	original := DefaultConfig()
	original.System.Hostname = "original"
	atomicWriteFile(s.PreMigratePath(), original)

	if err := s.RestorePreMigrate(); err != nil {
		t.Fatalf("RestorePreMigrate failed: %v", err)
	}
	if s.Current().System.Hostname != "original" {
		t.Errorf("hostname = %q, want %q", s.Current().System.Hostname, "original")
	}
}
