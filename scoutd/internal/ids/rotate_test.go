package ids

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRotateEVELog_UnderLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "eve.json")
	os.WriteFile(path, []byte("small"), 0644)

	if err := RotateEVELog(path, 1024, 3); err != nil {
		t.Fatalf("RotateEVELog: %v", err)
	}
	// File should still exist (not rotated).
	if _, err := os.Stat(path); err != nil {
		t.Error("original file should still exist when under limit")
	}
}

func TestRotateEVELog_OverLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "eve.json")

	// Write file exceeding 10-byte limit.
	os.WriteFile(path, make([]byte, 20), 0644)

	if err := RotateEVELog(path, 10, 3); err != nil {
		t.Fatalf("RotateEVELog: %v", err)
	}

	// Original should be gone.
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("original file should be rotated away")
	}
	// .1 should exist.
	if _, err := os.Stat(path + ".1"); err != nil {
		t.Error("rotated file .1 should exist")
	}
}

func TestRotateEVELog_ShiftsExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "eve.json")

	// Create .1 and .2 from previous rotations.
	os.WriteFile(path+".1", []byte("old1"), 0644)
	os.WriteFile(path+".2", []byte("old2"), 0644)

	// Write current file exceeding limit.
	os.WriteFile(path, []byte("current-data-exceeds"), 0644)

	if err := RotateEVELog(path, 10, 3); err != nil {
		t.Fatalf("RotateEVELog: %v", err)
	}

	// .1 should be the current file.
	data, _ := os.ReadFile(path + ".1")
	if string(data) != "current-data-exceeds" {
		t.Errorf(".1 = %q, want current data", string(data))
	}
	// .2 should be old .1.
	data, _ = os.ReadFile(path + ".2")
	if string(data) != "old1" {
		t.Errorf(".2 = %q, want old1", string(data))
	}
	// .3 should be old .2.
	data, _ = os.ReadFile(path + ".3")
	if string(data) != "old2" {
		t.Errorf(".3 = %q, want old2", string(data))
	}
}

func TestRotateEVELog_DeletesOldest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "eve.json")

	// Fill up all rotation slots.
	os.WriteFile(path+".1", []byte("r1"), 0644)
	os.WriteFile(path+".2", []byte("r2"), 0644)
	os.WriteFile(path+".3", []byte("r3"), 0644)

	// Write current file exceeding limit.
	os.WriteFile(path, make([]byte, 20), 0644)

	if err := RotateEVELog(path, 10, 3); err != nil {
		t.Fatalf("RotateEVELog: %v", err)
	}

	// .3 should be old .2 (old .3 was deleted, old .2 shifted to .3).
	data, _ := os.ReadFile(path + ".3")
	if string(data) != "r2" {
		t.Errorf(".3 = %q, want r2 (shifted from .2)", string(data))
	}
}

func TestRotateEVELog_FileDoesNotExist(t *testing.T) {
	err := RotateEVELog("/nonexistent/path/eve.json", 10, 3)
	if err != nil {
		t.Errorf("should return nil for nonexistent file, got: %v", err)
	}
}
