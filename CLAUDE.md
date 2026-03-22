# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ScoutUTM is a home network security appliance (UTM) running **ScoutOS**, a custom Alpine Linux-based operating system. It consists of a Go management daemon (`scoutd`), a Vue.js 3 frontend embedded into the Go binary, and a SquashFS-based image build system targeting the Odroid H4+ (x86_64).

**Key constraint:** Solo founder project. Keep things simple. No over-engineering.

## Architecture

**scoutd** is the single management daemon (control plane only). It does NOT handle data-plane traffic — packets flow through kernel nftables and Suricata directly. scoutd manages configuration, serves the REST API + embedded frontend, monitors stats, and orchestrates system daemons via OpenRC.

**Config flow:** User changes → REST API → `config.json` (single source of truth) → Go `text/template` renders daemon configs (nftables.conf, suricata.yaml, unbound.conf, etc.) → daemon reload via OpenRC/CLI.

**Frontend delivery:** Vue.js static assets are compiled into the scoutd binary via `//go:embed`. No separate web server.

**System daemons managed by scoutd:** nftables (firewall), Suricata (IDS), Unbound (DNS resolver + filtering), dnsmasq (DHCP only, DNS disabled), WireGuard (VPN), chrony (NTP).

## Build Commands

```bash
make frontend    # Build Vue.js → web/dist/ static assets
make scoutd      # Build Go binary (CGO_ENABLED=0, static)
make rootfs      # Alpine rootfs via alpine-make-rootfs + overlay
make squashfs    # Compress rootfs to SquashFS (zstd) + dm-verity
make initramfs   # Build initramfs with embedded dm-verity root hash
make image       # Full disk image (GPT, GRUB, partitions, squashfs)
make bundle      # RAUC OTA update bundle (signed)
make clean       # Remove build artifacts
```

## Development Commands

```bash
# Go backend (Tier 1 — host-native, ~1-2s iteration)
cd scoutd && go test ./...                  # Run all unit tests
cd scoutd && go test ./internal/config/...  # Run tests for one package
cd scoutd && go run ./cmd/scoutd --dev-mode # Start locally (stubs hardware ops)

# Vue.js frontend (Tier 1)
cd web && npm run dev                       # Vite dev server with HMR

# Vagrant VM (Tier 2 — full ScoutOS environment, ~30s reload)
vagrant up                                  # Create/boot Alpine VM
vagrant ssh                                 # Shell into VM
make scoutd && vagrant upload build/scoutd /usr/sbin/scoutd  # Deploy new binary
```

## Key Design Rules

- **No cgo.** `CGO_ENABLED=0` always. Pure static Go binary.
- **No database.** Config is a JSON file. Stats are in-memory ring buffers. Logs go to syslog.
- **No D-Bus.** Shell out to `rauc` CLI via `exec.Command`.
- **No cron.** All periodic tasks run via scoutd's internal scheduler (`internal/scheduler/`).
- **Single process.** This is an embedded appliance, not microservices.
- **Template safety.** Templates execute against strongly typed Go structs, never raw user input or `map[string]interface{}`. User values are validated by config schema before reaching templates.
- **`--dev-mode`** stubs hardware-dependent ops (nftables apply, OpenRC, netlink stats) but still renders templates to disk for inspection.

## Code Layout

- `scoutd/` — Go daemon. Standard `cmd/` + `internal/` layout.
  - `internal/api/` — REST handlers (chi router). One file per resource (firewall.go, dns.go, vpn.go, etc.)
  - `internal/config/` — JSON config schema (Go structs), load/save/validate, schema migration
  - `internal/render/` — `text/template` engine. One file per daemon config (nftables.go, suricata.go, etc.)
  - `internal/service/` — OpenRC service lifecycle (start/stop/reload)
  - `internal/ws/` — WebSocket hub for real-time stats + alerts
  - `internal/stats/` — System stats via netlink, conntrack, /proc
  - `internal/update/` — OTA via RAUC CLI
  - `internal/auth/` — Session tokens (in-memory) + bcrypt passwords
  - `internal/netmon/` — ARP scanning, DHCP lease parsing, device fingerprinting
  - `internal/scheduler/` — Goroutine-based tick loop for periodic tasks
  - `templates/` — `*.tmpl` files for daemon configs
- `web/` — Vue.js 3 + Vite + Tailwind CSS + Pinia frontend source
- `build/` — Image build system (shell scripts, Alpine package list, overlay files, initramfs)
- `tests/` — Integration test scripts (run in Vagrant VM)
- `tools/` — Dev utilities (vagrant-provision.sh, deploy.ps1, test-vm.ps1, etc.)

## Config Apply Flow (Three-Phase Commit)

1. **Write pending:** Save new config to `config.json.pending`
2. **Render + apply:** Generate all daemon configs from templates, reload daemons in dependency order
3. **Commit or rollback:** On success, rename pending → config.json. On failure, revert to previous config.

Daemon reload order matters: nftables → dnsmasq → unbound → suricata → wireguard → chrony.

## Testing

- **Unit tests:** `cd scoutd && go test ./...` — config validation, template rendering, API handlers (httptest), migrations
- **Integration tests:** Shell scripts in `tests/` — run in Vagrant VM with real daemons. Each script is self-contained and idempotent.
- Template rendering tests should verify the generated config output matches expected daemon config format.

## Important Technical Details

- **Dual DNS architecture:** Unbound handles all DNS (recursive, DNSSEC, RPZ filtering, DoT/DoH). dnsmasq handles DHCP only (its DNS is disabled). This is intentional separation of concerns.
- **Parental controls:** Per-device DNS filtering via Unbound's `access-control-tag` feature. Devices are pinned to static IPs via dnsmasq when assigned to parental profiles.
- **Boot:** UEFI → GRUB (ESP) → initramfs (dm-verity verify) → SquashFS + OverlayFS (tmpfs upper) → OpenRC → scoutd
- **Partitions:** ESP, rootfs-a (SquashFS), rootfs-b (standby), config (ext4, persistent), data (ext4, persistent)
- **OTA updates:** RAUC with A/B partition scheme. Post-install hook extracts kernel+initramfs to ESP slot directories.

## Reference Documents

- `PLAN.md` — Complete technical architecture (the authoritative spec for all implementation)
- `GTM.md` — Go-to-market strategy
- `RISKS.md` — Risk analysis
