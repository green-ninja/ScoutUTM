package service

// Manager controls system daemons via OpenRC.
type Manager struct {
	devMode bool
}

// NewManager creates a service manager.
func NewManager(devMode bool) *Manager {
	return &Manager{devMode: devMode}
}
