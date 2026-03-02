package netmon

import (
	"sync"
	"time"
)

// Device represents a discovered network device.
type Device struct {
	MAC       string    `json:"mac"`
	IP        string    `json:"ip"`
	Hostname  string    `json:"hostname"`
	Zone      string    `json:"zone"`
	Online    bool      `json:"online"`
	LastSeen  time.Time `json:"last_seen"`
	Name      string    `json:"name"`
	ProfileID string    `json:"profile_id"`
}

// Scanner discovers devices on the local network via ARP table and DHCP leases.
type Scanner struct {
	devMode   bool
	leaseFile string
	mu        sync.RWMutex
	devices   map[string]*Device
}

// NewScanner creates a Scanner.
func NewScanner(devMode bool, leaseFile string) *Scanner {
	return &Scanner{
		devMode:   devMode,
		leaseFile: leaseFile,
		devices:   make(map[string]*Device),
	}
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

// Scan reads the ARP table and DHCP leases, merging results into the device map.
// Devices seen in ARP are marked online; others are marked offline.
func (s *Scanner) Scan() error {
	var arpEntries []arpEntry
	var leases []dhcpLease
	var err error

	if s.devMode {
		arpEntries = syntheticARPEntries()
		leases = syntheticDHCPLeases()
	} else {
		arpEntries, err = readARPTable()
		if err != nil {
			return err
		}
		leases, err = readDHCPLeases(s.leaseFile)
		if err != nil {
			return err
		}
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

	// Merge ARP entries (these are currently online).
	for _, a := range arpEntries {
		d, ok := s.devices[a.MAC]
		if !ok {
			d = &Device{MAC: a.MAC}
			s.devices[a.MAC] = d
		}
		d.IP = a.IP
		d.Online = true
		d.LastSeen = now
		if h, ok := hostnames[a.MAC]; ok && h != "" {
			d.Hostname = h
		}
	}

	// Merge DHCP leases for devices not already seen via ARP.
	for _, l := range leases {
		if _, ok := s.devices[l.MAC]; !ok {
			s.devices[l.MAC] = &Device{
				MAC:      l.MAC,
				IP:       l.IP,
				Hostname: l.Hostname,
				Online:   false,
				LastSeen: time.Unix(l.Expiry, 0),
			}
		}
	}

	return nil
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
