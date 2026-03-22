# ScoutUTM Test Instructions

This document explains how to build, test, and develop ScoutUTM across both tiers of the development environment.

## Prerequisites

- **Go 1.22+** installed (on this system: `/usr/local/go/bin/go`, in WSL only)
- **VirtualBox** installed on Windows host (for Vagrant VM)
- **Vagrant** installed on Windows host (for Vagrant VM)
- **WSL2** for Go builds (Go is not installed on Windows)

---

## Quick Start

The fastest way to build and test everything:

```powershell
# From PowerShell (Windows) — builds via WSL, then provisions VM with tests
.\build.ps1
```

Or from WSL:

```bash
./build.sh
```

Both scripts: build the scoutd binary, copy templates, destroy any existing VM, provision a fresh VM, and run all integration tests automatically.

---

## Tier 1: Host-Native Development

Fast iteration (~1-2s). Runs on your host machine with no VM.

### Building scoutd

```bash
# From WSL — builds static Linux binary to build/scoutd
cd scoutd && CGO_ENABLED=0 /usr/local/go/bin/go build -ldflags "-X main.version=0.1.0-dev" -o ../build/scoutd ./cmd/scoutd/

# Copy templates into build directory (needed for Vagrant VM)
cp scoutd/templates/*.tmpl build/templates/
```

The binary is pure static Go (`CGO_ENABLED=0`, no cgo). Output goes to `build/scoutd`.

### Running Unit Tests

```bash
# All tests across all packages (284 tests, 12 packages)
cd scoutd && /usr/local/go/bin/go test ./...

# Single package:
cd scoutd && /usr/local/go/bin/go test ./internal/config/...
cd scoutd && /usr/local/go/bin/go test ./internal/render/...
cd scoutd && /usr/local/go/bin/go test ./internal/api/...
cd scoutd && /usr/local/go/bin/go test ./internal/service/...
cd scoutd && /usr/local/go/bin/go test ./internal/parental/...
cd scoutd && /usr/local/go/bin/go test ./internal/netmon/...
cd scoutd && /usr/local/go/bin/go test ./internal/ids/...

# Verbose output:
cd scoutd && /usr/local/go/bin/go test -v ./internal/api/...

# Run a specific test:
cd scoutd && /usr/local/go/bin/go test -v -run TestCreatePeer_SplitTunnel ./internal/api/...
```

**What the unit tests cover (284 tests):**
- `internal/api/` (94) -- REST API handlers via httptest (firewall CRUD, network/zones CRUD, VLANs, VPN/WireGuard, IDS, devices, parental, wizard, system, auth, config apply)
- `internal/render/` (60) -- Template rendering for all daemon configs (nftables, dnsmasq, chrony, unbound, wireguard, suricata)
- `internal/config/` (46) -- Config loading, saving, validation, schema migrations
- `internal/netmon/` (14) -- ARP scanning, DHCP lease parsing, device fingerprinting, OUI lookup
- `internal/parental/` (13) -- Schedule enforcement, blocked device IPs, time range detection
- `internal/auth/` (13) -- Session management, bcrypt password hashing
- `internal/ids/` (11) -- EVE event parsing, alert deduplication, log rotation
- `internal/dnslog/` (8) -- Unbound log parsing, ring buffer, log rotation
- `internal/ws/` (8) -- WebSocket hub broadcast, max clients, non-blocking send
- `internal/scheduler/` (6) -- Dev-mode task suppression, interval/RunAt timing
- `internal/stats/` (6) -- CPU, memory, uptime, throughput, conntrack (dev-mode synthetic data)
- `internal/service/` (5) -- Service manager dependency ordering

### Running scoutd Locally (Dev Mode)

```bash
cd scoutd && /usr/local/go/bin/go run ./cmd/scoutd --dev-mode
```

Dev mode changes:
- Listens on `:8443` (HTTPS) and `:8080` (HTTP) instead of `:443`/`:80`
- Uses local paths: `./config.json`, `./templates/`, `./output/`, `./tls/`
- Stubs all hardware operations (OpenRC reloads, nftables apply, netlink stats)
- Still renders templates to disk under `./output/` for inspection
- Generates a self-signed TLS certificate in `./tls/`

After starting, access the API at `https://localhost:8443/api/v1/health`.

---

## Tier 2: Vagrant VM Development

Full ScoutOS environment (~30s reload). Runs real daemons (nftables, dnsmasq, unbound, chrony) in an Alpine Linux VM.

### VM Setup

```powershell
# From PowerShell — the build scripts handle everything:
.\build.ps1
```

Or step by step from WSL + PowerShell:

```bash
# WSL: Build the binary
cd scoutd && CGO_ENABLED=0 /usr/local/go/bin/go build -ldflags "-X main.version=0.1.0-dev" -o ../build/scoutd ./cmd/scoutd/
cp scoutd/templates/*.tmpl build/templates/
```

```powershell
# PowerShell: Provision VM
vagrant destroy -f
vagrant up
```

The Vagrantfile configures two VMs:

**UTM VM (`utm`, primary):**
- **Box:** `generic/alpine319` (Alpine Linux 3.19)
- **Resources:** 4 CPUs, 8 GB RAM
- **Port forwards:** host `8443` -> guest `443` (HTTPS), host `8080` -> guest `80` (HTTP)
- **LAN interface:** `192.168.1.1/24` on VirtualBox internal network `scoutlan`
- **Synced folders:** `build/` -> `/opt/scout-build`, `tests/` -> `/opt/scout-tests`

**Client VM (`client`):**
- **Box:** `generic/alpine319` (lightweight LAN client)
- **Resources:** 1 CPU, 512 MB RAM
- **LAN interface:** DHCP from ScoutUTM on VirtualBox internal network `scoutlan`
- **No port forwards, no synced folders**

Provisioning (`tools/vagrant-provision.sh`) does:
1. Flushes nftables rules (SSH protection), enables IP forwarding
2. Installs Alpine packages (nftables, unbound, dnsmasq, chrony, wireguard-tools, etc.)
3. Installs scoutd binary and templates from `/opt/scout-build`
4. Installs default config, sets admin password to `admin`
5. Disables RPZ blocklists (no real blocklist data in dev VM)
6. Creates OpenRC service with `--dev-mode` and explicit paths
7. Starts scoutd (renders all daemon configs on startup)
8. Loads nftables rules, starts all daemons
9. Runs all integration tests automatically

### Accessing the VMs

```bash
vagrant ssh utm       # UTM appliance (default — also works with just `vagrant ssh`)
vagrant ssh client    # LAN client
```

### Accessing the Web UI

From your host browser:
- **HTTPS:** https://localhost:8443
- **Login:** `admin` / `admin`

The TLS certificate is self-signed, so you'll need to accept the browser warning.

### Deploying Code Changes to the VM

After modifying Go code:

```powershell
# Full rebuild + reprovision:
.\build.ps1

# Or just reprovision (if binary already built):
vagrant provision
```

### Integration Tests Run Automatically

All integration tests run at the end of `vagrant up` / `vagrant provision`. You'll see pass/fail results directly in the terminal output.

To re-run tests manually inside the UTM VM:

```bash
vagrant ssh utm
/opt/scout-tests/smoke_test.sh
/opt/scout-tests/api_test.sh
/opt/scout-tests/network_test.sh
/opt/scout-tests/vlan_test.sh
/opt/scout-tests/vpn_test.sh
/opt/scout-tests/parental_test.sh
/opt/scout-tests/ids_test.sh
/opt/scout-tests/wizard_test.sh
/opt/scout-tests/stress_test.sh
```

### Rebuilding the VM from Scratch

```powershell
.\build.ps1
```

---

## Integration Test Details

Each test script is self-contained and idempotent -- it cleans up any resources it creates. Tests that check nftables state reload rules via `nft -f /etc/nftables.conf` since scoutd runs in dev-mode (stubs service reloads).

### smoke_test.sh (8 checks)
Verifies the basics are working:
1. Health endpoint returns `healthy=true`
2. Version field present in health response
3. Config file exists at `/etc/scout/config.json` and is valid JSON
4. Config has all top-level keys (version, system, network, firewall, dns, vpn, ids, parental, update)
5. Services running: nftables, dnsmasq, chronyd, scoutd, suricata (warning if not running), unbound
6. TLS certificate and key exist

### api_test.sh (12 checks)
Exercises the REST API:
1. Login with admin password, receive session token
2. GET `/firewall` returns firewall config
3. GET `/network` returns network config
4. GET `/system` returns system config (without leaking `admin_hash`)
5. CRUD firewall rules (create, list, delete)
6. PUT `/system` partial update (hostname)
7. Protected zones cannot be deleted (wan, lan)
8. Logout invalidates token

### network_test.sh (9 checks)
Verifies rendered daemon configs and nftables rules:
1. `inet scout` nftables table exists
2. Zone device sets exist (`wan_devices`, `lan_devices`)
3. IPv6 drop rule in prerouting_raw
4. DNS redirect rule in prerouting
5. NAT masquerade on WAN
6. dnsmasq running with DNS disabled (`port=0`)
7. chronyd running

### vlan_test.sh (14 checks)
Tests VLAN and zone management end-to-end:
1. Create IoT zone via API
2. Create VLAN 20 on `eth1` assigned to IoT zone
3. Verify VLAN appears in list
4. Duplicate VLAN ID rejected
5. Invalid VLAN ID (5000) rejected
6. Create Guest zone + VLAN 30
7. nftables zone sets include VLAN sub-interfaces (`eth1.20`, `eth1.30`)
8. IoT zone has separate input chain with drop policy
9. dnsmasq has DHCP config for VLAN interface
10. dnsmasq has DHCP range for VLAN interface
11. DNS redirect rule exists for IoT zone
12. chrony allows VLAN subnet
13. Cleanup (remove test VLANs and zones)

### vpn_test.sh (26 checks)
Tests WireGuard VPN end-to-end:
1. WireGuard disabled by default
2. Enable WireGuard -- server keys auto-generated
3. WireGuard config file rendered at `/etc/wireguard.conf` with PrivateKey and ListenPort
4. nftables allows WireGuard UDP port (51820) on WAN
5. nftables has `input_vpn` chain (DNS, ICMP, NTP, HTTPS, accept)
6. nftables has VPN forwarding rules (`wg0` to WAN and LAN zones)
7. nftables has DNS redirect for VPN clients
8. Create peer -- returns name, tunnel IP, client config, QR code (base64 PNG)
9. Client config DNS points to LAN IP (`192.168.1.1`), not tunnel IP
10. Client config has server PublicKey and Endpoint
11. Sequential tunnel IP allocation (10.0.0.2, 10.0.0.3)
12. Split tunnel peer has LAN subnets in AllowedIPs (no `0.0.0.0/0`)
13. Full tunnel peer has `0.0.0.0/0` in AllowedIPs
14. Peer list shows correct count
15. Delete peer works, count decremented
16. Regenerate server keys changes public key and clears all peers
17. WireGuard config file exists (dev-mode, service management stubbed)

### parental_test.sh (15 checks)
Tests parental controls end-to-end:
1. Create parental profile (name, block categories, safe search, YouTube restricted, schedule)
2. List profiles -- verify profile appears
3. List devices -- check at least one available
4. Assign device to profile
5. Config has device with static IP
6. dnsmasq.conf has `dhcp-host=` static DHCP reservation
7. unbound.conf has `access-control-tag:` and `define-tag:` for parental filtering
8. unbound.conf has Google safe search CNAME override
9. unbound.conf has YouTube restricted mode CNAME override
10. unbound.conf has parental RPZ zones
11. Device response includes manufacturer field (OUI lookup)
12. DNS query log endpoint returns array
13. Cleanup (remove test profile)

### ids_test.sh (16 checks)
Tests Suricata IDS/IPS configuration and mode switching:
1. IDS enabled by default, mode is `ids`
2. suricata.yaml rendered at `/etc/suricata/suricata.yaml`
3. suricata.yaml has HOME_NET, fail-open, unix_dgram EVE output
4. Switch to IPS mode via API
5. nftables IPS chain exists with NFQUEUE fanout
6. Forward chain jumps to IPS chain
7. Switch back to IDS mode
8. IPS chain removed in IDS mode
9. Disable IDS via API
10. Re-enable IDS via API
11. Alerts endpoint returns empty array (no Suricata running in dev-mode)
12. Invalid mode rejected with error

### wizard_test.sh (10 checks)
Tests setup wizard end-to-end flow:
1. Pre-check: `setup_done` is false (fresh config)
2. Step 1: Welcome (returns 200)
3. Step 2: Set password (returns 200), short password rejected (returns 400)
4. Step 3: Network detection returns interfaces
5. Step 4: LAN config (returns 200)
6. Step 5: Security defaults -- IDS, malware/phishing blocking (returns 200)
7. Step 6: VLAN setup skipped (returns 200)
8. Step 7: Apply (returns 200)
9. Verify `setup_done` flipped to true
10. Wizard returns 403 after setup complete

### stress_test.sh (7 checks)
Tests WebSocket and HTTP concurrency:
1. 20 concurrent health endpoint requests complete
2. 10 concurrent authenticated device list requests complete
3. 10 concurrent alert endpoint requests complete
4. (With websocat) 10 concurrent WebSocket connections maintained
5. (With websocat) 11th WebSocket connection rejected (max clients)
6. (With websocat) Stats broadcast received within 6 seconds

Note: Falls back to HTTP-only tests if `websocat` is not installed.

### client_test.sh (7 checks)
Runs ON the client VM (not the UTM). Verifies end-to-end traffic flow through ScoutUTM:
1. Client has DHCP lease from ScoutUTM (IP in 192.168.1.100-253)
2. Default gateway is 192.168.1.1
3. DNS resolver is 192.168.1.1
4. DNS resolution works (nslookup via ScoutUTM's Unbound)
5. Internet access works through ScoutUTM (curl external site)
6. ScoutUTM web UI accessible at https://192.168.1.1
7. ScoutUTM health API returns healthy

To run manually:
```bash
vagrant ssh client -c "sh /opt/scout-tests/client_test.sh"
```

Note: This test is NOT run automatically during UTM provisioning (it runs on a different VM). It runs during client VM provisioning via `tools/client-provision.sh` connectivity checks.

---

## LAN Client VM

The client VM simulates a device on the home network with all traffic routed through ScoutUTM.

### Network Topology

```
Host Machine
  │
  ├─ Port Forward: 8443 → utm:443 (Web UI)
  ├─ Port Forward: 8080 → utm:80
  │
  ┌──────────────────────────────────────────┐
  │  VirtualBox NAT (10.0.2.x)              │
  │    ├─ utm:eth0 (WAN, DHCP)              │
  │    └─ client:eth0 (Vagrant SSH only)    │
  └──────────────────────────────────────────┘
  ┌──────────────────────────────────────────┐
  │  VirtualBox Internal Net "scoutlan"      │
  │    ├─ utm:eth1 = 192.168.1.1/24 (LAN)  │
  │    └─ client:eth1 = DHCP (192.168.1.x) │
  └──────────────────────────────────────────┘
```

Traffic flow: `client → eth1 → [intnet] → utm:eth1 → nftables → utm:eth0 → [NAT] → internet`

### What the Client Provisioning Does

1. Installs `curl` and `bind-tools`
2. Gets DHCP lease on eth1 from ScoutUTM's dnsmasq
3. Replaces default route to go through ScoutUTM (192.168.1.1)
4. Keeps eth0 route for Vagrant SSH (10.0.2.x)
5. Runs connectivity checks (ping, DNS, HTTP, HTTPS)

### Limitations

- scoutd runs in `--dev-mode`, so the device scanner returns synthetic devices — the client VM won't appear in the web UI's Devices page
- Internet access depends on VirtualBox NAT on the UTM's eth0 working for forwarded traffic

---

## Placeholder Test Scripts

The following test scripts exist but are not yet implemented (empty files for future months):

| Script | What It Tests | Month |
|--------|--------------|-------|
| `tests/dns_test.sh` | Unbound recursive resolution, DNSSEC, RPZ blocklists, custom allow/block | TBD (with real blocklist data) |
| `tests/rollback_test.sh` | Config apply failure triggers rollback, startup recovery, rapid save serialization | TBD (with multi-daemon rollback) |
| `tests/update_test.sh` | RAUC bundle install, reboot, version check, bad bundle rollback | 9 |
| `tests/qa_test.sh` | Full production QA (hardware, Month 10+) | 10 |
| `tests/run_all.sh` | Run all test scripts in sequence | TBD |

---

## Dev-Mode Behavior in Vagrant VM

scoutd runs with `--dev-mode` in the Vagrant VM. This is needed for:
- **SSH access**: Dev-mode adds firewall rules allowing port 22 on all zones (including WAN where Vagrant SSH arrives via eth0)
- **HTTPS/HTTP on WAN**: Dev-mode allows ports 443 and 80 on WAN for host-to-VM web UI access

Dev-mode stubs:
- **Service reloads**: OpenRC `rc-service restart` calls are stubbed. Integration tests that check nftables state manually reload rules with `nft -f /etc/nftables.conf`.
- **nftables apply**: `nft -f` is stubbed in scoutd but the provisioning script and tests load rules directly.
- **WireGuard service**: `wg syncconf` is stubbed. VPN tests verify config file rendering, not actual tunnel state.

Templates are still rendered to real paths (`--output-dir /etc`), so daemon config files at `/etc/nftables.conf`, `/etc/dnsmasq.conf`, `/etc/chrony/chrony.conf`, `/etc/unbound/unbound.conf`, `/etc/wireguard.conf` are all real and inspectable.

---

## Troubleshooting

### vagrant ssh hangs
The nftables firewall rules may be blocking SSH (port 22). Open the VirtualBox GUI, access the VM console, and run:
```
sudo nft flush ruleset
sudo rc-service sshd restart
```
The dev-mode nftables template includes SSH allow rules, but if scoutd failed to start or render templates, SSH will be blocked.

### Can't access web UI at https://localhost:8443
1. Check scoutd is running: `vagrant ssh -c "rc-service scoutd status"`
2. Check ports: `vagrant ssh -c "ss -tlnp | grep -E ':(80|443) '"`
3. Check logs: `vagrant ssh -c "cat /var/log/scout/scoutd.log | tail -30"`
4. Verify nftables allows port 443: `vagrant ssh -c "nft list chain inet scout input_wan 2>/dev/null"`

### Templates not found in VM
The `build.sh` / `build.ps1` scripts handle this automatically. If building manually, ensure templates are copied:
```bash
cp scoutd/templates/*.tmpl build/templates/
```

### Tests fail with "connection refused"
scoutd may not be running. Inside the VM:
```bash
sudo rc-service scoutd start
sleep 3
curl -sk https://localhost:443/api/v1/health
```
