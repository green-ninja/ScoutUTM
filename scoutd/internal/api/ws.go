package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/scoututm/scoutd/internal/ws"
	"github.com/coder/websocket"
)

const wsIdleTimeout = 5 * time.Minute

// handleWebSocket upgrades HTTP to WebSocket and registers the client with the Hub.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if s.Hub.ClientCount() >= 10 {
		w.Header().Set("Retry-After", "30")
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "too many connections"})
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{r.Host},
	})
	if err != nil {
		log.Printf("websocket accept: %v", err)
		return
	}

	client := ws.NewClient(s.Hub, conn, clientIP(r))
	if !s.Hub.Register(client) {
		conn.Close(websocket.StatusTryAgainLater, "too many connections")
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Start write pump in background.
	go client.WritePump(ctx)

	// Read loop with idle timeout: disconnect clients with no messages after 5 min.
	for {
		conn.SetReadLimit(512) // Client messages are small (pings).
		readCtx, readCancel := context.WithTimeout(ctx, wsIdleTimeout)
		_, _, err := conn.Read(readCtx)
		readCancel()
		if err != nil {
			break
		}
	}

	s.Hub.Unregister(client)
	conn.Close(websocket.StatusNormalClosure, "")
}
