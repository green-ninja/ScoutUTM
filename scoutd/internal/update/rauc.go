package update

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
)

// BundleInfo contains metadata extracted from a RAUC bundle after
// signature verification succeeds.
type BundleInfo struct {
	Compatible  string `json:"compatible"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Build       string `json:"build"`
}

// SlotStatus describes the state of an A/B partition slot.
type SlotStatus struct {
	Slot     string `json:"slot"`
	State    string `json:"state"`    // "booted", "inactive"
	Class    string `json:"class"`    // "rootfs"
	Bootname string `json:"bootname"` // "A" or "B"
}

// RAUC wraps the rauc CLI for firmware update operations.
//
// All operations rely on RAUC's built-in CMS signature verification
// against the system keyring (/etc/rauc/keyring.pem). The flags
// --no-verify and --trust-environment are intentionally NEVER used.
type RAUC struct {
	devMode bool
}

// NewRAUC creates a RAUC wrapper.
func NewRAUC(devMode bool) *RAUC {
	return &RAUC{devMode: devMode}
}

// VerifyBundle checks a bundle's CMS signature against the system
// keyring and returns its metadata. This MUST succeed before calling
// InstallBundle. A tampered, unsigned, or wrongly-signed bundle will
// cause this method to return an error.
func (r *RAUC) VerifyBundle(bundlePath string) (*BundleInfo, error) {
	if r.devMode {
		log.Printf("[dev-mode] would verify bundle: %s", bundlePath)
		return &BundleInfo{
			Compatible:  "scoututm-h4plus",
			Version:     "0.0.0-dev",
			Description: "dev-mode stub",
		}, nil
	}

	// rauc info verifies the CMS signature before returning metadata.
	// If the signature is invalid or the keyring is missing, rauc exits
	// with a non-zero status code.
	out, err := exec.Command("rauc", "info",
		"--output-format=json",
		bundlePath,
	).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("bundle verification failed (invalid signature): %s: %w",
			truncateOutput(out), err)
	}

	var info BundleInfo
	if err := json.Unmarshal(out, &info); err != nil {
		return nil, fmt.Errorf("parsing bundle info: %w", err)
	}
	return &info, nil
}

// InstallBundle installs a signed RAUC bundle to the inactive slot.
//
// RAUC re-verifies the CMS signature before writing to the partition.
// The caller SHOULD call VerifyBundle first to inspect metadata and
// confirm compatibility before committing to the install.
func (r *RAUC) InstallBundle(bundlePath string) error {
	if r.devMode {
		log.Printf("[dev-mode] would install bundle: %s", bundlePath)
		return nil
	}

	// rauc install:
	// 1. Verifies CMS signature against /etc/rauc/keyring.pem
	// 2. Writes image to inactive A/B slot
	// 3. Marks the slot for next boot
	//
	// --no-verify is intentionally NOT passed. Removing signature
	// verification would allow arbitrary firmware to be installed.
	out, err := exec.Command("rauc", "install", bundlePath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("bundle install failed: %s: %w",
			truncateOutput(out), err)
	}
	return nil
}

// GetSlotStatus returns the status of A/B partition slots.
func (r *RAUC) GetSlotStatus() ([]SlotStatus, error) {
	if r.devMode {
		return []SlotStatus{
			{Slot: "rootfs.0", State: "booted", Class: "rootfs", Bootname: "A"},
			{Slot: "rootfs.1", State: "inactive", Class: "rootfs", Bootname: "B"},
		}, nil
	}

	out, err := exec.Command("rauc", "status", "--output-format=json").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("getting slot status: %s: %w",
			truncateOutput(out), err)
	}

	var result struct {
		Slots []SlotStatus `json:"slots"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("parsing slot status: %w", err)
	}
	return result.Slots, nil
}

// MarkGood marks the currently booted slot as "good", confirming a
// successful update. Without this, RAUC will roll back on next boot.
func (r *RAUC) MarkGood() error {
	if r.devMode {
		log.Println("[dev-mode] would mark slot good")
		return nil
	}

	out, err := exec.Command("rauc", "status", "mark-good").CombinedOutput()
	if err != nil {
		return fmt.Errorf("marking slot good: %s: %w",
			truncateOutput(out), err)
	}
	return nil
}

// truncateOutput limits error output to a reasonable length for logs.
func truncateOutput(out []byte) string {
	const maxLen = 256
	s := string(out)
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
