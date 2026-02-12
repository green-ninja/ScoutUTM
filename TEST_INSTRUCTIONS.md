# ScoutUTM Test Instructions

This document explains how to build, test, and develop ScoutUTM across both tiers of the development environment.

## Prerequisites

- **Go 1.21+** installed (on this system: `/usr/local/go/bin/go`, in WSL only)
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
# All tests across all packages (170 tests, 5 packages)
cd scoutd && /usr/local/go/bin/go test ./...

# Single package:
cd scoutd && /usr/local/go/bin/go test ./internal/config/...
cd scoutd && /usr/local/go/bin/go test ./internal/render/...
cd scoutd && /usr/local/go/bin/go test ./internal/api/...
cd scoutd && /usr/local/go/bin/go test ./internal/service/...

# Verbose output:
cd scoutd && /usr/local/go/bin/go test -v ./internal/api/...

# Run a specific test:
cd scoutd && /usr/local/go/bin/go test -v -run TestCreatePeer_SplitTunnel ./internal/api/...
```

**What the unit tests cover (170 tests):**
- `internal/config/` -- Config loading, saving, validation, schema migrations
- `internal/render/` -- Template rendering for all daemon configs (nftables, dnsmasq, chrony, unbound, wireguard, suricata)
- `internal/api/` -- REST API handlers via httptest (firewall CRUD, network/zones CRUD, VLANs, VPN/WireGuard, system, auth, config apply)
- `internal/service/` -- Service manager dependency ordering
- `internal/auth/` -- Session management

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

The Vagrantfile configures:
- **Box:** `generic/alpine319` (Alpine Linux 3.19)
- **Resources:** 4 CPUs, 8 GB RAM
- **Port forwards:** host `8443` -> guest `443` (HTTPS), host `8080` -> guest `80` (HTTP)
- **Private network:** `192.168.56.10`
- **Synced folders:** `build/` -> `/opt/scout-build`, `tests/` -> `/opt/scout-tests`

Provisioning (`tools/vagrant-provision.sh`) does:
1. Flushes nftables rules (SSH protection)
2. Installs Alpine packages (nftables, unbound, dnsmasq, chrony, wireguard-tools, etc.)
3. Installs scoutd binary and templates from `/opt/scout-build`
4. Installs default config, sets admin password to `admin`
5. Disables RPZ blocklists (no real blocklist data in dev VM)
6. Creates OpenRC service with `--dev-mode` and explicit paths
7. Starts scoutd (renders all daemon configs on startup)
8. Loads nftables rules, starts all daemons
9. Runs all integration tests automatically

### Accessing the VM

```bash
vagrant ssh
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

To re-run tests manually inside the VM:

```bash
vagrant ssh
/opt/scout-tests/smoke_test.sh
/opt/scout-tests/api_test.sh
/opt/scout-tests/network_test.sh
/opt/scout-tests/vlan_test.sh
/opt/scout-tests/vpn_test.sh
```

### Rebuilding the VM from Scratch

```powershell
.\build.ps1
```

---

## Integration Test Details

Each test script is self-contained and idempotent -- it cleans up any resources it creates. Tests that check nftables state reload rules via `nft -f /etc/nftables.conf` since scoutd runs in dev-mode (stubs service reloads).

### smoke_test.sh (10 checks)
Verifies the basics are working:
1. Health endpoint returns `healthy=true`
2. Version field present in health response
3. Config file exists at `/etc/scout/config.json` and is valid JSON
4. Config has all top-level keys (version, system, network, firewall, dns, vpn, ids, parental, update)
5. Services running: nftables, dnsmasq, chronyd, scoutd
6. Suricata status (warning if not running -- needs rules, Month 5)
7. Unbound running
8. TLS certificate and key exist

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

---

## Placeholder Test Scripts

The following test scripts exist but are not yet implemented (empty files for future months):

| Script | What It Tests | Month |
|--------|--------------|-------|
| `tests/dns_test.sh` | Unbound recursive resolution, DNSSEC, RPZ blocklists, custom allow/block | 5 (with real blocklist data) |
| `tests/ids_test.sh` | Suricata alert on test signature, eve.json socket, fail-open | 5 |
| `tests/rollback_test.sh` | Config apply failure triggers rollback, startup recovery, rapid save serialization | 5 (with multi-daemon rollback) |
| `tests/stress_test.sh` | nmap alert storm, WebSocket rate limiting, no OOM or crash | 7 |
| `tests/update_test.sh` | RAUC bundle install, reboot, version check, bad bundle rollback | 9 |
| `tests/wizard_test.sh` | Setup wizard end-to-end: all 7 steps | 7 |
| `tests/qa_test.sh` | Full production QA (hardware, Month 10+) | 10 |
| `tests/run_all.sh` | Run all test scripts in sequence | 5 |

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
