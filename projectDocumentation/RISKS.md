BLOCKING — Will halt progress or kill the business

  ~~B1. CE marking is legally required BEFORE first EU sale (GTM)~~ ✅ RESOLVED
  Moved CE pre-compliance testing from Month 14-15 to Month 10-11, funded from contingency (€3K). Contingency reduced from €19,500 to €16,500. First sales
  shifted to Month 13-14 (after CE pre-compliance pass). CE pre-compliance pass is now a hard gate before any unit ships. Pre-compliance provides basis for
  self-declaration of conformity (legally sufficient for LVD/EMC self-assessment modules). Full test house certification deferred to Phase 2 funded by revenue.
  Note: ScoutUTM is system integrator/assembler -- CE applies to the finished product (board + enclosure + firmware), not to purchased components individually.

  ~~B2. No product liability insurance budgeted (GTM)~~ ⚠️ ACCEPTED RISK
  Acknowledged. Product liability insurance (€500-2,000/year) not budgeted. Accepted as a known risk given the €35K budget constraint.

  ~~B3. OTA update server architecture is completely undefined (PLAN)~~ ✅ RESOLVED
  Added PLAN.md section 7.5: self-hosted VPS (nginx/caddy) serving static files. Device GETs manifest.json (version, bundle filename, SHA-256, min_version).
  Publishing is manual SCP/rsync. No device auth, no channels, no registry. Security via ECDSA bundle signatures. No dynamic API.

  ~~B4. make-image.sh contradicts the boot architecture (PLAN)~~ ✅ RESOLVED
  Fixed section 8.4 step 5: changed "SquashFS loopback loading" to "ESP kernel extraction (see 7.1)" to match the actual architecture.

  ~~B5. dm-verity has no build system integration (PLAN)~~ ✅ RESOLVED
  Added section 8.3 (SquashFS + dm-verity Build) with veritysetup format step, root hash extraction, and hash offset calculation. Updated section 8.6
  (initramfs) with root hash embedding via template substitution. Updated section 8.5 (make-image.sh) to write SquashFS+verity to partition. Build chain
  is now: make squashfs → make initramfs (embed root hash) → make image.

  ~~B6. No internal scheduler architecture for scoutd (PLAN)~~ ✅ RESOLVED
  Added section 3.6 (Internal Scheduler): goroutine-based tick loop with 60s resolution. 7 registered tasks documented with intervals and timing
  (suricata-update, DNS blocklists, GeoIP, OTA check, log rotation, fake-hwclock, parental schedule enforcement). Added scheduler.go to project structure
  (3.3). Parental schedule enforcement runs every 60s, updates Unbound access-control per device based on time windows.

  ---
  HIGH — Will cause significant rework or business problems

  ~~H1. Per-device DNS filtering is architecturally undefined (PLAN)~~ ✅ RESOLVED
  Added section 5.6: uses Unbound's native access-control-tag feature. Each parental profile maps to a tag, RPZ zones are tag-restricted, devices mapped by
  source IP. Automatic static DHCP reservations pin device IPs when assigned to a profile. Rotating MAC randomization documented as known limitation with
  dashboard warning and detection. Added to section 14 scope limitations.

  ~~H2. IPv6 creates a silent security hole (PLAN)~~ ✅ RESOLVED
  Added explicit IPv6 drop-all rule to nftables prerouting hook: `meta nfproto ipv6 drop`. Blocks all IPv6 at the earliest hook point. Updated section 5.3
  and section 14 (scope limitations). GTM weakness #6 updated to note IPv6 is explicitly blocked, not silently ignored.

  ~~H3. GeoIP is listed as Core v1 AND as Phase 2 (GTM)~~ ✅ RESOLVED
  Removed GeoIP from GTM Core features and PLAN.md v1 architecture (config schema, nftables description, persistent data map, scheduler). GeoIP stays in
  Phase 2 roadmap (Months 15-24). Core feature list now only contains validated v1 features.

  ~~H4. "No subscription ever" model is unsustainable without financial modeling (GTM)~~ ✅ RESOLVED
  Added "Post-Launch Sustainability Model" to GTM financials. Infrastructure floor is ~€17/month. Per-unit gross profit ~€249. Explicitly acknowledges
  founder needs external income during Year 1-2. If sales stall, on-device automation continues, OTA server costs ~€10/month. Founder salary break-even
  at ~15-20 units/month sustained (Month 18-20).

  ~~H5. No warranty policy defined (GTM)~~ ✅ RESOLVED
  Added warranty policy to GTM section 7: 2-year legal guarantee per EU Directive 2019/771, RMA process (remote troubleshoot → ship back → repair/replace),
  warranty reserve of 2 units per batch (first batch: 23 saleable + 2 reserve), coverage scope defined, 5-day turnaround target.

  ~~H6. Shipping costs missing from unit economics (GTM)~~ ✅ RESOLVED
  Shipping charged separately at checkout (EU ~€10-15/unit). Pass-through cost, not absorbed into COGS -- gross margin unchanged. US shipping deferred
  until FCC certification. Added to unit economics section.

  ~~H7. Operating expenses are unrealistically low (GTM)~~ ✅ RESOLVED
  Clarified OpEx is business costs only -- founder salary not included. Founder needs separate income until ~Month 18-20. Added Year 2 OpEx breakdown
  (CE ~€10-15K, trademark ~€850, Amazon FBA ~€2K, packaging/shipping/returns ~€7-12K). €16,500 contingency covers personal runway during Year 1.

  ~~H8. WAN DHCP client is never specified (PLAN)~~ ✅ RESOLVED
  Documented WAN DHCP client in section 5.1: BusyBox udhcpc (included in BusyBox, no extra package). Configured via /etc/network/interfaces, runs as part
  of networking OpenRC service in boot runlevel. scoutd renders interfaces file from central config.

  ~~H9. Safe search enforcement is mentioned but never specified (PLAN)~~ ✅ RESOLVED
  Added SafeSearch and YouTube restricted mode implementation to section 5.6. Uses Unbound local-zone CNAME overrides with tag-based per-device application
  (Google → forcesafesearch.google.com, Bing → strict.bing.com, DuckDuckGo → safe.duckduckgo.com, YouTube → restrict.youtube.com). Standard DNS technique.
  DoH bypass limitation acknowledged with reference to M3.

  ~~H10. Config schema migration + A/B rollback creates a version mismatch problem (PLAN)~~ ✅ RESOLVED
  Added section 4.5: migration runs on first boot of new firmware (not during OTA install). Pre-migrate backup saved before migrating. If OTA rolls back,
  old firmware detects future config version and restores config.json.pre-migrate. Health check verifies config version matches firmware expectations.
  Added pre-migrate file to persistent data map (10.7).

  ~~H11. No GDPR compliance plan (GTM)~~ ✅ RESOLVED
  Added GDPR section to GTM section 17. Privacy-by-design: all personal data (DNS logs, DHCP, IDS alerts) stored on-device only, controlled by device owner.
  ScoutUTM is manufacturer, not data processor. OTA checks are anonymous. Standard e-commerce GDPR obligations (privacy policy, right to deletion of order
  data, breach notification). No DPA needed. Covered within existing €2K legal budget.

  ~~H12. No support process defined (GTM)~~ ✅ RESOLVED
  Added support model to GTM: email + Discord for Phase 1, response targets (24h critical, 48h normal — targets not SLAs), self-service first (docs, FAQ,
  USB recovery, config reset button), device-down bypass instructions in quick-start guide. Scout+ gets priority support in Phase 2.

  ---
  MEDIUM — Should be addressed

  ~~M1. "Two-phase commit" is actually three phases with two conflicting numbering schemes (PLAN)~~ ✅ RESOLVED
  Renamed to "Three-Phase Config Commit" (section 4.3). Replaced Phase A/B with Phase 2a/2b. Updated all references across both documents.

  ~~M2. No login brute-force protection (PLAN)~~ ✅ RESOLVED
  Added brute-force protection to section 6.4: per-IP limit of 5 failed attempts/min (5-min block), global limit of 20 failed attempts/min. In-memory
  counters, HTTP 429 response. Successful login resets counter.

  ~~M3. DNS-over-HTTPS bypass is unaddressed (PLAN)~~ ✅ RESOLVED
  Added partial mitigation in section 5.6: well-known DoH provider domains (cloudflare-dns.com, dns.google, etc.) added to parental RPZ blocklists, catching
  common browser defaults. Documented as known limitation in section 14 -- direct-to-IP DoH bypasses filtering. Industry-wide limitation for DNS-based products.

  ~~M4. GeoIP download/licensing mechanism undefined (PLAN)~~ ⏳ DEFERRED
  GeoIP moved to Phase 2 (see H3 resolution). Architecture will be defined when Phase 2 work begins.

  ~~M5. WireGuard client IP allocation range and server endpoint never defined (PLAN)~~ ✅ RESOLVED
  Added to section 4.2: tunnel subnet 10.0.0.0/24 (server .1, clients allocated sequentially). Endpoint uses current WAN IP with dashboard warning about
  dynamic IPs. QR code generation flow documented (keypair, tunnel IP, config, QR encode). Full tunnel vs split tunnel options. endpoint_override field
  for manual DDNS hostname.

  ~~M6. VAT treatment unclear in pricing (GTM)~~ ✅ RESOLVED
  All EUR prices now stated as VAT-inclusive (21% weighted average). Unit economics updated with ex-VAT revenue: Scout €412 ex-VAT, gross margin ~31%.
  All financial targets, breakeven, milestones, sustainability model recalculated on ex-VAT basis. Significant margin impact acknowledged honestly.

  ~~M7. Scout+ revenue in Year 1 is impossible (GTM)~~ ✅ RESOLVED
  Set Year 1 Scout+ revenue to €0. Scout+ launches in Phase 2 (Month 15+). Year 1 financial targets recalculated accordingly.

  ~~M8. Odroid H4+ may actually have an RTC (PLAN)~~ ✅ RESOLVED
  Updated PLAN.md to note H4+ likely has RTC (verify at Month 10). fake-hwclock kept as fallback regardless -- handles boards without RTC or dead RTC
  battery. Harmless if RTC exists.

  ~~M9. Session token loss on scoutd restart (PLAN)~~ ✅ RESOLVED
  Clarified in section 6.4: config apply does NOT restart scoutd (it reloads other daemons externally). Sessions only lost on reboot/OTA. In-memory
  sessions are intentional (no tokens on disk = smaller attack surface). Acceptable for home appliance.

  ~~M10. Suricata SCHED_FIFO thread priority race (PLAN)~~ ✅ RESOLVED
  Added 5-second delay before first priority-setting loop to allow thread spawning. Added background re-apply after 10 more seconds to catch late-spawning
  threads. Documents the race condition and the mitigation in section 10.9.

  ~~M11. No FCC certification for US sales (GTM)~~ ✅ RESOLVED
  Added EU-only restriction for Phase 1-2. USD pricing marked with footnote "for planning purposes, US sales require FCC (Phase 3+)". TAM figures
  updated to separate EU (addressable now) from US (addressable after FCC).

  ~~M12. Deciso is listed as competitor AND recommended ODM partner (GTM)~~ ✅ RESOLVED
  Added competitor conflict note to Deciso ODM section. Acknowledged they may decline. ODM engagement deferred to Phase 2+ when volumes justify it.
  Phase 1 uses Odroid H4+ (no ODM needed). Kontron/custom PCB as alternatives if Deciso declines.

  ~~M13. No business entity or legal formation mentioned (GTM)~~ ⏳ DEFERRED
  Not relevant until product is ready to certify and sell. Will be addressed alongside CE pre-compliance and first-sale preparation.

  ~~M14. syft is listed as on-device package but is a build-only tool (PLAN)~~ ✅ RESOLVED
  Removed syft from on-device Alpine packages (section 8.4). It's a build-time dependency only (used in make bundle).

  ~~M15. Backup API exports WireGuard private key (PLAN)~~ ✅ RESOLVED
  Intentional design: backup must include WG private key for restore to work (without it, all VPN peers break on restore). UI displays warning: "This
  backup file contains sensitive data including your VPN private key. Store it securely." Documented as known trade-off in PLAN.md section 14.

  ~~M16. No disk space monitoring (PLAN)~~ ✅ RESOLVED
  Added `disk-check` hourly scheduler task (section 3.6). Uses Go `statvfs` syscall on /data. Warns at 90%, auto-prunes oldest rotated logs at 95%.
  SMART/NVMe health monitoring deferred to Phase 2 (requires smartmontools).

  ~~M17. "Made in EU" claim needs legal vetting (GTM)~~ ⏳ DEFERRED
  Not relevant until product is ready for market. Will be addressed alongside legal entity formation (M13) and CE certification.

  ---
  LOW — Nice to have

  ~~L1. No i18n framework for Vue.js~~ ✅ RESOLVED
  v1 is English-only. Removed "Language" from setup wizard step 1. i18n deferred to Phase 2+. Added to PLAN.md section 14 scope limitations.

  ~~L2. No health check API endpoint~~ ✅ RESOLVED
  Added `GET /api/v1/health` (unauthenticated) to API endpoint table (section 6.3). Returns healthy/uptime/version JSON. Reuses OTA rollback health
  check logic (section 7.3).

  ~~L3. NVMe device paths hardcoded everywhere~~ ✅ RESOLVED
  Added PARTUUID note to section 7.1: build system writes stable PARTUUIDs, all runtime references use PARTUUID=. Hardcoded /dev paths in the document
  are for readability only.

  ~~L4. RAUC compatible string has a space~~ ✅ RESOLVED
  Changed RAUC compatible string from `ScoutUTM Scout` to `scout-utm`. Matches RAUC convention (lowercase, hyphenated).

  ~~L5. Rollback test table timing inconsistency~~ ✅ RESOLVED
  Fixed header from "Months 3-8" to "Months 2-9" to match actual table entries.

  ~~L6. HTTPS bootstrapping on first boot~~ ✅ RESOLVED
  Already documented in section 10.5: scoutd generates self-signed ECDSA P-256 cert on first boot before starting HTTPS listener. No race condition.

  ~~L7. Competitor pricing may be outdated~~ ⏳ DEFERRED
  Will be re-verified before any marketing materials are produced.

  ~~L8. No serial console specification~~ ✅ RESOLVED
  Added `console=tty0 console=ttyS0,115200n8` to GRUB kernel command line (section 7.2). Added serial console spec (115200 8N1) to emergency shell
  documentation (section 10.8).

  ~~L9. IPv6 exclusion could block some EU customers~~ ✅ RESOLVED
  Added "IPv4 internet connection" as explicit system requirement. DS-Lite/IPv6-only ISP limitation noted in PLAN.md section 14 and GTM weakness #6.
  Product page and quick-start guide will list this requirement.

  ~~L10. Payment processing fees absent~~ ✅ RESOLVED
  Phase 1 uses direct bank transfer (no Stripe/credit card). Zero processing fees. Stripe/card payments deferred to Phase 2+ when volume justifies it.

  ~~L11. TAM figures have no source citation~~ ⏳ DEFERRED
  Not relevant for internal planning document. Figures are directionally correct.

  ~~L12. Year 3 projections look aggressive~~ ✅ RESOLVED
  Added note to GTM financial targets: Year 3 figures are aspirational stretch targets, not forecasts. Business is sustainable at ~15-20 units/month.

  ---
  Summary

  All issues addressed:
  - ✅ RESOLVED: B1, B3, B4, B5, B6, H1-H12, M1-M3, M5-M12, M14-M16, L1-L6, L8-L10, L12
  - ⚠️ ACCEPTED RISK: B2 (product liability insurance)
  - ⏳ DEFERRED: M4 (GeoIP licensing), M13 (business entity), M17 ("Made in EU" legal), L7 (competitor pricing), L11 (TAM citations)
