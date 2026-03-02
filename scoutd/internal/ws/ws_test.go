package ws

import (
	"testing"
	"time"
)

func TestHub_RegisterUnregister(t *testing.T) {
	h := NewHub()

	c := &Client{Send: make(chan []byte, 16)}
	if !h.Register(c) {
		t.Fatal("Register returned false")
	}
	if h.ClientCount() != 1 {
		t.Errorf("ClientCount = %d, want 1", h.ClientCount())
	}

	h.Unregister(c)
	if h.ClientCount() != 0 {
		t.Errorf("after unregister: ClientCount = %d, want 0", h.ClientCount())
	}

	// Verify Send channel was closed.
	select {
	case _, ok := <-c.Send:
		if ok {
			t.Error("Send channel should be closed")
		}
	default:
		t.Error("Send channel should be closed and readable")
	}
}

func TestHub_ConnectionLimit(t *testing.T) {
	h := NewHub()

	clients := make([]*Client, maxClients)
	for i := range clients {
		clients[i] = &Client{Send: make(chan []byte, 16)}
		if !h.Register(clients[i]) {
			t.Fatalf("Register %d returned false", i)
		}
	}

	if h.ClientCount() != maxClients {
		t.Errorf("ClientCount = %d, want %d", h.ClientCount(), maxClients)
	}

	// One more should fail.
	extra := &Client{Send: make(chan []byte, 16)}
	if h.Register(extra) {
		t.Error("Register should return false at capacity")
	}

	// Remove one and try again.
	h.Unregister(clients[0])
	if !h.Register(extra) {
		t.Error("Register should succeed after unregister")
	}
}

func TestHub_Broadcast(t *testing.T) {
	h := NewHub()

	c1 := &Client{Send: make(chan []byte, 16)}
	c2 := &Client{Send: make(chan []byte, 16)}
	h.Register(c1)
	h.Register(c2)

	msg := []byte(`{"type":"stats"}`)
	h.Broadcast(msg)

	// Both clients should receive the message.
	select {
	case got := <-c1.Send:
		if string(got) != string(msg) {
			t.Errorf("c1 got %q, want %q", got, msg)
		}
	case <-time.After(time.Second):
		t.Error("c1 did not receive message")
	}

	select {
	case got := <-c2.Send:
		if string(got) != string(msg) {
			t.Errorf("c2 got %q, want %q", got, msg)
		}
	case <-time.After(time.Second):
		t.Error("c2 did not receive message")
	}
}

func TestHub_Broadcast_DropsSlowClient(t *testing.T) {
	h := NewHub()

	// Client with buffer size 1.
	slow := &Client{Send: make(chan []byte, 1)}
	h.Register(slow)

	// Fill the buffer.
	slow.Send <- []byte("fill")

	// Broadcast should not block even though client buffer is full.
	done := make(chan struct{})
	go func() {
		h.Broadcast([]byte("msg"))
		close(done)
	}()

	select {
	case <-done:
		// OK, didn't block.
	case <-time.After(time.Second):
		t.Error("Broadcast blocked on slow client")
	}
}

func TestHub_StatsInterval(t *testing.T) {
	h := NewHub()

	// 0 clients → 2s
	if h.StatsInterval() != 2*time.Second {
		t.Errorf("0 clients: interval = %v, want 2s", h.StatsInterval())
	}

	// 5 clients → 2s
	for i := 0; i < 5; i++ {
		h.Register(&Client{Send: make(chan []byte, 16)})
	}
	if h.StatsInterval() != 2*time.Second {
		t.Errorf("5 clients: interval = %v, want 2s", h.StatsInterval())
	}

	// 6 clients → 5s
	h.Register(&Client{Send: make(chan []byte, 16)})
	if h.StatsInterval() != 5*time.Second {
		t.Errorf("6 clients: interval = %v, want 5s", h.StatsInterval())
	}
}
