package stats

import (
	"math/rand"
	"time"
)

// Collector gathers system statistics.
type Collector struct {
	devMode   bool
	wanIface  string
	startTime time.Time
	prevCPU   cpuTimes
	prevNet   netCounters
	prevTime  time.Time

	// Dev-mode synthetic DNS counters (accumulated).
	devDNSQueries uint64
	devDNSBlocked uint64
}

// NewCollector creates a stats collector.
func NewCollector(devMode bool, wanIface string) *Collector {
	c := &Collector{
		devMode:   devMode,
		wanIface:  wanIface,
		startTime: time.Now(),
	}
	// Seed CPU baseline so the first Collect() returns a non-zero %.
	if !devMode {
		if cpu, err := readCPUTimes(); err == nil {
			c.prevCPU = cpu
		}
	}
	return c
}

// Collect gathers a stats snapshot.
// In dev mode, returns synthetic data.
func (c *Collector) Collect() (Snapshot, error) {
	now := time.Now()
	snap := Snapshot{
		Timestamp: now.Unix(),
	}

	if c.devMode {
		return c.collectDev(snap), nil
	}

	return c.collectReal(snap, now)
}

func (c *Collector) collectDev(snap Snapshot) Snapshot {
	snap.CPUPercent = 25 + rand.Float64()*10 // 25-35%
	snap.MemUsedMB = 512
	snap.MemTotalMB = 8192
	snap.UptimeSeconds = uint64(time.Since(c.startTime).Seconds())
	snap.WANRxBps = 1000000 + uint64(rand.Int63n(500000))  // ~1-1.5 MB/s
	snap.WANTxBps = 200000 + uint64(rand.Int63n(100000))   // ~200-300 KB/s
	snap.ConnCount = 50 + uint64(rand.Int63n(30))
	c.devDNSQueries += uint64(5 + rand.Int63n(20))
	c.devDNSBlocked += uint64(rand.Int63n(3))
	snap.DNSQueries = c.devDNSQueries
	snap.DNSBlocked = c.devDNSBlocked
	return snap
}

func (c *Collector) collectReal(snap Snapshot, now time.Time) (Snapshot, error) {
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
