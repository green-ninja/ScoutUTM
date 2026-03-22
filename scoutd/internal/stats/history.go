package stats

import "sync"

// ThroughputSample is a compact data point for chart history.
type ThroughputSample struct {
	Timestamp int64  `json:"timestamp"`
	RxBps     uint64 `json:"rx_bps"`
	TxBps     uint64 `json:"tx_bps"`
}

// History stores a fixed-size circular buffer of throughput samples.
type History struct {
	mu      sync.Mutex
	buf     []ThroughputSample
	head    int // next write position
	count   int
	maxSize int
}

// NewHistory creates a history buffer that holds up to maxSize samples.
func NewHistory(maxSize int) *History {
	return &History{
		buf:     make([]ThroughputSample, maxSize),
		maxSize: maxSize,
	}
}

// Record adds a snapshot to the history buffer.
func (h *History) Record(snap Snapshot) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.buf[h.head] = ThroughputSample{
		Timestamp: snap.Timestamp,
		RxBps:     snap.WANRxBps,
		TxBps:     snap.WANTxBps,
	}
	h.head = (h.head + 1) % h.maxSize
	if h.count < h.maxSize {
		h.count++
	}
}

// Query returns samples from the last `seconds` worth of history,
// ordered oldest to newest.
func (h *History) Query(seconds int) []ThroughputSample {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.count == 0 {
		return nil
	}

	// Read all entries in chronological order.
	ordered := make([]ThroughputSample, h.count)
	start := (h.head - h.count + h.maxSize) % h.maxSize
	for i := 0; i < h.count; i++ {
		ordered[i] = h.buf[(start+i)%h.maxSize]
	}

	// Find cutoff timestamp.
	latest := ordered[len(ordered)-1].Timestamp
	cutoff := latest - int64(seconds)

	// Binary search for the first entry >= cutoff.
	lo, hi := 0, len(ordered)
	for lo < hi {
		mid := (lo + hi) / 2
		if ordered[mid].Timestamp < cutoff {
			lo = mid + 1
		} else {
			hi = mid
		}
	}

	result := make([]ThroughputSample, len(ordered)-lo)
	copy(result, ordered[lo:])
	return result
}
