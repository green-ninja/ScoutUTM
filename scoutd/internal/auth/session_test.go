package auth

import (
	"testing"
	"time"
)

func TestCreateToken(t *testing.T) {
	m := NewSessionManager()
	token, err := m.Create()
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	// 32 bytes → 64 hex chars.
	if len(token) != 64 {
		t.Errorf("token length = %d, want 64", len(token))
	}
}

func TestValidateValid(t *testing.T) {
	m := NewSessionManager()
	token, _ := m.Create()
	if !m.Validate(token) {
		t.Error("Validate should return true for a fresh token")
	}
}

func TestValidateInvalid(t *testing.T) {
	m := NewSessionManager()
	if m.Validate("nonexistent-token") {
		t.Error("Validate should return false for unknown token")
	}
}

func TestValidateExpired(t *testing.T) {
	m := NewSessionManager()
	m.ttl = 1 * time.Millisecond
	token, _ := m.Create()
	time.Sleep(5 * time.Millisecond)
	if m.Validate(token) {
		t.Error("Validate should return false for expired token")
	}
}

func TestRevoke(t *testing.T) {
	m := NewSessionManager()
	token, _ := m.Create()
	m.Revoke(token)
	if m.Validate(token) {
		t.Error("Validate should return false after Revoke")
	}
}

func TestCleanup(t *testing.T) {
	m := NewSessionManager()
	m.ttl = 1 * time.Millisecond

	m.Create()
	m.Create()
	time.Sleep(5 * time.Millisecond)
	m.Cleanup()

	m.mu.Lock()
	count := len(m.sessions)
	m.mu.Unlock()
	if count != 0 {
		t.Errorf("sessions after cleanup = %d, want 0", count)
	}
}
