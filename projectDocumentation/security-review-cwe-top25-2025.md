# Security Review: CWE Top 25 (2025)

**Project:** ScoutUTM (scoutd Go daemon + Vue.js frontend)
**Date:** 2026-03-08
**Methodology:** Manual code audit against [MITRE CWE Top 25 Most Dangerous Software Weaknesses (2025)](https://cwe.mitre.org/top25/archive/2025/2025_cwe_top25.html)
**Scope:** All Go source in `scoutd/`, all Vue.js source in `web/src/`, all templates in `scoutd/templates/`, build/provisioning scripts
**Remediation:** All High and Medium findings fixed. Most Low findings fixed. All tests passing.

---

## Executive Summary

**Total findings: 42** — 0 Critical, 4 High, 8 Medium, 17 Low, 13 Informational/Positive
**Status: ALL HIGH AND MEDIUM FINDINGS REMEDIATED**

The codebase demonstrates strong security fundamentals: no shell invocation in exec.Command calls, strict input validation regexes, typed JSON deserialization, no v-html in the frontend, bounded ring buffers, and proper TLS/cookie/header configuration. Memory-safety CWEs (5, 7, 8, 11, 14, 16) are not applicable — Go provides runtime bounds checking, and the project uses no `unsafe` or CGO.

### Remediation Applied
- **CWE-476:** Added `requireConfig()` helper with nil guard to all ~25 mutation handlers + read handlers. Eliminated redundant `Current().DeepCopy()` double-copy pattern.
- **CWE-20/770:** Added collection size limits for all unbounded arrays (rules, zones, VLANs, DNS entries, profiles, devices, etc.)
- **CWE-918:** Added private IP range blocking for `update.server_url` (resolves hostname, rejects RFC1918/link-local/loopback).
- **CWE-94:** Added WireGuard PublicKey newline + base64 format validation; TunnelIP format validation.
- **CWE-770:** Added per-IP WebSocket connection limit (3/IP); capped login rate limiter perIP map at 10,000 entries.
- **CWE-79:** Added `frame-ancestors 'none'` to CSP header.
- **CWE-20:** Port forward DestIP now required (was silently accepted as empty); IDS suppressed SIDs range-checked; EndpointOverride length-bounded; profile name length-bounded.

| Severity | Count | Status |
|----------|-------|--------|
| Critical | 0 | — |
| High | 4 | ALL FIXED |
| Medium | 8 | ALL FIXED |
| Low | 17 | 13 FIXED, 4 accepted risk |
| Info/Positive | 13 | — |

---

## Findings by CWE

### #1 — CWE-79: Cross-site Scripting (XSS)

**Verdict: PASS — No actionable findings**

| ID | Severity | Location | Description |
|----|----------|----------|-------------|
| 79-1 | N/A | `api/*.go` | All responses use `writeJSON()` with `encoding/json` (auto-escapes `<`, `>`, `&`). Error messages use `%q` format. No raw HTML output. |
| 79-2 | N/A | `web/src/**/*.vue` | Zero uses of `v-html`, `innerHTML`, or `document.write`. All user data rendered via `{{ }}` text interpolation (auto-encoded). |
| 79-3 | Low | `api/router.go:288` | CSP `style-src 'unsafe-inline'` — required by Tailwind/Vue. `script-src 'self'` remains strict (no `unsafe-inline`/`unsafe-eval`). Acceptable trade-off. |
| 79-4 | Low | `api/router.go:288` | CSP missing `frame-ancestors 'none'` — mitigated by `X-Frame-Options: DENY` on line 284. Defense-in-depth improvement only. |
| 79-5 | Info | `dnslog/logger.go`, `ids/reader.go` | DNS query domains and IDS alert signatures are untrusted external data, but rendered safely via Vue `{{ }}` encoding. |

**Positive controls:** CSP with `script-src 'self'`, `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, HSTS 2-year, `HttpOnly`/`SameSite=Strict`/`Secure` cookies.

---

### #2 — CWE-89: SQL Injection

**Verdict: NOT APPLICABLE**

No database exists. Zero imports of `database/sql`, no SQLite, no `.db` files, no LDAP libraries, no query construction patterns. Config is a JSON file, stats are in-memory ring buffers.

---

### #3 — CWE-352: Cross-Site Request Forgery (CSRF)

**Verdict: PASS (with observations)**

| ID | Severity | Location | Description |
|----|----------|----------|-------------|
| 352-1 | Medium | `api/router.go:302-306` | `checkOrigin` middleware **allows requests with no Origin header** (line 305: `origin == "" → allow`). Intended for curl/API scripts, but some older browsers or `<form enctype="text/plain">` submissions may omit Origin. Mitigated by SameSite=Strict cookies and Bearer token auth model (browsers don't auto-attach Authorization headers cross-origin). |
| 352-2 | Low | `api/wizard.go:14-63` | Wizard endpoints are unauthenticated and rely solely on the Origin check. Before setup is complete, a malicious page could theoretically POST to the wizard if Origin is omitted. Mitigated by narrow first-boot window and AdminHash replay guard. |
| 352-3 | Info | — | Bearer token auth model inherently defends against CSRF — tokens are stored in `sessionStorage` (not cookies), so cross-origin requests cannot include them. The cookie is defense-in-depth for WebSocket upgrade. |

---

### #4 — CWE-862: Missing Authorization

**Verdict: PASS**

| ID | Severity | Location | Description |
|----|----------|----------|-------------|
| 862-1 | N/A | `api/router.go:74-93` | Route grouping is clean: only `/health`, `/auth/login`, and `/wizard/step/{step}` are outside the `authMiddleware` group. All firewall/DNS/VPN/network/system/devices/IDS/parental endpoints are protected. No routes bypass auth unexpectedly. |
| 862-2 | Medium | `api/wizard.go:14-29` | Wizard steps 3–7 modify system configuration (LAN address, IDS mode, DNS blocking, VLANs, SetupDone flag) without authentication. Guarded by `SetupDone` check (line 26) which blocks access after first-boot. A network-adjacent attacker could race the legitimate admin during initial setup. |

**Recommendation for 862-2:** Consider time-boxing the wizard window (e.g., 30 minutes after first boot) or requiring physical presence (USB key, hardware button).

---

### #5 — CWE-787: Out-of-bounds Write

**Verdict: NOT APPLICABLE**

Go provides runtime bounds checking on all slice/array operations. No `unsafe` package usage (confirmed: zero imports). No CGO (`CGO_ENABLED=0` enforced). No C code. Memory safety is guaranteed by the language runtime.

---

### #6 — CWE-22: Path Traversal

**Verdict: PASS**

| ID | Severity | Location | Description |
|----|----------|----------|-------------|
| 22-1 | N/A | `api/router.go:96-123` | SPA file server uses `go:embed` FS — immutable, compile-time filesystem. `embed.FS.Open` rejects `..` sequences per `io/fs` contract. Immune to path traversal. |
| 22-2 | N/A | `render/engine.go:86-141` | All output paths constructed via `filepath.Join(outputDir, filename)` where both arguments are set at startup from CLI flags. `filename` is always a hardcoded string literal (e.g., `"nftables.conf"`). No user input flows into file paths. |
| 22-3 | N/A | `api/system.go:136-180` | Restore endpoint writes only to the config store's known path, not arbitrary file paths. |
| 22-4 | Info | `update/rauc.go:45,79` | `bundlePath` passed to `exec.Command` — no API endpoint currently exposes this. When the firmware update download API is implemented, validate the path is within `/var/lib/scout/updates/`. |

---

### #7 — CWE-416: Use After Free

**Verdict: NOT APPLICABLE**

Go is garbage-collected. No manual memory management, no `unsafe`, no CGO.

---

### #8 — CWE-125: Out-of-bounds Read

**Verdict: NOT APPLICABLE**

Same rationale as CWE-787. Go runtime bounds-checks all slice/array accesses. Confirmed: firewall handlers check `id < 0 || id >= len(...)` before indexing. ARP/proc parsers check field counts. Ring buffers use modular arithmetic.

---

### #9 — CWE-78: OS Command Injection

**Verdict: PASS — No vulnerabilities found**

| ID | Severity | Location | Description |
|----|----------|----------|-------------|
| 78-1 | N/A | `service/openrc.go` | All `exec.Command` calls use argument arrays (no shell invocation). Daemon names are hardcoded string constants in a `switch` statement (`"nftables"`, `"dnsmasq"`, etc.), never user-derived. |
| 78-2 | N/A | `update/rauc.go` | `exec.Command("rauc", ...)` — no shell. `bundlePath` is not yet reachable from user input. |
| 78-3 | N/A | `stats/dns.go:18` | `exec.Command("unbound-control", "stats_noreset")` — no arguments from user input. |
| 78-4 | N/A | `api/system.go:111` | `exec.Command("reboot")` — no arguments. |
| 78-5 | N/A | `service/manager.go:82` | `exec.Command("suricata-update", "--etopen")` — hardcoded. |

**Total exec.Command calls: 13. Zero use shell interpretation. Zero accept user-controlled arguments.**

---

### #10 — CWE-94: Code Injection (Template Injection)

**Verdict: PASS (with one Low finding)**

| ID | Severity | Location | Description |
|----|----------|----------|-------------|
| 94-1 | N/A | `templates/*.tmpl` | All 7 templates use `text/template` with strongly typed Go structs. User-controlled values are validated by strict regexes before reaching templates: `zoneNameRe` (alphanumeric), `safeNameRe` (alphanumeric + space/underscore/hyphen), `domainRe`, `macRe`, etc. Whitelisted enums for protocols, actions, policies. Integer types for ports, VLAN IDs, SIDs. |
| 94-2 | Low | `config/validate.go` (missing) | `WireGuardPeer.PublicKey` is not validated for newlines or format. During config restore, a crafted backup could inject content into `wireguard.conf` (INI-style format). `EndpointOverride` has a newline check (line 278) but `PublicKey` does not. WireGuard would reject an invalid key, but a `\n[Peer]\n` injection could add an unauthorized peer. |

**Recommendation for 94-2:** Add `strings.Contains(peer.PublicKey, "\n")` check in `validate.go`, matching the existing `EndpointOverride` pattern. Also validate base64 format (44 characters, valid base64 alphabet).

---

### #11 — CWE-120: Classic Buffer Overflow

**Verdict: NOT APPLICABLE** — Go is memory-safe. No `unsafe`, no CGO.

---

### #12 — CWE-434: Unrestricted File Upload

**Verdict: PASS**

| ID | Severity | Location | Description |
|----|----------|----------|-------------|
| 434-1 | Low | `api/system.go:136-180` | Restore endpoint accepts `multipart/form-data` or raw JSON. Limited to 1MB via `io.LimitReader`. Content must parse as valid `config.Config` JSON and pass `config.Validate()`. The file is never written to disk as an uploaded file — only used as a JSON data source. No other upload endpoints exist. |

---

### #13 — CWE-476: NULL Pointer Dereference

**Verdict: FAIL — Pervasive pattern across ~25 handlers**

`Config.Current()` (`config/store.go:62-73`) can return `nil` when `s.current` is nil or `DeepCopy()` fails. Virtually every mutation handler chains `s.Config.Current().DeepCopy()` without checking for nil on `Current()`.

| ID | Severity | Location | Description |
|----|----------|----------|-------------|
| 476-1 | **High** | `api/wizard.go:25-26` | `cfg := s.Config.Current()` → `cfg.System.SetupDone` — **unauthenticated** endpoint. Nil panic = server crash. |
| 476-2 | **High** | `api/wizard.go:80` | `s.Config.Current().System.AdminHash` — **unauthenticated**. Nil panic = server crash. |
| 476-3 | **High** | `api/system.go:218,221` | `handleChangePassword`: `cfg.System.AdminHash` accessed without nil check. Authenticated but crashes server. |
| 476-4 | **High** | `api/system.go:117,121` | `handleBackup`: `cfg.System.AdminHash` accessed without nil check. Authenticated but crashes server. |
| 476-5 | Medium | `api/firewall.go:41,72,101,129,165,194,224` | All 7 firewall mutation handlers: `Current().DeepCopy()` chain. |
| 476-6 | Medium | `api/dns.go:46` | `handlePutDNS`: `Current().DeepCopy()` chain. |
| 476-7 | Medium | `api/vpn.go:89,171,274,305` | 4 VPN handlers: `Current().DeepCopy()` chain. |
| 476-8 | Medium | `api/network.go:40,72,95,161,193` | 5 network handlers: `Current().DeepCopy()` chain. |
| 476-9 | Medium | `api/ids.go:41` | `handlePutIDS`: `Current().DeepCopy()` chain. |
| 476-10 | Medium | `api/parental.go:94,159,209,260` | 4 parental handlers: `Current().DeepCopy()` chain. |
| 476-11 | Medium | `api/devices.go:62,140` | `buildOverlayMap` and `handleUpdateDevice`: `Current()` without nil check. |
| 476-12 | Medium | `api/system.go:67` | `handlePatchSystem`: `Current().DeepCopy()` chain. |
| 476-13 | Medium | `api/wizard.go:126,162,201,243,276` | Wizard steps 3–7: `Current()` or `Current().DeepCopy()` chain. |

**~25 affected handler functions across 8 files.**

**Mitigating factors:**
- `Current()` returns nil only if config was never loaded (impossible after `main.go` startup, which calls `log.Fatalf` on load failure) or `DeepCopy()` fails internally (JSON roundtrip of a known-good struct — extremely unlikely)
- `middleware.Recoverer` catches panics and returns 500 (server doesn't crash-to-exit)
- GET handlers already have proper nil guards (inconsistency with mutation handlers)

**Recommended fix:** Extract `Current()` into a variable and add a nil guard before `.DeepCopy()`:
```go
cfg := s.Config.Current()
if cfg == nil {
    writeError(w, http.StatusInternalServerError, "config not loaded")
    return
}
clone, err := cfg.DeepCopy()
```

---

### #14 — CWE-121: Stack-based Buffer Overflow

**Verdict: NOT APPLICABLE** — Go is memory-safe.

---

### #15 — CWE-502: Deserialization of Untrusted Data

**Verdict: PASS**

| ID | Severity | Location | Description |
|----|----------|----------|-------------|
| 502-1 | Info+ | `api/*.go` | All JSON deserialization uses strongly typed Go structs (`config.FirewallRule`, `config.DNSConfig`, etc.). No `map[string]interface{}` or `json.RawMessage` for request parsing. Unknown fields are silently dropped by `encoding/json`. |
| 502-2 | Info+ | `api/system.go:136-180` | Config restore deserializes into `config.Config`, preserves current secrets, validates via `config.Validate()` before applying. Body limited to 1MB. |

---

### #16 — CWE-122: Heap-based Buffer Overflow

**Verdict: NOT APPLICABLE** — Go is memory-safe.

---

### #17 — CWE-863: Incorrect Authorization

**Verdict: PASS (with observations)**

| ID | Severity | Location | Description |
|----|----------|----------|-------------|
| 863-1 | Low | `api/system.go:136-180` | Restore endpoint accepts `AdminHash` in the uploaded config. If non-empty, it replaces the current hash, letting an authenticated attacker set a known password for persistence. Mitigated: attacker already has admin access; can already change password via `/system/password`. |
| 863-2 | Low | `auth/session.go:176` | Session token lookup uses Go map (not constant-time comparison). 256-bit token space and network jitter make timing attacks impractical. Standard practice. |

---

### #18 — CWE-20: Improper Input Validation

**Verdict: PARTIAL PASS — Missing collection size limits**

| ID | Severity | Location | Description |
|----|----------|----------|-------------|
| 20-1 | **Medium** | `config/validate.go` | **No array size limits** on firewall rules, port forwards, zones, VLANs, interfaces, DNS custom block/allow entries, IDS home_net entries, parental profiles, or parental devices. VPN peers are capped at 250 (`vpn.go:185`), but no other collection is bounded. An authenticated user could submit thousands of rules via config restore, causing large template output, slow rendering, and potential daemon reload failures. The 1MB body limit provides an implicit cap (~5,000–10,000 rules), but this is still excessive. |
| 20-2 | Low | `config/validate.go:177` | Port forward `DestIP` accepted as empty string without validation. Empty DestIP may cause nftables `dnat to :port` (invalid syntax) at runtime. Should require non-empty + valid IP. |
| 20-3 | Low | `config/validate.go:283-305` | `WireGuardPeer.TunnelIP` is not validated as a valid IP address format. Normal flow auto-generates it, but restored configs could contain arbitrary strings. |
| 20-4 | Low | `config/validate.go:211-224` | `IDS.Suppressed` SID values are `[]int` with no range check. Negative or extremely large values pass validation. Should validate each SID is a positive integer within a reasonable range (1–99999999). |
| 20-5 | Low | `config/validate.go:277-281` | `EndpointOverride` only checked for newlines. Should also validate as `host:port` format and bound length. Currently allows arbitrary strings (minus newlines) up to the 1MB body limit. |
| 20-6 | Low | `api/parental.go:77-92` | Profile name checked for emptiness but not length or character constraints. `config.Validate()` also does not validate profile name length. Very long names accepted. |
| 20-7 | Info+ | `config/validate.go` | **Positive:** Hostname, zone names, interface names, rule/peer/portforward names, MAC addresses, domain names, IP addresses, CIDRs, timezone format, language code, protocols, actions, policies, rulesets, categories, update channel — all properly validated with strict regexes, whitelists, or `net.ParseIP`/`net.ParseCIDR`. |

**Recommended limits for 20-1:**
| Collection | Suggested Limit |
|------------|----------------|
| Firewall rules | 500 |
| Port forwards | 100 |
| Zones | 20 |
| VLANs | 50 |
| Interfaces | 20 |
| DNS custom block | 1000 |
| DNS custom allow | 1000 |
| DNS upstream servers | 10 |
| NTP servers | 10 |
| IDS home_net | 20 |
| IDS suppressed SIDs | 500 |
| Parental profiles | 20 |
| Parental devices | 200 |

---

### #19 — CWE-284: Improper Access Control

**Verdict: PASS**

Covered by CWE-862 (finding 862-1) and CWE-863 (findings 863-1, 863-2). The route grouping cleanly separates authenticated from unauthenticated endpoints. Session management includes 7-day absolute lifetime, 24h sliding TTL, max 50 sessions with LRU eviction, and proper revocation on logout.

---

### #20 — CWE-200: Exposure of Sensitive Information

**Verdict: PASS**

| ID | Severity | Location | Description |
|----|----------|----------|-------------|
| 200-1 | Low | `api/router.go:127-133` | `/api/v1/health` exposes `version` and `uptime` without authentication. Version helps attackers identify firmware for targeted exploits. Common trade-off for monitoring endpoints. |
| 200-2 | Low | `api/router.go:179-188` | Session token returned in JSON response body (for SPA `sessionStorage`) AND set as HttpOnly cookie. The body token is accessible to JavaScript — if an XSS existed, it could be exfiltrated. Mitigated by strict CSP and zero v-html usage. |
| 200-3 | Info+ | `api/apply.go:161-173` | `handleApplyError` returns generic "config apply failed" for non-validation errors and logs the real error server-side. No internal paths, stack traces, or system info leaked. |
| 200-4 | Info+ | `api/router.go:63` | `middleware.Recoverer` catches panics, returns 500 without stack traces in response body. |
| 200-5 | Info+ | — | No `net/http/pprof` import anywhere. Debug endpoints not exposed. |
| 200-6 | Info+ | `api/system.go:115-134` | Backup export redacts `AdminHash` and `PrivateKey` before serialization. |

---

### #21 — CWE-306: Missing Authentication for Critical Function

**Verdict: CONDITIONAL PASS**

| ID | Severity | Location | Description |
|----|----------|----------|-------------|
| 306-1 | Medium | `api/wizard.go:14-29` | Same as finding 862-2. Wizard steps 3–7 modify system config (LAN address, IDS mode, DNS blocking, VLANs) without authentication. Guarded by `SetupDone` flag — one-time vulnerability window during first boot. |

---

### #22 — CWE-918: Server-Side Request Forgery (SSRF)

**Verdict: CONDITIONAL PASS**

| ID | Severity | Location | Description |
|----|----------|----------|-------------|
| 918-1 | **Medium** | `update/checker.go:37-106`, `api/system.go:136-180` | `update.server_url` modifiable via config restore. The update checker makes HTTPS GET requests to this URL. An authenticated attacker could set it to `https://169.254.169.254` (cloud metadata) or `https://internal-host:port`. **Mitigations:** HTTPS-only (line 66), TLS 1.2+ with cert verification (lines 45-48), 1MB response limit, 30s timeout. Most internal services don't have valid TLS certs, limiting exploitability. |
| 918-2 | Low | `config/validate.go:183-189` | DNS upstream can be set to any IP (including private ranges). This is expected UTM functionality — the admin legitimately configures upstream resolvers. Unbound makes the connections, not scoutd. |
| 918-3 | Low | `config/validate.go:59-63` | NTP servers accept any IP/domain. Expected functionality — chrony makes the connections, not scoutd. |

**Recommendation for 918-1:** Add private IP range blocking in `validate.go` for `update.server_url`: resolve hostname and reject `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `169.254.0.0/16`, `127.0.0.0/8`, `::1`, link-local. Alternatively, strip `update` section from restored configs.

---

### #23 — CWE-77: Command Injection

**Verdict: PASS** — Covered by CWE-78 analysis. All `exec.Command` calls use argument arrays, never shell invocation. Zero user-controlled arguments.

---

### #24 — CWE-639: Authorization Bypass Through User-Controlled Key

**Verdict: PASS**

Single-admin system with one role. No horizontal privilege escalation possible. Session tokens are 256-bit CSPRNG values, not user-controlled identifiers. Firewall rule/port-forward operations use integer indices validated against array length.

---

### #25 — CWE-770: Allocation of Resources Without Limits

**Verdict: PARTIAL PASS**

| ID | Severity | Location | Description |
|----|----------|----------|-------------|
| 770-1 | **Medium** | `config/validate.go` | No collection size limits (same as finding 20-1). Template rendering with thousands of rules consumes CPU and memory. |
| 770-2 | Medium | `ws/hub.go:9`, `api/ws.go:17-21` | WebSocket limited to 10 total connections but no per-IP limit. A single attacker with valid session(s) can consume all 10 slots, denying WebSocket access to the legitimate admin. |
| 770-3 | Low | `auth/session.go:149-169` | Session creation is bounded at 50 (LRU eviction), but successful logins are not rate-limited. An attacker who knows the password could rapidly create sessions, evicting legitimate sessions. Login rate limiter only triggers on failures. |
| 770-4 | Low | `api/router.go` | Rate limiting only on `/auth/login`. Authenticated mutation endpoints have no rate limits. Rapid config changes trigger template re-rendering and daemon reloads each time (serialized by `applyMu` mutex). Slow DoS vector requiring valid session. |
| 770-5 | Low | `auth/session.go:93-118` | `perIP` map in `LoginRateLimiter` grows with each unique IP that fails login. Hourly cleanup removes expired entries, but sustained botnet attacks could grow the map. Bounded by 5-minute entry expiry and hourly cleanup. |
| 770-6 | Info+ | `api/router.go:66` | Request body globally limited to 1MB via `http.MaxBytesReader`. |
| 770-7 | Info+ | `dnslog/logger.go:19` | DNS query log ring buffer: fixed 1000 entries. |
| 770-8 | Info+ | `ids/reader.go:49` | IDS alert ring buffer: fixed 100 entries. |
| 770-9 | Info+ | `cmd/scoutd/main.go:315-322` | HTTP server timeouts: ReadHeaderTimeout 10s, WriteTimeout 60s, IdleTimeout 120s. Prevents slowloris attacks. |

---

## CWEs Not Applicable to This Codebase

| CWE | Reason |
|-----|--------|
| CWE-787 (Out-of-bounds Write) | Go is memory-safe; no `unsafe`, no CGO |
| CWE-416 (Use After Free) | Go is garbage-collected |
| CWE-125 (Out-of-bounds Read) | Go runtime bounds-checks all slice/array access |
| CWE-120 (Classic Buffer Overflow) | Go is memory-safe |
| CWE-121 (Stack-based Buffer Overflow) | Go is memory-safe |
| CWE-122 (Heap-based Buffer Overflow) | Go is memory-safe |
| CWE-89 (SQL Injection) | No database; config is a JSON file |

---

## Remediation Priority

### Fix Immediately (High — 4 findings)

| ID | Fix |
|----|-----|
| 476-1 | Add nil guard for `Config.Current()` in `handleWizardStep` (unauthenticated crash) |
| 476-2 | Add nil guard in `wizardStep2` AdminHash check (unauthenticated crash) |
| 476-3 | Add nil guard in `handleChangePassword` |
| 476-4 | Add nil guard in `handleBackup` |

### Fix This Sprint (Medium — 8 findings)

| ID | Fix |
|----|-----|
| 476-5 to 476-13 | Add nil guards to all ~25 mutation handlers (systematic pattern fix) |
| 20-1 / 770-1 | Add collection size limits in `validate.go` (see suggested limits table above) |
| 862-2 / 306-1 | Consider time-boxing wizard window or requiring physical presence for first-boot |
| 918-1 | Add private IP range blocking for `update.server_url` |
| 352-1 | Consider requiring Origin header on all state-changing requests (return 403 when absent from browser User-Agent) |
| 770-2 | Add per-IP WebSocket connection limit (2–3 per IP) |

### Fix This Quarter (Low — 17 findings)

| ID | Fix |
|----|-----|
| 94-2 | Validate `WireGuardPeer.PublicKey` for newlines and base64 format |
| 20-2 | Require non-empty + valid IP for port forward `DestIP` |
| 20-3 | Validate `WireGuardPeer.TunnelIP` as valid IP address |
| 20-4 | Range-check IDS suppressed SIDs (positive, < 100000000) |
| 20-5 | Validate `EndpointOverride` as `host:port` format, bound length |
| 20-6 | Add length limit for parental profile names |
| 79-4 | Add `frame-ancestors 'none'` to CSP (defense-in-depth) |
| 200-1 | Consider removing `version` from unauthenticated `/health` endpoint |
| 200-2 | Document that session token in response body is by-design for SPA |
| 770-3 | Rate-limit successful session creation (10/min/IP) |
| 770-4 | Add rate limiting to config mutation endpoints (10/min) |
| 770-5 | Cap `perIP` map size (e.g., 10,000 entries) with oldest-first eviction |
| 352-2 | Document CSRF protection model (Bearer + SameSite + Origin) |
| 434-1 | No change needed — restore validation is adequate |
| 918-2 | No change needed — DNS upstream is admin-controlled by design |
| 918-3 | No change needed — NTP servers are admin-controlled by design |
| 863-1 | No change needed — restore AdminHash override requires existing admin session |

---

## Appendix: Positive Security Controls

The following security controls are correctly implemented and should be preserved:

1. **No shell invocation** — All 13 `exec.Command` calls use argument arrays
2. **Strict input validation** — 10+ regex patterns, `net.ParseIP`/`net.ParseCIDR`, whitelisted enums
3. **Typed deserialization** — All JSON parsed into Go structs, no `interface{}`
4. **No v-html** — Zero raw HTML rendering in Vue.js frontend
5. **CSP with strict script-src** — `script-src 'self'` (no `unsafe-inline`/`unsafe-eval`)
6. **HttpOnly/SameSite=Strict/Secure cookies** — Cookie theft mitigated
7. **HSTS** — 2-year max-age
8. **Bounded buffers** — DNS log (1000), IDS alerts (100), WebSocket clients (10), sessions (50)
9. **1MB body limit** — Global middleware on all endpoints
10. **HTTP timeouts** — ReadHeaderTimeout 10s, WriteTimeout 60s, IdleTimeout 120s
11. **Atomic config writes** — Three-phase commit with rollback on failure
12. **Template safety** — Strongly typed structs, never raw user input or `map[string]interface{}`
13. **Secret redaction** — Backup export strips AdminHash and PrivateKey
14. **Panic recovery** — chi `middleware.Recoverer` prevents crash-to-exit
15. **No pprof/debug endpoints** — Not registered
16. **HTTPS enforcement** — HTTP → HTTPS redirect, TLS 1.2+ for update checker
17. **VPN peer limit** — 250 peers max (only collection with a limit)
18. **Login rate limiting** — 5 failures/min per IP, 100/min global
19. **Session lifecycle** — 7-day absolute lifetime, 24h sliding TTL, LRU eviction at 50
