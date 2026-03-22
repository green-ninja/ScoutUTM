# ScoutOS Technical Architecture Plan

> Version 0.9 | Created: 2026-02-11 | Updated: 2026-02-12 | Status: Draft

> **v0.9 changes:** Addressed remaining risk analysis. (1) Elevated Suricata throughput to highest-priority technical risk -- added explicit "throughput-defines-SKU" rule: no marketing throughput claims until Month 10 hardware benchmarks are complete, benchmark results define product specs (5.4). (2) Added 7-step hardware benchmark go/no-go gate protocol (5.4). (3) Added dm-verity for v1.0 rootfs integrity -- Merkle tree hash verification on every SquashFS block read, built into initramfs (10.10). (4) Replaced Secure Boot section with versioned integrity roadmap: v1.0 = signed OTA + dm-verity + read-only rootfs, v1.5 = full UEFI Secure Boot chain (10.10). (5) Added post-launch operational sustainability program with maintenance cadences, security update pipeline (72h critical CVE response), sustainability risks, and solo founder capacity planning (16). (6) Added cryptsetup/veritysetup to Alpine packages (8.3).
>
> **v0.8 changes:** Addressed engineering review (13-point technical analysis). (1) Documented musl libc allocator risk for Suricata multi-threading -- added jemalloc/mimalloc LD_PRELOAD as benchmark-gated mitigation, integrated into Month 5/10 benchmarks (5.4). (2) Noted musl DNS resolver irrelevance -- Unbound and Go's pure resolver sidestep musl's resolver limitations (5.5). (3) Added ext4 vs F2FS rationale -- NVMe has hardware FTL, F2FS unnecessary (7.1). (4) Added hardware watchdog section with `watchdog` package and Suricata thread priority tuning via SCHED_FIFO (10.9). (5) Added Secure Boot analysis -- deferred to v1.5 due to Alpine's lack of out-of-box shim/signed GRUB, documented path forward (10.10). (6) Added TPM 2.0 feasibility note -- deferred to v1.5, Alpine has tpm2-tools (10.11). (7) Added SBOM generation (syft/CycloneDX) in CI and GPL source distribution compliance as mandatory build steps (10.12). (8) Added DPDK, QAT, Secure Boot, TPM to deferred scope table (14). (9) Added new risks: musl allocator contention, Secure Boot on Alpine, QAT integration, GPL/SBOM compliance (15). (10) Documented Alpine vs OpenWrt strategic rationale (1).
>
> **v0.7 changes:** Addressed CTO/reviewer feedback round 2. (1) Adopted IDS-only as v1 default -- IPS is opt-in with throughput warning; eliminates late-discovery performance risk (5.4). (2) Added comprehensive rollback test plan with 10 specific scenarios scheduled across Months 2-9, plus "Reset to Previous Config" UI button (4.4). (3) Hardened RAUC ESP extraction with atomic temp-file-then-rename pattern and added GRUB rescue boot entry that always boots last-known-good slot (7.1). (4) Detailed integration test suite with 11 named scripts and what each covers, when each is added, and stress test with nmap alert storm (12.2). (5) Added WebSocket alert broadcast rate limiting at 10 alerts/sec per client (6.3). (6) Added fake-hwclock for RTC-less SBCs and scoutd time sanity check to handle boot-before-NTP-sync window (5.6). (7) Added Vue.js bundle size strategy: lazy-load routes, uPlot, CI size monitoring (6.5).
>
> **v0.6 changes:** Deferred hardware purchase to Month 9-10. Months 1-8 are pure software development using Tier 1 (host-native) and Tier 2 (Vagrant VM) only. All features validated in VMs before hardware spend. Alpha and Beta gates are VM-only. Suricata benchmarking split into VM phase (Month 5) and hardware phase (Month 10). Image build + OTA tested in QEMU first (Month 9). Added hardware portability as explicit design principle -- architecture is hardware-agnostic, can port to different x86 boards. Consolidated Month 9-10 (image build + OTA into one Month 9, hardware bring-up in Month 10). Added "late hardware surprise" risk with mitigation (5.4, 11.3, 13, 15).
>
> **v0.5 changes:** Addressed senior network engineer review. (1) Replaced SquashFS loopback boot with ESP-based kernel loading -- RAUC post-install hook extracts kernel+initramfs to slot-specific ESP subdirectories, GRUB loads from ESP natively (7.1-7.2). (2) Redesigned config apply flow to separate "write all files" from "reload all daemons" to minimize mixed-state race window, added dependency-ordered reload sequence (4.4). (3) Set conservative Suricata throughput expectations with benchmark-first task (5.4). (4) Extended timeline from 8 months to 12 months with explicit milestone gates (alpha at Month 4, beta at Month 8, RC at Month 10, v1.0 at Month 12) (13).
>
> **v0.4 changes:** Replaced hardware-dependent dev workflow with three-tier approach (11). Tier 1: host-native Go/Vue.js development with `--dev-mode` (~1-2s iteration, no VM). Tier 2: Vagrant VM with full Alpine + all daemons (~30s iteration, no hardware). Tier 3: Odroid H4+ hardware only for NIC testing and production QA. Added Vagrantfile, provisioning script, multi-VM LAN client testing, and "when to use which tier" guide. Added `--dev-mode` flag to scoutd design (3.2). Reduced hardware dependency from 2 boards to 1 for Months 1-6.
>
> **v0.3 changes:** Eliminated separate boot partition -- kernel+initramfs now inside SquashFS for atomic A/B updates (7.1-7.2). Reduced config partition from 256MB to 32MB (7.1). Added firmware downgrade protection via version monotonicity (7.5). Clarified Suricata eve.json via Unix datagram socket (5.4). Documented dnsmasq DHCP reload limitation (3.5). Added WebSocket connection limits (6.3). Added initramfs emergency fallback shell (8.5). Added Go template security guidelines (3.2). Added Alpine 3.21 lifecycle plan (11.5). Documented QEMU tap setup (11.4). Clarified DNS query log privacy -- full domain names, not URLs (10.7). Added WireGuard key rotation tracking and warning (10.3). Added explicit v1 scope limitations: IPv6, multi-WAN, scheduled backup, mobile app (14). Documented system logging destination -- BusyBox syslogd ring buffer (14). Added minimal backup/restore API (14). Added GRUB SquashFS loopback and Alpine kernel upgrade risks (15).
>
> **v0.2 changes:** Justified dual dnsmasq+Unbound DNS architecture (5.5). Added two-phase config commit with pending file (4.3-4.4). Added chrony/NTP throughout (stack, boot sequence, packages, services, templates). Documented all persistent data paths with retention policies and TLS cert lifecycle (10.7). Added USB recovery mode for password reset and factory reset (10.8).

## Context

ScoutUTM needs a technical architecture to go from GTM strategy (GTM.md v0.6) to a working prototype. This document defines the complete system architecture for ScoutOS -- the proprietary operating system powering the ScoutUTM appliance. The goal is a 12-month path from zero code to a beta-ready product on Odroid H4+ hardware.

**Constraints:** Solo founder, €35K budget, Alpine Linux base, proprietary/closed-source, ~50-80MB OS footprint, <15s boot, read-only rootfs, A/B OTA updates.

---

## 1. Technology Stack

| Layer | Choice | Rationale |
|-------|--------|-----------|
| Management daemon | **Go** | Single static binary, zero runtime deps on Alpine/musl, `//go:embed` for frontend, excellent networking stdlib, goroutines for concurrency |
| Frontend | **Vue.js 3 + Vite + Tailwind CSS** | Lightweight, small bundle (<500KB gzipped), good solo-dev DX |
| Frontend delivery | Embedded in Go binary via `//go:embed` | Single binary deployment, no separate web server |
| Config format | **JSON** (single file `/etc/scout/config.json`) | Human-readable, native Go marshaling, no database needed |
| Config-to-daemon rendering | Go `text/template` | In stdlib, no dependencies |
| Firewall | **nftables** (single `inet scout` table) | Modern, atomic rule replacement, named sets for zone model |
| IDS/IPS | **Suricata via NFQUEUE** | Router-mode compatible (not bridge), per-CPU queues, fail-open bypass |
| DNS resolver + filtering | **Unbound** (recursive + RPZ blocklists) | Lightweight, DNSSEC, Response Policy Zones for filtering |
| DHCP | **dnsmasq** (DHCP only, DNS disabled) | Battle-tested, tiny footprint |
| NTP | **chrony** | Lightweight, fast sync, essential for TLS/DNSSEC/IDS timestamps |
| VPN | **WireGuard** (kernel module) | In-kernel, fast, simple config |
| OTA updates | **RAUC** + GRUB A/B partition scheme | Industry standard for embedded Linux, automatic rollback |
| Root filesystem | **SquashFS + OverlayFS** (tmpfs upper) | Read-only, tamper-resistant, clean state on reboot |
| Init system | **OpenRC** | Alpine native, lightweight, no systemd |
| HTTP router (Go) | **chi** | Lightweight, stdlib-compatible, good middleware |
| Authentication | Local session tokens (in-memory) | Simple, no external auth provider needed |
| Build system | **Makefile + shell scripts** | Simple, transparent, no Yocto/Buildroot overhead |
| CI/CD | GitHub Actions | Free tier sufficient, Docker-based builds |

**Why Alpine Linux, not OpenWrt:** OpenWrt is a mature embedded Linux router OS with nftables, musl, and BusyBox -- similar to what ScoutOS builds on Alpine. However, Alpine is a better fit for a UTM appliance on x86:
- **Full Linux userland.** Alpine's package repository (~14,000 packages) includes Suricata, Unbound, WireGuard, chrony, and RAUC with no patching. OpenWrt's package set is optimized for constrained MIPS/ARM routers and would require porting or forking several components.
- **No unnecessary router abstraction.** OpenWrt's UCI/LuCI config system and procd init would fight against ScoutOS's own config management (scoutd + JSON + templates). Building on Alpine means there's no framework to work around.
- **x86 focus.** OpenWrt excels on MIPS/ath79/ARM targets with wireless drivers -- hardware ScoutOS doesn't target. Alpine on x86 is a well-tested, mainstream Linux environment.
- **Convergence.** OpenWrt has itself adopted APK (Alpine's package manager) and musl, reflecting that the two distributions are converging in their base tooling. Building on Alpine directly avoids the overhead of maintaining an OpenWrt fork.
- **Familiarity.** Alpine is a standard Linux distribution -- any Linux developer can `apk add`, edit configs, and debug. OpenWrt's custom build system (feeds, menuconfig) has a steeper onboarding curve for contributors.

The trade-off: OpenWrt provides built-in wireless driver support and a battle-tested router config model. If ScoutOS later targets wireless hardware or MIPS/ARM SBCs, an OpenWrt port could be considered. For v1 on x86 with wired NICs, Alpine is the simpler and more flexible base.

---

## 2. System Architecture

### 2.1 Component Diagram

```
+============================================================================+
|                           ScoutOS on Odroid H4+                            |
+============================================================================+
|                                                                            |
|  +------------------------------------------------------------------+     |
|  |                     Web Dashboard (Vue.js 3)                     |     |
|  |   Setup Wizard | Dashboard | Firewall Rules | DNS | VPN | Logs  |     |
|  +-------------------------------+----------------------------------+     |
|                                  |                                        |
|                          HTTPS / WebSocket                                |
|                                  |                                        |
|  +-------------------------------v----------------------------------+     |
|  |                        scoutd (Go binary)                        |     |
|  |                                                                  |     |
|  |  +------------+  +-----------+  +----------+  +--------------+  |     |
|  |  | REST API   |  | WebSocket |  | Config   |  | Service      |  |     |
|  |  | Server     |  | Hub       |  | Manager  |  | Orchestrator |  |     |
|  |  | /api/v1/*  |  | (stats,   |  | (JSON    |  | (start/stop/ |  |     |
|  |  |            |  |  alerts)  |  |  store)  |  |  reload)     |  |     |
|  |  +------------+  +-----------+  +----------+  +--------------+  |     |
|  |                                                                  |     |
|  |  +------------+  +-----------+  +----------+  +--------------+  |     |
|  |  | Template   |  | Auth      |  | Update   |  | Stats        |  |     |
|  |  | Engine     |  | Module    |  | Client   |  | Collector    |  |     |
|  |  | (text/tmpl)|  | (sessions)|  | (RAUC)   |  | (/proc,      |  |     |
|  |  |            |  |           |  |          |  |  unbound)    |  |     |
|  |  +------------+  +-----------+  +----------+  +--------------+  |     |
|  +------------------------------------------------------------------+     |
|         |                 |                |                |              |
|    generates         orchestrates      signals         monitors           |
|    configs           processes         reload          stats              |
|         |                 |                |                |              |
|  +------v-----------v----v--------v-------v--------v-------v--------+     |
|  |                    System Daemons (OpenRC managed)               |     |
|  |                                                                  |     |
|  |  +--------+ +--------+ +-------+ +-------+ +------+ +------+  |     |
|  |  |nftables| |Suricata| |Unbound| |dnsmasq| | WG   | |chrony|  |     |
|  |  |(fw+NAT)| |(IDS/IPS| |(DNS   | |(DHCP) | |(VPN) | |(NTP) |  |     |
|  |  |        | |NFQUEUE)| |filter)| |       | |      | |      |  |     |
|  |  +--------+ +--------+ +-------+ +-------+ +------+ +------+  |     |
|  +------------------------------------------------------------------+     |
|                                                                            |
|  +------------------------------------------------------------------+     |
|  |                    Linux Kernel (Hardened)                        |     |
|  |  nf_tables | netfilter | bridge | af_packet | wireguard | vlan  |     |
|  +------------------------------------------------------------------+     |
|                                                                            |
|  +---------------------------+  +-----------------------------------+     |
|  | eth0 (WAN - Intel I226-V) |  | eth1 (LAN - Intel I226-V)       |     |
|  +---------------------------+  +-----------------------------------+     |
+============================================================================+
```

### 2.2 Data Flow: Packet Path

```
Internet
    |
    v
+---+---+
| eth0  |  WAN interface
+---+---+
    |
    v
+---+-------------------+
| nftables PREROUTING   |  DNAT / port forwarding / DNS redirect
+---+-------------------+
    |
    v
+---+-------------------+
| nftables FORWARD      |  Zone-based policy (WAN->LAN, etc.)
| (inet scout table)    |  Conntrack, rate limiting
+---+-------------------+
    |
    v
+---+-------------------+
| NFQUEUE → Suricata    |  IPS inspection (queues 0-3, one per CPU core)
|                       |  Verdict: ACCEPT or DROP
+---+-------------------+
    |
    v
+---+-------------------+
| nftables POSTROUTING  |  SNAT / masquerade for outbound
+---+-------------------+
    |
    v
+---+---+
| eth1  |  LAN interface (or VLAN sub-interfaces)
+---+---+
    |
    v
LAN Devices
```

**DNS path:**
```
LAN Device → eth1 → nftables DNS redirect (port 53 → local)
  → Unbound (recursive resolver)
    → RPZ blocklists (malware, ads, parental)
    → upstream resolvers (DoT/DoH)
```

### 2.3 Control Flow

```
User (browser)
    |
    v
Vue.js SPA → REST API (HTTPS) / WebSocket (WSS)
    |
    v
scoutd REST handler / WS hub
    |
    v
Config Manager
    | 1. Validate input
    | 2. Update JSON config
    | 3. Render daemon config templates
    | 4. Write daemon configs atomically
    | 5. Signal daemon reload
    v
Service Orchestrator
    | rc-service X reload
    | nft -f /etc/nftables.conf
    | kill -HUP <pid>
    v
System Daemons (nftables, Suricata, Unbound, dnsmasq, WireGuard)
```

---

## 3. Core Daemon: scoutd

### 3.1 Overview

scoutd is the single management process. It is NOT the data plane. Packets flow through kernel nftables and Suricata directly -- scoutd only handles the control plane (configuration, API, monitoring, updates).

### 3.2 Key Design Rules

- **Single process, not microservices.** This is an embedded appliance, not a cloud platform.
- **Cooperates with OpenRC.** System daemons (Suricata, Unbound, etc.) are OpenRC services. scoutd manages their configuration and triggers reloads.
- **No database.** Config is a JSON file. Stats are in-memory ring buffers. Logs go to syslog.
- **No D-Bus.** Shell out to `rauc` CLI via `exec.Command` instead of linking D-Bus bindings. Keeps the binary pure Go (no cgo).
- **No cgo.** `CGO_ENABLED=0` for a fully static binary that runs on Alpine/musl without issues.
- **Embedded frontend.** Vue.js static assets compiled into the binary via `//go:embed`.
- **Template safety.** All daemon config templates are executed against **strongly typed Go structs** (`config.Firewall`, `config.DNS`, etc.), never against raw user input strings or `map[string]interface{}`. User-provided values (zone names, IP addresses, hostnames) are validated by the config schema before they ever reach a template. This prevents template injection -- a user cannot craft an input that escapes the template context and injects arbitrary nftables/Suricata configuration.
- **Dev mode.** `scoutd --dev-mode` starts the HTTP server and config management but stubs out hardware-dependent operations (nftables apply, OpenRC service orchestration, /proc stats). Templates are still rendered and written to disk for inspection, but not applied. This enables Tier 1 host-native development on any OS without root or Linux-specific features.

### 3.3 Internal Structure

```
scoutd/
├── cmd/scoutd/main.go              # Entry point, CLI flags, signal handling
├── internal/
│   ├── api/                        # REST API handlers
│   │   ├── router.go               # chi router setup, middleware
│   │   ├── auth.go                 # Login, session middleware
│   │   ├── firewall.go             # /api/v1/firewall/*
│   │   ├── dns.go                  # /api/v1/dns/*
│   │   ├── vpn.go                  # /api/v1/vpn/*
│   │   ├── network.go              # /api/v1/network/*
│   │   ├── system.go               # /api/v1/system/*
│   │   ├── devices.go              # /api/v1/devices/*
│   │   ├── ids.go                  # /api/v1/ids/*
│   │   ├── parental.go             # /api/v1/parental/*
│   │   └── wizard.go               # /api/v1/wizard/*
│   ├── ws/                         # WebSocket hub
│   │   ├── hub.go                  # Connection manager, broadcast
│   │   └── client.go               # Per-client read/write pumps
│   ├── config/                     # Configuration management
│   │   ├── schema.go               # Go structs matching JSON config
│   │   ├── store.go                # Load/save/validate (atomic writes)
│   │   ├── validate.go             # Config validation rules
│   │   └── migrate.go              # Schema version migration
│   ├── render/                     # Template rendering engine
│   │   ├── engine.go               # text/template rendering
│   │   ├── nftables.go             # nftables.conf template + helpers
│   │   ├── suricata.go             # suricata.yaml template
│   │   ├── unbound.go              # unbound.conf template
│   │   ├── dnsmasq.go              # dnsmasq.conf template
│   │   ├── wireguard.go            # wg0.conf template
│   │   └── chrony.go              # chrony.conf template
│   ├── service/                    # Service lifecycle
│   │   ├── manager.go              # Start/stop/reload/status
│   │   └── openrc.go               # OpenRC rc-service integration
│   ├── stats/                      # Real-time statistics
│   │   ├── collector.go            # Aggregator goroutine
│   │   ├── procnet.go              # Interface stats via /proc/net/dev
│   │   ├── conntrack.go            # Connection tracking
│   │   ├── dns.go                  # DNS query/block stats via unbound-control
│   │   └── system.go               # CPU, memory, uptime
│   ├── update/                     # OTA update client
│   │   ├── checker.go              # Periodic version check
│   │   └── rauc.go                 # RAUC CLI integration
│   ├── auth/                       # Authentication
│   │   ├── session.go              # Token management (in-memory)
│   │   └── password.go             # bcrypt hashing
│   ├── scheduler/                  # Internal task scheduler
│   │   └── scheduler.go            # Goroutine-based tick loop, registered tasks
│   └── netmon/                     # Network monitoring
│       ├── arp.go                  # ARP table scanning
│       ├── dhcp.go                 # DHCP lease parsing
│       └── fingerprint.go          # Device fingerprinting (OUI, DHCP options)
├── templates/                      # Daemon config templates (*.tmpl)
│   ├── nftables.conf.tmpl
│   ├── suricata.yaml.tmpl
│   ├── unbound.conf.tmpl
│   ├── dnsmasq.conf.tmpl
│   ├── wireguard.conf.tmpl
│   └── chrony.conf.tmpl
└── web/dist/                       # Built Vue.js assets (//go:embed)
```

### 3.4 Process Management

scoutd is an OpenRC service that starts after all system daemons:

```sh
#!/sbin/openrc-run
name="scoutd"
description="ScoutOS management daemon"
command="/usr/sbin/scoutd"
command_args="--config /etc/scout/config.json"
command_background=true
pidfile="/run/scoutd.pid"
depend() {
    need net nftables
    after suricata unbound dnsmasq
}
```

**Boot sequence:**
```
OpenRC sysinit → boot runlevel:
  ├── networking (interfaces up)
  ├── chronyd (NTP sync -- must run before security daemons)

OpenRC default runlevel:
  ├── nftables (load /etc/nftables.conf)
  ├── dnsmasq (DHCP)
  ├── unbound (DNS)
  ├── suricata (IDS/IPS)
  ├── wireguard (if configured)
  └── scoutd (starts last, depends on all above)
```

### 3.5 Daemon Reload Methods

| Daemon | Config File | Reload Method |
|--------|-------------|---------------|
| nftables | `/etc/nftables.conf` | `nft -f /etc/nftables.conf` (atomic) |
| Suricata | `/etc/suricata/suricata.yaml` | `suricatasc -c reload-rules` or SIGHUP |
| Unbound | `/etc/unbound/unbound.conf` | `unbound-control reload` |
| dnsmasq | `/etc/dnsmasq.conf` | `rc-service dnsmasq restart` (see note below) |
| WireGuard | `/etc/wireguard/wg0.conf` | `wg syncconf wg0 <(wg-quick strip wg0)` |
| chrony | `/etc/chrony/chrony.conf` | `rc-service chronyd restart` |

**dnsmasq reload note:** dnsmasq has no graceful reload -- it requires a full restart. During restart (~100ms), new DHCP requests will fail. Existing clients are unaffected (leases are cached client-side). dnsmasq reads its lease file on startup, so lease state survives the restart. DHCP config changes (subnet, range, static assignments) are infrequent operations, so the brief interruption is acceptable. The UI should warn: "Applying DHCP changes requires a brief service restart."

### 3.6 Internal Scheduler

ScoutOS runs no external scheduler (no cron -- see section 10.2). All periodic tasks are managed by scoutd's internal scheduler, a goroutine-based tick loop in `internal/scheduler/scheduler.go`.

**Design:** On startup, scoutd registers all scheduled tasks with their intervals. A single goroutine runs a `time.Ticker` loop, checking each task's next-run time on every tick (60-second resolution). Tasks run as goroutines so a slow task (e.g., GeoIP download) does not block others. Each task logs its start/completion and any errors to scoutd.log.

```go
type Task struct {
    Name     string
    Interval time.Duration     // e.g., 24 * time.Hour
    RunAt    string            // optional: "03:00" for daily-at-time tasks
    Fn       func(ctx context.Context) error
}
```

**Registered scheduled tasks:**

| Task | Interval | Default Time | What It Does |
|------|----------|-------------|--------------|
| `suricata-update` | Daily | 03:00 | Runs `suricata-update` to fetch latest ET Open rules, then `suricatasc -c reload-rules` |
| `dns-blocklist-update` | Daily | 03:30 | Downloads latest RPZ blocklist files, writes to `/var/lib/scout/blocklists/`, runs `unbound-control reload` |
| `ota-check` | Daily | 04:00 | GETs `manifest.json` from update server (see 7.5), notifies user if update available |
| `log-rotate` | Hourly | :00 | Checks size of scoutd.log, eve.json, dns-query.log; rotates if over limit (see 10.7) |
| `fake-hwclock-save` | Hourly | :30 | Writes current time to `/etc/scout/fake-hwclock.data` for RTC-less boot recovery |
| `parental-schedule-check` | Every 60s | -- | Evaluates parental control schedules against current time; enables/disables per-device DNS filtering profiles by updating Unbound access-control and reloading |
| `disk-check` | Hourly | :15 | Checks `/data` partition usage via `statvfs`. Warns at 90%. At 95%, prunes oldest rotated logs (eve.json.*, dns-query.log.*) to reclaim space |

**Disk space monitoring detail:** The `disk-check` task calls Go's `syscall.Statvfs` on `/data` to get usage percentage. If usage exceeds 90%, it logs a warning visible in the dashboard system log. If usage exceeds 95%, it auto-prunes the oldest rotated Suricata EVE and DNS query log files (oldest first) until usage drops below 90% or no rotated logs remain. NVMe SMART health monitoring deferred to Phase 2 (requires `smartmontools` package and NVMe-specific parsing).

**Parental schedule enforcement detail:** The `parental-schedule-check` task runs every 60 seconds. It reads the current time and each device's assigned parental profile schedule from config. If a device's schedule says "block internet 21:00-07:00" and the current time is 21:01, the task updates Unbound's configuration to apply the restrictive RPZ zone for that device's source IP and triggers `unbound-control reload`. When the schedule window ends (07:01), the next check removes the restriction. The 60-second resolution means enforcement is accurate to ±1 minute, which is acceptable for bedtime schedules.

**Staggered timing:** Daily tasks are spread across the 03:00-04:00 window to avoid simultaneous network activity and CPU load. All daily tasks run in the early morning when network usage is lowest.

---

## 4. Configuration Management

### 4.1 Central Config

Single JSON file at `/etc/scout/config.json`. This is the source of truth. All daemon configs are generated from it via templates. Never edited by hand -- always modified through scoutd's API.

### 4.2 Config Schema (Top-Level)

```go
type Config struct {
    Version    int             `json:"version"`      // Schema version for migrations
    System     SystemConfig    `json:"system"`       // Hostname, timezone, admin hash, language
    Network    NetworkConfig   `json:"network"`      // Interfaces, zones, VLANs
    Firewall   FirewallConfig  `json:"firewall"`     // Rules, forwardings, port forwards, GeoIP
    DNS        DNSConfig       `json:"dns"`          // Upstream, DoT/DoH, blocklists, custom entries
    VPN        VPNConfig       `json:"vpn"`          // WireGuard config + peers
    IDS        IDSConfig       `json:"ids"`          // Suricata mode, rulesets, home_net
    Parental   ParentalConfig  `json:"parental"`     // Profiles + device assignments
    Update     UpdateConfig    `json:"update"`       // Auto-check, channel, server URL
}
```

**Key sub-structures:**

- **SystemConfig**: `hostname`, `timezone`, `admin_hash` (bcrypt), `setup_done`, `language`, `ntp_servers[]` (default: `["pool.ntp.org"]`)
- **NetworkConfig**: `interfaces[]` (name, zone, type, ipv4 mode/address), `zones[]` (name, input/forward/output policy, masquerade), `vlans[]` (id, parent, subnet, dhcp_range)
- **FirewallConfig**: `forwardings[]` (src→dest zone), `rules[]` (custom rules with src/dest/proto/port/action), `port_forwards[]`
- **DNSConfig**: `upstream[]`, `use_dot`, `use_doh`, `block_malware`, `block_phishing`, `block_ads`, `custom_block[]`, `custom_allow[]`
- **VPNConfig → WireGuard**: `enabled`, `listen_port` (default: 51820), `private_key`, `key_created_at` (ISO 8601), `address` (default: `10.0.0.1/24`), `tunnel_subnet` (default: `10.0.0.0/24`), `peers[]` (name, public_key, allowed_ips, tunnel_ip), `endpoint_override` (optional: manual endpoint if WAN IP detection is unreliable)

**WireGuard peer management and QR code generation:**

The WireGuard server listens on `udp/51820` on the WAN interface. The tunnel subnet defaults to `10.0.0.0/24` — the server is `10.0.0.1`, and clients are allocated sequentially (`10.0.0.2`, `.3`, etc.) as peers are added. scoutd tracks allocated IPs to prevent collisions.

When a user adds a peer via the dashboard (`POST /api/v1/vpn/wireguard/peers`), scoutd:
1. Generates a client keypair (private + public key)
2. Allocates the next available tunnel IP from the subnet
3. Builds a standard WireGuard client config:
   ```ini
   [Interface]
   PrivateKey = <generated client private key>
   Address = 10.0.0.2/32
   DNS = 192.168.1.1   # ScoutUTM LAN IP -- routes DNS through Unbound

   [Peer]
   PublicKey = <server public key>
   Endpoint = <WAN IP>:51820
   AllowedIPs = 0.0.0.0/0   # Full tunnel (all traffic through home network)
   ```
4. Encodes the config as a QR code (displayed in the dashboard, scannable by the WireGuard mobile app)
5. The client private key is shown once (for QR/download) and then discarded — only the client public key is stored in config.json

**Server endpoint:** Uses the current WAN IP (from `udhcpc` lease on eth0) at the time of peer config generation. The dashboard warns: "Your VPN endpoint is set to your current WAN IP. If your ISP changes your IP address, VPN clients will need updated configs. Consider requesting a static IP from your ISP." The `endpoint_override` field allows the user to manually set a hostname or IP if they have a DDNS service configured outside of ScoutOS (DDNS is not in v1 scope).

**AllowedIPs options:** The dashboard offers two modes when creating a peer:
- **Full tunnel** (`AllowedIPs = 0.0.0.0/0`): All client traffic routes through ScoutUTM. Provides full DNS filtering and firewall protection remotely.
- **Split tunnel** (`AllowedIPs = 192.168.1.0/24, 10.0.0.0/24`): Only home network traffic routes through ScoutUTM. Internet traffic uses the client's local connection.
- **IDSConfig**: `enabled`, `mode` (ids/ips), `rulesets[]`, `home_net[]`, `suppressed[]` (SIDs)
- **ParentalConfig**: `profiles[]` (block_categories, safe_search, youtube_restricted, schedule), `devices[]` (mac → profile_id)
- **UpdateConfig**: `auto_check`, `auto_install`, `channel`, `server_url`

### 4.3 Three-Phase Config Commit

The original plan saved config.json before applying daemon changes. Problem: if the daemon reload fails (e.g., nftables rejects a rule), the config file already contains the bad state. This creates a divergence between config-on-disk and running-state.

ScoutOS uses a **three-phase commit** with a pending file:

**Phase 1: Stage** -- write the proposed config to `config.json.pending`
**Phase 2: Apply** -- render templates, reload daemons, test
**Phase 3: Commit** -- if all reloads succeed, promote pending to live

**Files involved:**
| File | Purpose |
|------|---------|
| `/etc/scout/config.json` | Live config. Always matches running system state. |
| `/etc/scout/config.json.pending` | Staged config. Exists only during apply. |
| `/etc/scout/config.json.bak` | Backup of previous live config (one-step manual rollback). |

**All file writes use the atomic write-tmp-rename pattern** (write to `.tmp`, fsync, rename).

### 4.4 Config Apply Flow

When a user changes something (e.g., adds a firewall rule):

```
 1. API handler receives PUT /api/v1/firewall/rules
 2. Merge change into in-memory copy of Config
 3. Validate entire Config (schema + semantic rules)
 4. STAGE: Write validated config to config.json.pending (atomic)
 5. Determine which daemons are affected (in this case: nftables)

--- PHASE 2a: Write all config files (no service disruption yet) ---
 6. Render ALL affected templates from the PENDING config
 7. Write ALL daemon config files atomically (write-tmp-rename)
    (e.g., /etc/nftables.conf.new, then rename to /etc/nftables.conf)

--- PHASE 2b: Reload all daemons (brief mixed-state window) ---
 8. Reload daemons in dependency order:
      nftables → dnsmasq → unbound → suricata → wireguard
    Each reload: execute, check exit code, verify service is running.
    Stop at first failure.
 9. TEST: Run post-apply health check (can we still reach WAN? DNS resolves? API up?)

ON SUCCESS (all daemons reloaded cleanly):
10. COMMIT: Copy config.json → config.json.bak (backup previous)
11. Rename config.json.pending → config.json (atomic promote)
12. Update in-memory Config to match
13. Return success to UI, push update via WebSocket

ON FAILURE (any daemon reload fails):
10. ROLLBACK: Re-render ALL templates from LIVE config.json (not pending)
11. Write ALL daemon config files from live templates
12. Reload ALL daemons from live configs (full rollback, same dependency order)
13. Delete config.json.pending
14. Return error to UI with specific failure reason
    (e.g., "nftables rejected rule: invalid port range")
```

**Race condition acknowledgment:** During Phase B, there is a brief window where some daemons are running the new config while others still have the old config (e.g., nftables reloaded but Unbound hasn't yet). This window is typically <1 second for a single-daemon change, up to ~3 seconds if all daemons need reloading. This is an inherent limitation of managing multiple independent daemons -- there is no atomic "reload everything at once" primitive in Linux.

**Why the write-then-reload approach minimizes risk:**
- **Phase 2a (write all files) is non-disruptive.** Writing config files changes nothing until daemons reload. If scoutd crashes during Phase 2a, daemons continue running with their old configs. On restart, scoutd finds the pending file, deletes it, and re-applies from live -- the system self-heals.
- **Phase 2b (reload all) is the smallest possible mixed-state window.** By writing ALL files first, we avoid the old pattern of "write + reload daemon A, then write + reload daemon B" which doubles the mixed-state duration. Now the window is only the serial reload time, not write+reload per daemon.
- **Dependency order matters.** nftables first (firewall is the foundation), then DNS/DHCP (dependent services), then IDS (inspects traffic shaped by nftables). This order ensures no transient security gap (firewall is always updated before services that depend on it).

**Why this matters:** The live config.json ALWAYS reflects the actual running state of the system. If scoutd crashes mid-apply, on restart it sees config.json (last known good) and the presence of config.json.pending (incomplete apply). It deletes the pending file and re-applies from the live config -- the system self-heals.

**Startup recovery logic:**
```go
func (s *Store) RecoverIfNeeded() error {
    // If a pending file exists at startup, a previous apply was interrupted
    if fileExists(s.pendingPath) {
        log.Warn("Found config.json.pending -- previous apply was interrupted, cleaning up")
        os.Remove(s.pendingPath)
        // Re-apply from live config to ensure daemons match config
        return s.ReapplyAll()
    }
    return nil
}
```

**Rollback testing plan (Months 2-9, in Vagrant VM):**

Config rollback is a critical safety mechanism. It must be tested systematically, not just when something happens to break:

| Scenario | When Tested | How |
|----------|------------|-----|
| nftables rejects invalid rule → rollback | Month 2 | Submit malformed firewall rule via API, verify rollback + error message |
| Unbound rejects bad config → rollback | Month 3 | Submit invalid DNS config, verify rollback restores previous DNS |
| scoutd killed mid-apply (SIGKILL) → startup recovery | Month 3 | Kill scoutd during config apply, restart, verify pending file cleaned up and live config re-applied |
| Multi-daemon change fails on 2nd daemon → full rollback | Month 5 | Change that affects nftables + Suricata, make Suricata fail, verify nftables also rolls back |
| Power loss during apply → recovery on reboot | Month 9 | In QEMU, snapshot VM, trigger config apply, force-kill VM, restore, verify recovery |
| Rapid config changes (user clicks "save" 3x fast) → serialization | Month 8 | Concurrent API calls, verify they serialize and don't corrupt config |

Each scenario is a shell script in `tests/rollback_test.sh` that runs in the Vagrant VM. These tests are part of the integration test suite and run at every milestone gate.

**UI "Reset to Last Known Good" button:** The dashboard Settings page includes a "Reset to Previous Config" button that applies `config.json.bak` via the same three-phase commit flow. This gives users a one-click escape hatch if they lock themselves out of something (e.g., firewall rule blocks their own access to the dashboard). This is separate from the USB recovery mode -- it works from the web UI when the user can still reach it.

### 4.5 Config Schema Migration and OTA Rollback

Config schema versions change across firmware releases (e.g., new fields, restructured sections). Migration must be safe across OTA updates and rollbacks.

**When migration runs:** Config migration runs **on first boot of new firmware**, not during OTA install. The RAUC bundle only writes the SquashFS to the inactive slot -- it does not touch the config partition. When the device reboots into the new slot, scoutd starts and handles migration.

**Migration flow on startup:**

```go
func (s *Store) MigrateIfNeeded() error {
    cfg := s.Load()
    expected := CurrentSchemaVersion  // compiled into this firmware's scoutd

    if cfg.Version == expected {
        return nil  // no migration needed
    }

    if cfg.Version > expected {
        // Config is from a NEWER firmware -- we're running after an OTA rollback
        log.Warn("Config version %d is newer than firmware expects %d -- restoring pre-migrate backup", cfg.Version, expected)
        return s.RestorePreMigrate()
    }

    // Config is older -- migrate forward
    // 1. Save pre-migration backup
    copyFile(s.configPath, s.preMigratePath)  // config.json → config.json.pre-migrate

    // 2. Run migration functions sequentially: v3→v4, v4→v5, etc.
    for v := cfg.Version; v < expected; v++ {
        if err := migrations[v](cfg); err != nil {
            // Migration failed -- restore pre-migrate backup
            log.Error("Migration v%d→v%d failed: %v -- restoring backup", v, v+1, err)
            return s.RestorePreMigrate()
        }
    }

    // 3. Save migrated config
    cfg.Version = expected
    return s.Save(cfg)
}
```

**Files involved:**

| File | Purpose |
|------|---------|
| `config.json` | Live config (migrated to current firmware's schema version) |
| `config.json.bak` | Previous config (one-step rollback for user config changes) |
| `config.json.pre-migrate` | Config snapshot from before the last schema migration. Used to restore when OTA rolls back to older firmware. |

**OTA rollback scenario:**
1. Device runs firmware v1.0 with config version 3
2. OTA installs firmware v1.1 to inactive slot, device reboots
3. New firmware's scoutd starts, migrates config from version 3 → 5, saves `config.json.pre-migrate` (version 3)
4. Health check fails → GRUB rolls back to firmware v1.0
5. Old firmware's scoutd starts, reads config version 5 -- higher than expected (3)
6. scoutd detects version mismatch, restores `config.json.pre-migrate` (version 3)
7. Device operates normally on old firmware with matching config

**OTA health check addition:** The post-boot health check (section 7.4 step 5) includes: "config.json version matches firmware's expected schema version range." If the config version is outside the expected range after migration, the health check fails and triggers rollback.

---

## 5. Network Architecture

### 5.1 Zone Model

Default zones created by the setup wizard:

| Zone | Interface | VLAN | Subnet | Default Policy |
|------|-----------|------|--------|----------------|
| WAN | eth0 | -- | DHCP from ISP | input:drop, forward:drop |
| LAN | eth1 (untagged) | -- | 192.168.1.0/24 | input:accept, forward→WAN |
| IoT | eth1.20 | 20 | 192.168.20.0/24 | input:drop, forward→WAN only, isolated from LAN |
| Guest | eth1.30 | 30 | 192.168.30.0/24 | input:drop, forward→WAN only, fully isolated |
| Kids | eth1.40 | 40 | 192.168.40.0/24 | input:accept, forward→WAN, parental DNS applied |

Users without a managed switch just use LAN (flat network) and can add VLANs later.

**WAN DHCP client:** The WAN interface (eth0) obtains its IP via BusyBox `udhcpc`, Alpine's default DHCP client (included in BusyBox -- no extra package). This is configured via `/etc/network/interfaces`:
```
auto eth0
iface eth0 inet dhcp
```
The `networking` OpenRC service in the boot runlevel brings up eth0 and runs `udhcpc` to obtain an IP, gateway, and DNS from the ISP. scoutd renders `/etc/network/interfaces` from the central config (supporting future static WAN IP configuration via the UI).

### 5.2 VLAN Topology

```
eth0 (WAN) ─────── ISP Router / Modem
eth1 (LAN trunk) ── Managed Switch
                       ├── Ports 1-4: VLAN 1 (untagged) → LAN zone
                       ├── Ports 5-6: VLAN 20 (tagged) → IoT zone
                       ├── Port 7:   VLAN 30 (tagged) → Guest WiFi AP
                       └── Port 8:   VLAN 40 (tagged) → Kids zone
```

### 5.3 nftables Ruleset Architecture

Single table `inet scout` (handles IPv4 + IPv6). Key features:

- **Named sets** for zone interfaces (e.g., `set lan_devices { type ifname; elements = { eth1 } }`)
- **IP sets** for blocklists (populated at runtime by scoutd; GeoIP sets added in Phase 2)
- **Per-zone input chains** (input_wan, input_lan, input_iot, etc.)
- **Zone forwarding rules** in the forward chain
- **DNS redirect** in prerouting: forces all LAN DNS queries through Unbound (prevents devices bypassing filtering with hardcoded DNS like 8.8.8.8)
- **IPS chain** queues to Suricata NFQUEUE 0-3 (one per CPU core, Intel N97 has 4 cores) with `fanout,bypass` flags
- **Masquerade** in postrouting for NAT on WAN-bound traffic
- **IPv6 drop-all** in prerouting at highest priority: `meta nfproto ipv6 drop`. v1 is IPv4-only -- without this rule, the `inet` table would pass IPv6 traffic through with no policy applied, creating an unfiltered path around the firewall on ISPs that assign IPv6. This rule ensures IPv6 is blocked at the earliest possible hook point. IPv6 dual-stack support is planned for v2.0.

### 5.4 Suricata IDS/IPS

**Mode: NFQUEUE** (not AF_PACKET inline).

Rationale: ScoutUTM IS the router. AF_PACKET inline requires a transparent bridge topology, which conflicts with routing. NFQUEUE lets nftables forward selected packets to Suricata for inspection, and Suricata returns an accept/drop verdict.

**Realistic throughput expectations (Intel N97, 4 cores):**

| Mode | Expected Throughput | Notes |
|------|-------------------|-------|
| Firewall only (nftables, no Suricata) | **2.5 Gbps** (line rate) | nftables is kernel-level, negligible CPU overhead |
| IDS mode (Suricata alert-only) | **1.5-2.0 Gbps** | Suricata inspects but doesn't verdict-block; less CPU per packet |
| IPS mode (Suricata inline verdict) | **1.0-1.5 Gbps** | Full inspection + verdict per packet; ~500-800 Mbps per core realistic |

These are conservative estimates based on real-world Suricata NFQUEUE deployments with moderate rulesets (~20K rules). The N97's 4 cores at 3.6GHz boost help, but AES-NI is irrelevant for IDS (it helps VPN, not packet inspection). **Do not market ScoutUTM as a "2.5 Gbps IDS/IPS appliance."** Market it as "2.5 Gbps firewall with IDS/IPS" and let throughput expectations be set honestly.

**THIS IS THE HIGHEST TECHNICAL RISK IN THE PROJECT.** If IPS caps at 800-1200 Mbps on hardware and we've marketed "2.5 Gbps IPS", credibility is destroyed. The throughput numbers from Month 10 hardware benchmarks define the product SKU -- not the other way around. No throughput claim appears on the website, packaging, or GTM materials until hardware benchmarks are complete and verified.

**Throughput-defines-SKU rule:**
- The benchmark results from Month 10 become the official product specifications
- If IPS delivers 900 Mbps, the product page says "up to 900 Mbps IPS throughput" -- not 1 Gbps, not 1.5 Gbps
- If IDS delivers 1.5 Gbps, the product page says "up to 1.5 Gbps IDS throughput"
- Firewall-only throughput (2.5 Gbps) is stated separately as "2.5 Gbps firewall throughput"
- All throughput claims include test conditions: ruleset size, traffic profile, measurement tool
- GTM.md must not be finalized until Month 10 benchmark data is available

**v1 ships IDS-only by default. IPS is opt-in.**

Real hardware throughput numbers won't be known until Month 10 (after hardware purchase). Rather than building the product around an assumption about IPS performance, v1 defaults to **IDS mode** (alert-only, no inline blocking). IDS has significantly lower per-packet overhead (1.5-2.0 Gbps vs 1.0-1.5 Gbps) and zero risk of Suricata becoming a bottleneck.

Users who want inline blocking can enable **IPS mode** via the dashboard, with a clear warning: "IPS mode inspects and blocks traffic in real-time. This may reduce throughput to ~1 Gbps depending on traffic patterns and ruleset size." This is an informed user choice, not a default.

**Why this is the right call:**
- IDS provides 90% of the security value (visibility, alerts, forensics) with no throughput penalty
- IPS throughput is hardware-dependent and unknown until Month 10 -- building the default around it is risky
- If hardware IPS benchmarks are strong (>1.2 Gbps), IPS-default can be reconsidered for v1.1
- No marketing claims need to be walked back

**Benchmark plan (two phases):**
1. **VM benchmark (Month 5, during Suricata integration):** Run Suricata NFQUEUE in the Vagrant VM with iperf3 between the scout VM and client VM. VM throughput will be lower than bare metal (virtual NICs, hypervisor overhead), but this validates the NFQUEUE pipeline, per-queue fanout, rule loading, and eve.json alert flow. Any architectural issues (queue drops, Suricata config problems, fail-open behavior) surface here.
2. **Hardware benchmark (Month 10, after hardware purchase):** Run the same tests on Odroid H4+ with real Intel I226-V NICs and iperf3 through the physical box. This produces the actual throughput numbers for documentation and marketing. If IPS throughput exceeds 1.2 Gbps, consider making IPS the default in a future release.

**Benchmark go/no-go gate (Month 10):** The hardware benchmark is a hard gate before any public throughput claims. The benchmark protocol:
1. iperf3 TCP throughput through the box (client → ScoutUTM → server) at each mode (firewall-only, IDS, IPS)
2. Each test run for 60 seconds, repeated 5 times, median reported
3. Tested with ET Open ruleset (~20K rules) at default settings
4. Tested with and without jemalloc LD_PRELOAD
5. Results documented in a benchmark report with exact versions (kernel, Suricata, ruleset date)
6. Benchmark report reviewed before any marketing copy references throughput numbers
7. If IPS throughput is below 800 Mbps, IPS mode gets a stronger dashboard warning and GTM materials position ScoutUTM as an IDS appliance with optional IPS

**Musl allocator risk:** Alpine's musl libc uses a simple memory allocator with per-arena locks. Under heavy multi-threaded workloads (Suricata runs 4+ packet processing threads), musl's malloc can suffer severe futex contention -- benchmarks show up to 7x slowdown vs glibc's malloc in thread-heavy servers. This is a real risk for IPS mode where every packet path allocates memory.

**Mitigation:** If Month 5 VM benchmarks or Month 10 hardware benchmarks show Suricata throughput significantly below expectations, the first step is to swap in a scalable allocator via `LD_PRELOAD`:
```sh
# In /etc/init.d/suricata (OpenRC service script)
export LD_PRELOAD=/usr/lib/libjemalloc.so.2  # or libmimalloc.so
```
jemalloc and mimalloc are both available in Alpine repos (`apk add jemalloc` / `apk add mimalloc`) and are designed for high-concurrency workloads. This swap requires no Suricata recompilation and no code changes -- it's a one-line environment variable. The benchmark plan (Month 5 and 10) will test both with and without the LD_PRELOAD to quantify the impact. If the allocator swap yields >20% throughput improvement, it ships as the default Suricata configuration.

Key configuration:
- `fail-open: yes` -- if Suricata crashes, packets pass through (availability over security)
- NFQUEUE fanout across 4 queues (one per CPU core) for parallel processing
- Rulesets: ET Open as baseline + curated ScoutUTM overlay (suppress noisy rules, enable IoT-specific signatures)
- `suricata-update` on a schedule for rule freshness

**Eve log integration:** Suricata outputs EVE JSON via a **Unix datagram socket** (`filetype: unix_dgram`, path: `/var/run/suricata/eve.sock`). scoutd opens this socket at startup and reads JSON lines as they arrive -- no polling, no file tailing. This is real-time, zero-latency alert delivery.

If scoutd restarts, it recreates the socket listener. Suricata's `unix_dgram` output silently drops messages when the socket has no reader (no backpressure, no Suricata stall). After scoutd reconnects, new alerts flow immediately. scoutd also writes a persistent copy of alerts to `/var/lib/scout/suricata/eve.json` for the 7-day retention log.

**Why Unix socket over file tailing:** File-based EVE output (`eve.json` on disk) requires polling with `inotify` or periodic reads, adds disk I/O, and creates a rotation coordination problem between Suricata and scoutd. The socket approach is cleaner -- Suricata pushes, scoutd receives, no intermediate file for real-time alerts.

### 5.5 DNS Architecture: Why Both dnsmasq and Unbound

ScoutOS runs two daemons for DNS/DHCP, each doing what it's best at. This is not redundancy -- it's separation of concerns.

**dnsmasq** handles DHCP only (its DNS listener is disabled):
- DHCP lease management with static assignments
- DHCP option distribution (gateway, DNS server, NTP server, domain)
- Lease file provides the device list (MAC → IP → hostname)
- scoutd parses `/var/lib/scout/dnsmasq.leases` for device monitoring
- Tiny footprint, rock-solid, 20+ years of field testing

**Unbound** handles all DNS:
- Full recursive resolver (not a forwarder) -- resolves from root servers
- DNSSEC validation (critical for trust, requires accurate time via chrony)
- Response Policy Zones (RPZ) for blocklist-based filtering (malware, ads, parental)
- DNS-over-TLS (DoT) and DNS-over-HTTPS (DoH) to upstream forwarders when configured
- Per-client query logging for parental controls and analytics
- `local-zone` and `local-data` for custom DNS records

**Why not consolidate into dnsmasq alone?** dnsmasq's DNS is a simple cache/forwarder. It cannot do recursive resolution, DNSSEC validation, RPZ blocklists, or DoH. These are all core requirements for a UTM's DNS filtering engine.

**Why not consolidate into Unbound alone?** Unbound has no DHCP server. DHCP is fundamentally different from DNS -- lease state, MAC address tracking, option negotiation. There is no DNS daemon that also does production-quality DHCP.

**How they connect:**
```
LAN device sends DNS query to port 53
  → nftables redirects to local Unbound (prevents DNS bypass)
  → Unbound checks RPZ blocklists → block or resolve
  → Response returned to device

LAN device sends DHCP DISCOVER
  → dnsmasq responds with IP lease
  → dnsmasq sets DNS server option = ScoutUTM's LAN IP (Unbound)
  → dnsmasq writes lease to /var/lib/scout/dnsmasq.leases
  → scoutd watches lease file for device list updates
```

This is the same pattern used by Pi-hole, AdGuard Home, and most professional DNS filtering deployments. It's proven and each daemon stays in its lane.

**Note on musl's DNS resolver:** musl libc's built-in DNS stub resolver has known limitations -- it uses UDP only (no TCP fallback for large responses), queries all nameservers in parallel (vs glibc's sequential fallback), and lacks EDNS support. These quirks cause real problems for applications that rely on the system resolver. **ScoutOS is not affected** because:
- Client DNS is handled entirely by Unbound, which implements full recursive resolution, TCP, EDNS, DNSSEC, and DoT/DoH independently of the C library.
- scoutd is built with `CGO_ENABLED=0`, so Go uses its own pure-Go DNS resolver (not musl's). Go's resolver supports TCP fallback and handles large responses correctly.
- dnsmasq has its own DNS implementation (though its DNS listener is disabled in ScoutOS).

No ScoutOS component relies on musl's `getaddrinfo()` for DNS resolution in production.

### 5.6 Per-Device DNS Filtering (Parental Controls)

Parental controls require per-device DNS filtering: a child's device blocks TikTok while an adult's allows it. This uses Unbound's native `access-control-tag` feature (available since Unbound 1.9) -- no DNS proxy in scoutd, no multiple Unbound instances.

**How it works:**

1. Each parental profile (child, teen, adult) maps to a **tag** in Unbound (e.g., `"child"`, `"teen"`)
2. Each tagged profile has associated RPZ zones (e.g., the `rpz-kids-block` zone applies only to queries tagged `"child"`)
3. Devices are mapped to tags by their source IP via `access-control-tag`
4. scoutd renders the Unbound config with the correct tag mappings whenever parental profiles or device assignments change

**Unbound config example (rendered by scoutd):**
```
# Tag definitions -- one per parental profile
define-tag: "child teen"

# Per-device IP → tag mapping
access-control-tag: 192.168.1.50/32 "child"
access-control-tag: 192.168.1.51/32 "teen"
# Devices with no tag get the default (no filtering beyond global malware/phishing RPZ)

# RPZ zones with tag restrictions
rpz:
    name: "rpz-kids-block"
    zonefile: "/var/lib/scout/blocklists/rpz-kids.zone"
    tags: "child"
    # Only applied to queries from IPs tagged "child"

rpz:
    name: "rpz-teen-block"
    zonefile: "/var/lib/scout/blocklists/rpz-teen.zone"
    tags: "child teen"
    # Applied to both child and teen

rpz:
    name: "rpz-malware"
    zonefile: "/var/lib/scout/blocklists/rpz-malware.zone"
    # No tags = applied to ALL queries (global)
```

**Automatic static DHCP reservations:** When a device is assigned to a parental profile, scoutd automatically creates a static DHCP reservation in dnsmasq for that device's MAC address, pinning it to a fixed IP from the zone's DHCP range. This ensures the Unbound `access-control-tag` mapping is stable across lease renewals. The user never manually configures static IPs -- it happens behind the scenes.

**Flow when a parent assigns a device to a profile:**
1. Parent selects "Timmy's iPad" → "Child" profile in the UI
2. scoutd looks up the device's MAC address (from ARP/DHCP lease tracking)
3. scoutd assigns a static IP from the device's zone DHCP range (or keeps its current IP)
4. scoutd updates dnsmasq config with the static reservation (MAC → fixed IP)
5. scoutd updates Unbound config with `access-control-tag` mapping (fixed IP → profile tag)
6. dnsmasq and Unbound are reloaded via the standard three-phase commit

**Edge case -- device currently online with a different IP:** scoutd maps both the current dynamic IP and the new static reservation in Unbound immediately. The UI notes: "DNS filtering is active. The device will receive its permanent IP on next reconnect or lease renewal."

**MAC address randomization handling:** Modern devices (iOS 14+, Android 10+, Windows 11) use per-network randomized MAC addresses by default. These randomized MACs are **stable per-network** -- the device uses the same random MAC every time it connects to the same network. scoutd identifies devices by whatever MAC they present, so stable random MACs work fine with static DHCP reservations.

**Rotating MAC randomization** (opt-in on iOS as "Rotate Wi-Fi Address") changes the MAC periodically even on the same network, breaking all MAC-based identification. This is a known industry-wide limitation that affects every router. ScoutOS handles it as follows:
- **Dashboard warning:** When a device is assigned to a parental profile, the UI shows: "For reliable parental controls, ensure this device does not rotate its Wi-Fi address. On iPhone: Settings → Wi-Fi → [network] → set Private Wi-Fi Address to Fixed."
- **Detection:** If scoutd sees a known hostname appear with a new MAC while the previously-associated MAC disappears, it flags the device in the dashboard: "This device may be using rotating MAC addresses. Parental controls may not apply reliably."
- This is documented as a known limitation (see section 14).

**SafeSearch and YouTube restricted mode enforcement:**

SafeSearch and YouTube restricted mode are enforced via DNS overrides in Unbound, applied through the same tag system as per-device filtering. When a parental profile has `safe_search: true` or `youtube_restricted: true`, scoutd renders additional Unbound `local-zone` entries for tagged devices:

```
# SafeSearch enforcement (applied to "child" and "teen" tagged devices)
# Google: CNAME to forced safe search
local-zone: "www.google.com" redirect
local-data: "www.google.com CNAME forcesafesearch.google.com"
local-zone-tag: "www.google.com" "child teen"

# Bing: CNAME to strict mode
local-zone: "www.bing.com" redirect
local-data: "www.bing.com CNAME strict.bing.com"
local-zone-tag: "www.bing.com" "child teen"

# DuckDuckGo: CNAME to safe mode
local-zone: "duckduckgo.com" redirect
local-data: "duckduckgo.com CNAME safe.duckduckgo.com"
local-zone-tag: "duckduckgo.com" "child teen"

# YouTube restricted mode: CNAME to restricted endpoint
local-zone: "www.youtube.com" redirect
local-data: "www.youtube.com CNAME restrict.youtube.com"
local-zone-tag: "www.youtube.com" "child teen"

# YouTube mobile
local-zone: "m.youtube.com" redirect
local-data: "m.youtube.com CNAME restrict.youtube.com"
local-zone-tag: "m.youtube.com" "child teen"

# YouTube API (used by apps)
local-zone: "youtube.googleapis.com" redirect
local-data: "youtube.googleapis.com CNAME restrict.youtube.com"
local-zone-tag: "youtube.googleapis.com" "child teen"
```

These are standard techniques used by Pi-hole, OpenDNS, CleanBrowsing, and every major parental control product. The DNS-level approach works across all browsers and apps on the device without requiring browser extensions or proxy configuration.

**DNS-over-HTTPS (DoH) bypass mitigation:** Port 53 redirection prevents devices from using alternative DNS servers, but DoH (port 443) blends with normal HTTPS traffic and cannot be blocked by port. Partial mitigation: the parental RPZ blocklists include well-known DoH provider domains (`dns.google`, `cloudflare-dns.com`, `dns.quad9.net`, `doh.opendns.com`, `mozilla.cloudflare-dns.com`, `dns.nextdns.io`). When a child's device tries to use DoH, the DNS lookup for the DoH server itself is blocked by Unbound. This catches the most common DoH configurations (Firefox uses `mozilla.cloudflare-dns.com` by default).

**Known limitation:** This does not catch DoH to IP addresses directly (no DNS lookup involved), custom DoH providers, or DoH configured via IP in browser settings. A technically sophisticated user can always bypass DNS-based filtering -- this is true of every DNS-based parental control product (Pi-hole, OpenDNS, CleanBrowsing, Firewalla). Full DoH blocking would require TLS inspection (MITM proxy), which is out of scope for v1 and raises significant privacy/trust concerns. See section 14.

### 5.7 NTP: chrony

Accurate time is essential for a security appliance:
- **TLS certificate validation** (expiry checks fail with wrong time)
- **DNSSEC signature validation** (signatures have validity windows)
- **Suricata alert timestamps** (forensic value of IDS logs depends on time accuracy)
- **Parental control schedules** (bedtime enforcement needs correct local time)
- **Log correlation** (meaningful timestamps across all services)
- **WireGuard** (handshake timestamps)

**chrony** is the right choice over ntpd:
- Faster initial sync (critical after power loss -- device could boot with wildly wrong time)
- Lower memory footprint than ntpd
- Handles intermittent connectivity well (important if WAN takes a few seconds to come up)
- Available in Alpine repos (`apk add chrony`)

Configuration: scoutd renders `/etc/chrony/chrony.conf` from the central config. Default upstream: `pool.ntp.org` with `iburst` for fast initial sync. Runs as an OpenRC service in the `boot` runlevel (before scoutd, Suricata, and Unbound -- they need time to be correct before they start).

**Boot time edge case:** chronyd starts in the boot runlevel, but WAN may not be up yet (DHCP takes 1-3 seconds). This is fine -- chrony doesn't block boot. It starts, attempts NTP sync in the background, and retries with increasing intervals until WAN is available. The `iburst` option sends 4 rapid requests on first contact for fast convergence.

**fake-hwclock fallback:** The Odroid H4+ likely has a battery-backed RTC (x86 boards with Intel SoCs typically include one -- verify during Month 10 hardware bring-up). Regardless, fake-hwclock is kept as a safety net for boards without RTC or if the RTC battery dies. After power loss without a functioning RTC, the system clock resets to the kernel's compiled-in epoch. To avoid booting with a wildly wrong time (which breaks TLS and DNSSEC):
- Install `fake-hwclock` (`apk add fake-hwclock`), which saves the current time to `/etc/scout/fake-hwclock.data` every hour and at shutdown
- On boot, `fake-hwclock` restores the last-saved time before chronyd starts -- the clock is at worst a few hours behind, not years
- chronyd then corrects to real NTP time within seconds of WAN coming up

**scoutd time sanity check:** On startup, scoutd checks if the system time is before 2025-01-01. If so, it logs a warning and delays DNSSEC validation and TLS certificate checks until chrony reports a synchronized source (`chronyc tracking | grep "Leap status.*Normal"`). This prevents false-positive certificate validation failures during the brief window between boot and NTP sync.

Boot sequence updated:
```
OpenRC boot runlevel:
  ├── networking (interfaces up)
  ├── fake-hwclock (restore last-known time from disk)
  ├── chronyd (NTP sync, iburst for fast initial time -- non-blocking)

OpenRC default runlevel:
  ├── nftables
  ├── dnsmasq
  ├── unbound
  ├── suricata
  ├── wireguard (if configured)
  └── scoutd (time sanity check on startup)
```

---

## 6. Web Dashboard

### 6.1 Backend (part of scoutd)

Go HTTP server using `chi` router. Serves:
- Static Vue.js assets from `//go:embed web/dist`
- REST API on `/api/v1/*`
- WebSocket on `/api/v1/ws`
- All on HTTPS (port 443), HTTP (80) redirects

Self-signed TLS cert generated at first boot, stored in `/etc/scout/tls/`.

### 6.2 REST API Summary

| Endpoint Group | Methods | Purpose |
|---------------|---------|---------|
| `/api/v1/health` | GET | Unauthenticated. Returns `{"healthy": true, "uptime": 3600, "version": "1.0.0"}`. Same checks as OTA rollback health (7.3) |
| `/api/v1/auth/login` | POST | Authenticate, get session token |
| `/api/v1/system/*` | GET, PUT, POST | Status, settings, reboot, updates |
| `/api/v1/network/*` | GET, PUT, POST, DELETE | Interfaces, zones, VLANs |
| `/api/v1/firewall/*` | GET, PUT, POST, DELETE | Rules, forwardings, port forwards |
| `/api/v1/dns/*` | GET, PUT | DNS settings, blocklists, query log |
| `/api/v1/vpn/wireguard/*` | GET, PUT, POST, DELETE | WG config, peers, QR code |
| `/api/v1/devices/*` | GET, PUT | Device list, naming, zone/profile assignment |
| `/api/v1/ids/*` | GET, PUT | IDS/IPS settings, alerts |
| `/api/v1/parental/*` | GET, PUT, POST | Profiles, schedules, assignments |
| `/api/v1/wizard/*` | POST | Setup wizard step processing |
| `/api/v1/ws` | GET (upgrade) | WebSocket for real-time data |

### 6.3 WebSocket Protocol

JSON messages over WSS. Client subscribes to topics. Server pushes:

- **stats** (every 2s): CPU, memory, uptime, WAN/LAN throughput, connection count, DNS query/block counts
- **devices** (on change): device list with MAC, IP, hostname, zone, online status, bandwidth
- **alert** (on event): IDS alerts with plain-language description, severity, source/dest, action taken

**Connection limits:**
- Max **10 concurrent WebSocket connections** (per device, this is a home appliance, not a SaaS platform)
- If >5 clients connected, stats interval increases from 2s to 5s (reduce CPU load)
- Idle clients disconnected after **5 minutes** of no client→server messages (ping/pong heartbeat)
- Connection 11+ receives HTTP 503 with `Retry-After: 30` header

**Alert broadcast rate limiting:** During an alert storm (e.g., nmap scan triggers 1000+ Suricata alerts in seconds), the WebSocket hub rate-limits alert messages to **max 10 alerts/second per client**. Excess alerts are buffered in a ring buffer (last 100) and delivered when the rate drops, or the client can fetch the full alert history via `GET /api/v1/ids/alerts`. This prevents a port scan from flooding the browser and causing UI freeze. Stats and device messages are unaffected by alert rate limiting.

### 6.4 Authentication

- Single local admin account (no multi-user in v1)
- Password set during setup wizard, stored as bcrypt hash
- Login returns random 256-bit session token
- Token as `Authorization: Bearer <token>` header or `scout_session` cookie
- Sessions expire after 24 hours of inactivity
- **Session persistence:** Sessions are in-memory only — lost on scoutd restart or reboot. This is intentional: no session tokens are written to disk, reducing the attack surface. Config apply does NOT restart scoutd (scoutd stays running and reloads other daemons externally), so normal config changes do not log the user out. Sessions are only lost during OTA updates (reboot) or manual service restart. Users re-login after reboot — acceptable for a home appliance.
- CSRF: SameSite=Strict cookies + origin checking
- **Brute-force protection:** Per-IP rate limit of 5 failed login attempts per minute -- after 5 failures, the IP is blocked for 5 minutes. Global rate limit of 20 failed attempts per minute across all IPs (protects against distributed brute-force from multiple LAN devices). Implemented as in-memory counters with expiring buckets in `internal/auth/session.go`. Successful login resets the per-IP counter. Blocked attempts return HTTP 429 with `Retry-After` header.

### 6.5 Frontend (Vue.js 3 SPA)

**Stack:**
- Vue 3 Composition API
- Vue Router (client-side routing)
- Pinia (state management)
- Tailwind CSS (utility-first, purged for small output)
- uPlot (lightweight charts)
- Server-side QR generation (WireGuard peer QR codes — private key never in browser)

**No heavy UI framework** (no Vuetify, no Element Plus). Target bundle: <500KB gzipped.

**Bundle size strategy:** 10+ views with charts and WebSocket make 500KB tight. Mitigations:
- **Lazy-load all routes** except Login and Dashboard (the two most-visited views). All other views load on-demand via Vue Router's `() => import('./views/Firewall.vue')`. Initial load is Login + Dashboard + shared framework only.
- **uPlot over Chart.js** -- uPlot is ~35KB min+gzip vs Chart.js ~65KB. For the simple time-series traffic graphs ScoutUTM needs, uPlot is sufficient.
- **Tailwind CSS purge** removes unused utility classes. Production CSS is typically 10-20KB gzipped.
- **Monitor bundle size in CI** -- `vite build` reports chunk sizes. Set a CI warning threshold at 400KB and a failure threshold at 600KB gzipped.

**Views:** Login, Dashboard, Firewall, DNS, VPN, Devices, IDS, Parental, Network, System, Setup Wizard (7 steps)

### 6.6 Setup Wizard

Runs on first boot when `config.system.setup_done == false`:

1. **Welcome** -- branding, brief product intro
2. **Admin Password** -- minimum 10 characters
3. **Network Detection** -- auto-detect WAN (DHCP) and LAN interfaces
4. **LAN Config** -- set IP range, enable/disable DHCP
5. **Security Defaults** -- IDS/IPS (recommended: yes, IDS mode), DNS filtering (malware + phishing), ad blocking (optional)
6. **VLAN Setup** (optional, can skip) -- create IoT/Guest/Kids zones if user has managed switch
7. **Summary + Apply** -- review all settings, apply, boot into operational mode

---

## 7. OTA Update System

### 7.1 Partition Layout

128GB NVMe SSD, GPT:

```
+--------+----------+----------+--------+----------+
|  ESP   | rootfs-a | rootfs-b | config |   data   |
| 256MB  |  640MB   |  640MB   |  32MB  | ~126GB   |
| vfat   | squashfs | squashfs | ext4   | ext4     |
| p1     | p2       | p3       | p4     | p5       |
+--------+----------+----------+--------+----------+
```

| Partition | Mount | Purpose |
|-----------|-------|---------|
| ESP (p1) | `/boot/efi` | GRUB EFI binary, grubenv |
| rootfs-a (p2) | `/` (via overlayfs) | SquashFS root A -- includes kernel + initramfs + entire OS |
| rootfs-b (p3) | (inactive) | SquashFS root B (standby for updates) |
| config (p4) | `/etc/scout` | Persistent config (survives updates) |
| data (p5) | `/var/lib/scout` | Suricata rules, logs, DHCP leases |

**Key design decisions:**

**No separate boot partition.** The kernel (`vmlinuz`) and initramfs are stored inside each SquashFS rootfs image at `/boot/` within the SquashFS. OTA updates atomically replace the entire system -- kernel, initramfs, and rootfs -- as a single unit. No risk of kernel/rootfs version mismatch. No separate boot-a/boot-b partitions needed.

**PARTUUID-based mounting.** The build system writes stable PARTUUIDs into the GPT table. The initramfs, GRUB config, and RAUC system.conf all reference partitions by `PARTUUID=` rather than `/dev/nvme0n1pN`. This is robust against device enumeration order changes. Hardcoded `/dev/nvme0n1pN` paths in this document are for readability only.

**Boot loading strategy: ESP extraction, not SquashFS loopback.** GRUB's built-in SquashFS loopback support (`loopback sqfs (hd0,gpt2)`) exists since GRUB 2.06, but it's uncommon in production embedded systems and a boot failure here means a bricked device with no easy recovery path. Instead, the build system extracts the kernel and initramfs from each SquashFS to slot-specific subdirectories in the ESP:

```
/boot/efi/EFI/scout/
├── grub.cfg
├── grubenv
├── slot-a/
│   ├── vmlinuz
│   └── initramfs
└── slot-b/
    ├── vmlinuz
    └── initramfs
```

GRUB loads the kernel from the ESP natively (well-tested, standard UEFI path) rather than loopback-mounting a raw SquashFS partition:

```
# grub.cfg snippet for slot A (standard ESP loading, no loopback)
linux /EFI/scout/slot-a/vmlinuz root=/dev/nvme0n1p2 rootfstype=squashfs ro rauc.slot=A console=tty0 console=ttyS0,115200n8
initrd /EFI/scout/slot-a/initramfs
```

**The kernel is still stored inside the SquashFS** for integrity (the SquashFS is the signed artifact). The ESP copies are extracted during the OTA install step. RAUC's post-install hook handles this:

```sh
#!/bin/sh
# /etc/rauc/post-install.sh -- called by RAUC after writing the SquashFS to a slot
set -eu
SLOT="$1"  # "A" or "B"
SLOT_DEV="$2"  # /dev/nvme0n1p2 or p3
SLOT_DIR=$(echo "$SLOT" | tr 'A-Z' 'a-z')  # "a" or "b"
ESP="/boot/efi/EFI/scout"
MOUNT=$(mktemp -d)

mount -t squashfs -o ro "$SLOT_DEV" "$MOUNT"

# Extract to temp files first, then atomic rename -- prevents partial writes on power loss
cp "$MOUNT/boot/vmlinuz" "${ESP}/slot-${SLOT_DIR}/vmlinuz.new"
cp "$MOUNT/boot/initramfs" "${ESP}/slot-${SLOT_DIR}/initramfs.new"

# Verify extracted files are valid (non-zero size, kernel magic bytes)
[ -s "${ESP}/slot-${SLOT_DIR}/vmlinuz.new" ] || { echo "ERROR: vmlinuz extraction failed"; exit 1; }
[ -s "${ESP}/slot-${SLOT_DIR}/initramfs.new" ] || { echo "ERROR: initramfs extraction failed"; exit 1; }

# Atomic rename (as atomic as FAT32 allows)
mv "${ESP}/slot-${SLOT_DIR}/vmlinuz.new" "${ESP}/slot-${SLOT_DIR}/vmlinuz"
mv "${ESP}/slot-${SLOT_DIR}/initramfs.new" "${ESP}/slot-${SLOT_DIR}/initramfs"
sync  # Flush to disk

umount "$MOUNT"
rmdir "$MOUNT"
```

**Mid-extraction failure protection:** If power is lost during the `cp` phase, the `.new` temp files are incomplete but the previous `vmlinuz`/`initramfs` are untouched. On the next RAUC install attempt, the hook overwrites the `.new` files. If power is lost during the `mv` phase, FAT32 rename is not truly atomic -- but GRUB's try-counter mechanism handles this: the new slot fails to boot (corrupt kernel), GRUB decrements the try counter, and after 3 failures falls back to the previous working slot.

**Rescue boot entry in GRUB:** In addition to the A/B slot entries, grub.cfg includes a **rescue entry** that always boots the currently-marked-good slot, bypassing the try-counter logic:

```
# grub.cfg -- rescue entry (always boots last known-good slot)
menuentry "ScoutOS Rescue (last known good)" {
    # Read which slot was last marked good
    load_env --file /EFI/scout/grubenv
    if [ "$A_OK" = "1" ]; then
        linux /EFI/scout/slot-a/vmlinuz root=/dev/nvme0n1p2 rootfstype=squashfs ro rauc.slot=A scout.rescue=1
        initrd /EFI/scout/slot-a/initramfs
    else
        linux /EFI/scout/slot-b/vmlinuz root=/dev/nvme0n1p3 rootfstype=squashfs ro rauc.slot=B scout.rescue=1
        initrd /EFI/scout/slot-b/initramfs
    fi
}
```

This entry is hidden from normal boot (GRUB auto-selects the active slot with a 3-second timeout) but is available via the GRUB menu (press Shift/Esc during boot). The `scout.rescue=1` kernel parameter tells scoutd it's a rescue boot -- scoutd logs this and shows a dashboard banner: "System booted in rescue mode from the previous firmware version."

**Power-loss testing (Month 9, in QEMU):** Simulate power loss during OTA install by force-killing the QEMU process at various stages of the post-install hook. Verify that: (a) the previous slot remains bootable, (b) GRUB falls back correctly, (c) re-running the OTA completes successfully.

**Why this is safer:** ESP loading is the most tested GRUB code path (every UEFI system boots this way). SquashFS loopback adds a fragile layer that's hard to debug if it fails. The trade-off is a small amount of duplication (~30MB of kernel+initramfs in the ESP alongside the SquashFS), which the 256MB ESP accommodates easily. ESP space: 2 slots x ~30MB + GRUB ~5MB + grubenv = ~65MB used of 256MB.

**Config partition is 32MB, not 256MB.** Actual persistent config data is <1MB (config.json, TLS certs, backup). 32MB provides 30x headroom for future needs (config snapshots, local CA certs, custom blocklist overrides) without wasting space. The saved 224MB + 128MB (former boot-a) goes to the data partition.

**Why ext4, not F2FS:** The Odroid H4+ uses an NVMe SSD with a hardware Flash Translation Layer (FTL) that handles wear leveling, garbage collection, and bad block management transparently. On storage with a hardware FTL (NVMe, eMMC, SATA SSD), ext4 is standard and the device firmware handles flash-specific concerns. F2FS is designed for raw NAND flash without an FTL, where the OS must manage erase block distribution -- this does not apply here. Using F2FS on an NVMe SSD provides negligible benefit and adds CPU overhead for log-structured writes that duplicate what the SSD controller already does. If ScoutOS is ever ported to a board with raw NAND (no FTL), F2FS should be reconsidered for the config and data partitions.

### 7.2 Boot Process

```
UEFI firmware → GRUB (from ESP)
  → Read grubenv (ORDER, A_TRY, B_TRY, A_OK, B_OK)
  → Select active slot (A or B)
  → Load kernel + initramfs from ESP (/EFI/scout/slot-a/ or slot-b/)
  → initramfs sets up OverlayFS:
      lower = SquashFS (read-only, from the matching slot partition)
      upper = tmpfs (volatile writes, lost on reboot)
  → Mount config partition at /etc/scout (persistent)
  → Mount data partition at /var/lib/scout (persistent)
  → switch_root → OpenRC starts services
```

GRUB loads the kernel from the ESP (standard UEFI path, no loopback):
```
# grub.cfg snippet for slot A
linux /EFI/scout/slot-a/vmlinuz root=/dev/nvme0n1p2 rootfstype=squashfs ro rauc.slot=A
initrd /EFI/scout/slot-a/initramfs
```

Kernel + initramfs in the ESP are extracted from the SquashFS by RAUC's post-install hook (see 7.1). The SquashFS remains the signed source of truth -- the ESP copies are derived artifacts. If the ESP copies become corrupted, re-installing the current slot via RAUC regenerates them.

### 7.3 RAUC Integration

RAUC handles:
- Bundle format with manifest, images, cryptographic verification
- Slot management (active vs. standby)
- GRUB grubenv variable updates
- Atomic slot switching

RAUC system config at `/etc/rauc/system.conf`:
```ini
[system]
compatible=scout-utm
bootloader=grub
grubenv=/boot/efi/EFI/scout/grubenv

[keyring]
path=/etc/rauc/keyring.pem

[slot.rootfs.0]
device=/dev/nvme0n1p2
type=raw
bootname=A

[slot.rootfs.1]
device=/dev/nvme0n1p3
type=raw
bootname=B
```

### 7.4 Update Flow

```
1. scoutd checks update server (periodic or manual)
   → GET https://update.scoututm.com/manifest.json (see 7.5)

2. Download RAUC bundle to /tmp (tmpfs), verify SHA-256
   → Progress pushed to UI via WebSocket

3. rauc install /tmp/scout-X.Y.Z.raucb
   → RAUC verifies ECDSA P-256 signature against baked-in keyring
   → Writes SquashFS to inactive slot
   → Updates grubenv: switch slot order, set try counter

4. User approves reboot (or auto-reboot in maintenance window)

5. GRUB boots new slot, scoutd runs health check:
   → nftables loaded?
   → WAN interface has IP?
   → DNS resolves?
   → LAN interface up?
   → scoutd API responds?

6. Health check passes → rauc status mark-good (commits update)
7. Health check fails after 3 attempts → GRUB falls back to previous slot
```

### 7.5 OTA Update Server

The update server is a static file host. No dynamic API, no device registry, no telemetry.

**Infrastructure:** A self-hosted VPS (nginx or caddy) serving static files over HTTPS at `update.scoututm.com`. This is the same VPS budgeted under "domain + hosting + infra" (€500). The server holds RAUC bundles and a manifest file -- nothing else.

**Manifest file:** `https://update.scoututm.com/manifest.json`

```json
{
  "version": "1.0.1",
  "bundle": "scout-1.0.1.raucb",
  "sha256": "a1b2c3d4...",
  "min_version": "1.0.0"
}
```

**Update check flow:**
1. scoutd periodically GETs `manifest.json` (interval configurable, default: daily)
2. Compare `version` against running firmware version
3. If newer and running version >= `min_version`, notify user (or auto-download if configured)
4. Download bundle file from same server, verify SHA-256 matches manifest
5. Pass bundle to RAUC for signature verification and installation (see 7.6)

**Publishing workflow (manual):** Founder builds a signed RAUC bundle locally, then uploads via SCP or rsync:
```sh
scp build/scout-1.0.1.raucb update-server:/var/www/update.scoututm.com/
# Update manifest.json with new version, filename, and sha256
scp build/manifest.json update-server:/var/www/update.scoututm.com/
```

**No device authentication.** Devices are anonymous -- the server cannot distinguish one device from another. The manifest and bundle are public static files. Security is at the cryptographic layer: RAUC verifies the ECDSA P-256 signature on the bundle before installing. An attacker who compromises the update server can serve a malicious bundle, but RAUC will reject it because it lacks a valid signature.

**No channels.** Beta testers receive a different `server_url` in their `UpdateConfig` pointing to a separate path or test server. No channel selection logic in the update client.

**Availability:** If the update server is down, devices continue operating normally. scoutd logs the failed check and retries on the next interval. Manual OTA via USB is always available as a fallback.

### 7.6 Firmware Signing & Downgrade Protection

**Signing:** ECDSA P-256 keypair. Public cert baked into rootfs as `/etc/rauc/keyring.pem`. Private key on build machine only, never shipped on devices. Prevents MITM firmware injection and unofficial firmware.

**Downgrade protection:** RAUC bundles include a `version` field in the manifest. scoutd enforces **version monotonicity** before calling `rauc install`:

```go
func (u *Updater) ValidateBundle(bundleVersion, currentVersion string) error {
    if semver.Compare(bundleVersion, currentVersion) <= 0 {
        return fmt.Errorf("refusing downgrade: bundle %s <= current %s", bundleVersion, currentVersion)
    }
    return nil
}
```

An attacker who obtains an old, validly-signed bundle cannot downgrade a device to a version with known vulnerabilities. The only way to force a downgrade is physical access (USB recovery mode with factory reset).

---

## 8. Image Build System

### 8.1 Build Targets

```makefile
make frontend    # Build Vue.js → static assets
make scoutd      # Build Go binary (CGO_ENABLED=0, static, ldflags for version)
make rootfs      # Alpine rootfs via alpine-make-rootfs + overlay + scoutd binary
make squashfs    # Compress rootfs to SquashFS (zstd) + dm-verity hash tree + root hash
make initramfs   # Build initramfs with embedded dm-verity root hash
make image       # Full disk image (GPT, partitions, GRUB, kernel, squashfs+verity, config)
make bundle      # RAUC update bundle (signed)
make clean       # Remove build artifacts
```

### 8.2 Rootfs Build (make-rootfs.sh)

1. Create minimal Alpine rootfs using `alpine-make-rootfs`
2. Install packages from `build/packages.txt`
3. Copy ScoutOS overlay files (OpenRC scripts, RAUC config, GRUB config)
4. Install scoutd binary to `/usr/sbin/scoutd`
5. Enable services in OpenRC default runlevel
6. Remove unnecessary files (man pages, docs, apk cache)
7. Create mount points for persistent partitions

### 8.3 SquashFS + dm-verity Build (make squashfs)

1. Compress rootfs directory to SquashFS image with zstd compression:
   ```sh
   mksquashfs build/rootfs build/scout-rootfs.sqfs -comp zstd -Xcompression-level 19
   ```
2. Generate dm-verity hash tree and root hash:
   ```sh
   SQFS_SIZE=$(stat -c %s build/scout-rootfs.sqfs)
   # Hash tree is appended after the SquashFS data in the same partition
   veritysetup format build/scout-rootfs.sqfs build/scout-rootfs.sqfs \
       --data-block-size=4096 --hash-offset="$SQFS_SIZE" \
       | tee build/verity-output.txt
   # Extract root hash and hash offset for initramfs embedding
   grep "Root hash:" build/verity-output.txt | awk '{print $3}' > build/roothash.txt
   echo "$SQFS_SIZE" > build/hashoffset.txt
   ```
3. The output is a single file (`scout-rootfs.sqfs`) containing SquashFS data followed by the verity hash tree. This file is written raw to the rootfs-a or rootfs-b partition (640MB, fits comfortably).

The root hash from `build/roothash.txt` is embedded into the initramfs by `make initramfs` (see 8.6). The initramfs is then packed into the SquashFS for the RAUC bundle, creating a dependency chain: `make squashfs` → `make initramfs` → rebuild SquashFS with new initramfs → recalculate verity. In practice, the initramfs is built first with a placeholder hash, then updated after verity generation and repacked.

### 8.4 Alpine Packages

```
# Kernel
linux-lts, linux-firmware-none

# Security
nftables, suricata, suricata-update, unbound, wireguard-tools

# Network
dnsmasq, iproute2, conntrack-tools

# Time
chrony, fake-hwclock

# System
openrc, busybox, alpine-baselayout, musl, e2fsprogs, squashfs-tools, grub-efi

# Reliability
watchdog, jemalloc

# Integrity
cryptsetup

# OTA
rauc

# TLS + Utils
openssl, jq, curl, ca-certificates
```

### 8.5 Disk Image Assembly (make-image.sh)

1. Create sparse image file (2GB -- data partition expands on first boot)
2. GPT partition table (5 partitions per layout in 7.1)
3. Format: ESP (vfat), config (ext4), data (ext4)
4. Write SquashFS + dm-verity hash tree to rootfs-a partition (raw -- SquashFS data followed by verity hash tree at the calculated offset)
5. Install GRUB to ESP, configure A/B boot with ESP kernel extraction (see 7.1)
6. Extract kernel + initramfs (with baked-in root hash) from SquashFS to ESP slot-a directory
7. Write default config.json to config partition
8. Compress image with zstd for distribution

### 8.6 Custom initramfs

Handles dm-verity verification and OverlayFS setup at boot. The dm-verity root hash is embedded in the initramfs at build time by `make initramfs`.

**Build step (`make initramfs`):**
```sh
# After 'make squashfs' produces the root hash:
ROOTHASH=$(cat build/roothash.txt)
HASHOFFSET=$(cat build/hashoffset.txt)

# Substitute root hash and offset into the initramfs init script template
sed "s/@ROOTHASH@/${ROOTHASH}/; s/@HASHOFFSET@/${HASHOFFSET}/" \
    build/initramfs/init.template > build/initramfs/init

# Pack initramfs (cpio + gzip)
cd build/initramfs && find . | cpio -o -H newc | gzip > ../initramfs.img
```

**Boot sequence in initramfs init script:**
```
# dm-verity: verify rootfs integrity before mounting
veritysetup open /dev/nvme0n1pX scout-root /dev/nvme0n1pX "$ROOTHASH" \
    --hash-offset="$HASHOFFSET"
mount SquashFS via dm-verity → read-only /mnt/lower (/dev/mapper/scout-root)
create tmpfs → /mnt/upper (volatile writes)
mount OverlayFS → /mnt/root (lower=squashfs, upper=tmpfs)
mount config partition → /mnt/root/etc/scout (persistent)
mount data partition → /mnt/root/var/lib/scout (persistent, noexec,nosuid,nodev)
switch_root → /sbin/init
```

**Emergency fallback:** If any mount step fails (corrupted SquashFS, missing partition, OverlayFS error), the initramfs drops to a minimal BusyBox emergency shell with a diagnostic message:

```
*** ScoutOS boot failed ***
Error: failed to mount SquashFS from /dev/nvme0n1p2
This may indicate a corrupted firmware image.

Recovery options:
  1. Reboot to try the alternate slot (type: reboot)
  2. Insert USB recovery drive and reboot
  3. Use this shell to inspect partitions

Type 'help' for available commands.
/ #
```

This requires a keyboard + monitor (or serial console) connected to the Odroid H4+. **Serial console:** 115200 baud, 8N1 (`ttyS0`). Kernel boot parameters include `console=tty0 console=ttyS0,115200n8` so kernel messages are available on both HDMI and serial. The emergency shell includes `mount`, `fdisk`, `cat`, and other BusyBox utilities for diagnosis. This is a last-resort debug tool, not a normal user path.

**initramfs maintenance:** The initramfs script is pinned to specific partition UUIDs (set during image build) and tested against the exact Alpine LTS kernel version shipped in each release. The build system generates the initramfs as part of `make rootfs`, so it's always in sync with the kernel. Alpine LTS kernel upgrades (e.g., 6.6.x → 6.12.x) require initramfs re-testing as part of the OTA update QA process.

---

## 9. Project Structure

```
ScoutUTM/
├── GTM.md                         # Go-to-market strategy
├── PLAN.md                        # This document
├── Makefile                       # Top-level build orchestration
│
├── scoutd/                        # Go management daemon
│   ├── go.mod
│   ├── cmd/scoutd/main.go
│   ├── internal/
│   │   ├── api/                   # REST API handlers
│   │   ├── ws/                    # WebSocket hub
│   │   ├── config/                # Config store + schema + validation
│   │   ├── render/                # Template engine for daemon configs
│   │   ├── service/               # Service lifecycle (OpenRC)
│   │   ├── stats/                 # Real-time stats (/proc, conntrack, DNS)
│   │   ├── update/                # OTA client (RAUC)
│   │   ├── auth/                  # Session auth + bcrypt
│   │   └── netmon/                # ARP, DHCP, device fingerprinting
│   ├── templates/                 # Daemon config templates (*.tmpl)
│   └── web/dist/                  # Embedded Vue.js assets
│
├── web/                           # Vue.js 3 frontend source
│   ├── src/
│   │   ├── views/                 # Page components
│   │   ├── components/            # Reusable UI components
│   │   ├── stores/                # Pinia state stores
│   │   ├── api/                   # API client
│   │   └── composables/           # Vue composables (WebSocket, auth)
│   ├── package.json
│   ├── vite.config.ts
│   └── tailwind.config.js
│
├── build/                         # Image build system
│   ├── scripts/                   # make-rootfs.sh, make-image.sh, make-bundle.sh, flash.sh
│   ├── packages.txt               # Alpine packages to install
│   ├── overlay/                   # Files overlaid onto rootfs
│   │   └── etc/
│   │       ├── init.d/scoutd      # OpenRC init script
│   │       ├── rauc/system.conf
│   │       └── grub/grub.cfg
│   ├── defaults/config.json       # Default first-boot config
│   ├── initramfs/init             # Custom initramfs script
│   └── keys/                      # Signing certs (private key NOT committed)
│
├── tests/                         # Integration + QA tests
│   ├── smoke_test.sh              # Boot smoke test
│   ├── network_test.sh            # Connectivity tests
│   ├── api_test.sh                # REST API tests
│   └── qa_test.sh                 # Production QA (runs on every unit)
│
├── Vagrantfile                    # Vagrant VM for Tier 2 development
├── tools/                         # Dev utilities
│   ├── vagrant-provision.sh       # Alpine VM provisioning script
│   └── dev-vm.sh                  # Launch ScoutOS image in QEMU (alternative to Vagrant)
│
└── docs/                          # Documentation
    ├── architecture.md
    ├── api.md
    └── building.md
```

---

## 10. Security Architecture

### 10.1 Read-Only Root

SquashFS is inherently immutable. OverlayFS upper layer is tmpfs (volatile). Even if an attacker gains root, they cannot modify system binaries. Reboot restores pristine state.

### 10.2 Minimal Attack Surface

**Running services on deployed device:**

| Service | Listening | Accessible From |
|---------|----------|-----------------|
| scoutd (web UI + API) | tcp/443, tcp/80 | LAN only |
| Unbound (DNS) | udp/53, tcp/53 | LAN only |
| dnsmasq (DHCP) | udp/67 | LAN only |
| WireGuard | udp/51820 | WAN (if enabled) |
| chronyd (NTP) | udp/123 (client only) | Outbound only (not listening) |
| Suricata | N/A (NFQUEUE) | Not listening |
| nftables | N/A (kernel) | Not listening |

**Not running:** No SSH, no Telnet, no SNMP, no FTP, no cron, no avahi/mDNS.

### 10.3 Secrets Management

| Secret | Storage | Protection |
|--------|---------|------------|
| Admin password | config.json (bcrypt hash) | 0600 file permissions |
| WireGuard private key | config.json | Never exposed via API. `key_created_at` tracked for rotation. |
| TLS private key | /etc/scout/tls/ | 0600 file permissions |
| Session tokens | In-memory only | Lost on reboot, 24h expiry |
| RAUC signing key | Build machine only | Never on device |

**WireGuard key rotation:** The WG private key is generated once during setup wizard (or when VPN is first enabled) and stored with a `key_created_at` timestamp. scoutd shows a dashboard warning if the key is >12 months old: "Your VPN keys were created over a year ago. Consider regenerating them." `POST /api/v1/vpn/wireguard/regenerate-keys` generates a new keypair and invalidates all existing peer configs (peers must re-download their configs). This is manual, not automatic -- automatic rotation would silently break all VPN clients.

### 10.4 WAN Lockdown

nftables drops all WAN input except WireGuard (if enabled) and rate-limited ICMP. Management ports (443, 80, 53, 67) are only accessible from LAN zones.

### 10.5 DNS Bypass Prevention

nftables prerouting redirects all port 53 traffic from LAN to local Unbound. Devices cannot bypass DNS filtering by using hardcoded resolvers (8.8.8.8, 1.1.1.1, etc.).

### 10.6 Persistent Partition Hardening

Config and data partitions mounted with `noexec,nosuid,nodev` options.

### 10.7 Persistent Data Map

Everything outside the two persistent partitions is volatile (tmpfs overlay, lost on reboot). This table is the complete inventory of persistent state:

**Config partition** (`/etc/scout/`, 32MB, ext4, `noexec,nosuid,nodev`):

| Path | Contents | Lifecycle |
|------|----------|-----------|
| `/etc/scout/config.json` | Central configuration (all settings) | Written by scoutd on every config change |
| `/etc/scout/config.json.bak` | Previous config (one-step rollback) | Overwritten on each commit |
| `/etc/scout/config.json.pending` | Staged config during three-phase commit | Transient -- deleted after commit or rollback |
| `/etc/scout/config.json.pre-migrate` | Config snapshot before last schema migration | Overwritten on each OTA migration, used for rollback (see 4.5) |
| `/etc/scout/tls/server.crt` | Self-signed TLS certificate | Generated on first boot, valid 10 years. Regenerate via API if needed. |
| `/etc/scout/tls/server.key` | TLS private key (0600) | Generated with cert. Never leaves device. |
| `/etc/scout/tls/ca.crt` | (Future) Custom CA for client certs | Not used in v1 |

**Data partition** (`/var/lib/scout/`, ~126GB, ext4, `noexec,nosuid,nodev`):

| Path | Contents | Lifecycle | Retention |
|------|----------|-----------|-----------|
| `/var/lib/scout/dnsmasq.leases` | Active DHCP leases | Written by dnsmasq continuously | Automatic (dnsmasq manages) |
| `/var/lib/scout/suricata/rules/` | Downloaded IDS/IPS rulesets | Updated by suricata-update (scheduled by scoutd) | Overwritten on each update |
| `/var/lib/scout/suricata/eve.json` | Suricata alert/flow log | Rotated by scoutd | **7 days**, max 500MB. Oldest files deleted first. |
| `/var/lib/scout/logs/scoutd.log` | scoutd application log | Rotated by scoutd | **7 days**, max 100MB |
| `/var/lib/scout/logs/dns-query.log` | DNS query log (for parental reports) | Rotated by scoutd | **7 days**, max 200MB. See privacy note below. |
| `/var/lib/scout/blocklists/` | DNS blocklist files (RPZ format) | Updated daily by scoutd | Overwritten on each update |
| `/var/lib/scout/update/` | Temporary download location for OTA bundles | Cleaned after install | Deleted after successful or failed install |

**DNS query log privacy:** The dns-query.log records **full domain names** (e.g., `analytics.google.com`, `youtube.com`) per device, but NOT full URLs or paths. This level of detail is necessary for parental controls to be meaningful -- a parent needs to see that a device visited `tiktok.com`, not just `com`. The log entry format: `timestamp | device_mac | device_name | domain | action(allowed/blocked) | category`. No request payload, no response content, no IP addresses of upstream servers. The DNS query log can be disabled entirely via `config.dns.query_logging: false`. When the parental profile for a device is "adult" (no restrictions), queries from that device are not logged.

**Log retention policy:** All logs are managed by scoutd using a simple rotation goroutine. When a log file exceeds its size limit, scoutd rotates it (rename to `.1`, delete `.7`). No external log rotation daemon needed. Total maximum disk usage for logs: ~800MB. On a 126GB data partition this is negligible.

**TLS certificate lifecycle:**
1. **First boot:** scoutd generates a self-signed ECDSA P-256 certificate + key at `/etc/scout/tls/`. Valid for 10 years. Subject: `CN=scout.local`.
2. **Normal operation:** Certificate is served for HTTPS on the web UI. Users see a browser warning (expected for appliance self-signed certs). The setup wizard explains this.
3. **Renewal:** The 10-year validity means renewal is not a concern for the product's expected life. If needed, `POST /api/v1/system/regenerate-cert` generates a new keypair and restarts scoutd's HTTPS listener.
4. **Custom certs:** Future Scout+ feature could allow uploading a custom certificate (e.g., from a local CA or Let's Encrypt via reverse proxy).

**What is NOT persistent (by design):**
- `/tmp` -- tmpfs, used for OTA downloads during install
- `/var/log` (system) -- volatile overlay, syslog goes to ring buffer only
- `/run` -- tmpfs, PID files and sockets
- All binaries and libraries -- read-only SquashFS, restored on every boot

### 10.8 Password Recovery

If the admin password is lost, the device must be recoverable without returning it to the manufacturer. ScoutOS provides a **USB recovery mode** accessed through the GRUB boot menu.

**Recovery flow:**

```
1. User creates a recovery USB stick:
   - Download scout-recovery.img from ScoutUTM.com
   - Flash to USB drive (dd or Rufus)

2. Boot the ScoutUTM device with USB inserted:
   - Connect keyboard + monitor to Odroid H4+
   - Power on, press ESC/F2 during UEFI to select boot device
   - Select USB drive

3. Recovery environment boots (minimal Alpine):
   - Mounts the config partition (/dev/nvme0n1p4)
   - Presents a menu:
     a) Reset admin password (set new password)
     b) Factory reset (wipe config.json, restore defaults)
     c) Export config (copy config.json to USB for backup)
     d) Import config (restore config.json from USB)
     e) Exit

4. "Reset admin password" option:
   - Prompts for new password (with confirmation)
   - Reads config.json, updates admin_hash with new bcrypt hash
   - Writes config.json back atomically
   - User removes USB, reboots, logs in with new password
```

**Why USB recovery instead of a hardware reset button:**
- Odroid H4+ has no user-accessible reset button on its case
- A reset button could be accidentally pressed (especially by children)
- USB recovery is more secure (requires physical access + the recovery image)
- Allows multiple recovery operations (not just password reset)

**The recovery image is published on ScoutUTM.com** and documented in the quick-start guide. It does not contain any secrets -- it's a minimal Alpine environment with a shell script that mounts the config partition and offers the menu.

**Factory reset** wipes config.json and replaces it with the default first-boot config. On next boot, the setup wizard runs again. All persistent data on the data partition (logs, leases, rules) is optionally preserved or wiped (user chooses).

### 10.9 Hardware Watchdog and Process Priority Tuning

A production appliance must recover from system hangs without human intervention. ScoutOS enables the hardware watchdog and tunes critical process priorities.

**Hardware watchdog:** The Intel N97 SoC includes an iTCO watchdog timer. ScoutOS configures it via the `watchdog` daemon (Alpine package: `apk add watchdog`):

```
# /etc/watchdog.conf
watchdog-device = /dev/watchdog
watchdog-timeout = 60
interval = 15

# Process health checks -- reboot if critical services die
pidfile = /var/run/scoutd.pid
pidfile = /var/run/suricata.pid
```

- The watchdog daemon kicks the hardware timer every 15 seconds
- If the system hangs (kernel panic, deadlock, OOM freeze), the timer expires after 60 seconds and the hardware forces a cold reboot
- Process monitoring: if scoutd or Suricata stop running, the watchdog daemon triggers a reboot
- The watchdog OpenRC service starts in the `boot` runlevel (before scoutd)

**Suricata thread priority tuning:** Suricata's packet processing threads are latency-sensitive -- they hold NFQUEUE verdicts and delay packets while inspecting them. To prevent other processes (log rotation, suricata-update, background tasks) from starving Suricata:

```sh
# In /etc/init.d/suricata (OpenRC start function), after starting Suricata:
# Wait for Suricata to finish rule loading and spawn all threads.
# Suricata spawns threads asynchronously during startup -- applying priorities
# too early misses late-spawning packet processing threads.
sleep 5
# Set all Suricata threads to SCHED_FIFO priority 50
for tid in $(ls /proc/$(cat /var/run/suricata.pid)/task/); do
    chrt -f -p 50 "$tid" 2>/dev/null || true
done
# Re-apply after 10 more seconds to catch any late-spawning threads
(sleep 10 && for tid in $(ls /proc/$(cat /var/run/suricata.pid)/task/); do
    chrt -f -p 50 "$tid" 2>/dev/null || true
done) &
```

This gives Suricata's threads real-time scheduling priority, ensuring they preempt normal processes. `SCHED_FIFO` priority 50 is above default processes (0) but below kernel threads and the watchdog. scoutd itself runs at normal priority -- it's not latency-sensitive (API responses of 10-50ms are fine).

**Month 5 benchmark note:** Thread priority tuning is tested as part of the Suricata VM benchmark. The benchmark compares throughput with and without `SCHED_FIFO` to quantify the benefit.

### 10.10 Integrity and Secure Boot Roadmap

ScoutOS takes a layered approach to system integrity, adding protections incrementally across releases. This roadmap is explicit so that customers, reviewers, and the team know exactly what is shipped and what is planned.

**v1.0: Signed OTA + dm-verity + read-only rootfs**

v1.0 ships with three integrity mechanisms that do not require UEFI Secure Boot:

1. **Signed OTA bundles.** RAUC verifies ECDSA P-256 signatures on every firmware update. Only ScoutUTM-signed images can be installed. Downgrade protection via version monotonicity.

2. **dm-verity for rootfs.** The SquashFS rootfs partition is protected by dm-verity -- a Linux kernel feature that computes a Merkle tree hash of every 4KB block and verifies them on read. If any block is tampered with (bit flip, deliberate modification), the kernel returns an I/O error instead of corrupted data.

   dm-verity integration:
   - The build system (`make squashfs`) generates the SquashFS image, then runs `veritysetup format` to produce the hash tree and root hash
   - The root hash is embedded in the initramfs (which is inside the signed RAUC bundle)
   - At boot, the initramfs runs `veritysetup open` before mounting the SquashFS, passing the root hash
   - The kernel verifies every block read from the SquashFS against the hash tree
   - If verification fails, the kernel returns EIO -- the block cannot be read, and the system logs the corruption

   ```sh
   # In initramfs init script, before mounting SquashFS:
   ROOTHASH="<baked-in-at-build-time>"
   veritysetup open /dev/nvme0n1p2 scout-root /dev/nvme0n1p2 "$ROOTHASH" \
       --hash-offset=<offset>
   mount -t squashfs -o ro /dev/mapper/scout-root /mnt/lower
   ```

   dm-verity adds negligible performance overhead (hashes are cached after first read) and requires no hardware support. It catches both accidental corruption (bad blocks) and deliberate tampering.

3. **Read-only SquashFS + noexec persistent partitions.** SquashFS is inherently immutable. Config and data partitions are mounted `noexec,nosuid,nodev`. tmpfs overlay is volatile.

**v1.0 does NOT include UEFI Secure Boot.** The boot chain (UEFI → GRUB → kernel) is not cryptographically verified at boot time. An attacker with physical access could theoretically replace GRUB or the kernel on the ESP. However:
- This requires physical access to the NVMe SSD (opening the case, removing the drive)
- Once the kernel boots, dm-verity protects the rootfs from tampering
- OTA updates are still signed -- a compromised bootloader cannot install unsigned firmware via the normal update path

**v1.5: Full UEFI Secure Boot chain**

Full chain-of-trust from firmware to kernel:

1. **Custom shim** signed by ScoutUTM's key, enrolled via MOK (Machine Owner Key)
2. **Signed GRUB EFI binary** verified by shim
3. **Signed Linux kernel** verified by GRUB
4. **Kernel module signing** -- only signed modules can be loaded (`CONFIG_MODULE_SIG_FORCE=y`)

**The challenge:** Alpine Linux does not ship a signed shim or signed GRUB. Implementing Secure Boot on Alpine requires building and maintaining a custom signing pipeline. Every kernel update requires re-signing. This is achievable but represents ongoing maintenance burden.

**Why this order is correct:**
- dm-verity (v1.0) protects the most valuable target -- the rootfs containing all binaries and configuration templates. It works today, with zero hardware requirements.
- Secure Boot (v1.5) closes the remaining gap -- the boot chain. It requires signing infrastructure, key management, and Alpine-specific workarounds that are better tackled after v1.0 stabilizes.
- Together (v1.5), they provide a complete chain: UEFI verifies GRUB → GRUB verifies kernel → kernel + dm-verity verifies rootfs → RAUC verifies OTA updates.

### 10.11 TPM 2.0 (Deferred to v1.5)

**Status: Not implemented in v1.0, but feasible on Alpine.**

The Odroid H4+ does not include a TPM chip on-board (an external TPM module would need to be added via header). Alpine supports TPM 2.0 via `tpm2-tools` and `tpm2-tss` packages, and the kernel includes TPM drivers (`tpm_tis`, `tpm_crb`).

**Potential uses for ScoutOS:**
- **Entropy source:** Feed the TPM's hardware RNG into the system entropy pool via `rngd`, improving cryptographic key generation quality
- **Key sealing:** Seal the WireGuard private key or TLS private key to the TPM, so they cannot be extracted even with physical disk access
- **Measured boot:** Record boot measurements (PCR values) to detect firmware tampering

**Why deferred:** TPM adds hardware cost, requires a TPM module on the Odroid H4+, and the key sealing integration is non-trivial. The v1.0 security model (read-only rootfs, signed OTA, noexec partitions) provides strong protection without TPM. TPM will be evaluated for v1.5 alongside Secure Boot, as the two features complement each other (Secure Boot verifies the boot chain, TPM measures it).

### 10.12 SBOM and GPL Compliance

ScoutOS ships GPL-licensed software (Linux kernel, Suricata, nftables, Unbound, dnsmasq, WireGuard tools, GRUB, BusyBox, chrony). Compliance with open-source licenses and emerging regulatory requirements is mandatory.

**GPL source distribution:** GPLv2 (Linux kernel, Suricata, BusyBox) requires that corresponding source code be made available to anyone who receives the binary. ScoutOS complies via:

1. **Source archive on ScoutUTM.com:** For each firmware release, publish a tarball of all GPL source packages at their exact versions (e.g., `scout-1.0.0-sources.tar.gz`). This includes the Alpine APK source packages (`apk fetch --stdout`) and any patches applied during the build.
2. **Written offer in documentation:** The quick-start guide and `GET /api/v1/system/about` include a notice: "ScoutOS contains open-source software. Source code is available at https://scoututm.com/opensource".
3. **Build reproducibility:** The Docker-based build (section 11.5) pins Alpine package versions. The exact package list and versions are recorded in the build output for traceability.

**Bootloader unlocking:** ScoutOS uses GPLv2-licensed software (Linux kernel, GRUB). GPLv2 does not require providing "installation information" (that's a GPLv3 requirement -- "anti-tivoization"). The device is not legally required to allow custom firmware installation. However, the USB recovery mode (section 10.8) provides physical-access recovery, and the build system is documented so that technically capable users can build their own images.

**SBOM generation:** Every firmware build generates a Software Bill of Materials in CycloneDX format:

```makefile
# In Makefile, as part of 'make bundle'
make sbom:
	syft /path/to/rootfs -o cyclonedx-json > build/scout-$(VERSION)-sbom.json
```

- **Tool:** `syft` (open-source, by Anchore) scans the rootfs and produces a CycloneDX or SPDX JSON listing every package, version, and license
- **When generated:** Automatically during `make bundle`, included alongside the RAUC bundle as a build artifact
- **Published:** SBOM is published on ScoutUTM.com alongside each firmware release and included in the `GET /api/v1/system/about` response as a download link
- **Why this matters:** EU Cyber Resilience Act (CRA) and U.S. Executive Order 14028 increasingly require SBOMs for security appliances sold to government and enterprise. Having SBOM generation from day one avoids retroactive compliance work

**CI integration:** GitHub Actions build pipeline includes:
1. `syft` scan → CycloneDX JSON
2. `grype` vulnerability scan against the SBOM (warns on known CVEs in shipped packages)
3. License check (flag any non-OSI-approved or copyleft-incompatible licenses)
4. SBOM artifact uploaded to GitHub Releases alongside the firmware bundle

---

## 11. Development Workflow

Development is organized in **three tiers**, from fastest/lightest to slowest/most realistic. Most daily work happens in Tier 1 and 2 -- hardware is only needed for NIC-specific testing and final QA.

| Tier | Tool | Iteration | What It Tests | % of Daily Work |
|------|------|-----------|---------------|-----------------|
| **1. Host-native** | `go run` / `go test` / `npm run dev` | **~1-2s** | API, config, templates, WebSocket, frontend | ~60-70% |
| **2. Vagrant VM** | `vagrant up` / `vagrant provision` | **~30s** reload | nftables, DNS, IDS/IPS, DHCP, VPN, VLANs, OTA, full boot | ~25-35% |
| **3. Hardware** | Odroid H4+ (flash + boot) | **~10min** | Intel I226-V driver, 2.5GbE throughput, thermal, production QA | ~5-10% |

### 11.1 Tier 1: Host-Native Development (No VM, No Hardware)

For the majority of daily coding. scoutd is a standard Go program -- most of its logic (config parsing, template rendering, API handlers, WebSocket, auth) can be developed and tested on the workstation with no VM or hardware.

**Go backend:**
```
1. Edit Go code
2. go test ./...                    # Run unit tests (~2s)
3. go run ./cmd/scoutd --dev-mode   # Start scoutd locally (~1s)
4. Test API with curl or browser at https://localhost:8443
```

`--dev-mode` flag: scoutd starts its HTTP server and config management but skips hardware-dependent operations (nftables apply, service orchestration, stats collection from /proc/net). It loads config from a local file and serves the API + frontend. Template rendering still works -- it just doesn't execute `nft -f` or `rc-service`. This lets you test the entire control plane without root or Linux-specific features.

**Vue.js frontend:**
```
cd web && npm run dev               # Vite dev server with HMR (~200ms reload)
```
Vite proxies `/api/*` to the local scoutd instance. Hot module replacement means UI changes appear instantly.

**What this tier covers:**
- All REST API handler logic (request parsing, validation, response formatting)
- Config schema validation and migration
- Template rendering (generate nftables.conf, suricata.yaml, etc. from config -- verify output in tests)
- WebSocket hub logic
- Auth (login, sessions, bcrypt)
- Setup wizard flow
- All Vue.js UI development
- Unit tests for all of the above

**What this tier cannot test:**
- Actual nftables rule loading (needs Linux kernel)
- Suricata, Unbound, dnsmasq, WireGuard (need the daemons running)
- Network traffic flow (needs virtual or physical NICs)
- Boot process, OverlayFS, OTA updates

### 11.2 Tier 2: Vagrant VM (Full ScoutOS in a VM)

For integration testing. A Vagrant-managed VM runs the complete ScoutOS environment with all daemons, virtual NICs, and the full network stack. No Odroid hardware needed.

**Setup:**
```
.\build.ps1         # PowerShell: builds via WSL, provisions VM, runs integration tests (~3-5 min)
# OR:
./build.sh          # WSL: same as above
# OR step-by-step:
vagrant up          # First run: creates VM, provisions Alpine, starts all services, runs tests
vagrant ssh         # Shell access for debugging (dev/test only, not in production images)
```

**Iterating on scoutd inside the VM:**
```
.\build.ps1                           # Full rebuild + reprovision + tests (~3-5 min)
# OR: just reprovision (if binary already built):
vagrant provision                     # Reinstalls binary, restarts, runs tests (~1-2 min)
```

**Vagrantfile:**
```ruby
Vagrant.configure("2") do |config|
  config.vm.box = "generic/alpine319"
  config.vm.hostname = "scoututm-dev"

  config.vm.provider "virtualbox" do |vb|
    vb.cpus = 4
    vb.memory = 8048
    vb.name = "scoututm-dev"
  end

  # Port forwards -- map host ports to VM ports.
  config.vm.network "forwarded_port", guest: 443, host: 8443, host_ip: "127.0.0.1"
  config.vm.network "forwarded_port", guest: 80,  host: 8080, host_ip: "127.0.0.1"

  # Private network for simulated LAN.
  config.vm.network "private_network", ip: "192.168.56.10"

  # Sync build artifacts and tests into the VM.
  config.vm.synced_folder "build/", "/opt/scout-build"
  config.vm.synced_folder "tests/", "/opt/scout-tests"

  # Provision with ScoutOS packages + run integration tests.
  config.vm.provision "shell", path: "tools/vagrant-provision.sh"
end
```

**Implementation notes vs. original plan:**
- Box changed from `generic/alpine321` (unavailable) to `generic/alpine319`.
- Synced folder uses default VirtualBox shared folders (rsync doesn't work on Windows/WSL2).
- VM resources increased to 4 CPUs / 8GB RAM for faster provisioning.
- scoutd runs in `--dev-mode` to allow SSH/HTTPS on WAN (Vagrant's eth0). Templates still render to real paths (`--output-dir /etc`).
- Integration tests run automatically at the end of provisioning. Tests that check nftables state reload rules manually with `nft -f` since dev-mode stubs service reloads.

**What this tier covers:**
- Full nftables firewall (rule loading, zone policies, NAT, DNS redirect)
- Unbound DNS resolution (RPZ blocklists disabled in dev VM -- no real blocklist data)
- dnsmasq DHCP (VM's host-only network acts as LAN)
- WireGuard VPN config rendering and API management
- chrony NTP sync
- Template rendering to real daemon config paths
- Integration test suite (5 active suites, 71 checks, run automatically on provision)

**What this tier cannot test:**
- Intel I226-V NIC driver behavior
- Real 2.5GbE line-rate throughput
- Hardware-specific UEFI boot
- Thermal performance under load
- Physical assembly and packaging QA

**Multi-VM testing (optional, for advanced scenarios):**
```ruby
# In Vagrantfile, add a LAN client VM:
config.vm.define "client" do |client|
  client.vm.box = "generic/alpine321"
  client.vm.hostname = "lan-client"
  client.vm.network "private_network", ip: "192.168.56.10",
    virtualbox__intnet: "scout-lan"
  client.vm.provider "virtualbox" do |vb|
    vb.memory = 512
    vb.cpus = 1
  end
end
```
This creates a second VM on the same host-only network, acting as a LAN device. Useful for testing DHCP lease assignment, DNS filtering from a client perspective, and parental controls.

### 11.3 Tier 3: Hardware (Odroid H4+)

**Hardware is deferred until the software PoC is validated in VMs.** The vast majority of ScoutOS features (nftables, Suricata, Unbound, WireGuard, scoutd, the web UI) are not hardware-specific. The entire stack runs identically on any x86_64 Linux -- Alpine in a VM is functionally equivalent to Alpine on an Odroid H4+ for software validation. If needed, the solution can be ported to different hardware with minimal changes (different NIC driver, different partition device paths).

**When to purchase hardware: Month 9-10** (after Beta gate at Month 8). By then, all features are working in VMs and the image build pipeline is ready. Hardware testing validates the last 5-10% -- NIC drivers, real throughput, UEFI boot, thermals.

**Hardware is only needed for:**
- Intel I226-V NIC driver compatibility with Alpine kernel
- Real 2.5GbE line-rate throughput benchmarks (IDS/IPS, VPN, NAT)
- UEFI boot on actual Odroid H4+ firmware
- Thermal behavior under sustained load
- Production QA on assembled units before shipping

**Hardware setup (Month 10+):**
- **One Odroid H4+** board for development + testing (Alpine in read-write mode, SSH enabled)
- **One Odroid H4+** board for production image testing only needed closer to launch (Month 11-12)

**Workflow:**
```
1. make image                    # Full disk image (~3-5 min)
2. Flash to NVMe (via USB adapter or from running dev board)
3. Boot, run tests/qa_test.sh    # Full production QA (~2 min)
```

### 11.4 When to Use Which Tier

| Task | Tier |
|------|------|
| Writing a new API endpoint | 1 (host-native) |
| Fixing a Vue.js component | 1 (host-native) |
| Debugging config validation | 1 (host-native, `go test`) |
| Testing nftables template output correctness | 1 (unit test renders template, asserts output) |
| Testing that nftables rules actually load | 2 (Vagrant) |
| Testing DNS filtering blocks a domain | 2 (Vagrant, or multi-VM) |
| Testing DHCP serves leases to a client | 2 (Vagrant multi-VM) |
| Testing Suricata alerts on malicious traffic | 2 (Vagrant) |
| Testing WireGuard tunnel establishes | 2 (Vagrant) |
| Testing OTA update installs and reboots | 2 (Vagrant) |
| Testing setup wizard end-to-end | 2 (Vagrant) |
| Benchmarking IDS/IPS throughput (VM baseline) | 2 (Vagrant, Month 5) |
| Benchmarking IDS/IPS throughput at 2.5Gbps | 3 (hardware, Month 10+) |
| Validating Intel I226-V driver on Alpine kernel | 3 (hardware, Month 10+) |
| Running QA script before shipping a unit | 3 (hardware, Month 10+) |

### 11.5 Build Dependencies

- Go 1.22+, Node.js 20+
- Vagrant 2.4+ with VirtualBox or libvirt provider
- `alpine-make-rootfs`, `mksquashfs`, `sgdisk`, `grub-install`, `rauc`, `openssl`
- QEMU + OVMF (used by libvirt provider and for direct image testing)

Reproducible build via Docker (pin the digest for reproducibility):
```dockerfile
FROM alpine:3.21
# Pin with: docker pull alpine:3.21 && docker inspect --format='{{index .RepoDigests 0}}' alpine:3.21
# Then use: FROM alpine:3.21@sha256:<digest>
RUN apk add --no-cache go nodejs npm alpine-make-rootfs \
    squashfs-tools sgdisk grub-efi e2fsprogs rauc openssl make bash curl
```

**Alpine 3.21 lifecycle:** Alpine 3.21 was released ~November 2025 and is supported until approximately November 2027 (~2 years). ScoutOS must track Alpine releases and migrate before EOL. Plan: Alpine 3.21 for v1.0 (2026), migrate to Alpine 3.23 or latest LTS before 3.21 EOL. Each Alpine migration is an OTA update that ships a new SquashFS built against the new Alpine version -- the A/B update system makes this safe.

---

## 12. Testing Strategy

### 12.1 Unit Tests (Go)

- Config validation logic
- Template rendering (config → nftables.conf, verify output)
- API handlers (httptest)
- Config migration logic
- Run: `cd scoutd && go test ./...`

### 12.2 Integration Tests (shell scripts, run in Vagrant VM)

All integration tests run automatically at the end of `vagrant up` / `vagrant provision`. The provisioning script iterates over `tests/*_test.sh`, skipping empty placeholder scripts, and reports pass/fail for each suite. These run at every milestone gate (Alpha M4, Beta M8, RC M10).

| Script | What It Tests | When Added | Status (Month 4) |
|--------|--------------|------------|-------------------|
| `smoke_test.sh` | Boot, scoutd API responds, services running, config.json valid, TLS cert | Month 1 | **10 checks, passing** |
| `network_test.sh` | nftables rules loaded (zone sets, IPv6 drop, DNS redirect, masquerade), dnsmasq DHCP, chrony | Month 2 | **9 checks, passing** |
| `api_test.sh` | REST endpoints: CRUD operations, auth, validation errors, protected zones | Month 2 (grows each month) | **12 checks, passing** |
| `rollback_test.sh` | Config apply failures trigger rollback, startup recovery from pending file, rapid save serialization | Month 5 | Placeholder (deferred -- requires multi-daemon rollback scenarios from Month 5) |
| `dns_test.sh` | Unbound recursive resolution, DNSSEC validation, RPZ blocklist blocks test domains, custom allow/block | Month 5 | Placeholder (deferred -- requires real blocklist data and Suricata integration context) |
| `vlan_test.sh` | VLAN creation, zone isolation (IoT has separate input chain), nftables zone sets with VLAN sub-interfaces, dnsmasq DHCP per VLAN, DNS redirect per zone, chrony allows VLAN subnets | Month 4 | **14 checks, passing** |
| `vpn_test.sh` | WireGuard enable/disable, key generation, nftables VPN rules (WAN port, input_vpn chain, forwarding, DNS redirect), peer CRUD, tunnel IP allocation, split/full tunnel, client config (LAN DNS, server info), key regeneration, QR codes | Month 4 | **26 checks, passing** |
| `ids_test.sh` | Suricata alert on test signature, eve.json socket delivery, fail-open when Suricata stopped | Month 5 | Placeholder |
| `stress_test.sh` | nmap scan through the firewall (triggers alert storm), verify WebSocket rate limiting, verify no OOM or crash | Month 7 | Placeholder |
| `update_test.sh` | RAUC bundle install, reboot, version check, health check pass, deliberate bad bundle → rollback | Month 9 | Placeholder |
| `wizard_test.sh` | Setup wizard end-to-end: all 7 steps, config generated, services configured | Month 7 | Placeholder |

**Month 4 Alpha gate totals:** 170 unit tests (Go) + 71 integration checks (5 suites) across 5 active test scripts.

**Test automation strategy:** Each test script is self-contained and idempotent -- it sets up its preconditions, runs assertions, and cleans up. Tests use `curl` for API assertions and standard CLI tools (`nft`, `grep`, `jq`) for config and nftables verification. Since scoutd runs in `--dev-mode` in the VM (needed for SSH access), tests that verify nftables state manually reload rules via `nft -f /etc/nftables.conf` after API changes.

### 12.3 Production QA (every unit before shipping)

`tests/qa_test.sh` runs on real hardware (Month 10+) and checks:
- All services running (nftables, suricata, unbound, dnsmasq, chronyd, scoutd)
- API responds on HTTPS
- WAN link up, WAN IP assigned
- DNS resolves (external domain via Unbound)
- NTP synchronized (chronyc tracking shows valid source)
- nftables rules loaded (correct chain count, zone sets populated)
- Suricata running in IDS mode, eve socket connected
- Memory usage <2GB at idle (of 8GB total)
- Boot time <15s (measured from GRUB to scoutd API responding)
- Both NIC LEDs active (Intel I226-V link up)

---

## 13. Development Timeline (12 Months)

The original 8-month estimate assumed no setbacks, no learning curves, and full-time dedicated focus. Industry experience says solo embedded projects take 2-3x initial estimates. This 12-month plan builds in realistic buffer while defining hard milestone gates.

**Hardware purchase is deferred until Month 9.** Months 1-8 use only Tier 1 (host-native) and Tier 2 (Vagrant VM) development. All features are validated in VMs before any hardware spend. This preserves budget, avoids premature hardware coupling, and proves the software PoC independently.

| Month | Focus | Key Deliverables | Milestone |
|-------|-------|-----------------|-----------|
| **1** | Foundation + Dev environment | scoutd skeleton (main.go, config schema, config store), Vagrant VM dev environment with Alpine, basic project structure | |
| **2** | Firewall core | nftables template rendering, two-zone firewall (WAN/LAN) with NAT, masquerade, dnsmasq DHCP, firewall verified in Vagrant VM | |
| **3** | DNS + Config | Unbound integration (recursive + DNSSEC + RPZ blocklists), DNS filtering (malware, ads), two-phase config commit, atomic config save/load | |
| **4** | Zones + VPN | VLAN creation + zone assignment (IoT, Guest, Kids), WireGuard VPN server + peer management + QR codes, chrony NTP | **Gate: Alpha -- firewall + DNS + VPN working in Vagrant VM** |
| **5** | IDS/IPS | Suricata NFQUEUE integration (queues, fail-open, eve socket), rule management (suricata-update), IDS alert pipeline to scoutd, VM-based Suricata benchmark | |
| **6** | Frontend (Dashboard) | Vue.js project scaffold, API client, Dashboard view, Firewall view, DNS view, VPN view, real-time WebSocket stats | |
| **7** | Frontend (Remaining) | Setup wizard (7 steps), Devices view, IDS alerts view, Parental controls, Network/System settings views | |
| **8** | Parental + Polish | Per-device parental profiles, schedules, safe search enforcement, DNS query logging, device fingerprinting | **Gate: Beta -- all features functional in Vagrant VM** |
| **9** | Image build + OTA | SquashFS rootfs build, disk image assembly, GRUB A/B slot boot, custom initramfs with OverlayFS, ESP kernel extraction, RAUC integration, bundle signing, OTA update flow, downgrade protection -- all tested in QEMU VM first | |
| **10** | Hardware bring-up | **Purchase Odroid H4+.** Flash production image, Alpine kernel + Intel I226-V NIC validation, Suricata hardware throughput benchmarks (iperf3), UEFI boot validation, USB recovery mode testing. **Submit finished product (board + enclosure + firmware) for CE pre-compliance testing at EU test lab (~€3K).** | **Gate: Release Candidate -- boots and passes QA on hardware** |
| **11** | Hardware QA + Hardening + CE | Full QA test suite on hardware, boot time optimization, thermal testing under sustained load, bug fixing, edge case hardening, performance tuning. **CE pre-compliance results received -- must pass before first sale.** | **Gate: CE pre-compliance pass** |
| **12** | Ship prep | Quick-start guide, production image pipeline, second Odroid H4+ for production image validation, order first sales batch (25 units), assemble + flash + test + package | **Gate: v1.0 -- ready for beta testers** |

**Key design principles:**
- **No hardware until Month 10.** Months 1-8 are pure software development on the workstation and in VMs. The Alpha gate (Month 4) and Beta gate (Month 8) are both validated entirely in Vagrant VMs. This keeps the focus on building features, not debugging hardware.
- **Image build tested in QEMU first (Month 9).** The SquashFS build, GRUB A/B boot, initramfs, and RAUC OTA flow can all be validated in a QEMU VM with virtual disks before touching real hardware. QEMU + OVMF provides a real UEFI boot environment.
- **Hardware spend is deferred.** The Odroid H4+ (~€200-300 with NVMe, RAM, case, power supply) isn't purchased until Month 9-10 when the software PoC is proven. If the project pivots to different hardware (e.g., different SBC, different NIC count), no money was wasted.
- **Frontend gets 2 months** (6-7) instead of being crammed into the same sprint as IDS/IPS.
- **Buffer is built in.** If Month 3 runs late, Month 4's gate catches it. The alpha gate at Month 4 is the first real checkpoint.
- **Hardware portability.** The architecture is hardware-agnostic by design. nftables, Suricata, Unbound, WireGuard, and scoutd work on any x86_64 Linux. Hardware-specific items are isolated to: NIC driver (kernel module), partition device paths (config), and UEFI boot (GRUB config). Porting to a different x86 board is a build-config change, not an architecture change.

---

## 14. Explicit v1 Scope Limitations

These features are intentionally excluded from v1.0 to keep scope manageable for a solo founder. Each is documented here so the omission is a conscious decision, not an oversight.

| Feature | v1 Status | Rationale | Target |
|---------|-----------|-----------|--------|
| **IPv6** | **Out of scope** | IPv4-only for v1. All IPv6 traffic is explicitly dropped at the nftables prerouting hook (`meta nfproto ipv6 drop`) to prevent unfiltered IPv6 bypass on ISPs that assign IPv6. **Customers on IPv6-only or DS-Lite ISPs (common in Germany/France) cannot use ScoutUTM v1 — "IPv4 internet connection" is a system requirement.** IPv6 support (SLAAC, DHCPv6, dual-stack firewall rules, IPv6 DNS) adds significant complexity to nftables templates, the zone model, and the setup wizard. | v2.0 |
| **Multi-WAN** | **Out of scope** | Single WAN (eth0) only. Multi-WAN failover and load balancing requires WAN health checking, policy routing, connection tracking migration, and complex nftables rules. The Odroid H4+ only has 2 NICs anyway -- multi-WAN would require a USB NIC. | v2.0 |
| **Scheduled config backup** | **Out of scope** | No automatic backup to cloud/USB. Manual export via USB recovery mode and `GET /api/v1/system/backup` (returns config.json download) are the v1 options. Scheduled cloud sync adds cloud infrastructure dependency. | v1.5 |
| **Mobile app** | **Out of scope** | Web UI is responsive/mobile-friendly via Tailwind CSS. Native iOS/Android apps deferred -- the web UI accessed via phone browser is the v1 mobile experience. | v2.0+ (Phase 4) |
| **Custom DNS upstream per zone** | **Out of scope** | Single Unbound config serves all zones. Per-zone DNS upstreams (e.g., Kids zone uses family-safe resolvers) adds Unbound configuration complexity. Parental controls use RPZ blocklists instead, which is simpler and more effective. | v1.5 if needed |
| **DoH bypass of DNS filtering** | **Known limitation** | DNS-over-HTTPS on port 443 cannot be blocked by port. Partial mitigation: well-known DoH provider domains are included in parental RPZ blocklists, catching common browser defaults. Direct-to-IP DoH or obscure providers bypass filtering. Full DoH blocking requires TLS inspection (MITM proxy) -- out of scope, significant privacy concerns. Industry-wide limitation for all DNS-based filtering products. | v2.0 if needed |
| **Rotating MAC address resistance** | **Known limitation** | Devices with rotating MAC randomization (iOS "Rotate Wi-Fi Address") break all MAC-based identification including parental controls and static DHCP. Per-network stable random MACs (the default on iOS/Android/Windows) work fine. Dashboard warns users to disable rotation for managed devices. Industry-wide limitation -- no MAC-independent solution without 802.1X or captive portal auth. | v2.0 if needed |
| **UEFI Secure Boot** | **Out of scope** | Alpine lacks signed shim/GRUB out of the box. Implementing a full chain-of-trust (shim → signed GRUB → signed kernel) requires custom key enrollment and ongoing signing infrastructure. v1.0 security relies on read-only rootfs, signed OTA bundles, and noexec partitions. See section 10.10. | v1.5 |
| **TPM 2.0 integration** | **Out of scope** | Odroid H4+ lacks onboard TPM. Alpine supports tpm2-tools, so integration is feasible when TPM hardware is available. Useful for key sealing and measured boot. See section 10.11. | v1.5 |
| **DPDK acceleration** | **Out of scope** | DPDK is supported on Alpine/musl since v21.05, but adds complexity (hugepages, poll-mode drivers, dedicated cores) not needed at v1 throughput targets. Could be revisited if IPS throughput needs exceed what kernel nftables + NFQUEUE delivers. | v2.0 if needed |
| **Intel QAT acceleration** | **Out of scope** | Intel QuickAssist crypto offload for VPN/TLS. QAT's user-space libraries do not build cleanly on musl (glibc-specific macros). Would require patching Intel's libraries or using gcompat. Not needed at v1 VPN throughput targets -- WireGuard's kernel implementation saturates 2.5GbE without hardware offload. | v2.0 if needed |
| **Internationalization (i18n)** | **Out of scope** | v1 is English-only. No vue-i18n or translation framework. Localization deferred until sales data validates demand for specific languages. | Phase 2+ |

### System Logging Destination

ScoutOS uses **BusyBox syslogd** (Alpine's default) for system-level logging:
- Logs to a **64KB circular ring buffer** in memory (`/dev/log` → syslog → `logread`)
- **No persistent system logs.** System logs (kernel, OpenRC, daemon stderr) are volatile -- lost on reboot. This is by design: the read-only rootfs has no writable `/var/log`, and persistent logs are handled at the application level (scoutd.log, eve.json, dns-query.log on the data partition).
- `logread` provides access to the current ring buffer for live debugging
- `logread -f` streams new entries in real-time (useful during development)

For operational troubleshooting after a reboot, the relevant data is in the persistent logs on the data partition (scoutd.log, Suricata eve.json). If a boot failure occurs and the system never reaches scoutd, the serial console output (kernel messages) is the diagnostic path.

### Config Backup/Restore (v1 minimal)

While scheduled backup is out of scope, v1 provides basic mechanisms:
- `GET /api/v1/system/backup` -- downloads `config.json` as a file (UI: "Download Backup"). **Note:** The backup includes the WireGuard private key so that restore fully works (a backup without the key would break all VPN peers on restore). The UI warns: "This backup file contains sensitive data including your VPN private key. Store it securely."
- `POST /api/v1/system/restore` -- uploads a `config.json`, validates it, applies via three-phase commit (UI: "Restore Backup")
- USB recovery mode offers export/import (see Section 10.8)
- Every config change creates `config.json.bak` (one-step rollback)

Users are advised in the quick-start guide to download a backup after initial setup and after any major config changes.

---

## 15. Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Suricata RAM usage on 8GB device | Medium | High | Limit thread count, `detect.profile: low`, monitor RSS. VM benchmark in Month 5, hardware benchmark in Month 10. |
| Suricata IPS throughput below expectations | Medium | Medium | VM benchmark in Month 5 catches architectural issues early. Hardware benchmark in Month 10 gives real numbers. Fallback: IDS-only default, IPS opt-in. |
| Boot time exceeds 15s | Low | Medium | Alpine is inherently fast; test in QEMU (Month 9) and hardware (Month 10); profile with bootchart if needed |
| RAUC + Alpine integration issues | Medium | Medium | Test in QEMU VM in Month 9; fallback to custom A/B script |
| RAUC post-install hook fails (ESP kernel extraction) | Low | High | Hook is simple (mount + cp + umount). Validate extracted files with checksum before marking install complete. Test in QEMU first. |
| SquashFS + OverlayFS + persistent mounts complexity | Medium | High | Test in Vagrant VM early (Month 2). Pinewall project proves this pattern works on Alpine. |
| Intel I226-V driver issues on Alpine kernel | Low | High | Intel igc driver is mainline Linux. Tested in Month 10 when hardware is purchased. If driver fails, alternative x86 boards with Intel NICs exist. |
| Late hardware surprise blocks launch | Low | High | Architecture is hardware-agnostic. If Odroid H4+ has issues (discontinued, driver problems), the software runs on any x86_64 board with 2 NICs. Hardware-specific items are isolated to build config. |
| Vue.js bundle too large | Low | Low | Tailwind purge + code splitting |
| Alpine LTS kernel upgrade breaks initramfs | Low | High | Pin kernel version per release; test initramfs as part of OTA QA |
| Config apply race window causes transient issues | Low | Low | Write-all-then-reload minimizes window to <3s. Dependency-ordered reloads ensure firewall updates before dependent services. Acceptable for a home appliance with infrequent config changes. |
| 12-month timeline slips | Medium | Medium | Hard milestone gates (alpha M4, beta M8, RC M10). If alpha gate missed by >2 weeks, re-scope v1 features. |
| Musl allocator contention degrades Suricata throughput | Medium | Medium | Benchmark with and without jemalloc LD_PRELOAD in Month 5 (VM) and Month 10 (hardware). If >20% improvement, ship jemalloc as default. IDS-only default already provides headroom. |
| Secure Boot required by customer or regulation | Low | Medium | Deferred to v1.5. v1.0 security model (read-only rootfs, signed OTA, noexec) is strong without Secure Boot. If a customer requires it before v1.5, custom shim + MOK enrollment is the path. |
| Intel QAT libraries incompatible with musl | Low | Low | QAT is not needed for v1 throughput targets. WireGuard's kernel crypto saturates 2.5GbE. If future products need QAT, libraries must be patched for musl or run via gcompat. |
| GPL compliance failure (source distribution) | Low | High | Automated: build pipeline publishes source tarball per release, SBOM generated via syft, license check in CI. Source archive URL embedded in firmware (`/api/v1/system/about`). |
| Missing SBOM blocks government/enterprise sales | Low | Medium | CycloneDX SBOM generated automatically in `make bundle`. Published alongside each firmware release. grype CVE scan in CI catches known vulnerabilities before shipping. |

---

*Next step: Begin implementation. Start with Month 1 -- scoutd skeleton (main.go, config schema, config store), Vagrant VM dev environment with Alpine, and basic project structure. No hardware needed.*

---

## 16. Post-Launch Operational Sustainability

Building ScoutOS is a 12-month project. **Sustaining it is an ongoing program.** After v1.0 ships, the product requires continuous security engineering -- not just feature development. This section makes the operational burden explicit so it's planned for, not discovered after launch.

The viability question shifts from "Can we build it?" to "Can we sustain it?"

### 16.1 Ongoing Maintenance Cadences

| Activity | Cadence | Owner | Effort | Automation |
|----------|---------|-------|--------|------------|
| **Suricata rule updates** (ET Open) | Daily (automated) | scoutd (suricata-update cron) | None (automated on device) | Fully automated |
| **DNS blocklist updates** (RPZ) | Daily (automated) | scoutd | None (automated on device) | Fully automated |
| **GeoIP database updates** | Monthly (automated) | scoutd | None (automated on device) | Fully automated |
| **CVE monitoring** (grype scan) | Weekly (CI) | Founder | ~1 hour/week to triage | GitHub Actions, Slack/email alerts |
| **Alpine security patches** | As released (~weekly) | Founder | ~2-4 hours/patch cycle | Manual: review advisory → rebuild → test → OTA |
| **Suricata version upgrades** | Quarterly (~4 releases/year) | Founder | ~1-2 days per upgrade | Manual: test in VM → build → OTA |
| **Alpine major version migration** | Every ~2 years (before EOL) | Founder | ~1-2 weeks | Manual: full regression test required |
| **SBOM regeneration** | Every firmware release | CI pipeline | None (automated) | make bundle includes syft |
| **GPL source archive** | Every firmware release | CI pipeline | None (automated) | Build script archives sources |
| **OTA infrastructure** | Continuous | Founder | ~2 hours/month (monitoring, cert renewal) | update.scoututm.com uptime monitoring |
| **Linux kernel LTS tracking** | Every ~6 months (LTS point releases) | Founder | ~1-2 days per kernel bump | Manual: test initramfs + dm-verity + drivers |

### 16.2 The Security Update Pipeline

When a CVE is discovered in a shipped component, the response process is:

```
1. grype CI scan flags CVE (or manual discovery via advisory)
2. Assess severity and exploitability for ScoutOS (not all CVEs apply)
3. If Critical/High and exploitable:
   a. Patch within 72 hours (rebuild with patched Alpine package)
   b. Test in Vagrant VM (smoke + affected subsystem tests)
   c. Build signed RAUC bundle
   d. Push OTA update to update.scoututm.com
   e. Devices with auto-update check within 24 hours
   f. Publish security advisory on ScoutUTM.com
4. If Medium/Low or not exploitable:
   a. Queue for next scheduled firmware release (~monthly cadence)
```

**Worst-case scenario:** A critical Suricata or kernel CVE requires a same-day patch. The full cycle (rebuild → VM test → sign → publish OTA) should take <4 hours once the pipeline is mature. This is feasible for a solo founder because the build system is automated -- the human work is triage, review, and the decision to ship.

### 16.3 Sustainability Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Founder burnout from constant CVE triage | Automate triage: grype filters by severity, ignore informational. Most Alpine patches are not exploitable in ScoutOS's context (no SSH, no user shells, no web apps beyond scoutd). |
| Alpine 3.21 EOL before migration is ready | Track Alpine release dates. Begin migration testing 6 months before EOL. Alpine's 2-year support window provides ample lead time. |
| OTA server downtime prevents security updates | Devices continue operating normally and retry on next interval. Manual update via USB is always available. OTA server is a self-hosted VPS with nginx serving static files (see 7.5). |
| Suricata upstream introduces breaking changes | Pin Suricata version per ScoutOS release. Test upstream releases in VM before adopting. Maintain ability to carry patches if upstream drops NFQUEUE support. |
| SBOM/compliance requirements expand beyond current scope | CycloneDX and syft are extensible. If VEX (Vulnerability Exploitability eXchange) or other formats are required, syft supports them. The foundation is in place. |

### 16.4 Solo Founder Capacity Planning

Realistic time allocation after v1.0 ships:

| Activity | Hours/Month |
|----------|-------------|
| Security maintenance (CVE triage, patches, OTA) | 8-12 |
| Customer support (beta testers, bug reports) | 8-12 |
| Feature development (v1.1, v1.5) | 40-60 |
| Infrastructure (OTA server, build pipeline, CI) | 4-8 |
| **Total** | **60-90** |

The security maintenance load is the non-negotiable baseline. If feature development must be cut to maintain security patch cadence, that is the correct trade-off. A UTM appliance that ships known vulnerabilities is worse than one that ships late features.

---

## 17. Glossary

| Term | Definition |
|------|------------|
| **A/B Updates** | Dual-partition update scheme where the system has two root filesystem slots (A and B). Updates are written to the inactive slot while the active one keeps running. If the update fails, the system falls back to the previous working slot. |
| **AES-NI** | Advanced Encryption Standard New Instructions. CPU hardware acceleration for AES encryption/decryption. Relevant for VPN (WireGuard) throughput but not for IDS packet inspection. |
| **AF_PACKET** | A Linux socket type that provides raw access to network frames at the data link layer. Used by Suricata in bridge/tap deployments but not suitable for router-mode operation (ScoutOS uses NFQUEUE instead). |
| **Alpine Linux** | A lightweight, security-oriented Linux distribution built on musl libc and BusyBox. Used as the base OS for ScoutOS due to its small footprint (~5MB base), fast boot, and apk package manager. |
| **APK** | Alpine Package Keeper. Alpine Linux's package manager, used to install and manage software packages (e.g., `apk add suricata`). |
| **ARP** | Address Resolution Protocol. Maps IP addresses to MAC addresses on a local network. scoutd scans the ARP table to discover devices connected to the LAN. |
| **bcrypt** | A password hashing algorithm designed to be computationally expensive, making brute-force attacks slow. Used to hash the admin password stored in config.json. |
| **BusyBox** | A single binary that provides stripped-down implementations of common Unix utilities (sh, ls, mount, etc.). Alpine Linux uses it as the default shell and utility set, keeping the OS footprint small. |
| **chi** | A lightweight, idiomatic HTTP router for Go. Compatible with Go's standard `net/http` library and supports middleware chaining. Used by scoutd for REST API routing. |
| **CI/CD** | Continuous Integration / Continuous Deployment. Automated pipeline that builds, tests, and deploys code on every commit. ScoutOS uses GitHub Actions for this. |
| **cgo** | Go's mechanism for calling C code from Go programs. ScoutOS disables cgo (`CGO_ENABLED=0`) to produce a fully static binary with no C library dependencies, ensuring compatibility with Alpine's musl libc. |
| **conntrack** | Connection tracking. A Linux kernel subsystem that tracks the state of network connections (TCP sessions, UDP flows). Used by nftables for stateful firewall rules and by scoutd for connection count statistics. |
| **CycloneDX** | An open standard for Software Bill of Materials (SBOM). ScoutOS generates CycloneDX JSON as part of each firmware build to document all included packages and their licenses. |
| **CSRF** | Cross-Site Request Forgery. An attack where a malicious website tricks a user's browser into making unintended requests to another site. Mitigated in ScoutOS with SameSite=Strict cookies and origin header checking. |
| **DHCP** | Dynamic Host Configuration Protocol. Automatically assigns IP addresses, gateway, DNS server, and other network settings to devices joining the network. Handled by dnsmasq in ScoutOS. |
| **DNAT** | Destination Network Address Translation. Rewrites the destination IP/port of incoming packets. Used for port forwarding (e.g., forwarding WAN port 8080 to a LAN server on port 80). |
| **DNS** | Domain Name System. Translates human-readable domain names (e.g., `google.com`) into IP addresses. ScoutOS uses Unbound as its DNS resolver. |
| **DNSSEC** | DNS Security Extensions. Cryptographic signatures on DNS records that allow resolvers to verify that responses haven't been tampered with. Requires accurate system time (provided by chrony). |
| **dnsmasq** | A lightweight daemon that provides DHCP and DNS services. In ScoutOS, only its DHCP functionality is used (DNS is handled by Unbound). |
| **DoH** | DNS over HTTPS. Encrypts DNS queries inside standard HTTPS connections (port 443). Harder to block than DoT since it blends with normal web traffic. |
| **DoT** | DNS over TLS. Encrypts DNS queries using TLS on a dedicated port (853). Prevents ISPs and network observers from seeing which domains are being resolved. |
| **dm-verity** | Device-mapper verity. A Linux kernel feature that provides transparent integrity checking of block devices using a Merkle tree of hashes. ScoutOS uses dm-verity to verify every block read from the SquashFS rootfs partition, detecting corruption or tampering at the block level. |
| **DPDK** | Data Plane Development Kit. A set of libraries for fast packet processing that bypasses the kernel networking stack. Supported on Alpine/musl since v21.05 but not used in ScoutOS v1 -- kernel-based nftables + NFQUEUE is sufficient for the target throughput. |
| **ECDSA** | Elliptic Curve Digital Signature Algorithm. A public-key cryptography algorithm used for digital signatures. ScoutOS uses ECDSA P-256 for signing OTA update bundles and for the self-signed TLS certificate. |
| **ESP** | EFI System Partition. A FAT32 partition on the disk that stores bootloader files (GRUB), kernel images, and boot configuration. Required for UEFI booting. |
| **ET Open** | Emerging Threats Open. A free, community-maintained ruleset for Suricata/Snort IDS that detects malware, exploits, and suspicious network traffic. Used as the baseline IDS ruleset in ScoutOS. |
| **EVE JSON** | Extensible Event Format in JSON. Suricata's structured log output format containing alerts, flow records, DNS events, and other network metadata. scoutd reads EVE JSON from a Unix socket for real-time alert processing. |
| **ext4** | Fourth Extended Filesystem. A widely-used Linux journaling filesystem. Used for the config and data partitions in ScoutOS. |
| **F2FS** | Flash-Friendly File System. A Linux filesystem designed for NAND flash storage without a hardware Flash Translation Layer (FTL). Not used in ScoutOS because the NVMe SSD has a hardware FTL that handles wear leveling. |
| **FTL** | Flash Translation Layer. Firmware in an SSD/eMMC controller that translates logical block addresses to physical NAND locations, handling wear leveling, garbage collection, and bad block management. NVMe and eMMC devices include an FTL, making OS-level flash-aware filesystems (F2FS) unnecessary. |
| **gcompat** | A glibc compatibility layer for Alpine Linux/musl. Allows running some glibc-linked binaries on musl systems without recompilation. Used as a fallback if a vendor provides only glibc-linked user-space tools. |
| **GeoIP** | Geographic IP filtering. Blocking or allowing network traffic based on the geographic location associated with an IP address. Uses a database (MaxMind/DB-IP) to map IPs to countries. |
| **`//go:embed`** | A Go compiler directive that embeds files into the compiled binary at build time. ScoutOS uses it to bundle the Vue.js frontend assets directly into the scoutd binary, eliminating the need for a separate web server. |
| **GPT** | GUID Partition Table. A modern disk partitioning scheme that replaces MBR. Supports large disks and more partitions. Required for UEFI boot. |
| **grype** | An open-source vulnerability scanner by Anchore that checks SBOMs or container images against known CVE databases. Used in ScoutOS's CI pipeline to detect known vulnerabilities in shipped packages before release. |
| **GRUB** | GRand Unified Bootloader. A widely-used bootloader that loads the Linux kernel and initramfs. ScoutOS uses GRUB for A/B slot selection, try-counter-based fallback, and the rescue boot entry. |
| **HMR** | Hot Module Replacement. A development feature where code changes are applied to a running application without a full page reload. Provided by Vite during Vue.js frontend development. |
| **IDS** | Intrusion Detection System. Monitors network traffic for malicious activity and generates alerts, but does not block traffic. ScoutOS defaults to IDS mode using Suricata. |
| **igc** | Intel I225/I226 network driver. The Linux kernel module that drives the Intel I226-V 2.5GbE NICs on the Odroid H4+. Mainline in the Linux kernel -- no proprietary blobs required. |
| **initramfs** | Initial RAM Filesystem. A minimal filesystem loaded into memory during early boot, before the real root filesystem is mounted. ScoutOS uses a custom initramfs to set up the OverlayFS stack (SquashFS + tmpfs). |
| **iperf3** | A network performance measurement tool that generates TCP/UDP traffic between a client and server to measure throughput, jitter, and packet loss. Used for benchmarking IDS/IPS throughput. |
| **IPS** | Intrusion Prevention System. Like IDS but actively blocks malicious traffic in real-time. In ScoutOS, IPS mode uses Suricata's NFQUEUE verdicts to drop packets matching threat signatures. Opt-in due to throughput impact. |
| **jemalloc** | A general-purpose memory allocator designed for high-concurrency workloads. Can be swapped in via `LD_PRELOAD` on Alpine to replace musl's default allocator, which suffers contention under heavy multi-threading (e.g., Suricata). |
| **`LD_PRELOAD`** | A Linux environment variable that loads a shared library before all others, overriding functions (like `malloc`) in the default C library. Used in ScoutOS to optionally swap musl's allocator with jemalloc for Suricata. |
| **MAC Address** | Media Access Control address. A unique 48-bit hardware identifier assigned to every network interface (e.g., `AA:BB:CC:DD:EE:FF`). Used by DHCP for lease assignment and by scoutd for device identification. |
| **masquerade** | A form of NAT where outgoing packets have their source IP rewritten to the router's WAN IP. Essential for allowing LAN devices with private IPs to communicate with the internet. |
| **Merkle tree** | A cryptographic data structure where each leaf node contains a hash of a data block, and each parent node contains a hash of its children. Used by dm-verity to efficiently verify the integrity of individual blocks without hashing the entire disk. |
| **mimalloc** | A compact, high-performance memory allocator by Microsoft. Like jemalloc, it can replace musl's default allocator via `LD_PRELOAD` for better multi-threaded performance. |
| **MOK** | Machine Owner Key. A key database in UEFI Secure Boot that allows device owners to enroll custom signing keys alongside the default Microsoft/vendor keys. Required for booting non-major-distribution Linux with Secure Boot enabled. |
| **musl** | A lightweight C standard library designed for static linking and small binary size. Alpine Linux uses musl instead of glibc, which is why ScoutOS builds Go with `CGO_ENABLED=0` to avoid musl compatibility issues. |
| **NAT** | Network Address Translation. Rewrites IP addresses in packet headers as they pass through the router, allowing multiple LAN devices to share a single public WAN IP address. |
| **NFQUEUE** | Netfilter Queue. A Linux kernel mechanism that passes packets from the firewall (nftables) to a userspace program (Suricata) for inspection. The userspace program returns a verdict (accept/drop) for each packet. |
| **nftables** | The modern Linux kernel firewall framework, replacing iptables. Provides packet filtering, NAT, and traffic classification using a single unified syntax. ScoutOS uses a single `inet scout` table for all firewall rules. |
| **nmap** | A network scanning tool used for security auditing and network discovery. In ScoutOS testing, nmap scans are used to trigger IDS alert storms and validate rate limiting. |
| **NTP** | Network Time Protocol. Synchronizes system clocks over the network. Accurate time is critical for TLS, DNSSEC, IDS timestamps, and parental control schedules. Handled by chrony in ScoutOS. |
| **NVMe** | Non-Volatile Memory Express. A high-speed storage interface for SSDs connected directly to the CPU via PCIe. The Odroid H4+ uses an NVMe SSD for the ScoutOS disk. |
| **Odroid H4+** | A single-board computer (SBC) by Hardkernel with an Intel N97 CPU, dual Intel I226-V 2.5GbE NICs, and support for NVMe storage. The target hardware platform for ScoutOS. |
| **OpenRC** | A dependency-based init system used by Alpine Linux (as an alternative to systemd). Manages service startup order, dependencies, and lifecycle (start/stop/restart/reload). |
| **OTA** | Over-The-Air. Delivering software updates to a device remotely over the network, without requiring physical access or USB drives. ScoutOS uses RAUC for OTA updates. |
| **OUI** | Organizationally Unique Identifier. The first 3 bytes of a MAC address, assigned to manufacturers by IEEE. scoutd uses OUI lookups to identify device manufacturers (e.g., "Apple", "Samsung") for device fingerprinting. |
| **OverlayFS** | A union filesystem that layers a writable upper directory on top of a read-only lower directory. ScoutOS uses OverlayFS with SquashFS (lower, read-only) and tmpfs (upper, volatile) so the root filesystem appears writable but reverts to a pristine state on every reboot. |
| **OVMF** | Open Virtual Machine Firmware. An open-source UEFI firmware implementation for virtual machines. Used with QEMU to test ScoutOS's UEFI boot process without physical hardware. |
| **Pinia** | The official state management library for Vue.js 3. Provides reactive stores for sharing state between components (e.g., authentication state, device list, real-time stats). |
| **QAT** | QuickAssist Technology. Intel hardware acceleration for cryptographic operations (encryption, compression). QAT's user-space libraries have glibc-specific dependencies that do not build cleanly on musl/Alpine. Not used in ScoutOS v1 -- WireGuard's kernel crypto is sufficient. |
| **QEMU** | Quick EMUlator. An open-source machine emulator that can run full operating systems in a virtual environment. Used for testing ScoutOS disk images and UEFI boot before deploying to physical hardware. |
| **RAUC** | Robust Auto-Update Controller. An open-source framework for managing A/B partition OTA updates on embedded Linux systems. Handles bundle verification, slot management, and bootloader integration. |
| **REST API** | Representational State Transfer Application Programming Interface. A web API design pattern using standard HTTP methods (GET, POST, PUT, DELETE) on resource URLs. scoutd exposes a REST API at `/api/v1/*` for all management operations. |
| **RPZ** | Response Policy Zones. A DNS mechanism that allows a resolver (Unbound) to override responses for specific domains. Used for DNS-based content filtering -- blocked domains return NXDOMAIN instead of their real IP address. |
| **RTC** | Real-Time Clock. A battery-backed hardware clock that keeps time when the system is powered off. The Odroid H4+ lacks an RTC, so ScoutOS uses fake-hwclock to preserve approximate time across reboots. |
| **SBOM** | Software Bill of Materials. A formal inventory of all software components, versions, and licenses in a product. Required by EU Cyber Resilience Act and U.S. Executive Order 14028. ScoutOS generates SBOMs in CycloneDX format using syft. |
| **SKU** | Stock Keeping Unit. A distinct product variant, often differentiated by performance tier. In ScoutOS context, throughput benchmarks define what performance claims each hardware configuration's SKU can make. |
| **SBC** | Single-Board Computer. A complete computer built on a single circuit board, including CPU, memory, and I/O. Examples: Raspberry Pi, Odroid H4+. |
| **SCHED_FIFO** | A Linux real-time scheduling policy that gives a thread fixed priority and allows it to run until it voluntarily yields or is preempted by a higher-priority thread. Used for Suricata's packet processing threads to prevent latency spikes from lower-priority processes. |
| **Secure Boot** | A UEFI firmware feature that verifies the cryptographic signature of the bootloader and kernel before executing them, preventing unauthorized code from running at boot. Deferred to ScoutOS v1.5 due to Alpine's lack of out-of-box signed bootloader support. |
| **semver** | Semantic Versioning. A versioning scheme using MAJOR.MINOR.PATCH format (e.g., 1.2.3). Used by ScoutOS for firmware version numbering and downgrade protection checks. |
| **shim** | A small UEFI application signed by Microsoft that chain-loads a distribution's own bootloader (GRUB). Required for Secure Boot on non-Microsoft operating systems. Alpine does not ship a signed shim, which is why Secure Boot is deferred. |
| **SID** | Suricata Rule ID. A unique numeric identifier for each IDS/IPS detection rule. Users can suppress noisy SIDs to reduce false-positive alerts. |
| **SNAT** | Source Network Address Translation. Rewrites the source IP of outgoing packets. Masquerade is a special case of SNAT where the source is dynamically set to the outgoing interface's IP. |
| **SPDX** | Software Package Data Exchange. An open standard for communicating software bill of materials (SBOM) information. An alternative to CycloneDX. ScoutOS uses CycloneDX but syft can output either format. |
| **SPA** | Single Page Application. A web application that loads a single HTML page and dynamically updates content via JavaScript, without full page reloads. The ScoutOS dashboard is a Vue.js SPA. |
| **SquashFS** | A compressed, read-only Linux filesystem. ScoutOS packages the entire root filesystem as a SquashFS image, providing tamper resistance and enabling atomic A/B updates (the whole OS is a single file). |
| **Suricata** | An open-source network threat detection engine capable of IDS, IPS, and network security monitoring. Inspects network traffic against signature rules to detect malware, exploits, and suspicious activity. |
| **syft** | An open-source tool by Anchore that generates Software Bills of Materials (SBOMs) by scanning container images or filesystem directories. Used in ScoutOS's build pipeline to produce CycloneDX JSON inventories. |
| **switch_root** | A Linux command used during early boot to change the root filesystem from the initramfs to the real root filesystem. Part of ScoutOS's boot sequence after OverlayFS is assembled. |
| **Tailwind CSS** | A utility-first CSS framework that provides low-level CSS classes (e.g., `p-4`, `text-blue-500`) instead of pre-built components. Produces small production CSS via unused class purging. |
| **TLS** | Transport Layer Security. A cryptographic protocol that provides encrypted communication over a network. Used for HTTPS (web UI), DoT (DNS encryption), and WireGuard handshakes. |
| **tmpfs** | A temporary filesystem stored entirely in RAM. Fast but volatile -- all data is lost on reboot. Used as the OverlayFS upper layer in ScoutOS so runtime writes don't persist. |
| **TPM** | Trusted Platform Module. A hardware security chip that provides cryptographic functions (key generation, key sealing, random number generation) and tamper-resistant storage. The Odroid H4+ does not include onboard TPM. Deferred to ScoutOS v1.5. |
| **UEFI** | Unified Extensible Firmware Interface. The modern replacement for BIOS that initializes hardware and boots the operating system. The Odroid H4+ uses UEFI to boot GRUB from the ESP. |
| **Unbound** | A validating, recursive DNS resolver. Unlike simple DNS forwarders, Unbound resolves queries from the DNS root servers and validates DNSSEC signatures. Also supports RPZ for content filtering. |
| **uPlot** | A lightweight, high-performance JavaScript charting library (~35KB min+gzip). Used in the ScoutOS dashboard for real-time traffic graphs. Chosen over Chart.js for its smaller bundle size. |
| **Vagrant** | A tool for building and managing virtual machine environments using simple configuration files (Vagrantfile). ScoutOS uses Vagrant with VirtualBox or libvirt for Tier 2 integration testing. |
| **vfat** | Virtual File Allocation Table. A filesystem format compatible with FAT32, required for the EFI System Partition (ESP) by the UEFI specification. |
| **Vite** | A fast build tool and development server for modern web projects. Provides instant HMR during development and optimized production builds. Used for the Vue.js frontend in ScoutOS. |
| **VLAN** | Virtual Local Area Network. A technology that segments a physical network into isolated logical networks using 802.1Q tags. Requires a managed switch. ScoutOS uses VLANs to create separate zones (IoT, Guest, Kids) on a single physical LAN port. |
| **Vue.js** | A progressive JavaScript framework for building user interfaces. ScoutOS uses Vue 3 with the Composition API for the web dashboard SPA. |
| **watchdog** | A hardware timer that must be periodically "kicked" by software. If the software hangs and stops kicking, the timer expires and forces a hardware reboot. ScoutOS uses the Intel N97's iTCO watchdog to recover from system hangs. |
| **WebSocket** | A protocol providing full-duplex, persistent communication channels over a single TCP connection. ScoutOS uses WebSocket (WSS) to push real-time stats, device updates, and IDS alerts to the dashboard without polling. |
| **WireGuard** | A modern, high-performance VPN protocol implemented as a Linux kernel module. Simpler and faster than OpenVPN or IPsec. ScoutOS uses WireGuard for remote access VPN with QR-code-based peer configuration. |
