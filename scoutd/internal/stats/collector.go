package stats

import (
	"time"
)

// Collector gathers system statistics.
type Collector struct {
	wanIface string
	prevCPU  cpuTimes
	prevNet  netCounters
	prevTime time.Time
}

// NewCollector creates a stats collector.
func NewCollector(wanIface string) *Collector {
	c := &Collector{
		wanIface: wanIface,
	}
	// Seed CPU baseline so the first Collect() returns a non-zero %.
	if cpu, err := readCPUTimes(); err == nil {
		c.prevCPU = cpu
	}
	return c
}

// Collect gathers a stats snapshot from /proc and system tools.
func (c *Collector) Collect() (Snapshot, error) {
	now := time.Now()
	snap := Snapshot{
		Timestamp: now.Unix(),
	}

	// CPU
	cpu, err := readCPUTimes()
	if err != nil {
		return snap, err
	}
	if c.prevCPU.total > 0 && cpu.total > c.prevCPU.total {
		totalDelta := cpu.total - c.prevCPU.total
		idleDelta := uint64(0)
		if cpu.idle > c.prevCPU.idle {
			idleDelta = cpu.idle - c.prevCPU.idle
		}
		if totalDelta > 0 {
			snap.CPUPercent = float64(totalDelta-idleDelta) / float64(totalDelta) * 100
		}
	}
	c.prevCPU = cpu

	// Memory
	mem, err := readMemInfo()
	if err != nil {
		return snap, err
	}
	snap.MemTotalMB = mem.totalMB
	if mem.totalMB > mem.availableMB {
		snap.MemUsedMB = mem.totalMB - mem.availableMB
	}

	// Uptime
	uptime, err := readUptime()
	if err != nil {
		return snap, err
	}
	snap.UptimeSeconds = uptime

	// Network throughput
	if c.wanIface != "" {
		net, err := readNetCounters(c.wanIface)
		if err == nil {
			elapsed := now.Sub(c.prevTime).Seconds()
			if !c.prevTime.IsZero() && elapsed > 0 &&
				net.rxBytes >= c.prevNet.rxBytes && net.txBytes >= c.prevNet.txBytes {
				snap.WANRxBps = uint64(float64(net.rxBytes-c.prevNet.rxBytes) / elapsed)
				snap.WANTxBps = uint64(float64(net.txBytes-c.prevNet.txBytes) / elapsed)
			}
			c.prevNet = net
		}
	}
	c.prevTime = now

	// Conntrack
	snap.ConnCount = readConntrackCount()

	// DNS stats from Unbound
	dns := readUnboundStats()
	snap.DNSQueries = dns.totalQueries
	snap.DNSBlocked = dns.blockedRPZ

	return snap, nil
}
