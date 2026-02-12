package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func tempConfigPath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "config.json")
}

func TestLoad_CreateDefault(t *testing.T) {
	path := tempConfigPath(t)
	s := NewStore(path)

	if err := s.Load(true); err != nil {
		t.Fatalf("Load(createDefault=true) failed: %v", err)
	}

	cfg := s.Current()
	if cfg == nil {
		t.Fatal("Current() returned nil after load")
	}
	if cfg.Version != CurrentSchemaVersion {
		t.Errorf("version = %d, want %d", cfg.Version, CurrentSchemaVersion)
	}
	if cfg.System.Hostname != "scoututm" {
		t.Errorf("hostname = %q, want %q", cfg.System.Hostname, "scoututm")
	}

	// File should exist on disk.
	if _, err := os.Stat(path); err != nil {
		t.Errorf("config file not created: %v", err)
	}
}

func TestLoad_ExistingFile(t *testing.T) {
	path := tempConfigPath(t)
	cfg := DefaultConfig()
	cfg.System.Hostname = "custom"
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(path, data, 0644)

	s := NewStore(path)
	if err := s.Load(false); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if s.Current().System.Hostname != "custom" {
		t.Errorf("hostname = %q, want %q", s.Current().System.Hostname, "custom")
	}
}

func TestLoad_MissingFile_NoCreate(t *testing.T) {
	path := tempConfigPath(t)
	s := NewStore(path)
	if err := s.Load(false); err == nil {
		t.Fatal("Load(createDefault=false) on missing file should fail")
	}
}

func TestSave_AtomicWrite(t *testing.T) {
	path := tempConfigPath(t)
	s := NewStore(path)
	s.Load(true)

	s.current.System.Hostname = "modified"
	if err := s.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Reload and verify.
	s2 := NewStore(path)
	s2.Load(false)
	if s2.Current().System.Hostname != "modified" {
		t.Errorf("hostname after save/reload = %q, want %q", s2.Current().System.Hostname, "modified")
	}
}

func TestStagePending_CommitPending(t *testing.T) {
	path := tempConfigPath(t)
	s := NewStore(path)
	s.Load(true)

	// Stage a modified config.
	newCfg := DefaultConfig()
	newCfg.System.Hostname = "staged"
	if err := s.StagePending(newCfg); err != nil {
		t.Fatalf("StagePending failed: %v", err)
	}

	// Pending file should exist.
	if _, err := os.Stat(path + ".pending"); err != nil {
		t.Fatalf("pending file not created: %v", err)
	}

	// Current should still be the old config.
	if s.Current().System.Hostname != "scoututm" {
		t.Errorf("Current() changed before commit")
	}

	// Commit.
	if err := s.CommitPending(); err != nil {
		t.Fatalf("CommitPending failed: %v", err)
	}

	if s.Current().System.Hostname != "staged" {
		t.Errorf("hostname after commit = %q, want %q", s.Current().System.Hostname, "staged")
	}

	// Pending file should be gone.
	if _, err := os.Stat(path + ".pending"); !os.IsNotExist(err) {
		t.Error("pending file still exists after commit")
	}

	// Backup should exist.
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Error("backup file not created after commit")
	}
}

func TestStagePending_Rollback(t *testing.T) {
	path := tempConfigPath(t)
	s := NewStore(path)
	s.Load(true)

	newCfg := DefaultConfig()
	newCfg.System.Hostname = "staged"
	s.StagePending(newCfg)

	if err := s.RollbackPending(); err != nil {
		t.Fatalf("RollbackPending failed: %v", err)
	}

	// Pending file should be gone.
	if _, err := os.Stat(path + ".pending"); !os.IsNotExist(err) {
		t.Error("pending file still exists after rollback")
	}

	// Current should be unchanged.
	if s.Current().System.Hostname != "scoututm" {
		t.Errorf("hostname changed after rollback")
	}
}

func TestRecoverIfNeeded(t *testing.T) {
	path := tempConfigPath(t)

	// Create a leftover pending file.
	os.WriteFile(path+".pending", []byte(`{"version":1}`), 0644)

	s := NewStore(path)
	if err := s.RecoverIfNeeded(); err != nil {
		t.Fatalf("RecoverIfNeeded failed: %v", err)
	}

	if _, err := os.Stat(path + ".pending"); !os.IsNotExist(err) {
		t.Error("pending file still exists after recovery")
	}
}

func TestRecoverIfNeeded_NoPending(t *testing.T) {
	path := tempConfigPath(t)
	s := NewStore(path)
	if err := s.RecoverIfNeeded(); err != nil {
		t.Fatalf("RecoverIfNeeded with no pending file should not fail: %v", err)
	}
}

func TestConcurrentReads(t *testing.T) {
	path := tempConfigPath(t)
	s := NewStore(path)
	s.Load(true)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cfg := s.Current()
			if cfg == nil {
				t.Error("Current() returned nil during concurrent read")
			}
		}()
	}
	wg.Wait()
}
