package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"sync"

	"github.com/scoututm/scoutd/internal/config"
)

// applyMu serializes concurrent config applies.
var applyMu sync.Mutex

// applyConfig performs the three-phase config commit:
// 1. Validate and stage pending config
// 2. Render affected daemon configs and reload daemons
// 3. Commit pending → live (or rollback on failure)
func (s *Server) applyConfig(newCfg *config.Config) error {
	applyMu.Lock()
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
		s.rollback(oldCfg)
		return fmt.Errorf("rendering configs: %w", err)
	}

	if err := s.Services.ReloadDaemons(daemons); err != nil {
		s.rollback(oldCfg)
		return fmt.Errorf("reloading daemons: %w", err)
	}

	// Post-apply health check (PLAN section 4.4 step 9).
	if err := s.Services.HealthCheck(daemons); err != nil {
		s.rollback(oldCfg)
		return fmt.Errorf("post-apply health check: %w", err)
	}

	// Phase 3: Commit.
	if err := s.Config.CommitPending(); err != nil {
		s.rollback(oldCfg)
		return fmt.Errorf("committing config: %w", err)
	}

	// Notify WebSocket clients of config change.
	s.broadcastConfigChange(daemons)

	return nil
}

// broadcastConfigChange notifies WebSocket clients that config was updated.
func (s *Server) broadcastConfigChange(affectedDaemons []string) {
	msg, err := json.Marshal(map[string]interface{}{
		"type":    "config_changed",
		"daemons": affectedDaemons,
	})
	if err != nil {
		return
	}
	s.Hub.Broadcast(msg)
}

// rollback re-renders from the old config and removes the pending file.
func (s *Server) rollback(oldCfg *config.Config) {
	log.Println("rolling back config apply")
	if oldCfg != nil {
		if err := s.Render.RenderAll(oldCfg); err != nil {
			log.Printf("rollback render error: %v", err)
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

	// nftables: affected by Firewall, Network (zones/interfaces/VLANs), IDS changes.
	if !reflect.DeepEqual(old.Firewall, new.Firewall) ||
		!reflect.DeepEqual(old.Network, new.Network) ||
		!reflect.DeepEqual(old.IDS, new.IDS) {
		daemons = append(daemons, "nftables")
	}

	// dnsmasq: affected by Network (interfaces/VLANs), Parental (device → static IP) changes.
	if !reflect.DeepEqual(old.Network, new.Network) ||
		!reflect.DeepEqual(old.Parental, new.Parental) {
		daemons = append(daemons, "dnsmasq")
	}

	// chrony: affected by NTP server changes.
	if !reflect.DeepEqual(old.System.NTPServers, new.System.NTPServers) {
		daemons = append(daemons, "chrony")
	}

	return daemons
}

// ValidationErrors wraps config validation errors for API responses.
type ValidationErrors struct {
	Errors []config.ValidationError
}

func (e *ValidationErrors) Error() string {
	return fmt.Sprintf("%d validation error(s)", len(e.Errors))
}

// handleApplyError writes the appropriate error response for an apply failure.
func handleApplyError(w http.ResponseWriter, err error) {
	if ve, ok := err.(*ValidationErrors); ok {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"error":  "validation failed",
			"errors": ve.Errors,
		})
		return
	}
	writeJSON(w, http.StatusInternalServerError, map[string]string{
		"error": err.Error(),
	})
}
