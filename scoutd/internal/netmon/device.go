package netmon

import (
	"sync"
	"time"
)

// OfflineTimeout is how long after a device was last confirmed REACHABLE
// before it is considered offline. The Linux kernel transitions ARP entries
// from REACHABLE to STALE after ~30s of silence, but STALE entries persist
// indefinitely on small networks. This timeout ensures devices that have
// truly gone offline are reflected in the UI.
const OfflineTimeout = 5 * time.Minute

// Device represents a discovered network device.
type Device struct {
	MAC          string    `json:"mac"`
	IP           string    `json:"ip"`
	Hostname     string    `json:"hostname"`
	Zone         string    `json:"zone"`
	Online       bool      `json:"online"`
	LastSeen     time.Time `json:"last_seen"`
	Name         string    `json:"name"`
	ProfileID    string    `json:"profile_id"`
	Manufacturer string    `json:"manufacturer"`
}

// Scanner discovers devices on the local network via the kernel neighbor
// table (ip neigh) and DHCP leases.
type Scanner struct {
	leaseFile     string
	neighFile     string // test override; empty = run `ip neigh` command
	mu            sync.RWMutex
	devices       map[string]*Device
	lastReachable map[string]time.Time // MAC → last time seen in REACHABLE state
}

// NewScanner creates a Scanner.
func NewScanner(leaseFile string) *Scanner {
	return &Scanner{
		leaseFile:     leaseFile,
		devices:       make(map[string]*Device),
		lastReachable: make(map[string]time.Time),
	}
}

// SetNeighFile overrides the neighbor table source (for testing).
// The file should contain `ip -4 neigh show` format output.
func (s *Scanner) SetNeighFile(path string) {
	s.neighFile = path
}

// Devices returns a snapshot of all discovered devices.
func (s *Scanner) Devices() []Device {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Device, 0, len(s.devices))
	for _, d := range s.devices {
		result = append(result, *d)
	}
	return result
}

// Scan reads the kernel neighbor table and DHCP leases, merging results
// into the device map. Only devices in REACHABLE/DELAY/PROBE NUD state
// are marked online. STALE devices remain online for up to OfflineTimeout
// after they were last REACHABLE.
func (s *Scanner) Scan() error {
	var entries []neighEntry
	var err error
	if s.neighFile != "" {
		entries, err = parseNeighFile(s.neighFile)
	} else {
		entries, err = readNeighTable()
	}
	if err != nil {
		return err
	}

	leases, err := readDHCPLeases(s.leaseFile)
	if err != nil {
		return err
	}

	// Build hostname lookup from DHCP leases.
	hostnames := make(map[string]string) // MAC → hostname
	for _, l := range leases {
		hostnames[l.MAC] = l.Hostname
	}

	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Mark all existing devices as offline first.
	for _, d := range s.devices {
		d.Online = false
	}

	// Merge neighbor entries.
	for _, e := range entries {
		d, ok := s.devices[e.MAC]
		if !ok {
			d = &Device{MAC: e.MAC, Manufacturer: LookupOUI(e.MAC)}
			s.devices[e.MAC] = d
		}
		d.IP = e.IP

		switch e.State {
		case "REACHABLE", "DELAY", "PROBE":
			d.Online = true
			d.LastSeen = now
			s.lastReachable[e.MAC] = now
		case "STALE":
			// STALE means the kernel hasn't confirmed reachability recently.
			// Keep the device online if it was REACHABLE within the timeout.
			if lr, ok := s.lastReachable[e.MAC]; ok && now.Sub(lr) < OfflineTimeout {
				d.Online = true
				d.LastSeen = now
			}
		// FAILED, INCOMPLETE → stays offline
		}

		if h, ok := hostnames[e.MAC]; ok && h != "" {
			d.Hostname = h
		}
	}

	// Merge DHCP leases for devices not already seen via neighbor table.
	for _, l := range leases {
		if _, ok := s.devices[l.MAC]; !ok {
			s.devices[l.MAC] = &Device{
				MAC:          l.MAC,
				IP:           l.IP,
				Hostname:     l.Hostname,
				Online:       false,
				LastSeen:     time.Unix(l.Expiry, 0),
				Manufacturer: LookupOUI(l.MAC),
			}
		}
	}

	return nil
}

// LookupByIP finds a device by IP and returns its MAC and name.
func (s *Scanner) LookupByIP(ip string) (mac, name string, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, d := range s.devices {
		if d.IP == ip {
			return d.MAC, d.Name, true
		}
	}
	return "", "", false
}

// SetDeviceName sets the user-assigned name for a device by MAC.
func (s *Scanner) SetDeviceName(mac, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d, ok := s.devices[mac]; ok {
		d.Name = name
	}
}

// SetDeviceProfile sets the parental profile for a device by MAC.
func (s *Scanner) SetDeviceProfile(mac, profileID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d, ok := s.devices[mac]; ok {
		d.ProfileID = profileID
	}
}
