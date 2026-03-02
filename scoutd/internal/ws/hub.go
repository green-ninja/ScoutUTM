package ws

import (
	"sync"
	"time"
)

const maxClients = 10

// Hub manages WebSocket client connections and broadcasts.
type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]bool
}

// NewHub creates a Hub.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[*Client]bool),
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
