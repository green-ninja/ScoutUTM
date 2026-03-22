package ws

import (
	"context"

	"github.com/coder/websocket"
)

// Client represents a WebSocket connection.
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	Send chan []byte
	IP   string // Remote IP for per-IP connection limiting.
}

// NewClient creates a Client for the given connection.
func NewClient(hub *Hub, conn *websocket.Conn, ip string) *Client {
	return &Client{
		hub:  hub,
		conn: conn,
		Send: make(chan []byte, 16),
		IP:   ip,
	}
}

// WritePump drains the Send channel and writes to the WebSocket connection.
// Returns when the context is cancelled or an error occurs.
func (c *Client) WritePump(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-c.Send:
			if !ok {
				return
			}
			if err := c.conn.Write(ctx, websocket.MessageText, msg); err != nil {
				return
			}
		}
	}
}
