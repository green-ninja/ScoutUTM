package ws

// Client represents a WebSocket connection.
type Client struct {
	Hub  *Hub
	Send chan []byte
}
