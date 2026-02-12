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
