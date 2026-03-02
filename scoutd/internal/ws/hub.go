package ws

import (
	"encoding/json"
	"sync"
	"time"
)

const maxClients = 10

// Hub manages WebSocket client connections and broadcasts.
type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]bool

	// Alert rate limiting: per-client token buckets.
	alertMu     sync.Mutex
	alertTokens map[*Client]*tokenBucket
}

// NewHub creates a Hub.
func NewHub() *Hub {
	return &Hub{
		clients:     make(map[*Client]bool),
		alertTokens: make(map[*Client]*tokenBucket),
	}
}

// Register adds a client. Returns false if at capacity.
func (h *Hub) Register(c *Client) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.clients) >= maxClients {
		return false
	}
	h.clients[c] = true

	h.alertMu.Lock()
	h.alertTokens[c] = newTokenBucket(10, 10) // 10 alerts/sec, burst 10
	h.alertMu.Unlock()

	return true
}

// Unregister removes a client and closes its Send channel.
func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c.Send)
	}

	h.alertMu.Lock()
	delete(h.alertTokens, c)
	h.alertMu.Unlock()
}

// Broadcast sends a message to all connected clients.
// Non-blocking: drops message for slow clients.
func (h *Hub) Broadcast(msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		select {
		case c.Send <- msg:
		default:
			// Client too slow, drop message.
		}
	}
}

// BroadcastDevices sends a devices update to all clients.
func (h *Hub) BroadcastDevices(devices interface{}) {
	msg, err := json.Marshal(map[string]interface{}{
		"type": "devices",
		"data": devices,
	})
	if err != nil {
		return
	}
	h.Broadcast(msg)
}

// BroadcastAlert sends an alert to all clients, subject to per-client rate limiting.
// Max 10 alerts/sec per client. Excess silently dropped.
func (h *Hub) BroadcastAlert(alert interface{}) {
	msg, err := json.Marshal(map[string]interface{}{
		"type": "alert",
		"data": alert,
	})
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	h.alertMu.Lock()
	defer h.alertMu.Unlock()

	for c := range h.clients {
		bucket, ok := h.alertTokens[c]
		if !ok {
			continue
		}
		if !bucket.allow() {
			continue // Rate limited.
		}
		select {
		case c.Send <- msg:
		default:
		}
	}
}

// ClientCount returns the number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// StatsInterval returns the broadcast interval based on client count.
// 2s for <=5 clients, 5s for more.
func (h *Hub) StatsInterval() time.Duration {
	if h.ClientCount() <= 5 {
		return 2 * time.Second
	}
	return 5 * time.Second
}

// tokenBucket implements a simple token-bucket rate limiter.
type tokenBucket struct {
	rate     float64
	burst    float64
	tokens   float64
	lastTime time.Time
}

func newTokenBucket(rate, burst float64) *tokenBucket {
	return &tokenBucket{
		rate:     rate,
		burst:    burst,
		tokens:   burst,
		lastTime: time.Now(),
	}
}

func (tb *tokenBucket) allow() bool {
	now := time.Now()
	elapsed := now.Sub(tb.lastTime).Seconds()
	tb.tokens += elapsed * tb.rate
	if tb.tokens > tb.burst {
		tb.tokens = tb.burst
	}
	tb.lastTime = now

	if tb.tokens < 1 {
		return false
	}
	tb.tokens--
	return true
}
