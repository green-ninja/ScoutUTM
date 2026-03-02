package stats

// Snapshot holds a point-in-time system stats reading.
type Snapshot struct {
	Timestamp     int64   `json:"timestamp"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemUsedMB     uint64  `json:"mem_used_mb"`
	MemTotalMB    uint64  `json:"mem_total_mb"`
	UptimeSeconds uint64  `json:"uptime_seconds"`
	WANRxBps      uint64  `json:"wan_rx_bps"`
	WANTxBps      uint64  `json:"wan_tx_bps"`
	ConnCount     uint64  `json:"conn_count"`
	DNSQueries    uint64  `json:"dns_queries"`
	DNSBlocked    uint64  `json:"dns_blocked"`
}
