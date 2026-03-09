# OWASP Top 10 2025 — Security Code Review Report

**Project:** ScoutUTM (`scoutd` daemon + build system)
**Date:** 2026-03-08
**Scope:** Full codebase review — Go backend, Vue.js frontend dependencies, build scripts, Vagrant provisioning, OTA update system
**Methodology:** 10 independent parallel reviews, one per OWASP 2025 category

---

## Executive Summary

ScoutUTM's `scoutd` daemon demonstrates strong security fundamentals for a solo-founder embedded appliance: 256-bit CSPRNG session tokens, bcrypt password hashing, SameSite=Strict cookies, CSRF origin checking, HTTPS enforcement, atomic config writes, RAUC signature verification, three-phase config commit with rollback, and comprehensive input validation regexes for fields flowing into daemon configs.

However, the review identified **137 total findings** across 10 OWASP categories:

| Severity | Count |
|----------|-------|
| Critical | 7 |
| High | 26 |
| Medium | 61 |
| Low | 43 |

**Verdicts:** 2 PASS, 1 FAIL, 7 NEEDS REVIEW

The most systemic issue is the **complete absence of security audit logging** (A09 — FAIL). For a UTM security appliance, the inability to detect brute-force attacks, track configuration tampering, or investigate incidents is a critical gap. Other high-priority issues include a wizard race condition enabling unauthenticated admin takeover (A06), a gitignored `package-lock.json` undermining reproducible builds (A03), nil dereference risks across all API GET handlers (A10), and an empty RAUC system.conf rendering firmware signature verification non-functional (A08).

---

## Findings by Category

---

### A01:2025: Broken Access Control

**Verdict: PASS**

The router correctly separates unauthenticated routes (`/health`, `/auth/login`, wizard) from authenticated routes behind `authMiddleware`. Session management uses 256-bit cryptographic randomness, CSRF protection via Origin checking and SameSite=Strict cookies is in place. The wizard is properly guarded by a `SetupDone` check. No IDOR, path traversal, or SSRF vectors were found.

| # | File | Line(s) | Severity | Description | Recommendation |
|---|------|---------|----------|-------------|----------------|
| 1 | `internal/api/system.go` | 109-127 | Medium | Backup export mutates config object returned by `Current()`. If `Current()` is ever changed to return a reference instead of a copy, this would destroy live credentials (AdminHash, WG PrivateKey). | Create an explicit deep copy at the start of `handleBackup` rather than relying on an implicit contract. |
| 2 | `internal/api/wizard.go` | 24-29 | Medium | TOCTOU race in wizard's `SetupDone` guard. The check occurs outside `applyMu`, allowing concurrent wizard completions to race. | Move the `SetupDone` check inside the lock scope, or add a dedicated mutex spanning the handler. |
| 3 | `internal/api/wizard.go` | 14-16 | Medium | Wizard endpoints are fully unauthenticated before `SetupDone`. Any LAN device can race to complete setup. | Consider IP pinning, a time window, or physical-presence challenge. |
| 4 | `internal/api/router.go` | 277-286 | Low | Missing `Content-Security-Policy` header. | Add a restrictive CSP: `default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'`. |
| 5 | `internal/api/router.go` | 95-101 | Low | SPA static assets served without authentication, exposing API endpoint paths. | Acceptable for most threat models; document risk acceptance. |
| 6 | `internal/api/router.go` | 290-313 | Low | Origin check allows both `http://` and `https://` origins. Since HSTS is enforced, `http://` is unnecessary. | Remove the `http://` origin check. |
| 7 | `internal/api/router.go` | 296-301 | Low | Origin check skipped when Origin header is absent. SameSite=Strict is the primary defense. | Acceptable; document the design decision. |
| 8 | `internal/api/wizard.go` | 19-22 | Low | Wizard progress stored in package-level global state. | Move into the `Server` struct. |

---

### A02:2025: Security Misconfiguration

**Verdict: NEEDS REVIEW**

Security headers are well-implemented (HSTS, X-Frame-Options, X-Content-Type-Options, Referrer-Policy, Permissions-Policy). TLS 1.2+ is enforced. Error messages avoid leaking internals. Secrets are redacted from backups. However, gaps exist around CSP, hardcoded dev credentials, TLS directory permissions, and IPS fail-open behavior.

| # | File | Line(s) | Severity | Description | Recommendation |
|---|------|---------|----------|-------------|----------------|
| 1 | `tools/vagrant-provision.sh` | 90-94 | High | Hardcoded password "admin" with bcrypt hash in version-controlled provisioning script. If used in production, the appliance ships with a trivially guessable credential. | Move to a separate gitignored script or environment variable. |
| 2 | `internal/api/router.go` | 277-286 | Medium | Missing `Content-Security-Policy` header. | Add restrictive CSP. |
| 3 | `cmd/scoutd/main.go` | 380 | Medium | TLS directory created with 0755 (world-readable). | Change to `os.MkdirAll(dir, 0700)`. |
| 4 | `internal/config/validate.go` | 346-350 | Medium | Update server URL validation accepts plaintext `http://`. | Restrict to `https://` only. |
| 5 | `templates/suricata.yaml.tmpl` | 75-76 | Medium | Suricata NFQ `fail-open: yes` silently accepts all traffic when IPS engine is down. | Set `fail-open: no` for a security appliance. |
| 6 | `templates/nftables.conf.tmpl` | 157 | Medium | nftables IPS queue uses `bypass` flag — second independent fail-open path. | Remove `bypass` flag in IPS mode. |
| 7 | `cmd/scoutd/main.go` | 411 | Low | TLS cert file written at 0644. | Acceptable for public certs; ensure directory is 0700. |
| 8 | `internal/config/schema.go` | 213 | Low | Default config uses cleartext DNS upstream (DoT/DoH disabled). | Default `UseDOT` to `true`. |
| 9 | `internal/config/schema.go` | 219 | Low | DNS query logging enabled by default (privacy concern). | Default to `false`; let users opt in. |
| 10 | `internal/api/router.go` | 127-133 | Low | Health endpoint exposes version and uptime without auth. | Return only `{"healthy": true}` unauthenticated. |
| 11 | `cmd/scoutd/main.go` | 302-305 | Low | TLS config does not restrict cipher suites. | Explicitly restrict to AEAD ciphers. |
| 12 | `internal/render/engine.go` | 96 | Low | Rendered config output directories created at 0755. | Verify daemon-specific permissions. |

---

### A03:2025: Software Supply Chain Failures

**Verdict: NEEDS REVIEW**

The project shows supply chain awareness: RAUC bundle signature verification, HTTPS enforcement for OTA, `go mod verify` in build scripts, `npm ci --ignore-scripts` for frontend. However, a critical gap exists: the npm lockfile is gitignored, undermining reproducible builds. No automated vulnerability scanning is configured.

| # | File | Line(s) | Severity | Description | Recommendation |
|---|------|---------|----------|-------------|----------------|
| 1 | `.gitignore` | 18 | Critical | `web/package-lock.json` is gitignored. `npm ci` in build scripts will fail or use locally-generated lockfiles, undermining reproducible builds. | Remove from `.gitignore` and commit the lockfile. |
| 2 | `scoutd/go.mod` | 7 | Medium | `skip2/go-qrcode` pinned to a pseudo-version from June 2020 (6+ years old, no releases). | Evaluate maintained alternatives or document risk acceptance. |
| 3 | `internal/update/checker.go` | 19-20 | Medium | `BundleSHA` field received from server but never verified against downloaded bundle. | Implement `DownloadBundle()` with SHA-256 verification. |
| 4 | `web/package.json` | 11-24 | Medium | All npm deps use caret (`^`) ranges. Combined with gitignored lockfile, builds are non-deterministic. | Pin exact versions or commit the lockfile. |
| 5 | `tools/vagrant-provision.sh` | 35-47 | Medium | Alpine packages installed without version pinning. | Pin versions (e.g., `suricata=7.0.3-r0`). |
| 6 | (Project-wide) | N/A | Medium | No automated dependency vulnerability scanning (Dependabot, govulncheck, npm audit). | Add `.github/dependabot.yml` and `npm audit` to build scripts. |
| 7 | `tools/client-provision.sh` | 18 | Low | `apk add` without `--no-scripts` (inconsistent with UTM provisioning). | Add `--no-scripts` for consistency. |
| 8 | `Vagrantfile` | 4-5 | Low | Vagrant box lacks `box_download_checksum`. | Add SHA-256 checksum. |
| 9 | `Makefile` | 7 | Low | Makefile does not run `go mod verify` before building. | Add `go mod verify` prerequisite. |
| 10 | `PLAN.md` | 1900-1904 | Low | Planned Docker build uses unpinned `FROM alpine:3.21`. | Use `FROM alpine:3.21@sha256:<digest>`. |

---

### A04:2025: Cryptographic Failures

**Verdict: PASS**

All critical cryptographic primitives are correctly chosen: `crypto/rand` for token generation, bcrypt for passwords, ECDSA P-256 for TLS, Curve25519 for WireGuard, TLS 1.2+ with cert verification. Secrets are protected with 0600 permissions and redacted from exports. The update channel enforces HTTPS.

| # | File | Line(s) | Severity | Description | Recommendation |
|---|------|---------|----------|-------------|----------------|
| 1 | `internal/render/engine.go` | 100-131 | Medium | WireGuard private key may be briefly visible at final path between rename and chmod (race window). | Apply `chmod(0600)` on temp file *before* `os.Rename`. |
| 2 | `internal/auth/session.go` | 161-175 | Medium | Session token validation uses Go map lookup (non-constant-time). Theoretical timing side-channel. | Accept as low-risk given 256-bit entropy, or use `subtle.ConstantTimeCompare`. |
| 3 | `internal/config/validate.go` | 346-349 | Medium | Config validation accepts `http://` for update server URL (runtime enforces HTTPS). | Restrict to `https://` at schema level. |
| 4 | `internal/api/vpn.go` | 230-252 | Medium | Client WireGuard private key returned in HTTP response body. Design trade-off. | Document; key is never stored server-side. |
| 5 | `cmd/scoutd/main.go` | 380 | Low | TLS directory at 0755. | Use 0700. |
| 6 | `internal/auth/password.go` | 7 | Low | bcrypt at cost 10 (minimum recommended). OWASP recommends 12. | Consider increasing to cost 12. |
| 7 | `cmd/scoutd/main.go` | 302-305 | Low | TLS cipher suites not explicitly restricted. | Explicitly list AEAD ciphers. |
| 8 | `internal/api/router.go` | 236-241 | Low | WebSocket token in query parameter (mitigated by redaction middleware). | Known trade-off; consider single-use tickets. |

---

### A05:2025: Injection

**Verdict: NEEDS REVIEW**

The codebase uses `exec.Command` with proper argument separation (no shell invocation), typed template data, and validation regexes for most fields flowing into configs. However, several user-controlled fields lack validation before reaching daemon configs or generated client configs.

| # | File | Line(s) | Severity | Description | Recommendation |
|---|------|---------|----------|-------------|----------------|
| 1 | `internal/config/validate.go` | — | High | No validation for `IPv4Address` field on interfaces. Flows into multiple templates via `ParseCIDR` calls. | Add CIDR format validation for static interfaces. |
| 2 | `internal/config/validate.go` | — | High | No validation for `IPv4Mode` field. No whitelist. | Validate: must be `"static"`, `"dhcp"`, or `""`. |
| 3 | `internal/config/validate.go` | 260-279 | High | WireGuard `peer.Name` not validated — only checked for empty. | Add format validation using `safeNameRe`. |
| 4 | `internal/config/validate.go` | — | Medium | `EndpointOverride` not validated. Newlines could inject directives into generated WireGuard client config. | Validate with hostname/IP regex; reject newlines. |
| 5 | `internal/api/vpn.go` | 230-238 | Medium | WireGuard client config built with `fmt.Sprintf` — `EndpointOverride` is user-supplied. | Validate input; reject newlines. |
| 6 | `internal/config/validate.go` | 49-51 | Medium | `Timezone` only checked for empty. No format validation. | Add regex or validate via `time.LoadLocation()`. |
| 7 | `internal/config/validate.go` | — | Medium | No validation for `Language` field. | Add whitelist (`"en"`, `"de"`, `"fr"`, etc.). |
| 8 | `internal/config/validate.go` | 346-349 | Medium | `ServerURL` validation is minimal — only checks prefix. | Parse with `url.ParseRequestURI()`. |
| 9-20 | (various templates) | various | Low | Template interpolation of zone names, interface names, NTP servers, domains, ports — all adequately validated by existing regexes. | No action required. |

---

### A06:2025: Insecure Design

**Verdict: NEEDS REVIEW**

Strong controls exist for login rate limiting, body size limits, WebSocket caps, atomic config writes, and session revocation. However, critical design gaps include the wizard race condition, missing rate limiting on password change, unbounded resource creation, and bcrypt truncation.

| # | File | Line(s) | Severity | Description | Recommendation |
|---|------|---------|----------|-------------|----------------|
| 1 | `internal/api/wizard.go` | 18-63 | Critical | Wizard password overwrite race: unauthenticated admin takeover. Steps 1-6 replayable until step 7 sets `SetupDone=true`. Any LAN device can overwrite the admin password. | Prevent step 2 replay if `AdminHash` is already non-empty. |
| 2 | `internal/api/system.go` | 202-246 | High | No rate limiting on password change endpoint. Authenticated attacker can brute-force current password. | Apply rate limiting keyed on session token or IP. |
| 3 | `internal/auth/session.go` | 136-158 | High | No upper bound on concurrent sessions. Unbounded memory allocation. | Add maximum session count (e.g., 50). |
| 4 | `internal/api/vpn.go` | 151-253 | Medium | No limit on VPN peer count. Each creation triggers key generation, QR rendering, config apply. | Enforce maximum (e.g., 50 peers). |
| 5 | `internal/api/firewall.go` | 56-75 | Medium | No limit on firewall rules or port forwards. | Add configurable maximums. |
| 6 | `internal/auth/password.go` | 6-12 | Medium | bcrypt 72-byte truncation not enforced. Two passwords with same first 72 bytes authenticate identically. | Add max password length check (72 chars). |
| 7 | `internal/api/system.go` | 129-172 | Medium | Config restore can downgrade security settings in one request with no confirmation. | Add "dangerous changes" warning for security-impacting fields. |
| 8 | `internal/api/wizard.go` | 140-174 | Medium | Wizard step 4 `lan_address` lacks format validation. | Validate as CIDR before applying. |
| 9 | `internal/api/system.go` | 96-107 | Medium | Reboot endpoint has no confirmation parameter. | Require `{"confirm": true}`. |
| 10 | `internal/api/wizard.go` | 222-261 | Low | Wizard step 6 accepts unbounded VLANs/zones. | Cap at reasonable limits. |
| 11 | `internal/auth/session.go` | 161-175 | Low | Session TTL is sliding only — no absolute maximum lifetime. | Add `CreatedAt` with 7-day absolute max. |
| 12 | `internal/api/router.go` | 291-313 | Low | Origin check allows missing Origin header. | Acceptable; SameSite=Strict is primary defense. |

---

### A07:2025: Authentication Failures

**Verdict: NEEDS REVIEW**

Session tokens use 256-bit CSPRNG, passwords are bcrypt-hashed, brute-force protection exists with per-IP and global rate limiting, sessions are revoked on logout and password change, login errors are unified to prevent enumeration. However, sliding TTL has no absolute max, password change lacks rate limiting, and the rate limiter may be bypassed due to port in `RemoteAddr`.

| # | File | Line(s) | Severity | Description | Recommendation |
|---|------|---------|----------|-------------|----------------|
| 1 | `internal/auth/session.go` | 10, 169-173 | Medium | Sliding 24h TTL with no absolute maximum. Stolen token valid indefinitely if used periodically. | Add `CreatedAt` with absolute max (e.g., 7 days). |
| 2 | `internal/auth/session.go` | 143-157 | Medium | No max concurrent session count. Unbounded memory allocation. | Cap at 10-20 sessions. |
| 3 | `internal/api/system.go` | 202-245 | Medium | Password change endpoint has no rate limiting. | Apply rate limiter to `/system/password`. |
| 4 | `internal/api/router.go` | 185 | Medium | Login response returns token in both cookie AND JSON body. JSON token readable by JS (XSS risk). | Consider cookie-only for browser clients. |
| 5 | `internal/api/wizard.go` | 24-29 | Medium | Wizard password-setting endpoint is unauthenticated and unrate-limited. | Add rate limiting to wizard endpoints. |
| 6 | `internal/api/router.go` | 236-242 | Low | WebSocket token in query parameter. Mitigated by redaction middleware. | Consider single-use WebSocket tickets. |
| 7 | `internal/auth/password.go` | 6-11 | Low | No password complexity rules beyond 10-char minimum. | Check against common breached password list (NIST SP 800-63B). |
| 8 | `internal/api/router.go` | 136-137 | Low | Rate limiter uses `r.RemoteAddr` which includes port — may allow bypass. | Strip port with `net.SplitHostPort()`. |

---

### A08:2025: Software and Data Integrity Failures

**Verdict: NEEDS REVIEW**

RAUC firmware updates use CMS signature verification with explicit documentation that `--no-verify` is never used. Build pipeline uses `go mod verify` and `npm ci --ignore-scripts`. Config writes use atomic tmp+fsync+rename. However, the RAUC system.conf is empty, bundle SHA-256 is never verified, and config files have no integrity protection at rest.

| # | File | Line(s) | Severity | Description | Recommendation |
|---|------|---------|----------|-------------|----------------|
| 1 | `internal/config/validate.go` | 346-349 | High | Config validation accepts `http://` for update server URL. | Restrict to `https://` only. |
| 2 | `internal/update/checker.go` | 20 | High | `BundleSHA` received from server but never verified against downloaded bundle. | Implement download + SHA-256 verification. |
| 3 | `build/overlay/etc/rauc/system.conf` | 1 | High | Empty file — RAUC cannot perform signature-verified updates without valid system.conf and keyring. | Populate with correct partition layout and keyring path. |
| 4 | `internal/config/store.go` | 34-56 | Medium | Config file loaded with no HMAC/signature. Attacker with filesystem access can tamper. | Add HMAC verification with device-specific key. |
| 5 | `internal/render/engine.go` | 86-93 | Medium | Template files loaded from disk with no integrity verification. | Embed templates via `go:embed` for production, or verify checksums. |
| 6 | `internal/api/system.go` | 129-171 | Medium | Config restore deserializes untrusted JSON with incomplete validation. | Add validation for `admin_hash` format, `private_key` length, array size limits. |
| 7 | `internal/service/manager.go` | 77-83 | Low | `suricata-update` integrity relies on its internal TLS + checksum verification. | Document the trust boundary. |
| 8 | `build.sh` | 21 | Low | No `npm audit` step in build pipeline. | Add `npm audit --production`. |
| 9 | `build.sh` | 28 | Low | `go mod verify` correctly invoked. Positive finding. | No action needed. |
| 10 | `tools/vagrant-provision.sh` | 36 | Low | APK `--no-scripts` and key verification. Positive finding. | No action needed. |

---

### A09:2025: Logging and Alerting Failures

**Verdict: FAIL**

The application has **zero audit logging** for security-critical events. Authentication (login/failure/logout), configuration changes, password changes, firmware updates, reboot, backup/restore, firewall modifications, VPN peer management, and IDS config changes all produce no audit log entries. The application uses unstructured stderr logging with no syslog integration.

| # | File | Line(s) | Severity | Description | Recommendation |
|---|------|---------|----------|-------------|----------------|
| 1 | `internal/api/router.go` | 136-186 | Critical | No audit log for login success or failure. | Log with source IP on both outcomes. |
| 2 | `internal/api/router.go` | 139-143 | Critical | No audit log when rate limiter blocks login. Brute-force attacks invisible. | Log rate-limit triggers. |
| 3 | `internal/api/system.go` | 202-246 | Critical | No audit log for password change. | Log with source IP. |
| 4 | `internal/api/system.go` | 96-107 | Critical | No audit log for reboot (production branch). | Log before issuing reboot command. |
| 5 | `internal/api/system.go` | 129-172 | Critical | No audit log for config restore. Attacker could silently replace entire config. | Log with source IP. |
| 6 | `internal/auth/session.go` | 88-113 | High | No log when rate limiter threshold is crossed (block triggered). | Log when per-IP block activates. |
| 7 | `internal/api/system.go` | 56-94 | High | No audit log for system config changes (hostname, timezone, NTP). | Log with changed fields and source IP. |
| 8 | `internal/api/system.go` | 109-127 | High | No audit log for config backup export. Data exfiltration goes unrecorded. | Log with source IP. |
| 9 | `internal/api/system.go` | 174-183 | High | No audit log for TLS cert regeneration. | Log with source IP. |
| 10 | `internal/api/firewall.go` | 30-221 | High | No audit log for firewall rule CRUD. Highest-value target on a UTM. | Log all mutations. |
| 11 | `internal/api/vpn.go` | 151-311 | High | No audit log for VPN peer create/delete or key regeneration. | Log all mutations. |
| 12 | `internal/api/ids.go` | 30-69 | High | No audit log for IDS/IPS config changes (enable/disable, mode switch). | Log with new state and source IP. |
| 13 | `internal/api/wizard.go` | 24-286 | High | No audit log for initial setup wizard (unauthenticated, sets password). | Log each step with source IP. |
| 14 | `internal/update/rauc.go` | 74-98 | High | No audit log for firmware install operations. | Log before and after install. |
| 15 | `internal/api/dns.go` | 35-56 | Medium | No audit log for DNS config changes (malware/phishing blocking). | Log with source IP. |
| 16 | `internal/api/router.go` | 189-206 | Medium | No audit log for logout. | Log for session lifecycle trail. |
| 17 | `internal/api/router.go` | 209-218 | Medium | No log for authorization failures (401 responses). | Log with path and source IP. |
| 18 | `cmd/scoutd/main.go` | 1-483 | Medium | No structured logging or syslog integration. Logs lost on reboot (tmpfs). | Use `log/syslog` or `log/slog` with syslog handler. |
| 19 | `internal/api/network.go` | 29-227 | Medium | No audit log for network topology changes (zones, VLANs). | Log all mutations. |
| 20 | `internal/api/parental.go` | 67-290 | Medium | No audit log for parental control changes. | Log all mutations. |
| 21 | `internal/config/store.go` | 86-132 | Medium | No audit log for config commit/rollback lifecycle. | Log commit and rollback events. |
| 22 | `cmd/scoutd/main.go` | 1-483 | Medium | No log tamper protection (no signing, no append-only, no remote forwarding). | Forward to remote syslog or persistent partition. |
| 23 | `internal/api/router.go` | 291-312 | Low | No log for cross-origin request blocks. | Log CSRF blocks for incident investigation. |
| 24 | `internal/ids/reader.go` | 1-251 | Low | IDS alerts only in 100-entry ring buffer. No alerting threshold. | Add rate-based alert summary warnings. |
| 25 | `internal/api/apply.go` | 169 | Low | Config apply errors may leak internal paths in server-side logs. | Acceptable; client receives generic message. |

---

### A10:2025: Mishandling of Exceptional Conditions

**Verdict: NEEDS REVIEW**

The three-phase config commit has proper rollback, API handlers consistently return appropriate HTTP status codes, and file operations use atomic writes with cleanup. However, a systemic pattern exists: all GET handlers dereference `Config.Current()` without nil checks, creating crash risk if `DeepCopy()` fails under memory pressure.

| # | File | Line(s) | Severity | Description | Recommendation |
|---|------|---------|----------|-------------|----------------|
| 1 | `internal/api/firewall.go` | 26-28 | High | `handleGetFirewall` nil dereference on `Config.Current()`. | Add `if cfg == nil` guard. |
| 2 | `internal/api/network.go` | 24-27 | High | `handleGetNetwork`, `handleGetZones`, `handleGetVLANs` — same nil risk. | Add nil guard. |
| 3 | `internal/api/vpn.go` | 41-42 | High | `handleGetWireGuard`, `handleGetPeers` — same nil risk. | Add nil guard. |
| 4 | `internal/api/dns.go` | 18-21 | High | `handleGetDNS` — same nil risk. | Add nil guard. |
| 5 | `internal/api/ids.go` | 16-18 | High | `handleGetIDS` — same nil risk. | Add nil guard. |
| 6 | `internal/api/parental.go` | 43-51 | High | `handleGetParental`, `handleListProfiles` — same nil risk. | Add nil guard. |
| 7 | `internal/api/system.go` | 37-46 | High | `handleGetSystem` — same nil risk. | Add nil guard. |
| 8 | `internal/config/store.go` | 68-72 | Medium | `Current()` silently returns nil when `DeepCopy()` fails. Masks data corruption. | Return `(*Config, error)` or log failure. |
| 9 | `internal/ids/reader.go` | 178-181 | Medium | `processLine` silently swallows JSON unmarshal errors. | Log periodic parse failure summary. |
| 10 | `internal/ids/reader.go` | 247-249 | Medium | `appendLog` ignores write errors. EVE alerts silently dropped on disk full. | Check and log write errors. |
| 11 | `cmd/scoutd/main.go` | 143, 225 | Medium | `json.Marshal` errors discarded in goroutines. | Check and log errors. |
| 12 | `internal/api/system.go` | 105 | Medium | `exec.Command("reboot").Run()` return value ignored. | Log reboot command errors. |
| 13 | `cmd/scoutd/main.go` | 366-367 | Medium | `Shutdown()` errors completely ignored. | Check and log. |
| 14 | `internal/api/router.go` | 249 | Medium | `writeJSON` ignores `json.Encode` errors. | Log write errors at debug level. |
| 15 | `internal/api/system.go` | 126 | Medium | `w.Write(data)` in backup handler ignores error. | Check return value. |
| 16 | `internal/api/parental.go` | 62-64 | Medium | `generateID()` discards `rand.Read(b)` error. CSPRNG failure yields all-zero ID. | Check error and propagate. |
| 17 | `internal/dnslog/logger.go` | 129 | Medium | `strconv.ParseInt` error discarded — malformed timestamp becomes epoch 0. | Check error; skip entry. |
| 18 | `internal/dnslog/logger.go` | 227 | Medium | `f.Seek` return value ignored — failure causes stale data re-read. | Check error. |
| 19 | `internal/dnslog/logger.go` | 257 | Low | `f.Stat()` error discarded — prevents rotation detection. | Log the error. |
| 20 | `internal/ids/rotate.go` | 27-28 | Low | `os.Rename` errors during log rotation silently ignored. | Check and log. |
| 21 | `internal/service/openrc.go` | 55 | Medium | `runCmd` includes raw command output in error messages. Could leak to API if error chain changes. | Document invariant; ensure errors stay server-side. |
| 22 | `internal/api/ws.go` | 55 | Low | `conn.Close()` error unchecked (library handles double-close gracefully). | Minor; library is safe. |
| 23 | `internal/render/engine.go` | 106 | Low | Non-sensitive configs rendered at 0600 may be too restrictive for daemon users. | Set 0644 for non-sensitive, 0600 for sensitive. |
| 24 | `internal/api/devices.go` | 96 | Low | `url.PathUnescape` error discarded — malformed MAC yields wrong lookup. | Check error; return 400. |

---

## Top 5 Priority Fixes

### 1. Add Security Audit Logging (A09 — 5 Critical, 10 High findings)
The single largest gap. Zero security events are logged. For a UTM appliance, this is the foundational capability that enables incident detection and response. **Action:** Add `log.Printf("audit: ...")` calls to all authentication, authorization, and config mutation handlers. Configure syslog output for log persistence.

### 2. Fix Wizard Password Overwrite Race (A06 #1 — Critical)
An unauthenticated attacker on the LAN can overwrite the admin password during the setup window by replaying wizard step 2. **Action:** Once `AdminHash` is set in step 2, prevent re-execution of that step (check `AdminHash != ""` in addition to `SetupDone`).

### 3. Commit `package-lock.json` to Git (A03 #1 — Critical)
The npm lockfile is gitignored, making `npm ci` in build scripts non-functional and frontend builds non-reproducible. **Action:** Remove `web/package-lock.json` from `.gitignore` and commit the lockfile.

### 4. Add nil Guards to All Config GET Handlers (A10 #1-7 — 7 High findings)
All API GET handlers dereference `Config.Current()` without nil checks. If `DeepCopy()` fails (memory pressure on embedded device), the daemon panics. **Action:** Add `if cfg == nil { writeJSON(w, 500, ...); return }` to every handler, or fix `Current()` to never return nil.

### 5. Populate RAUC system.conf and Implement Bundle SHA-256 Verification (A08 #2-3 — 2 High)
The RAUC system.conf is empty (0 bytes) and the bundle SHA-256 hash from the update server is never verified. Firmware updates — the highest-value supply chain target — are non-functional in production. **Action:** Populate system.conf with partition layout and keyring; implement bundle download with SHA-256 verification before RAUC install.

---

## Remediation Roadmap

### Fix Immediately (before next deployment)

| Finding | Category | Severity | Description |
|---------|----------|----------|-------------|
| A09 #1-5 | Logging | Critical | Add audit logging for login, password change, reboot, config restore |
| A06 #1 | Design | Critical | Fix wizard password overwrite race |
| A03 #1 | Supply Chain | Critical | Commit `package-lock.json` |
| A10 #1-7 | Exceptions | High | Add nil guards to all GET handlers |
| A07 #8 | Auth | Low | Strip port from `RemoteAddr` in rate limiter (silently defeats brute-force protection) |

### Fix This Sprint

| Finding | Category | Severity | Description |
|---------|----------|----------|-------------|
| A09 #6-14 | Logging | High | Audit logging for all config mutations (firewall, VPN, IDS, wizard, firmware) |
| A08 #2-3 | Integrity | High | RAUC system.conf + bundle SHA-256 verification |
| A06 #2-3 | Design | High | Rate-limit password change; cap concurrent sessions |
| A02 #1 | Misconfig | High | Remove hardcoded "admin" password from version-controlled script |
| A05 #1-3 | Injection | High | Validate `IPv4Address`, `IPv4Mode`, WireGuard peer name |
| A02 #4 | Misconfig | Medium | Reject HTTP for update server URL in validation |
| A05 #4-5 | Injection | Medium | Validate `EndpointOverride` (reject newlines) |
| A02 #5-6 | Misconfig | Medium | IPS fail-open → fail-closed |

### Fix This Quarter

| Finding | Category | Severity | Description |
|---------|----------|----------|-------------|
| A01 #4, A02 #2 | Access/Misconfig | Medium | Add Content-Security-Policy header |
| A02 #3, A04 #5 | Misconfig/Crypto | Medium | TLS directory 0700; chmod before rename |
| A06 #4-6 | Design | Medium | Resource limits (VPN peers, firewall rules, password max length) |
| A07 #1 | Auth | Medium | Add absolute session lifetime (7 days) |
| A09 #18 | Logging | Medium | Syslog integration for log persistence |
| A08 #4-5 | Integrity | Medium | Config HMAC; template integrity verification |
| A03 #2, 6 | Supply Chain | Medium | Evaluate go-qrcode replacement; add vulnerability scanning |
| A10 #9-18 | Exceptions | Medium | Fix all silently swallowed errors |
| A05 #6-8 | Injection | Medium | Validate timezone, language, server URL format |

---

*Generated by 10 parallel OWASP category review agents. Each agent independently reviewed the full codebase for its assigned category.*
