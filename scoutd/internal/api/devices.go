package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/scoututm/scoutd/internal/config"
	"github.com/scoututm/scoutd/internal/netmon"
)

func devicesRoutes(r chi.Router, s *Server) {
	r.Get("/devices", s.handleListDevices)
	r.Get("/devices/{mac}", s.handleGetDevice)
	r.Put("/devices/{mac}", s.handleUpdateDevice)
}

type deviceResponse struct {
	MAC          string `json:"mac"`
	IP           string `json:"ip"`
	Hostname     string `json:"hostname"`
	Zone         string `json:"zone"`
	Online       bool   `json:"online"`
	LastSeen     string `json:"last_seen"`
	Name         string `json:"name"`
	ProfileID    string `json:"profile_id"`
	Manufacturer string `json:"manufacturer"`
}

func toDeviceResponse(d netmon.Device, cfg deviceOverlay) deviceResponse {
	r := deviceResponse{
		MAC:          d.MAC,
		IP:           d.IP,
		Hostname:     d.Hostname,
		Zone:         d.Zone,
		Online:       d.Online,
		LastSeen:     d.LastSeen.Format("2006-01-02T15:04:05Z"),
		Name:         d.Name,
		Manufacturer: d.Manufacturer,
	}
	// Overlay config-persisted name and profile.
	if cfg.Name != "" {
		r.Name = cfg.Name
	}
	if cfg.ProfileID != "" {
		r.ProfileID = cfg.ProfileID
	}
	return r
}

type deviceOverlay struct {
	Name      string
	ProfileID string
}

// buildOverlayMap creates a MAC→overlay from config's parental device list
// and a hypothetical device name map. Device names are stored in scanner memory
// (not persisted in config), so we just return profile assignments.
func (s *Server) buildOverlayMap() map[string]deviceOverlay {
	cfg := s.Config.Current()
	overlays := make(map[string]deviceOverlay)
	for _, pd := range cfg.Parental.Devices {
		mac := strings.ToLower(pd.MAC)
		o := overlays[mac]
		o.ProfileID = pd.ProfileID
		overlays[mac] = o
	}
	return overlays
}

// lookupDeviceIP returns the current IP for a device MAC from the scanner.
func (s *Server) lookupDeviceIP(mac string) string {
	for _, d := range s.Scanner.Devices() {
		if strings.ToLower(d.MAC) == mac {
			return d.IP
		}
	}
	return ""
}

func (s *Server) handleListDevices(w http.ResponseWriter, r *http.Request) {
	devices := s.Scanner.Devices()
	overlays := s.buildOverlayMap()

	result := make([]deviceResponse, len(devices))
	for i, d := range devices {
		result[i] = toDeviceResponse(d, overlays[strings.ToLower(d.MAC)])
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleGetDevice(w http.ResponseWriter, r *http.Request) {
	raw, _ := url.PathUnescape(chi.URLParam(r, "mac"))
	mac := strings.ToLower(raw)
	overlays := s.buildOverlayMap()

	for _, d := range s.Scanner.Devices() {
		if strings.ToLower(d.MAC) == mac {
			writeJSON(w, http.StatusOK, toDeviceResponse(d, overlays[mac]))
			return
		}
	}

	writeJSON(w, http.StatusNotFound, map[string]string{"error": "device not found"})
}

type updateDeviceRequest struct {
	Name      *string `json:"name,omitempty"`
	ProfileID *string `json:"profile_id,omitempty"`
}

func (s *Server) handleUpdateDevice(w http.ResponseWriter, r *http.Request) {
	raw, _ := url.PathUnescape(chi.URLParam(r, "mac"))
	mac := strings.ToLower(raw)

	var req updateDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	// Update in-memory scanner name.
	if req.Name != nil {
		s.Scanner.SetDeviceName(mac, *req.Name)
	}

	// Update parental profile assignment in config.
	if req.ProfileID != nil {
		cfg, err := s.Config.Current().DeepCopy()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		// Validate profile ID exists (unless clearing assignment).
		if *req.ProfileID != "" {
			found := false
			for _, p := range cfg.Parental.Profiles {
				if p.ID == *req.ProfileID {
					found = true
					break
				}
			}
			if !found {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown profile_id"})
				return
			}
		}

		// Update or add the device assignment.
		found := false
		for i, pd := range cfg.Parental.Devices {
			if strings.ToLower(pd.MAC) == mac {
				if *req.ProfileID == "" {
					// Remove assignment.
					cfg.Parental.Devices = append(cfg.Parental.Devices[:i], cfg.Parental.Devices[i+1:]...)
				} else {
					cfg.Parental.Devices[i].ProfileID = *req.ProfileID
					if cfg.Parental.Devices[i].IP == "" {
						cfg.Parental.Devices[i].IP = s.lookupDeviceIP(mac)
					}
				}
				found = true
				break
			}
		}
		if !found && *req.ProfileID != "" {
			cfg.Parental.Devices = append(cfg.Parental.Devices, config.ParentalDevice{
				MAC:       mac,
				ProfileID: *req.ProfileID,
				IP:        s.lookupDeviceIP(mac),
			})
		}

		if err := s.applyConfig(cfg); err != nil {
			handleApplyError(w, err)
			return
		}
	}

	// Return updated device.
	overlays := s.buildOverlayMap()
	for _, d := range s.Scanner.Devices() {
		if strings.ToLower(d.MAC) == mac {
			writeJSON(w, http.StatusOK, toDeviceResponse(d, overlays[mac]))
			return
		}
	}

	writeJSON(w, http.StatusNotFound, map[string]string{"error": "device not found"})
}
