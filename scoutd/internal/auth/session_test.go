package auth

import (
	"fmt"
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

func TestRateLimiter_AllowsInitial(t *testing.T) {
	rl := NewLoginRateLimiter()
	if !rl.Allow("192.168.1.10") {
		t.Error("should allow initial attempt")
	}
}

func TestRateLimiter_BlocksAfterMaxFailures(t *testing.T) {
	rl := NewLoginRateLimiter()
	ip := "192.168.1.10"

	for i := 0; i < 5; i++ {
		rl.RecordFailure(ip)
	}

	if rl.Allow(ip) {
		t.Error("should block after 5 failures")
	}
}

func TestRateLimiter_AllowsDifferentIP(t *testing.T) {
	rl := NewLoginRateLimiter()

	for i := 0; i < 5; i++ {
		rl.RecordFailure("192.168.1.10")
	}

	if !rl.Allow("192.168.1.11") {
		t.Error("should allow different IP")
	}
}

func TestRateLimiter_GlobalLimit(t *testing.T) {
	rl := NewLoginRateLimiter()

	// 100 failures from different IPs to hit the global limit.
	for i := 0; i < 100; i++ {
		rl.RecordFailure(fmt.Sprintf("10.%d.%d.%d", i/256/256%256, i/256%256, i%256))
	}

	// 101st from a new IP should be blocked by global limit.
	if rl.Allow("172.16.0.1") {
		t.Error("should block after global limit reached")
	}
}

func TestRateLimiter_ResetIPClearsBlock(t *testing.T) {
	rl := NewLoginRateLimiter()
	ip := "192.168.1.10"

	for i := 0; i < 5; i++ {
		rl.RecordFailure(ip)
	}

	rl.ResetIP(ip)

	if !rl.Allow(ip) {
		t.Error("should allow after ResetIP")
	}
}
