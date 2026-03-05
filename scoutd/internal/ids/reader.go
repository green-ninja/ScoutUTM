package ids

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

// Alert represents a Suricata EVE alert event.
type Alert struct {
	Timestamp time.Time   `json:"timestamp"`
	SrcIP     string      `json:"src_ip"`
	SrcPort   int         `json:"src_port"`
	DestIP    string      `json:"dest_ip"`
	DestPort  int         `json:"dest_port"`
	Proto     string      `json:"proto"`
	EventType string      `json:"event_type"`
	Alert     AlertDetail `json:"alert"`
}

// AlertDetail holds the nested alert fields from Suricata EVE.
type AlertDetail struct {
	Action      string `json:"action"`
	GID         int    `json:"gid"`
	SignatureID int    `json:"signature_id"`
	Rev         int    `json:"rev"`
	Signature   string `json:"signature"`
	Category    string `json:"category"`
	Severity    int    `json:"severity"`
}

// eveEvent is the raw JSON structure from Suricata EVE log.
type eveEvent struct {
	Timestamp string      `json:"timestamp"`
	SrcIP     string      `json:"src_ip"`
	SrcPort   int         `json:"src_port"`
	DestIP    string      `json:"dest_ip"`
	DestPort  int         `json:"dest_port"`
	Proto     string      `json:"proto"`
	EventType string      `json:"event_type"`
	Alert     AlertDetail `json:"alert"`
}

const ringCapacity = 100

// RingBuffer is a fixed-size circular buffer for alerts.
type RingBuffer struct {
	mu    sync.RWMutex
	items []Alert
	pos   int
	full  bool
}

// NewRingBuffer creates a ring buffer with fixed capacity.
func NewRingBuffer() *RingBuffer {
	return &RingBuffer{
		items: make([]Alert, ringCapacity),
	}
}

// Add appends an alert to the ring buffer, overwriting the oldest if full.
func (rb *RingBuffer) Add(a Alert) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.items[rb.pos] = a
	rb.pos = (rb.pos + 1) % ringCapacity
	if rb.pos == 0 || rb.full {
		rb.full = true
	}
}

// All returns all alerts in chronological order (oldest first).
func (rb *RingBuffer) All() []Alert {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	if !rb.full {
		result := make([]Alert, rb.pos)
		copy(result, rb.items[:rb.pos])
		return result
	}
	result := make([]Alert, ringCapacity)
	copy(result, rb.items[rb.pos:])
	copy(result[ringCapacity-rb.pos:], rb.items[:rb.pos])
	return result
}

// Reader reads Suricata EVE alerts from a Unix datagram socket.
type Reader struct {
	socketPath string
	logPath    string
	alerts     *RingBuffer

	// OnAlert is called when a new alert is parsed, before ring buffer insert.
	OnAlert func(Alert)

	logMu   sync.Mutex
	logFile *os.File
}

// NewReader creates an EVE alert reader.
func NewReader(socketPath, logPath string) *Reader {
	return &Reader{
		socketPath: socketPath,
		logPath:    logPath,
		alerts:     NewRingBuffer(),
	}
}

// Alerts returns the ring buffer for API access.
func (r *Reader) Alerts() *RingBuffer {
	return r.alerts
}

// Run starts reading EVE alerts from a Unix datagram socket.
func (r *Reader) Run(ctx context.Context) {
	if r.logPath != "" {
		r.openLog()
		defer r.closeLog()
	}

	for {
		if err := r.listen(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("EVE reader error: %v, retrying in 5s", err)
			select {
			case <-time.After(5 * time.Second):
			case <-ctx.Done():
				return
			}
		}
	}
}

// listen opens the Unix datagram socket and reads events.
func (r *Reader) listen(ctx context.Context) error {
	// Remove stale socket file.
	os.Remove(r.socketPath)

	addr, err := net.ResolveUnixAddr("unixgram", r.socketPath)
	if err != nil {
		return err
	}

	conn, err := net.ListenUnixgram("unixgram", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Close socket when context is cancelled.
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			conn.Close()
		case <-done:
		}
	}()

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 64*1024), 64*1024)
	for scanner.Scan() {
		r.processLine(scanner.Bytes())
	}
	return scanner.Err()
}

// processLine parses a single EVE JSON line and adds alerts to the ring buffer.
func (r *Reader) processLine(line []byte) {
	var ev eveEvent
	if err := json.Unmarshal(line, &ev); err != nil {
		return
	}
	if ev.EventType != "alert" {
		return
	}

	ts, err := time.Parse("2006-01-02T15:04:05.999999-0700", ev.Timestamp)
	if err != nil {
		// Suricata 7+ may use colon in timezone offset.
		ts, err = time.Parse("2006-01-02T15:04:05.999999-07:00", ev.Timestamp)
		if err != nil {
			ts = time.Now()
		}
	}

	alert := Alert{
		Timestamp: ts,
		SrcIP:     ev.SrcIP,
		SrcPort:   ev.SrcPort,
		DestIP:    ev.DestIP,
		DestPort:  ev.DestPort,
		Proto:     ev.Proto,
		EventType: ev.EventType,
		Alert:     ev.Alert,
	}

	// Notify callback before ring buffer insert.
	if r.OnAlert != nil {
		r.OnAlert(alert)
	}

	r.alerts.Add(alert)

	// Append to persistent log file.
	if r.logPath != "" {
		r.appendLog(line)
	}
}

// openLog opens the persistent EVE log file for appending.
func (r *Reader) openLog() {
	f, err := os.OpenFile(r.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("EVE log open error: %v", err)
		return
	}
	r.logMu.Lock()
	r.logFile = f
	r.logMu.Unlock()
}

// closeLog closes the persistent EVE log file.
func (r *Reader) closeLog() {
	r.logMu.Lock()
	defer r.logMu.Unlock()
	if r.logFile != nil {
		r.logFile.Close()
		r.logFile = nil
	}
}

// appendLog appends raw JSON to the persistent EVE log file.
func (r *Reader) appendLog(line []byte) {
	r.logMu.Lock()
	defer r.logMu.Unlock()
	if r.logFile == nil {
		return
	}
	r.logFile.Write(line)
	r.logFile.Write([]byte("\n"))
}
