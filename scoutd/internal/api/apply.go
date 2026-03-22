package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"sync"
	"syscall"
	"time"

	"github.com/scoututm/scoutd/internal/config"
)

// applyMu serializes concurrent config applies.
var applyMu sync.Mutex

// ErrApplyInProgress is returned when a config apply is already running.
var ErrApplyInProgress = fmt.Errorf("config apply already in progress")

// applyConfig performs the three-phase config commit:
// 1. Validate and stage pending config
// 2. Render affected daemon configs and reload daemons
// 3. Commit pending → live (or rollback on failure)
func (s *Server) applyConfig(newCfg *config.Config) error {
	if !applyMu.TryLock() {
		return ErrApplyInProgress
	}
	defer applyMu.Unlock()

	// Validate.
	if errs := config.Validate(newCfg); len(errs) > 0 {
		return &ValidationErrors{Errors: errs}
	}

	oldCfg := s.Config.Current()

	// Phase 1: Stage pending.
	if err := s.Config.StagePending(newCfg); err != nil {
		return fmt.Errorf("staging pending config: %w", err)
	}

	// Determine which daemons need re-render/reload.
	var daemons []string
	if oldCfg == nil {
		daemons = s.Render.DaemonNames()
	} else {
		daemons = affectedDaemons(oldCfg, newCfg)
	}

	// Phase 2: Render + reload.
	if err := s.Render.RenderDaemons(newCfg, daemons); err != nil {
		s.rollback(oldCfg, newCfg, daemons)
		return fmt.Errorf("rendering configs: %w", err)
	}

	// Provision/deprovision VLAN sub-interfaces before daemon reload
	// so that nftables/dnsmasq can reference them.
	var oldVLANs []config.VLANConfig
	if oldCfg != nil {
		oldVLANs = oldCfg.Network.VLANs
	}
	if err := s.Services.SyncVLANs(oldVLANs, newCfg.Network.VLANs); err != nil {
		s.rollback(oldCfg, newCfg, daemons)
		return fmt.Errorf("syncing VLAN interfaces: %w", err)
	}

	// Apply interface IP addresses before daemon reload so that
	// daemons (dnsmasq DHCP, unbound access-control) see the correct
	// addresses on the wire.
	var oldIfaces []config.InterfaceConfig
	if oldCfg != nil {
		oldIfaces = oldCfg.Network.Interfaces
	}
	if err := s.Services.ApplyInterfaceAddresses(oldIfaces, newCfg.Network.Interfaces); err != nil {
		s.rollback(oldCfg, newCfg, daemons)
		return fmt.Errorf("applying interface addresses: %w", err)
	}

	if err := s.Services.ReloadDaemons(daemons); err != nil {
		s.rollback(oldCfg, newCfg, daemons)
		return fmt.Errorf("reloading daemons: %w", err)
	}

	// Post-apply health check (PLAN section 4.4 step 9).
	if err := s.Services.HealthCheck(daemons); err != nil {
		s.rollback(oldCfg, newCfg, daemons)
		return fmt.Errorf("post-apply health check: %w", err)
	}

	// Phase 3: Commit.
	if err := s.Config.CommitPending(); err != nil {
		s.rollback(oldCfg, newCfg, daemons)
		return fmt.Errorf("committing config: %w", err)
	}

	// Notify WebSocket clients of config change.
	s.broadcastConfigChange(daemons)

	// If interface IPs changed, scoutd must restart to regenerate the TLS cert
	// with the new IP SANs. Schedule a self-restart after the response is sent.
	// Skip in dev mode — no OpenRC to restart the process.
	if !s.DevMode && interfaceIPsChanged(oldCfg, newCfg) {
		log.Println("network interface IPs changed, scheduling scoutd restart for TLS cert regeneration")
		go func() {
			time.Sleep(1 * time.Second) // Let the HTTP response complete.
			p, _ := os.FindProcess(os.Getpid())
			p.Signal(syscall.SIGTERM)
		}()
	}

	return nil
}

// broadcastConfigChange notifies WebSocket clients that config was updated.
func (s *Server) broadcastConfigChange(affectedDaemons []string) {
	msg, err := json.Marshal(map[string]interface{}{
		"type":    "config_changed",
		"daemons": affectedDaemons,
	})
	if err != nil {
		log.Printf("config change broadcast marshal error: %v", err)
		return
	}
	s.Hub.Broadcast(msg)
}

// rollback re-renders from the old config, reverts VLAN interfaces,
// reloads all affected daemons, and removes the pending file.
// Per PLAN section 4.4: on failure, re-render ALL templates from
// live config, reload ALL daemons, then delete pending.
func (s *Server) rollback(oldCfg *config.Config, newCfg *config.Config, daemons []string) {
	log.Println("rolling back config apply")
	if oldCfg != nil {
		if err := s.Render.RenderAll(oldCfg); err != nil {
			log.Printf("rollback render error: %v", err)
		}
		// Revert VLAN interfaces to old state.
		var newVLANs []config.VLANConfig
		var newIfaces []config.InterfaceConfig
		if newCfg != nil {
			newVLANs = newCfg.Network.VLANs
			newIfaces = newCfg.Network.Interfaces
		}
		if err := s.Services.SyncVLANs(newVLANs, oldCfg.Network.VLANs); err != nil {
			log.Printf("rollback VLAN sync error: %v", err)
		}
		// Restore old interface addresses.
		if err := s.Services.ApplyInterfaceAddresses(newIfaces, oldCfg.Network.Interfaces); err != nil {
			log.Printf("rollback interface address error: %v", err)
		}
		if err := s.Services.ReloadDaemons(daemons); err != nil {
			log.Printf("rollback reload error: %v", err)
		}
	}
	if err := s.Config.RollbackPending(); err != nil {
		log.Printf("rollback pending error: %v", err)
	}
}

// affectedDaemons compares old and new configs to determine which daemons need re-render/reload.
// Caller must ensure old is not nil (nil case is handled by applyConfig using Engine.DaemonNames).
func affectedDaemons(old, new *config.Config) []string {
	var daemons []string

	// nftables: affected by Firewall, Network (zones/interfaces/VLANs), IDS, VPN changes.
	if !reflect.DeepEqual(old.Firewall, new.Firewall) ||
		!reflect.DeepEqual(old.Network, new.Network) ||
		!reflect.DeepEqual(old.IDS, new.IDS) ||
		!reflect.DeepEqual(old.VPN, new.VPN) {
		daemons = append(daemons, "nftables")
	}

	// dnsmasq: affected by Network (interfaces/VLANs), Parental (device → static IP) changes.
	if !reflect.DeepEqual(old.Network, new.Network) ||
		!reflect.DeepEqual(old.Parental, new.Parental) {
		daemons = append(daemons, "dnsmasq")
	}

	// chrony: affected by NTP server changes or network changes (allow list includes VLAN subnets).
	if !reflect.DeepEqual(old.System.NTPServers, new.System.NTPServers) ||
		!reflect.DeepEqual(old.Network, new.Network) {
		daemons = append(daemons, "chrony")
	}

	// unbound: affected by DNS, Network (access-control subnets), Parental (per-device tags) changes.
	if !reflect.DeepEqual(old.DNS, new.DNS) ||
		!reflect.DeepEqual(old.Network, new.Network) ||
		!reflect.DeepEqual(old.Parental, new.Parental) {
		daemons = append(daemons, "unbound")
	}

	// suricata: affected by IDS config changes (mode, rulesets, home_net, suppressed).
	if !reflect.DeepEqual(old.IDS, new.IDS) {
		daemons = append(daemons, "suricata")
	}

	// wireguard: affected by VPN config changes (peers, listen port, address).
	if !reflect.DeepEqual(old.VPN, new.VPN) {
		daemons = append(daemons, "wireguard")
	}

	return daemons
}

// interfaceIPsChanged returns true if any non-WAN interface IPv4 address differs.
func interfaceIPsChanged(old, new *config.Config) bool {
	if old == nil {
		return false
	}
	oldIPs := make(map[string]string, len(old.Network.Interfaces))
	for _, iface := range old.Network.Interfaces {
		if iface.Zone != "wan" {
			oldIPs[iface.Name] = iface.IPv4Address
		}
	}
	newIPs := make(map[string]string, len(new.Network.Interfaces))
	for _, iface := range new.Network.Interfaces {
		if iface.Zone != "wan" {
			newIPs[iface.Name] = iface.IPv4Address
		}
	}
	if len(oldIPs) != len(newIPs) {
		return true
	}
	for name, addr := range oldIPs {
		if newIPs[name] != addr {
			return true
		}
	}
	return false
}

// ValidationErrors wraps config validation errors for API responses.
type ValidationErrors struct {
	Errors []config.ValidationError
}

func (e *ValidationErrors) Error() string {
	return fmt.Sprintf("%d validation error(s)", len(e.Errors))
}

// handleApplyError writes the appropriate error response for an apply failure.
// Validation errors return details; other errors return a generic message and
// log the real error server-side to avoid leaking internal paths and state.
func handleApplyError(w http.ResponseWriter, err error) {
	if err == ErrApplyInProgress {
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "config apply already in progress",
		})
		return
	}
	if ve, ok := err.(*ValidationErrors); ok {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"error":  "validation failed",
			"errors": ve.Errors,
		})
		return
	}
	log.Printf("config apply error: %v", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{
		"error": "config apply failed",
	})
}
