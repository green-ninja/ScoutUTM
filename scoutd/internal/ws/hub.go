package ws

import "sync"

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

// Register adds a client.
func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = true
}

// Unregister removes a client.
func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, c)
}

// Broadcast sends a message to all connected clients (stub — no-op for now).
func (h *Hub) Broadcast(msg []byte) {
	// Will be implemented when WebSocket upgrade handling is added.
}

// Run is a stub goroutine for the hub event loop.
func (h *Hub) Run() {
	// Will be implemented with actual WebSocket handling.
}
