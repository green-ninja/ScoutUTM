# Month 6 Quality Control Report

> Generated: 2026-03-02 | Scope: Months 4–6 deliverables | 205 Go tests passing, frontend builds clean

---

## 1. Deliverable Completeness

All Month 4, 5, and 6 deliverables from PLAN.md Section 13 are implemented.

| Month | Focus | Key Deliverables | Status |
|-------|-------|-----------------|--------|
| **4** | Zones + VPN | VLAN creation + zone assignment (IoT, Guest, Kids), WireGuard VPN server + peer management + QR codes, chrony NTP | **COMPLETE** |
| **5** | IDS/IPS | Suricata NFQUEUE integration (queues, fail-open, eve socket), rule management (suricata-update), IDS alert pipeline to scoutd | **COMPLETE** |
| **6** | Frontend (Dashboard) | Vue.js project scaffold, API client, Dashboard view, Firewall view, DNS view, VPN view, real-time WebSocket stats | **COMPLETE** |

### Month 4 Detail

- **Config schema**: `VLANConfig`, `ZoneConfig`, `WireGuardConfig`, `WireGuardPeer` structs in `schema.go`
- **API endpoints**: Full CRUD for zones, VLANs, WireGuard (including peer management, QR codes, key regeneration)
- **Templates**: nftables (VLAN zone sets, VPN rules), wireguard.conf, chrony.conf, dnsmasq (per-VLAN DHCP)
- **Validation**: VLAN + WireGuard rules in `validate.go`
- **Integration tests**: `vlan_test.sh` (11 checks), `vpn_test.sh` (26 checks)

### Month 5 Detail

- **nftables template**: IPS chain with `queue num 0-3 fanout,bypass`, conditional on IDS mode
- **suricata.yaml template**: NFQUEUE mode, fail-open, unix_dgram EVE socket, detect profile low, threshold.config for suppressed SIDs
- **EVE reader**: Unix datagram socket listener, JSON parsing, 100-alert ring buffer, persistent log, auto-reconnect
- **EVE log rotation**: Size-based rotation (50MB, 7 copies), hourly scheduler task
- **IDS API**: GET/PUT config, GET alerts
- **Scheduler tasks**: `suricata-update` (daily 03:00), `eve-log-rotate` (hourly)
- **Integration test**: `ids_test.sh` (12 checks)

### Month 6 Detail

- **Vue.js scaffold**: Vite 5 + Vue 3.4 + Tailwind 3 + TypeScript 5 + Pinia 2 + uPlot 1.6
- **Build pipeline**: `npm run build` → `scoutd/web/dist/` → `//go:embed all:dist` → Go binary
- **API client**: `apiFetch<T>()` wrapper with Bearer auth, 401 auto-redirect, typed modules for auth/firewall/dns/vpn/system
- **WebSocket**: Hub (max 10 clients, adaptive 2s/5s interval, non-blocking broadcast), stats collector (/proc readers + dev-mode synthetic), composable with auto-reconnect
- **Views**: Login, Dashboard (stat cards + uPlot throughput chart + VPN/DNS summaries), Firewall (rules + port forwards tabs with CRUD modals), DNS (toggles + upstream/block/allow textareas), VPN (WireGuard settings + peers + QR code modal)
- **SPA serving**: `spaHandler` with index.html fallback for client-side routing
- **Shared components**: Modal, Toast, ToggleSwitch, StatCard, ThroughputChart, NavLink

---

## 2. Issues

### 2.1 CRITICAL (2)

#### C1. Auth store `isAuthenticated` is NOT reactive

**File:** `web/src/stores/auth.ts:6-8`

```typescript
const isAuthenticated = computed(() => {
  return sessionStorage.getItem('scout_token') !== null
})
```

`sessionStorage` is not a reactive Vue data source. `computed()` evaluates once and never re-evaluates when the token changes because Vue's reactivity system cannot track `sessionStorage.getItem()` calls. After `login()` stores the token, `isAuthenticated` may still return `false`. The router guard at `router/index.ts:48` depends on this value, so authentication checks can be stale.

**Fix:** Use a reactive `ref<string | null>` as the source of truth, synced to sessionStorage:

```typescript
const token = ref<string | null>(sessionStorage.getItem('scout_token'))
const isAuthenticated = computed(() => token.value !== null)

async function login(password: string) {
  const t = await authApi.login(password)
  sessionStorage.setItem('scout_token', t)
  token.value = t
}

function clearSession() {
  sessionStorage.removeItem('scout_token')
  token.value = null
}
```

#### C2. WireGuardConfig TypeScript type mismatches backend response

**File:** `web/src/api/types.ts:41-50`

The interface includes `private_key` (which the backend **never** sends — it would be a security leak) and is missing `public_key` (which the backend **does** send via `wireGuardResponse` in `vpn.go:29-38`). `VpnView.vue:138` uses `(config as any).public_key` as a type escape to work around this.

**Fix:** Remove `private_key`, add `public_key?: string` to the interface.

---

### 2.2 HIGH (8)

#### H1. Session tokens never cleaned up — unbounded memory growth

**File:** `scoutd/internal/auth/session.go`

`SessionManager.Cleanup()` exists and correctly removes expired sessions, but it is never called from `main.go` or the scheduler. Over weeks of uptime, the `sessions` map grows unboundedly as tokens expire but are never pruned. On an embedded appliance with limited RAM, this is a slow memory leak.

**Fix:** Register a periodic task in the scheduler:

```go
sched.Register(scheduler.Task{
    Name:     "session-cleanup",
    Interval: 1 * time.Hour,
    Fn: func(ctx context.Context) error {
        sessions.Cleanup()
        return nil
    },
})
```

#### H2. LoginRateLimiter `perIP` map never pruned

**File:** `scoutd/internal/auth/session.go`

The `perIP` map stores an entry for every IP that fails a login attempt. Entries are only removed on successful login (`ResetIP`). If an attacker sends failed login attempts from many different source IPs, the map grows without bound — a memory exhaustion vector on an embedded appliance.

**Fix:** Add a `Cleanup()` method that removes entries whose `resetAt` is in the past, called alongside session cleanup on the same scheduler interval.

#### H3. No `ReadHeaderTimeout` on HTTP servers — Slowloris vulnerability

**File:** `scoutd/cmd/scoutd/main.go:210-223`

Both `httpsServer` and `httpServer` use default `http.Server` settings with `ReadHeaderTimeout == 0` (infinite). An attacker can open connections and send headers extremely slowly, tying up goroutines and file descriptors indefinitely. Go's `gosec` linter flags this as G112.

**Fix:**

```go
httpsServer := &http.Server{
    Addr:              addr,
    Handler:           router,
    TLSConfig:         tlsConfig,
    ReadHeaderTimeout: 10 * time.Second,
    IdleTimeout:       120 * time.Second,
}
httpServer := &http.Server{
    Addr:              httpAddr,
    Handler:           /* redirect handler */,
    ReadHeaderTimeout: 5 * time.Second,
    IdleTimeout:       30 * time.Second,
}
```

#### H4. WebSocket 5-minute idle timeout NOT implemented (PLAN.md deviation)

**File:** `scoutd/internal/api/ws.go`

PLAN.md Section 6.3 specifies: "Idle clients disconnected after 5 minutes of no client→server messages (ping/pong heartbeat)." The read loop in `ws.go` has no deadline — connections persist indefinitely. Stale WebSocket connections accumulate and count against the 10-client limit.

**Fix:** Set a read deadline that resets on each received client message, using a context with timeout or `SetReadDeadline`.

#### H5. Dashboard lazy-loaded, PLAN says eager

**File:** `web/src/router/index.ts:24`

PLAN.md Section 6.5: "Lazy-load all routes **except** Login and Dashboard (the two most-visited views)." Login is correctly eagerly imported (line 3), but Dashboard uses `() => import(...)`. After login, the user must wait for the Dashboard chunk to load before seeing content.

**Fix:** Import DashboardView eagerly at the top of the file.

#### H6. WebSocket reconnects infinitely after auth failure

**File:** `web/src/composables/useWebSocket.ts:44-46`

```typescript
ws.onclose = () => {
  cleanup()
  reconnectTimer = setTimeout(connect, RECONNECT_DELAY)
}
```

When the server rejects the WebSocket (token expired, 401), `onclose` fires and the composable reconnects every 3 seconds forever — even after the user logs out and the token is removed from sessionStorage. This causes continuous network spam in the background.

**Fix:** Check for a valid token before reconnecting:

```typescript
ws.onclose = () => {
  cleanup()
  if (!sessionStorage.getItem('scout_token')) return
  reconnectTimer = setTimeout(connect, RECONNECT_DELAY)
}
```

#### H7. Toast setTimeout not cleaned up on unmount

**File:** `web/src/components/Toast.vue:13-15`

The `setTimeout` is never cleared. If the Toast unmounts before 3 seconds (user navigates away, parent re-renders), the callback fires on a destroyed component, calling `emit('dismiss')` which may trigger state changes on a now-unmounted parent.

**Fix:**

```typescript
let timer: ReturnType<typeof setTimeout>
onMounted(() => { timer = setTimeout(() => emit('dismiss'), 3000) })
onUnmounted(() => clearTimeout(timer))
```

#### H8. Router guard doesn't redirect authenticated users from login page

**File:** `web/src/router/index.ts:46-51`

An authenticated user can navigate to `/login` and see the login form. If they submit, they get a second token stored in sessionStorage. UX issue that can cause confusion.

**Fix:** Add to the beforeEach guard:

```typescript
if (to.name === 'login' && auth.isAuthenticated) {
  return { name: 'dashboard' }
}
```

---

### 2.3 MEDIUM (13)

#### M1. Token in URL query param logged by chi middleware.Logger

**File:** `scoutd/internal/api/router.go:57,200`

The `extractToken()` function accepts a session token via `?token=` to support WebSocket upgrades (browsers cannot set headers on WS handshake). However, `middleware.Logger` logs the full request URL including the query string — session tokens appear in plaintext in logs.

**Fix:** Add middleware before the logger that redacts the token query param:

```go
r.Use(func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        q := r.URL.Query()
        if q.Get("token") != "" {
            q.Set("token", "REDACTED")
            r.URL.RawQuery = q.Encode()
        }
        next.ServeHTTP(w, r)
    })
})
```

#### M2. HTTP redirect uses unvalidated `r.Host` — open redirect

**File:** `scoutd/cmd/scoutd/main.go:219-221`

```go
target := "https://" + r.Host + r.URL.RequestURI()
```

`r.Host` comes from the `Host` header of the HTTP request, which is attacker-controlled. An attacker can set `Host: evil.com` and get a redirect to `https://evil.com/...`. Practical impact is limited on a LAN appliance.

**Fix:** Use a relative redirect: `http.Redirect(w, r, r.URL.RequestURI(), http.StatusMovedPermanently)`.

#### M3. `middleware.RealIP` trusts spoofed headers

**File:** `scoutd/internal/api/router.go:56`

`middleware.RealIP` trusts `X-Forwarded-For` and `X-Real-IP` headers. On a LAN appliance with no reverse proxy, an attacker can spoof these headers to bypass rate limiting — each request appears to come from a different IP.

**Fix:** Remove `middleware.RealIP` since the appliance terminates connections directly.

#### M4. WebSocket `InsecureSkipVerify: true` disables origin checking

**File:** `scoutd/internal/api/ws.go:20`

Cross-origin WebSocket connections are allowed from any website. Mitigated by `SameSite: Strict` cookies (browser won't send cookie cross-origin) and the token-based auth requiring the attacker to already have the token.

**Fix:** Consider restricting `OriginPatterns` to known appliance addresses for defense-in-depth.

#### M5. Integer underflow in stats delta calculations

**File:** `scoutd/internal/stats/collector.go:60-63,89-90`

`uint64` subtraction on /proc counters. If counters wrap or the interface resets/renames, the new value could be less than the previous, causing unsigned integer underflow and a massive spike value sent to the dashboard.

**Fix:** Guard with `if cpu.total > c.prevCPU.total { ... }`, same for network byte counters.

#### M6. Network throughput sentinel uses `rxBytes > 0` instead of checking `prevTime`

**File:** `scoutd/internal/stats/collector.go:88`

```go
if c.prevNet.rxBytes > 0 && elapsed > 0 {
```

If the WAN interface genuinely has `rxBytes == 0` (freshly booted, no traffic), this guard prevents the throughput calculation on the second sample. The check is used as a sentinel for "first call" but zero is a legitimate counter value.

**Fix:** Use `!c.prevTime.IsZero()` as the sentinel instead.

#### M7. WebSocket `Retry-After` header should be 30, not 5

**File:** `scoutd/internal/api/ws.go:14`

PLAN.md Section 6.3: "Connection 11+ receives HTTP 503 with `Retry-After: 30` header." Implementation uses `Retry-After: 5`, causing clients to retry more aggressively than intended.

**Fix:** Change to `w.Header().Set("Retry-After", "30")`.

#### M8. ThroughputChart watch may miss updates when ring buffer is full

**File:** `web/src/components/ThroughputChart.vue:80-87`

The watch tracks `props.history.length`. Once the stats store ring buffer is full at 60 entries, every `push()` is followed by `splice()`. Vue may batch these mutations in the same tick, seeing length go 60→60, and the watcher may not fire.

**Fix:** In the stats store, replace the array instead of mutating in-place:

```typescript
history.value = [...history.value.slice(-(MAX_HISTORY - 1)), snap]
```

Or watch the array with `{ deep: false }` to trigger on reference change.

#### M9. DNS textarea refs not re-synced after save

**File:** `web/src/views/DnsView.vue:34-41`

After saving, `config.value` is updated from the server response, but `upstreamText`, `customBlockText`, and `customAllowText` refs are not re-assigned. If the server normalizes data, the textareas show stale values.

**Fix:** Re-assign the textarea refs after the API response:

```typescript
config.value = await api.updateDns(updated)
upstreamText.value = config.value.upstream.join('\n')
customBlockText.value = config.value.custom_block.join('\n')
customAllowText.value = config.value.custom_allow.join('\n')
```

#### M10. No input validation on port numbers in Firewall forms

**File:** `web/src/views/FirewallView.vue:245,279,289`

Port number inputs have no `min`/`max` constraints. Users could enter 0, negative numbers, or values > 65535. Backend validates, but client-side validation provides immediate feedback.

**Fix:** Add `min="1" max="65535"` to port input fields.

#### M11. Modal missing Escape key handler

**File:** `web/src/components/Modal.vue`

The modal can be closed by clicking the backdrop or the X button, but there is no `keydown` listener for Escape. No focus trap for keyboard accessibility.

**Fix:** Add Escape key listener via `onMounted`/`onUnmounted` event handlers.

#### M12. DNS query/block counts always 0 in stats (PLAN deviation)

**File:** `scoutd/internal/stats/stats.go`

PLAN.md Section 6.3 specifies stats include "DNS query/block counts." The `DNSQueries` and `DNSBlocked` fields exist in the `Snapshot` struct but are never populated by the collector. Dashboard would show 0 for DNS metrics.

**Note:** Requires Unbound stats integration (`unbound-control stats_noreset`), which can be added as a follow-up.

#### M13. Stats broadcast runs even with 0 WebSocket clients

**File:** `scoutd/cmd/scoutd/main.go:123-145`

The stats collection goroutine reads `/proc/stat`, `/proc/meminfo`, `/proc/uptime`, `/proc/net/dev`, and conntrack every 2 seconds even when no dashboard is open. Unnecessary syscalls on an embedded appliance.

**Fix:** Add `if hub.ClientCount() == 0 { continue }` at the top of the ticker case.

---

### 2.4 LOW (12)

| # | File | Issue |
|---|------|-------|
| L1 | `stats/system.go:31-40` | Unparseable /proc/stat fields silently skipped — produces incorrect CPU percentage without indication |
| L2 | `stats/collector.go` | CPU shows 0% on first collection (no baseline). Could initialize `prevCPU` in `NewCollector()` |
| L3 | `scoutd/web/embed.go` | `all:dist` includes dotfiles; consider `dist` without `all:` prefix unless dotfiles needed |
| L4 | `cmd/scoutd/main.go:123-145` | Stats ticker lag when clients drop below 5 — adjusts from 5s→2s only on next tick |
| L5 | `components/NavLink.vue:16` | Exact path match (`route.path === props.to`) won't highlight for future sub-routes |
| L6 | `router/index.ts` | No catch-all/404 route — unknown paths show blank AppShell. Add `/:pathMatch(.*)*` redirect |
| L7 | `views/DashboardView.vue:15-23` | API errors silently swallowed in `catch {}`, "Loading..." shown forever on persistent failures |
| L8 | `views/VpnView.vue:96-99` | `navigator.clipboard.writeText()` error not handled — shows "Copied" even on failure |
| L9 | `layouts/AppShell.vue:16-18` | Redundant `onUnmounted(disconnect)` — the composable already registers its own cleanup |
| L10 | `components/ToggleSwitch.vue` | Missing `aria-label` on the button when `label` prop is absent |
| L11 | `views/FirewallView.vue` | No loading indicator during initial `getFirewall()` fetch — content area is empty |
| L12 | `router.go` (Section 6.4 CSRF) | PLAN says "SameSite=Strict cookies + origin checking" — origin checking not implemented. SameSite=Strict alone provides primary defense |

---

## 3. PLAN.md Deviations Summary

| # | Deviation | Spec Reference | Severity | Notes |
|---|-----------|---------------|----------|-------|
| D1 | WebSocket idle timeout not implemented | Section 6.3 | Moderate | Stale connections count against 10-client limit |
| D2 | `Retry-After: 5` instead of `30` | Section 6.3 | Minor | More aggressive client retry |
| D3 | Dashboard lazy-loaded instead of eager | Section 6.5 | Minor | Extra network round-trip after login |
| D4 | DNS query/block counts always 0 | Section 6.3 | Minor | Requires Unbound stats integration |
| D5 | Alert broadcast rate limiting missing | Section 6.3 | N/A | Correctly deferred — alerts not yet broadcast over WebSocket |
| D6 | `netlink.go` replaced by `procnet.go` | Section 3.3 | N/A | Acceptable simplification — avoids cgo, functionally equivalent |
| D7 | Server-side QR instead of qrcode.vue | Section 6.5 | N/A | Improvement — smaller bundle, private key never in browser JS |
| D8 | Global rate limit 100/min not 20/min | Section 6.4 | N/A | Intentional, documented in code — avoids DoS amplification |
| D9 | CSRF origin checking absent | Section 6.4 | Minor | SameSite=Strict provides primary defense |

---

## 4. Correctly Deferred Items (Month 7+)

The following items are **not** in the Month 6 codebase and are correctly deferred per PLAN.md:

| Item | Target Month | Current State |
|------|-------------|--------------|
| Setup wizard view (7 steps) | 7 | API stub returns 501 |
| Devices view + netmon ARP scanner | 7 | API stub returns 501, `netmon/` files are package declarations only |
| IDS alerts view | 7 | Alerts available via REST API only, no Vue view |
| Parental controls view | 7 | API stub returns 501 |
| Network/System settings views | 7 | Backend API handlers exist, no dedicated Vue views |
| WebSocket `devices` and `alert` topics | 7 | Only `stats` topic implemented |
| Alert storm rate limiting (10/sec/client) | 7 | No alerts broadcast over WebSocket yet |
| `stress_test.sh`, `wizard_test.sh` | 7 | Placeholder scripts |
| Per-device parental profiles + schedules | 8 | Schema exists, API stubs return 501 |
| Device fingerprinting | 8 | `netmon/` package stubs exist |
| `dns-blocklist-update` scheduler task | 8 | RPZ download mechanism not yet implemented |
| `parental-schedule-check` scheduler task | 8 | Not registered |

---

## 5. Recommended Fix Priority

### Immediate (before Month 7 work begins)

| Priority | Issue | Impact |
|----------|-------|--------|
| 1 | **C1** — Fix auth store reactivity | Login flow can break |
| 2 | **C2** — Fix WireGuardConfig TypeScript type | Type safety violation, `as any` escape |
| 3 | **H1+H2** — Add session + rate limiter cleanup | Memory leak on embedded appliance |
| 4 | **H3** — Set `ReadHeaderTimeout` on HTTP servers | Slowloris DoS vulnerability |
| 5 | **H4** — Add WebSocket idle timeout | PLAN.md deviation, connection leak |
| 6 | **H6** — Stop WebSocket reconnect after logout | Network spam |

### Soon (during Month 7)

| Priority | Issue | Impact |
|----------|-------|--------|
| 7 | **M1** — Redact token from access logs | Token exposure in logs |
| 8 | **M3** — Remove `middleware.RealIP` | Rate limit bypass via header spoofing |
| 9 | **M5+M6** — Guard uint64 underflow in stats | Dashboard spike on counter wrap |
| 10 | **M8** — Fix chart watch for full ring buffer | Chart stops updating after 2 minutes |
| 11 | **H5** — Eagerly import DashboardView | PLAN.md alignment |
| 12 | **H7** — Clean up Toast setTimeout | Component lifecycle correctness |
| 13 | **H8** — Redirect authenticated users from login | UX polish |

### Backlog

All remaining MEDIUM and LOW issues.

---

## 6. Test Coverage

| Package | Tests | Notes |
|---------|-------|-------|
| `internal/api` | 8 | Router, auth, health, rate limiting |
| `internal/auth` | 12 | Sessions, passwords, rate limiter |
| `internal/config` | 45 | Schema, store, validation, migration |
| `internal/ids` | 12 | EVE reader, rotation |
| `internal/render` | 78 | All 6 template renderers |
| `internal/scheduler` | 6 | Task registration, execution |
| `internal/service` | 8 | Manager, OpenRC integration |
| `internal/stats` | 3 | Collector dev-mode |
| `internal/ws` | 5 | Hub register/unregister, broadcast, connection limit, interval |
| **Total Go** | **205** | Up from 197 (Month 5) |
| Integration (VM) | 71 checks | 5 test suites: firewall, network, vlan, vpn, ids |
| Frontend | 0 | No unit tests (acceptable for Month 6 scaffold) |

---

## 7. Build Verification

```
Go build:    scoutd binary compiles (CGO_ENABLED=0)
Go tests:    205/205 passing
Vue tsc:     Type-check passes (0 errors)
Vite build:  17 chunks, completes in <1s
Bundle size: ~200KB total JS (~80KB gzipped), well under 500KB target
Embed:       Go binary embeds frontend assets, serves SPA at /
```
