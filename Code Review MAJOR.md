# Code Review — ScoutUTM (through Month 8)

> Reviewed: 2026-03-02 | Scope: All Go source (51 files, 7,610 LOC), all Go tests (32 files, 6,243 LOC), all Vue.js frontend (23 components/views, 11 API modules), all templates (7 `.tmpl` files), all integration tests (15 scripts), PLAN.md, GTM.md, RISKS.md

---

## 1. PLAN.md Alignment — Deliverables Checklist (Months 1–8)

### Fully Implemented

| Deliverable | Status |
|---|---|
| scoutd skeleton (main.go, config schema, store) | Done |
| nftables template rendering + NAT/masquerade | Done |
| dnsmasq DHCP (+ static DHCP for parental) | Done |
| Unbound (recursive + DNSSEC + RPZ + parental tags) | Done |
| Three-phase config commit with rollback | Done |
| VLAN creation + zone assignment | Done |
| WireGuard VPN + peers + QR codes + split/full tunnel | Done |
| chrony NTP template | Done |
| Suricata NFQUEUE + EVE socket + alert pipeline | Done |
| Vue.js 11 views (Login through Setup Wizard) | Done |
| WebSocket real-time stats + device list + alerts | Done |
| Parental per-device DNS filtering (access-control-tag) | Done |
| Parental schedules (60s enforcement) | Done |
| SafeSearch + YouTube restricted mode (DNS CNAME) | Done |
| DNS query logging (Unbound tail + ring buffer) | Done |
| Device fingerprinting (OUI lookup, ~886 vendors) | Done |
| Automatic static DHCP reservations | Done |
| Setup wizard (7 steps) | Done |
| 284 unit tests passing, 0 failures, go vet clean | Done |

### Missing or Incomplete

| Item | PLAN.md Requirement | Current State | Severity |
|---|---|---|---|
| **dns-blocklist-update scheduler task** | Daily at 03:30 — downloads RPZ blocklist files, writes to `/var/lib/scout/blocklists/`, reloads Unbound | **Not registered in main.go.** Only `suricata-update`, `eve-log-rotate`, `session-cleanup`, `dns-log-rotate`, and `parental-schedule-check` are registered. | **HIGH** |
| **ota-check scheduler task** | Daily at 04:00 — GETs manifest.json from update server | **Not registered.** `internal/update/` is a complete stub (1-line `rauc.go`, 12-line `checker.go` with no methods). | MEDIUM (Month 9+ deliverable) |
| **fake-hwclock-save scheduler task** | Hourly — writes time to `/etc/scout/fake-hwclock.data` | **Not registered.** | LOW (hardware-dependent) |
| **disk-check scheduler task** | Hourly — checks /data partition usage, warns at 90%, prunes at 95% | **Not registered.** | LOW (hardware-dependent) |
| **rollback_test.sh** | nftables reject rollback, Unbound bad config rollback, SIGKILL recovery, multi-daemon rollback, rapid serialization | **Empty file (0 bytes).** Go unit test `TestPutDNS_RapidConcurrentChanges` covers serialization, but the Vagrant integration test is missing. | MEDIUM |
| **dns_test.sh** | Unbound recursive, DNSSEC, RPZ blocklist blocks, custom allow/block | **Empty file (0 bytes).** | MEDIUM |
| **run_all.sh** | Test runner | **Empty file (0 bytes).** | LOW |
| **DHCP options fingerprinting** | GTM: "Device fingerprinting (OUI manufacturer lookup, DHCP options)" | **OUI lookup is implemented. DHCP options fingerprinting is not.** | LOW |
| **DNS query log viewer in frontend** | Backend `GET /dns/query-log` exists and returns entries | **No UI to view query logs.** DnsView only has the toggle switch. | MEDIUM |
| **"Reset to Previous Config" button** | PLAN 4.4: "Dashboard Settings page includes a Reset to Previous Config button" | **Not in SystemView.vue.** Backup/restore exists, but no one-click rollback to `config.json.bak`. | MEDIUM |

### Scheduler Task Summary

| PLAN.md Task | Registered in main.go? |
|---|---|
| suricata-update (daily 03:00) | YES |
| dns-blocklist-update (daily 03:30) | **NO** |
| ota-check (daily 04:00) | **NO** (update package is stub) |
| eve-log-rotate (hourly) | YES |
| dns-log-rotate (hourly) | YES |
| fake-hwclock-save (hourly) | **NO** |
| parental-schedule-check (60s) | YES |
| disk-check (hourly) | **NO** |
| session-cleanup (hourly) | YES (not in PLAN but good practice) |

---

## 2. GTM.md Alignment — Marketing Claims vs Code

| GTM Claim | Code Reality | Risk |
|---|---|---|
| "Automatic threat intelligence updates (Suricata rules daily, DNS blocklists daily)" | Suricata-update task exists. **DNS blocklist update task does NOT exist.** | **HIGH** — Half of this claim is unsupported. |
| "Plain-language security alerts" (e.g. "Your smart TV tried to contact a known malware server") | IDS alerts pass through raw Suricata EVE fields (signature, category, severity). **No translation layer to human-friendly messages.** | **MEDIUM** — GTM explicitly uses this example. Alert `Message` field from Suricata is a technical signature name, not plain language. |
| "Per-device content filtering via DNS blocklists" | Fully implemented via Unbound access-control-tag + RPZ. | OK |
| "Screen time schedules and bedtime enforcement" | Schedule enforcement blocks all DNS (access-control deny), effectively blocking internet. | OK |
| "SafeSearch enforcement" + "YouTube restricted mode" | Implemented via CNAME overrides. YouTube uses `restrict.youtube.com` (strict). | OK |
| "QR code peer configuration" | Implemented via skip2/go-qrcode. Server-side generation (private key never in browser). | OK |
| "Real-time dashboard (traffic stats, device list, IDS alerts)" | All three push via WebSocket. Stats every 2s (5s when >5 clients). Device push on change. Alert push on event. | OK |
| "DNS query activity reporting (domains only, not URLs)" | Implemented. Log Entry has `Domain` field only, no URL/path. `DeviceMAC` included per PLAN format. | OK |
| "No cloud dependency for core functions" | No outbound calls except Suricata rule updates (optional). All core functions local. | OK |
| "No telemetry without consent" | Zero telemetry code found anywhere. | OK |
| "No subscription fees for core security features" | No license check, no subscription logic. All features unconditionally available. | OK |
| "Set it up in 15 minutes" | 7-step wizard exists and works. Realistic for the happy path. | OK |
| "IDS mode by default, IPS opt-in" | Default config sets `IDS.Mode: "ids"`. UI has mode selector. | OK |
| "Device fingerprinting (OUI manufacturer lookup, DHCP options)" | **OUI only. DHCP options parsing is not implemented.** | LOW — OUI covers the common case. |
| "VLAN support with guided segmentation wizard" | Wizard step 6 is a skip checkbox, not a guided VLAN creation flow. NetworkView has full VLAN CRUD separately. | LOW |

---

## 3. Code Security

### Strengths

- **Session tokens:** 256-bit crypto/rand, hex-encoded. Strong.
- **Password hashing:** bcrypt. Industry standard.
- **CSRF protection:** SameSite=Strict cookies + Origin header check middleware.
- **Request body limit:** 1MB via `MaxBytesReader`. Prevents request flooding.
- **Token redaction:** Query param tokens redacted from access logs before logging.
- **TLS:** ECDSA P-256, MinVersion TLS 1.2. Self-signed cert with 10-year validity.
- **TLS key permissions:** Written with 0600. Correct.
- **HTTP-to-HTTPS redirect:** Properly implemented.
- **No admin hash exposure:** `systemResponse` struct explicitly excludes `AdminHash`.
- **WebSocket:** Max 10 clients, origin validation, idle timeout.
- **No XSS vectors:** Vue auto-escapes, no `v-html` usage found anywhere in frontend.
- **Template safety:** All template data structs use concrete Go types, not `map[string]interface{}`. Matches PLAN.md's template safety rule.

### Issues

#### S3. `RemoteAddr` includes port — login rate limiting is bypassed

**Severity: HIGH**

In `router.go:133`, the login handler uses `ip := r.RemoteAddr` as the rate limit key. `RemoteAddr` is `IP:port` (e.g., `192.168.1.50:52341`), not just the IP address. Each new TCP connection from the same client IP gets a different rate limit key, **completely bypassing per-IP rate limiting**.

An attacker can send unlimited login attempts from a single IP because every connection uses a different source port.

**Fix:** Strip the port before rate limiting:
```go
ip, _, _ := net.SplitHostPort(r.RemoteAddr)
```

#### S1. Backup endpoint exposes WireGuard private key

`handleBackup` marshals the entire `config.Config` including `VPN.WireGuard.PrivateKey`. PLAN.md and RISKS.md explicitly acknowledge this as an intentional trade-off (RISKS M15): "backup must include WG private key for restore to work." The frontend shows a confirmation dialog warning the user.

**Status:** Accepted risk, properly mitigated per RISKS.md.

#### S2. Backup includes AdminHash (bcrypt)

The bcrypt hash of the admin password is included in the backup file. While not the raw password, it could be used for offline brute-force attacks. The frontend confirmation dialog is the primary mitigation.

**Status:** Accepted risk, consistent with PLAN/RISKS.

---

## 4. Business Logic

### Validated (correct)

- **Parental schedule enforcement:** Correct handling of overnight windows (21:00-07:00), same-day windows, day-of-week, empty days = every day.
- **Static DHCP:** Auto-populated when device is assigned to profile via `lookupDeviceIP`.
- **Unbound tags:** Sorted categories for deterministic RPZ output (fixed in previous review).
- **Safe search:** Google, Bing, DuckDuckGo, YouTube strict mode per PLAN.
- **Three-phase commit:** Stage pending -> render + reload -> commit. Rollback on failure re-renders ALL templates from live config. Matches PLAN 4.4.
- **Affected daemon detection:** `reflect.DeepEqual` correctly identifies which daemons need reload.
- **VPN key generation:** Curve25519 via `crypto/ecdh`. Tunnel IP allocation scans existing peers.
- **WireGuard QR codes:** Server-side generation, private key shown once then discarded.
- **Config serialization:** `applyMu sync.Mutex` serializes concurrent applies. Tested via `TestPutDNS_RapidConcurrentChanges`.

### Issue

#### B1. `prevBlockedIPs` closure has potential data race

In `main.go:263`, the `prevBlockedIPs` variable is captured by closure in the parental-schedule-check task function. The scheduler runs tasks as goroutines (`go func() { task.Fn(ctx) }()`). If the parental-schedule-check takes longer than 60s (extremely unlikely but possible), two invocations could race on `prevBlockedIPs`.

**Severity:** LOW — The function completes in < 1ms in practice.

**Fix:** Either guard with a mutex, or have the scheduler skip a task if its previous invocation is still running.

---

## 5. Code Quality

### Strengths

- **Minimal dependencies.** 4 direct deps (chi, crypto, go-qrcode, websocket), 9 total. Exactly right for an embedded appliance.
- **Zero panic/os.Exit calls** in library code. All error handling through returns.
- **Zero TODO/FIXME/HACK comments.** Codebase is clean of technical debt markers.
- **82% test-to-source ratio** by line count (6,243 test LOC / 7,610 source LOC).
- **Consistent API handler pattern** across all resources: decode -> deep copy -> mutate -> applyConfig -> respond.
- **Frontend:** All 11 PLAN.md views implemented. No hardcoded credentials, no XSS vectors, proper token handling.

### Issues

#### Q1. Stale "Stub" comments in `openrc.go`

Three comments say "Stub -- will be implemented in Month N" but the code is actually implemented:
```go
case Unbound:
    // Stub -- will be implemented in Month 3.
    return runCmd("unbound-control", "reload")
```

**Fix:** Remove the outdated comments.

#### Q2. Dead code: `handleNotImplemented` in `router.go:219-222`

Defined but never called. No route references it.

**Fix:** Remove.

#### Q3. Vestigial file: `internal/api/auth.go`

Contains only a package declaration and a comment redirecting to `router.go`. Serves no purpose.

**Fix:** Remove.

#### Q4. `scheduler/scheduler.go:parseHHMM` missing digit validation

The scheduler's `parseHHMM` (line 99-111) does not validate that characters are digits before performing arithmetic. The `parental/schedule.go` version was fixed in the previous review session to add `isDigit()` checks, but the scheduler's copy was not updated.

```go
// Current — no digit check, could produce wrong values from non-digit ASCII
h := int(s[0]-'0')*10 + int(s[1]-'0')
```

**Fix:** Add digit validation to match the parental/schedule.go version.

#### Q5. Global login rate limit deviates from PLAN.md

PLAN.md 6.4 specifies "Global limit of 20 failed attempts per minute." Code uses `maxGlob: 100` with an explicit comment: "Global limit is set high (100/min) to avoid a DoS amplification vector where an attacker hitting the global limit locks out legitimate users."

**Status:** Deliberate, documented deviation. The reasoning is sound — not a bug.

---

## 6. Integration Tests Gap Analysis

| Script | Required by PLAN | Status |
|---|---|---|
| smoke_test.sh | Month 1 | Implemented (2.6 KB) |
| network_test.sh | Month 2 | Implemented (2.3 KB) |
| api_test.sh | Month 2 | Implemented (4.1 KB) |
| vlan_test.sh | Month 4 | Implemented (5.6 KB) |
| vpn_test.sh | Month 4 | Implemented (8.1 KB) |
| dns_test.sh | Month 5 | **Empty (0 bytes)** |
| rollback_test.sh | Month 5 | **Empty (0 bytes)** |
| ids_test.sh | Month 5 | Implemented (4.9 KB) |
| stress_test.sh | Month 7 | Implemented (3.6 KB) |
| wizard_test.sh | Month 7 | Implemented (3.9 KB) |
| parental_test.sh | Month 8 | Implemented (6.3 KB) |
| run_all.sh | Runner | **Empty (0 bytes)** |
| qa_test.sh | QA | **Empty (0 bytes)** |
| update_test.sh | OTA | **Empty (0 bytes)** |

**8 of 11 PLAN-required test scripts are implemented.** `dns_test.sh` and `rollback_test.sh` are overdue.

---

## 7. Strategic Alignment (GTM Business Goals)

### Aligned

- **"No subscription ever"** — No subscription check, no license validation, no cloud dependency. Architecturally enforced.
- **"Prosumer sweet spot"** — Enterprise feature set (IDS, VLANs, DNSSEC, WireGuard, parental) with consumer-friendly UI (wizard, toggles, dashboard).
- **"Honest performance claims"** — No throughput numbers in the code or UI. Benchmark-gated per PLAN.md.
- **"Privacy-respecting"** — All data local, DNS logs record domains only, zero telemetry, sessions in-memory only.
- **"Family-first design"** — Parental controls are deep and first-class (profiles, categories, schedules, safe search, per-device).
- **EU sovereignty** — No cloud dependency, no US vendor lock-in.

### Concerns

#### G1. DNS blocklist staleness risk

The GTM's core competitor differentiator is "automatic threat intelligence updates." Without the `dns-blocklist-update` scheduler task, RPZ blocklists are never refreshed after initial setup. A deployed ScoutUTM would have stale malware/phishing blocklists within weeks.

**Impact:** Directly undermines the core value proposition. Every competitor (Firewalla, pfSense + pfBlockerNG, Pi-hole) auto-updates their blocklists.

#### G2. Plain-language alerts are missing

The GTM specifically highlights "Your smart TV tried to contact a known malware server" as a key differentiator vs pfSense/OPNsense ("no plain-language alerts"). Current alerts show raw Suricata signature text (e.g., `ET MALWARE Win32/Kryptik.ABCD`). This is the exact "enterprise complexity" the product claims to eliminate.

**Impact:** Reduces the "family simplicity" value proposition for the primary target audience.

**Recommendation:** Add a category-to-message mapping for the top 20-30 Suricata alert categories. Doesn't need to be exhaustive — cover the common patterns (malware callback, phishing, port scan, etc.) and fall back to the raw signature for uncommon alerts.

#### G3. DNS query log viewer missing from frontend

The backend records queries and has a working API endpoint (`GET /dns/query-log`). The GTM promises "DNS query activity reporting." But the frontend has no way to view this data. Parents cannot see what domains their kids' devices are accessing — which is the primary use case for query logging in the parental controls context.

#### G4. "Reset to Previous Config" button missing

PLAN.md 4.4 explicitly specifies: "The dashboard Settings page includes a Reset to Previous Config button that applies config.json.bak via the same three-phase commit flow. This gives users a one-click escape hatch if they lock themselves out." This safety feature is not in SystemView.vue.

---

## 8. Summary

### Must Fix (before Beta QA)

| # | Finding | Severity | Effort |
|---|---|---|---|
| **S3** | `RemoteAddr` includes port — login rate limiting is completely bypassed | HIGH | 1 line |
| **G1** | `dns-blocklist-update` scheduler task not implemented — blocklists go stale | HIGH | ~50 lines |

### Should Fix (before launch)

| # | Finding | Severity | Effort |
|---|---|---|---|
| **G3** | DNS query log viewer missing from frontend | MEDIUM | ~100 lines (new tab/view) |
| **G4** | "Reset to Previous Config" button missing from SystemView | MEDIUM | ~30 lines |
| **G2** | Plain-language alert translation missing | MEDIUM | ~100 lines (mapping table) |
| -- | `dns_test.sh` and `rollback_test.sh` integration tests are empty | MEDIUM | ~200 lines each |

### Should Clean Up

| # | Finding | Severity | Effort |
|---|---|---|---|
| **Q1** | Stale "Stub" comments in `openrc.go` | LOW | Delete 3 comments |
| **Q2** | Dead code `handleNotImplemented` in `router.go` | LOW | Delete 3 lines |
| **Q3** | Vestigial `auth.go` file (3 lines, no code) | LOW | Delete file |
| **Q4** | `scheduler/parseHHMM` missing digit validation (inconsistent with parental/schedule.go) | LOW | 4 lines |
| **B1** | `prevBlockedIPs` closure has theoretical data race | LOW | Add mutex or skip-if-running |

### Accepted / Deferred (not bugs)

| Item | Rationale |
|---|---|
| OTA update package stub (`internal/update/`) | Month 9+ deliverable |
| fake-hwclock-save, disk-check scheduler tasks | Hardware-dependent, Month 9-10 |
| DHCP options fingerprinting | OUI covers the common case |
| Wizard step 6 VLAN simplified | NetworkView has full CRUD |
| Global rate limit 100 vs PLAN's 20 | Deliberate, documented deviation with sound reasoning |
| Backup includes WG private key + admin hash | Intentional, documented in RISKS.md M15 |

### Metrics

| Metric | Value |
|---|---|
| Go source files | 51 (7,610 LOC) |
| Go test files | 32 (6,243 LOC) |
| Test-to-source ratio | 82% |
| Vue components/views | 23 |
| Tests passing | 284 / 284 |
| Go vet issues | 0 |
| Direct Go dependencies | 4 |
| Hardcoded secrets found | None |
| TODO/FIXME/HACK comments | None |
| Integration test scripts (of 11 required) | 8 implemented, 3 empty |
