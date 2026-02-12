package stats

// Collector gathers system statistics.
type Collector struct {
	devMode bool
}

// NewCollector creates a stats collector.
func NewCollector(devMode bool) *Collector {
	return &Collector{devMode: devMode}
}
