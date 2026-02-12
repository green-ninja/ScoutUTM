package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const defaultTTL = 24 * time.Hour

type session struct {
	Token    string
	LastSeen time.Time
}

// SessionManager holds active sessions in memory.
type SessionManager struct {
	mu       sync.Mutex
	sessions map[string]*session
	ttl      time.Duration
}

// loginAttempt tracks failed login attempts for rate limiting.
type loginAttempt struct {
	count   int
	resetAt time.Time
}

// LoginRateLimiter enforces brute-force protection on login attempts.
// Per-IP: 5 failures per minute, then blocked for 5 minutes.
// Global: 20 failures per minute across all IPs.
type LoginRateLimiter struct {
	mu       sync.Mutex
	perIP    map[string]*loginAttempt
	global   *loginAttempt
	maxPerIP int
	maxGlob  int
	window   time.Duration
	blockDur time.Duration
}

// NewLoginRateLimiter creates a rate limiter with PLAN-specified defaults.
func NewLoginRateLimiter() *LoginRateLimiter {
	return &LoginRateLimiter{
		perIP:    make(map[string]*loginAttempt),
		global:   &loginAttempt{},
		maxPerIP: 5,
		maxGlob:  20,
		window:   1 * time.Minute,
		blockDur: 5 * time.Minute,
	}
}

// Allow checks whether a login attempt from the given IP is permitted.
// Returns true if allowed, false if rate-limited.
func (l *LoginRateLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	// Check per-IP block.
	if a, ok := l.perIP[ip]; ok {
		if now.Before(a.resetAt) && a.count >= l.maxPerIP {
			return false
		}
		if now.After(a.resetAt) {
			a.count = 0
		}
	}

	// Check global limit.
	if now.After(l.global.resetAt) {
		l.global.count = 0
	}
	if l.global.count >= l.maxGlob {
		return false
	}

	return true
}

// RecordFailure records a failed login from the given IP.
func (l *LoginRateLimiter) RecordFailure(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	a, ok := l.perIP[ip]
	if !ok {
		a = &loginAttempt{}
		l.perIP[ip] = a
	}
	if now.After(a.resetAt) {
		a.count = 0
		a.resetAt = now.Add(l.window)
	}
	a.count++
	// Once the per-IP limit is hit, extend the block duration.
	if a.count >= l.maxPerIP {
		a.resetAt = now.Add(l.blockDur)
	}

	if now.After(l.global.resetAt) {
		l.global.count = 0
		l.global.resetAt = now.Add(l.window)
	}
	l.global.count++
}

// ResetIP resets the failure counter for a given IP (called on successful login).
func (l *LoginRateLimiter) ResetIP(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.perIP, ip)
}

// NewSessionManager creates a SessionManager with the default 24h TTL.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*session),
		ttl:      defaultTTL,
	}
}

// Create generates a new 256-bit session token and stores it.
func (m *SessionManager) Create() (string, error) {
	b := make([]byte, 32) // 256 bits
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)

	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[token] = &session{
		Token:    token,
		LastSeen: time.Now(),
	}
	return token, nil
}

// Validate checks if a token exists and hasn't expired. Updates LastSeen on success.
func (m *SessionManager) Validate(token string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.sessions[token]
	if !ok {
		return false
	}
	if time.Since(s.LastSeen) > m.ttl {
		delete(m.sessions, token)
		return false
	}
	s.LastSeen = time.Now()
	return true
}

// Revoke removes a session token.
func (m *SessionManager) Revoke(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, token)
}

// Cleanup removes all expired sessions.
func (m *SessionManager) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for token, s := range m.sessions {
		if now.Sub(s.LastSeen) > m.ttl {
			delete(m.sessions, token)
		}
	}
}
