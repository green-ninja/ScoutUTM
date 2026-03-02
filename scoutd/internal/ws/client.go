package ws

import (
	"context"

	"nhooyr.io/websocket"
)

// Client represents a WebSocket connection.
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	Send chan []byte
}

// NewClient creates a Client for the given connection.
func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:  hub,
		conn: conn,
		Send: make(chan []byte, 16),
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
