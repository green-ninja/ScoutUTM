# ScoutUTM - Go-To-Market Strategy

## Executive Summary

ScoutUTM is a unified threat management appliance built for families and family-run businesses who need enterprise-grade network security without enterprise complexity or subscription fatigue. It fills the gap between overly technical open-source firewalls (pfSense, OPNsense) and consumer devices that lack depth (Firewalla), delivering powerful protection through a modern, approachable interface at a one-time cost.

**All hardware is designed and assembled in the European Union**, giving ScoutUTM a verifiable, trustworthy supply chain -- a strategic differentiator as EU regulations (Cyber Resilience Act, NIS2) increasingly demand supply chain transparency and digital sovereignty.

---

## 1. Product Vision

**One-liner:** Enterprise security for the rest of us.

**Problem:** Home networks now host 20-50+ connected devices -- laptops, phones, smart TVs, IoT sensors, security cameras, work-from-home setups, and kids' devices. The attack surface rivals small businesses, but the security tools available are either too complex (pfSense), too expensive with recurring fees (Fortinet, SonicWall), or too limited (consumer routers).

**Solution:** A purpose-built appliance that combines:
- Deep packet inspection and intrusion detection at multi-gigabit firewall speeds
- Intuitive, mobile-friendly web management
- Zone-based network segmentation with VLAN support
- Parental controls that actually work (DNS filtering, schedules, per-device profiles)
- No subscription fees for core security features
- EU-designed and EU-assembled hardware with a transparent, auditable supply chain

---

## 2. Target Audience

### Primary: Tech-Comfortable Families (TAM: ~35M EU households; US ~18M households addressable after FCC certification in Phase 3+)
- Households with 15-50+ connected devices
- At least one technically inclined member (not necessarily IT professional)
- Privacy-conscious, aware of cyber threats (GDPR-aware in EU)
- Willing to invest 300-700 EUR/USD once for peace of mind
- Tired of configuring consumer routers that don't deliver
- Often have kids who need age-appropriate internet access
- EU customers who prefer locally manufactured, trustworthy hardware

### Secondary: Family-Run Small Businesses (TAM: ~23M EU businesses; US ~6M addressable after FCC in Phase 3+)
- 1-25 employees, often operating from home or small office
- Handle sensitive client data (accounting, legal, medical, retail)
- Need compliance-ready security but can't afford managed IT
- NIS2 and GDPR compliance pressure in EU markets
- Dual-use networks (business + personal)
- Value simplicity -- the owner is often the "IT department"

### Tertiary: Remote/Hybrid Workers
- Professionals working from home who need to secure their home network
- Employers increasingly requiring endpoint and network security for WFH setups
- VPN needs for secure access to corporate resources
- EU employers with data sovereignty requirements for remote staff

### Anti-Personas (who this is NOT for):
- Large enterprises with dedicated security teams
- Users wanting a $50 plug-and-play box with zero configuration
- Network engineers who prefer raw CLI-only tools

---

## 3. Competitive Landscape

### Direct Competitors

| Product | Price | Subscription | Ease of Use | Power | Sweet Spot |
|---------|-------|-------------|-------------|-------|------------|
| **Firewalla Gold Plus** | $589 | None | High | Medium | Privacy-conscious families |
| **Ubiquiti UDM Pro** | $379-599 | Optional | Medium | High | UniFi ecosystem users |
| **Netgate 2100 (pfSense)** | $412 | None | Low | Very High | Technical users |
| **OPNsense (DIY)** | $100-600 | None | Low | Very High | DIY enthusiasts |
| **Fortinet FortiGate 30G** | $250+ | $500-950/yr | Low | Very High | Small business (budget exists) |
| **Sophos XG Home** | Free (SW) | None | Medium | High | Tech-savvy home users |
| **Untangle NG** | $299+ | $240/yr | Medium | Medium | Small business |
| **SonicWall TZ** | $330+ | Required | Low | High | SMBs with IT support |
| **ScoutUTM** | **$449-649** | **None** | **High** | **High** | **Families & family biz** |

### Competitive Positioning Map

```
                        HIGH POWER
                            |
               pfSense  ----+---- Fortinet
              OPNsense       |      SonicWall
                            |
                     ScoutUTM *
                            |
         COMPLEX -----------+----------- SIMPLE
                            |
                   Ubiquiti  |  Firewalla
                            |
                            |
                        LOW POWER
```

**ScoutUTM's position:** Upper-right quadrant -- high power AND high simplicity. This is the underserved zone.

### Why Competitors Fall Short

| Competitor | Gap ScoutUTM Fills |
|-----------|-------------------|
| **Firewalla** | Limited IDS depth, email-only support, weaker DNS filtering, no VLAN segmentation wizard |
| **pfSense/OPNsense** | Intimidating UI, no guided setup, no parental controls, no plain-language alerts |
| **Ubiquiti** | Locked to UniFi ecosystem, CyberSecure subscription for full features |
| **Fortinet/SonicWall** | Subscription costs ($500-950/yr), enterprise complexity, overkill UI |
| **Sophos XG Home** | Requires dedicated PC, home-use license only, no commercial use |
| **Protectli** | Hardware only -- still need to configure everything yourself |
| **Deciso (OPNsense)** | Dutch-made, but software is complex and UI is dated |
| **LANCOM / R&S** | German-made, BSI-certified, but enterprise pricing and complexity |
| **All non-EU competitors** | No EU supply chain transparency; cannot claim "Made in EU" trust |

---

## 4. Product Features

### Platform

ScoutUTM runs **ScoutOS**, a proprietary operating system built on Alpine Linux. Closed-source, purpose-built for network security appliances.

| Property | Detail |
|----------|--------|
| Base | Alpine Linux (musl libc, BusyBox) |
| Kernel | Hardened Linux LTS, custom config |
| Root filesystem | Read-only SquashFS + dm-verity integrity verification + tmpfs overlay |
| Init system | OpenRC (lightweight, no systemd) |
| Package delivery | OTA firmware images (not individual packages), ECDSA-signed |
| Footprint | ~50-80MB installed OS (vs. ~500MB+ for pfSense/OPNsense) |
| Boot time | Target <15 seconds to operational |
| Update model | A/B partition scheme with RAUC, automatic rollback on failed health check |
| Integrity | dm-verity Merkle tree hash on rootfs, signed OTA bundles, noexec persistent storage |
| Reliability | Hardware watchdog (auto-reboot on hang), process health monitoring |
| Compliance | SBOM (CycloneDX) generated per release, GPL source archive published |
| Licensing | Proprietary. Not open-source. (GPL source for included open-source components published per release.) |

**Why Alpine over Debian/Ubuntu:** 10x smaller attack surface, faster boot, no systemd complexity, musl libc is more security-auditable than glibc, and Alpine is already the standard base for security-focused container images. Full rationale in PLAN.md section 1.

**Why not fork OPNsense/pfSense:** Their UIs are deeply coupled to FreeBSD, their codebases carry decades of legacy, and pfSense's license restricts redistribution. Starting clean on Alpine lets us build the UX we want without inheriting technical debt.

**Why Alpine over OpenWrt:** Alpine provides a full Linux userland with Suricata, Unbound, WireGuard, and RAUC available as standard packages. OpenWrt is optimized for constrained MIPS/ARM routers -- its UCI config system and build model would fight against ScoutOS's custom control plane. Alpine on x86 is a standard Linux environment any developer can work with. See PLAN.md section 1 for detailed rationale.

### Core (Free Forever -- included with hardware)

*All features listed here are implemented and validated in PLAN.md. No aspirational features.*

**Network Security**
- Stateful firewall with zone-based policies (nftables)
- IDS with curated rulesets (Suricata, IDS mode by default -- alert and monitor)
- IPS mode available as opt-in (inline blocking with throughput trade-off -- see throughput note below)
- DNS filtering and sinkholing (malware, phishing, ads) via Unbound RPZ
- Automatic threat intelligence updates (Suricata rules daily, DNS blocklists daily)
- dm-verity rootfs integrity verification (tamper detection on every block read)

**Network Management**
- VLAN support with guided segmentation wizard ("LAN", "Kids", "IoT", "Guest")
- DHCP server (dnsmasq) with static assignments
- Recursive DNS resolver (Unbound) with DNSSEC validation
- DNS-over-TLS (DoT) and DNS-over-HTTPS (DoH) to upstream resolvers

**VPN**
- WireGuard VPN server with peer management
- QR code peer configuration (scan-and-connect from phone)

**Family Controls**
- Per-device content filtering via DNS blocklists
- Age-based profiles (child, teen, adult)
- Screen time schedules and bedtime enforcement
- SafeSearch enforcement
- YouTube restricted mode
- DNS query activity reporting (domains only, not URLs -- privacy-respecting)

**Device Awareness**
- Automatic device discovery (ARP + DHCP lease parsing)
- Device fingerprinting (OUI manufacturer lookup, DHCP options)
- Per-device zone and profile assignment

**Interface**
- Modern web dashboard (responsive, mobile-friendly via Tailwind CSS)
- Guided setup wizard (7-step onboarding)
- Plain-language security alerts ("Your smart TV tried to contact a known malware server")
- Real-time dashboard (WebSocket: traffic stats, device list, IDS alerts)

**Throughput note:** Exact throughput numbers depend on hardware benchmarks (Month 10 of development). Conservative estimates for the Odroid H4+ (Intel N97, 4 cores): 2.5 Gbps firewall-only, 1.5-2.0 Gbps IDS mode, 1.0-1.5 Gbps IPS mode. **Final product specifications will be based on verified benchmark data, not estimates.** See PLAN.md section 5.4.

### Scout+ (Optional -- one-time $99 upgrade, NOT a subscription)

*Scout+ features are planned for Phase 2 (Months 12-24). Not available at v1.0 launch.*

- Advanced reporting and analytics (90-day log retention, up from 7 days)
- Multi-site management (manage parents' ScoutUTM remotely)
- Priority email and chat support
- Early access to new features
- Custom TLS certificates (upload your own or integrate with Let's Encrypt)

### Future Considerations (v1.5 / v2+)

**v1.5 (post-launch, funded by initial sales):**
- UEFI Secure Boot (full chain-of-trust: shim → signed GRUB → signed kernel)
- TPM 2.0 integration (key sealing, hardware entropy)
- OpenVPN server/client support
- Multi-WAN / failover support

**v2.0+ (funded by sustained revenue):**
- Native mobile app (iOS + Android)
- Built-in Wi-Fi 7 AP option (ScoutUTM AX variant)
- QoS / traffic shaping with application awareness
- Smart home protocol awareness (Matter, Thread)
- Home Assistant integration
- IPv6 dual-stack support
- UPnP with security guardrails
- Dynamic DNS
- Web Application Firewall (WAF) for self-hosted services
- Reverse proxy with automatic SSL (Let's Encrypt)
- Network topology visualization
- AI-driven behavioral analysis and anomaly detection
- PoE ports for cameras/APs
- Cellular failover (4G/5G modem slot)

---

## 5. Strengths (Why ScoutUTM Wins)

1. **No subscription fees** -- Core security is a one-time purchase. Period. This alone differentiates from Fortinet, SonicWall, Untangle, and Ubiquiti's CyberSecure.

2. **The "prosumer sweet spot"** -- Enterprise features (IDS, VLANs, DNSSEC, WireGuard) wrapped in a consumer-friendly UI. Nobody else nails both.

3. **Honest performance claims** -- Throughput specs are based on verified hardware benchmarks, not marketing estimates. Every number on the product page comes from reproducible iperf3 tests with documented conditions.

4. **Family-first design** -- Parental controls are a first-class feature, not an afterthought bolted onto a business firewall.

5. **Transparent, honest pricing** -- No hidden costs, no "call for quote," no artificial feature gating on hardware you already own.

6. **Modern UX** -- Mobile-friendly web dashboard, guided 7-step setup wizard, plain-language alerts. Designed in 2026, not 2006.

7. **Privacy-respecting** -- Local processing, no cloud dependency for core functions, no telemetry without consent. DNS query logs record domains only (not URLs), and can be disabled entirely.

8. **Verifiable integrity** -- dm-verity cryptographic verification on every rootfs block read, ECDSA-signed OTA updates, hardware watchdog for hang recovery. The security of the appliance itself is as important as the security it provides.

9. **Dual-use design** -- Handles family AND small business needs on the same box with proper segmentation.

10. **Made in the EU** -- Designed, assembled, and supported within the European Union. Verifiable supply chain, no black-box offshore manufacturing. Aligns with EU Cyber Resilience Act and NIS2. Among consumer-facing UTMs, almost nobody else can claim this.

11. **Regulatory tailwind** -- EU Cyber Resilience Act (CRA) compliance deadlines in 2026-2027 will force all products with digital elements to prove supply chain security. ScoutUTM ships CycloneDX SBOMs and GPL source archives from day one, while competitors scramble to retrofit.

---

## 6. Weaknesses & Risks (Honest Assessment)

1. **New brand, zero trust** -- Firewalla has years of community trust, pfSense has decades. We start from zero. Mitigation: transparent security audits (independent third-party), community engagement, "Made in EU" trust signal, and published security architecture documentation.

2. **EU manufacturing cost premium** -- EU assembly is 20-40% more expensive than Asian manufacturing, squeezing margins. Mitigation: position the premium as a feature ("you're paying for trust, not just a box"), use cost-effective EU regions (Poland, Czech Republic, Hungary) for assembly, and offset with Scout+ software revenue.

3. **Component dependency on non-EU silicon** -- CPUs (Intel/AMD), NICs, and memory are not made in the EU. "EU-origin" means EU-designed and EU-assembled, not EU-fabbed silicon. Mitigation: be transparent about what "Made in EU" means; pursue EU PCB fabrication and final assembly; monitor EU Chips Act progress for future opportunities.

4. **Performance vs. price tension** -- IDS/IPS throughput is hardware-limited and unknown until benchmarked. If IPS caps at ~800 Mbps, the product must be positioned accordingly. Mitigation: benchmark-first approach (no marketing claims until Month 10 hardware tests), tiered positioning (firewall throughput vs. IDS throughput vs. IPS throughput stated separately), and IDS-default mode that preserves higher throughput for most users.

5. **Support burden** -- Home users expect more hand-holding than enterprise customers. Mitigation: excellent documentation, community forums, video tutorials, 7-step guided setup wizard.

6. **Feature scope creep** -- Trying to be everything to everyone. Mitigation: ruthless prioritization, clear "not in v1" boundaries documented in PLAN.md section 14 (no multi-WAN, no IPv6, no mobile app, no QoS in v1). IPv6 is explicitly dropped at the firewall to prevent security bypass -- not silently ignored. **Note:** IPv6-only or DS-Lite ISP customers (increasingly common in Germany/France) cannot use ScoutUTM v1. "IPv4 internet connection" is listed as a system requirement on the product page and quick-start guide.

7. **Open-source competition** -- pfSense and OPNsense are free and open-source. Mitigation: compete on experience, polish, and time saved -- not raw feature count. Our value is curation, UX, EU trust, and "it just works." Open-source users are rarely our target buyer anyway; they enjoy the tinkering.

8. **Market education** -- Many families don't know they need a UTM. Mitigation: content marketing focused on real-world threat scenarios, not technical jargon.

9. **Regulatory/compliance** -- If targeting businesses handling health/financial data, may need certifications. CE marking mandatory for EU hardware sales from unit #1. Mitigation: CE pre-compliance testing in Month 10-11 (€3K from contingency) provides basis for self-declaration of conformity before first sale. Design for compliance from day one (SBOM from first build, GPL source archive automated). Full CE certification and BSI pursuit funded by revenue. Note: ScoutUTM is a system integrator/assembler — the Odroid H4+ is a purchased component. CE conformity assessment applies to the finished product (board + enclosure + firmware as a system).

10. **EU ODM partner risk** -- Fewer EU hardware partners than Asia means less negotiating leverage and potential single-vendor dependency. Mitigation: qualify at least two EU assembly partners, maintain designs that are portable across manufacturers.

11. **Post-launch operational sustainability** -- After v1.0 ships, the product requires continuous security engineering: CVE monitoring, Alpine patches, Suricata rule updates, OTA infrastructure, SBOM regeneration. This is not a one-time build -- it becomes an ongoing security program requiring 8-12 hours/month minimum. Mitigation: automate everything possible (daily rule/blocklist updates on-device, CVE scanning in CI, SBOM in build pipeline), maintain a documented security update process with 72-hour target for critical CVEs. See PLAN.md section 16 for full operational plan.

12. **Musl libc allocator risk** -- Alpine's musl malloc can suffer contention under heavy multi-threading (Suricata). Mitigation: benchmark with jemalloc LD_PRELOAD during Month 5/10 testing; if >20% improvement, ship jemalloc as default. IDS-default mode provides throughput headroom regardless.

---

## 7. Pricing Strategy

### Hardware Tiers

ScoutUTM is a physical product. Every unit is assembled, tested, and shipped from the EU. There is no software-only option -- customers buy a complete, ready-to-use appliance. This is non-negotiable for building brand trust as a new entrant.

All prices include EU assembly. **All EUR prices are VAT-inclusive** (as required by EU consumer law). VAT rate varies by EU country (17-27%); 21% used as weighted average for financial modeling. USD prices are pre-tax (US sales tax varies by state). VAT is remitted via the EU One-Stop Shop (OSS) scheme.

| Model | Platform | Specs | EUR | USD | Target |
|-------|----------|-------|-----|-----|--------|
| **Scout** | Odroid H4+ | 2x 2.5GbE, Intel N97, 8GB DDR5, 128GB NVMe | €499 | $529* | Families, home offices |
| **Scout Pro** | Odroid H4 Ultra | 2x 2.5GbE, Intel i3-N305 8C, 16GB DDR5, 256GB NVMe | €699 | $739* | Power users, family businesses |

*\*Phase 1-2 sales are EU-only. US sales require FCC certification (deferred to Phase 3, Months 24+). USD pricing shown for planning purposes.*

*"Designed and assembled in the European Union. Board manufactured in South Korea by Hardkernel."*
*Pricing reflects EU assembly, QA, branding, packaging, and support. Transparent supply chain published on ScoutUTM.com.*

### Software

| Tier | Price | Includes |
|------|-------|---------|
| **Core** | Included | All security features, parental controls, VPN, IoT awareness |
| **Scout+** | €99 / $99 one-time | Advanced analytics (90-day retention), multi-site management, custom TLS certs, priority support |

### Pricing Philosophy
- No subscriptions. Ever. For any security feature.
- Hardware margins target 35-45% gross margin (adjusted for EU manufacturing premium)
- Scout+ is pure software margin (near 100%) -- critical to offset lower hardware margins
- "Made in EU" premium is justified and marketed, not hidden
- Compete on total cost of ownership over 3 years:

**3-Year TCO Comparison (EUR):**
| Product | Year 1 | Year 2 | Year 3 | Total | Made in EU? |
|---------|--------|--------|--------|-------|-------------|
| ScoutUTM Scout + Scout+ | €598 | €0 | €0 | **€598** | Assembled in EU |
| Firewalla Gold Plus | €549 | €0 | €0 | €549 | No (USA/China) |
| Deciso DEC750 (OPNsense) | €699 | €0 | €0 | €699 | Yes (NL) |
| Fortinet FortiGate 30G | €700 | €470 | €470 | €1,640 | No (Asia) |
| LANCOM R&S UF-160 | €800 | €350 | €350 | €1,500 | Yes (DE) |
| Ubiquiti UDM Pro + CyberSecure | €470 | €95 | €95 | €660 | No (Asia) |
| SonicWall TZ270 | €590 | €375 | €375 | €1,340 | No (Asia) |

*ScoutUTM is the most affordable EU-assembled option with the best UX. The cheaper EU competitor (Deciso) ships complex OPNsense without guided setup or parental controls. Subscription-based competitors cost 2-3x more over 3 years.*

### Warranty Policy

**2-year legal guarantee** per EU Directive 2019/771. This is non-negotiable for EU hardware sales.

**Coverage:**
- Manufacturing defects (solder joints, component failure, DOA)
- Component failure under normal use within 2 years
- **Not covered:** Physical damage, water damage, power surge damage, opened/modified units, software bugs (addressed via OTA updates)

**RMA process:**
1. Customer contacts support (email/Discord) with issue description
2. Remote troubleshooting (firmware reflash via USB recovery, config reset) -- resolves ~50% of cases
3. If hardware fault confirmed: customer ships unit back (customer pays return shipping during Phase 1; ScoutUTM prepaid label at scale)
4. Unit is repaired/reflashed and returned, or replaced from reserve stock
5. Turnaround target: 5 business days from receipt

**Warranty reserve:** 2 units per batch held as RMA stock. First batch: 25 ordered, 23 saleable + 2 reserve. At small scale, defective units are repaired and returned rather than replaced with new units.

---

## 8. Go-To-Market Channels

### Phase 1: Build + First Hardware Sales (Months 1-14) -- €0 marketing budget

*Months 1-12 are pure software development. Hardware purchased Month 9-10. CE pre-compliance Month 10-11. First sales Month 13-14 (after CE pre-compliance pass). See PLAN.md section 13 for development timeline.*

**Launch Strategy: Build in Public (free channels only)**
- Personal blog / dev log (static site, free hosting)
- GitHub for community engagement (bug reports, feature requests, security disclosures)
- Reddit presence (r/homelab, r/homeautomation, r/selfhosted, r/netsec, r/smarthome, r/europrivacy)
- Discord community (free tier)
- Hacker News Show HN posts at key milestones (Alpha Month 4, Beta Month 8, Launch Month 12)
- No paid advertising, no influencer campaigns -- earn attention through the product

**Sales Model: Hardware appliance, direct only**
- ScoutUTM.com (simple storefront -- bank transfer payments in Phase 1, Stripe/card payments added in Phase 2+ when volume justifies it)
- Sell complete, ready-to-use ScoutUTM appliances (assembled and tested in EU)
- First batch: ~25 units from €35K budget (see Section 10)
- Ship from home / workshop -- founder does assembly, QA, and fulfillment
- EU VAT via OSS (One-Stop Shop) scheme to avoid per-country registration

### Phase 2: Growth + Marketplace (Months 14-24) -- funded by Phase 1 revenue

- Larger hardware batches (50-100 units, better component pricing)
- Amazon.de as first marketplace (FBA for logistics)
- Targeted YouTube tech reviewer outreach (send 3-5 units to relevant creators)
- Regional MSP/VAR partnerships for EU business customers
- Scout+ tier launch (€99 one-time upgrade)

### Phase 3: Scale (Months 24-36) -- funded by Phase 2 revenue

- Broader EU Amazon marketplaces (NL, FR, IT, ES)
- EU retail channels (Alternate, Reichelt, Coolblue, LDLC)
- US market expansion (Amazon.com, FCC certification when budget allows)
- Consider EU grant applications with traction data to fund growth

---

## 9. Marketing Strategy

### Brand Voice
- Approachable, not patronizing
- Confident, not aggressive
- Technical when needed, plain language by default
- "We're the security nerds so you don't have to be"

### Content Pillars

1. **Threat Awareness** -- "Did you know your smart TV phones home 5,000 times per day?" Real-world scenarios that make security tangible.
2. **Family Digital Safety** -- Parental controls, screen time, online predator awareness. Emotionally resonant.
3. **Small Business Security** -- Compliance, data breaches, insurance requirements. Fear + solution.
4. **How-To / Education** -- Network setup guides, VPN explainers, security best practices. SEO-driven.
5. **Build in Public** -- Development updates, transparent decision-making, community input. Trust-building.

### Content Pillar (EU-specific)

6. **Made in Europe** -- Supply chain transparency, digital sovereignty, CRA/NIS2 readiness. Resonates with EU privacy-conscious audience and B2B buyers with compliance needs. Differentiates from every US/Asian competitor.

### Key Marketing Messages
- "Enterprise security. Family simplicity. Zero subscriptions. Made in Europe."
- "Protect everything on your network. Set it up in 15 minutes."
- "Your smart home is smart. Make it safe too."
- "The firewall that speaks human."
- "Know where your security comes from." (EU trust message)
- "Designed and assembled in the EU. No backdoors. No subscriptions. No compromises."

### Launch Campaign (zero-budget)
- Pre-launch landing page with email capture (static site, free hosting)
- Show HN post + Reddit launch posts across target subreddits
- Dev blog post: "Why I built a UTM appliance on Alpine Linux in my living room"
- Product Hunt launch
- Reach out to 5-10 niche bloggers / YouTubers who cover home networking (free unit if they review)
- Word-of-mouth referral program once first customers are happy

### Support Model

**Phase 1 (launch, solo founder):**

| Channel | Availability | Response Target |
|---------|-------------|-----------------|
| Email (support@scoututm.com) | Business hours | Critical (device down): 24h. Normal: 48h. |
| Discord community | Best effort | Community-assisted + founder when available |

These are **targets, not SLAs.** Solo founder, no guarantees. The founder is the only escalation path.

**Self-service first (reduces support load):**
- Quick-start guide in the box
- FAQ and troubleshooting docs on ScoutUTM.com
- USB recovery mode for emergencies (password reset, factory reset — no support ticket needed)
- "Reset to Previous Config" button in the dashboard for config mistakes
- Plain-language error messages in the UI (not raw stack traces)

**Device-down urgency:** A ScoutUTM failure means no internet for the household. The quick-start guide includes: "If your ScoutUTM is not working, bypass it by connecting your ISP modem/router directly to your computer or switch. This restores internet while you troubleshoot." This buys time without requiring immediate support response.

**Phase 2 (Scout+ tier):** Priority email support with 12-hour response target for critical issues. Chat support during business hours.

---

## 10. Financial Goals & Business Viability

### Runway

**Total available budget: €35,000.** No external funding. No crowdfunding. Every euro must be justified.

### Budget Allocation (€35,000)

*Hardware purchase is deferred until Month 9-10 (after Beta gate). Months 1-8 are pure software development using host-native tools and Vagrant VMs -- no hardware spend. See PLAN.md section 13.*

| Category | Amount | When Spent | Notes |
|----------|--------|-----------|-------|
| **Development hardware** | €500 | Month 9-10 | 2x Odroid H4+ boards + RAM + SSD for dev/test |
| **Domain + hosting + infra** | €500 | Month 1 | Domain, VPS for OTA updates, static site hosting |
| **Legal / trademark** | €2,000 | Month 12+ | EUIPO trademark filing (~€850), terms of service, privacy policy |
| **CE pre-compliance testing** | €3,000 | Month 10-11 | EMC/safety pre-scan at EU test lab. Must pass before first unit ships. Funded from contingency. |
| **First sales batch (25 units)** | €7,500 | Month 11-12 | 25x Odroid H4+ kits (board + RAM + SSD + case) @ ~€300 each |
| **Packaging + branding** | €1,500 | Month 11-12 | Branded boxes, printed quick-start guide, labels |
| **Software licenses + tools** | €500 | Month 1 | Code signing cert, CI/CD, build infra |
| **Contingency / runway** | €16,500 | Months 1-12 | Living expenses + emergency buffer for 9-12 months |
| **Total** | **€35,000** | |

*This budget assumes the founder is the sole worker for the first 12+ months, doing development, assembly, QA, support, marketing, and fulfillment. No salaries, no contractors, no paid marketing. The deferred hardware approach preserves budget -- months 1-8 cost only ~€1,000 (domain + tools), leaving the bulk of the budget intact until the software is proven in VMs.*

### Revenue Model

ScoutUTM is a hardware appliance business. Every euro of revenue comes from selling a physical product. There is no software-only option.

- **Primary (85%+ of revenue):** ScoutUTM appliance sales (€499 / €699)
- **Secondary (10-15%):** Scout+ one-time upgrade (€99)
- **No software-only sales.** No licenses. No subscriptions. You buy a box, it works.

### Unit Economics (Odroid H4+ platform, EU-assembled)

| | **Scout** | **Scout Pro** |
|--|----------|--------------|
| Retail price (VAT-inclusive) | €499 | €699 |
| VAT at 21% (remitted to tax authority) | -€87 | -€121 |
| **Revenue ex-VAT** | **€412** | **€578** |
| Odroid H4+ / H4 Ultra (EU distributor) | ~€200 | ~€300 |
| 8GB / 16GB DDR5 SO-DIMM | ~€25 | ~€50 |
| 128GB / 256GB NVMe SSD | ~€15 | ~€25 |
| Case (Hardkernel Type 1) | ~€15 | ~€15 |
| Assembly labor (flash, test, burn-in) | ~€15 | ~€15 |
| Packaging + quick-start guide | ~€10 | ~€10 |
| Shipping materials | ~€5 | ~€5 |
| **Total landed cost** | **~€285** | **~€420** |
| **Gross margin (on ex-VAT revenue)** | **~31%** | **~27%** |
| **Gross profit per unit** | **~€127** | **~€158** |

**Shipping:** Charged separately at checkout (not included in retail price). EU tracked+insured: ~€10-15/unit. Shipping is pass-through -- not absorbed into COGS, does not affect gross margin. US shipping deferred until FCC certification (Phase 3+).

Scout+ attachment rate target: 35% of customers at €99 VAT-incl (€82 ex-VAT) = ~€28.70 blended ex-VAT revenue per unit (near 100% margin)

**VAT impact on margins:** The 31% gross margin (ex-VAT) is significantly lower than the 43% headline number. This is the real margin the business operates on. At bootstrap scale with EU-only sales, VAT is a major factor. Mitigations: Scout+ software revenue (near 100% margin after VAT), and potential future price adjustment once brand is established.

### Year 1-3 Financial Targets (EUR) -- Hardware-First Bootstrap

*Year 1 is almost entirely development (12 months). First sales are Month 12-14. Revenue ramps in Year 2.*

| Metric | Year 1 (M1-12) | Year 2 (M13-24) | Year 3 (M25-36) |
|--------|--------|--------|--------|
| Units sold | 10-15 | 200-300 | 1,000-1,500 |
| Avg selling price (VAT-incl) | €530 | €550 | €560 |
| Avg revenue ex-VAT (at 21%) | €438 | €455 | €463 |
| Hardware revenue (ex-VAT) | €4.4-6.6K | €91-136K | €463-694K |
| Scout+ revenue (35% attach, ex-VAT) | €0 | €5.7-8.6K | €28.7-43K |
| **Total revenue (ex-VAT)** | **€4.4-6.6K** | **€97-145K** | **€492-737K** |
| COGS | €3.1-4.7K | €62-93K | €310-465K |
| **Gross profit** | **€1.3-1.9K** | **€35-52K** | **€182-272K** |
| Operating expenses (business only, see note) | €2K | €25K | €80K |
| **Net income** | **-€0.7 to -€0.1K** | **€10-27K** | **€102-192K** |

*Year 1 is build + prove: 12 months of development, first 10-15 units sold in the final months. Months 1-8 are pure software development with zero revenue and minimal spend (~€1K). Hardware purchased Month 9-10, CE pre-compliance Month 10-11, first batch ordered Month 11-12, first sales Month 13-14 (after CE pre-compliance pass). Year 1 revenue is intentionally small -- the goal is a working, tested, CE-compliant product, not premature sales. Year 2 is the inflection point: reinvest profits into larger batches, reach Amazon, get reviews, launch Scout+. Year 3 is where it becomes a real business. **Note:** Year 3 projections (83-125 units/month) are aspirational stretch targets, not forecasts. The business is sustainable at much lower volumes (~15-20 units/month). These figures assume successful Amazon launch, review momentum, and growing brand awareness -- all of which are uncertain from a standing start.*

**OpEx note:** Operating expenses are **business costs only** -- founder salary/living expenses are NOT included. The founder needs separate income (freelance, savings, day job) until ~Month 18-20 when the business can sustain a salary. The €16,500 contingency covers personal runway during Year 1. Year 2 OpEx breakdown: infrastructure ~€200/year, full CE certification ~€10-15K, EUIPO trademark ~€850, Amazon FBA setup ~€2K, packaging/shipping/returns ~€7-12K.

### The First 25 Units: Make-or-Break

The entire business hinges on selling the first batch of 25 units (ordered Month 11-12, sold Month 13-14 after CE pre-compliance pass):

| | Amount |
|--|--------|
| Cost to build 25 units | ~€7,500 |
| Revenue if all 25 sell (avg €530 VAT-incl, €438 ex-VAT) | ~€10,950 ex-VAT |
| Gross profit from first batch (ex-VAT) | ~€3,450 |
| **This funds:** | Next batch of ~15-20 units |

2 units held as warranty/RMA reserve. 5-10 go to beta testers (from Discord community) at launch. Remaining 13-18 are for sale. If the first batch sells within 2-3 months, the business is self-sustaining. Each batch funds the next, slightly larger batch. If they don't sell, the product needs rethinking before spending more.

### Milestones That Unlock Growth

| Milestone | Units Sold | What It Enables |
|-----------|-----------|-----------------|
| First 25 units sold | 25 | Validates product-market fit, funds next batch |
| 50 cumulative units | 50 | CE pre-compliance testing, EUIPO trademark |
| 100 cumulative units | 100 | Amazon.de listing (FBA), first reviewer units |
| €50K cumulative revenue (ex-VAT) | ~115 units | Consider first part-time contractor |
| €200K cumulative revenue (ex-VAT) | ~460 units | Full CE certification, WEEE registration |
| €500K cumulative revenue (ex-VAT) | ~1,140 units | First full-time hire, custom PCB exploration |

### Breakeven
- Every unit sold is profitable from unit #1 (~31% gross margin ex-VAT)
- Initial investment (€7,500 first batch) recovered after selling ~25 units (ex-VAT profit ~€127/unit)
- Company breakeven (covering all startup costs + operating expenses): ~unit 50-60
- Founder salary possible: ~Month 18-24 (when cumulative profit allows it)

### Post-Launch Sustainability Model

**The hard truth:** ScoutUTM is not self-sustaining in Year 1. The founder needs external income (freelance, savings, or a day job) until Year 2 volume covers living expenses. This is implicit in the €16,500 contingency/runway but should be stated plainly.

**Ongoing operational costs (monthly, post-launch):**

| Item | Monthly Cost |
|------|-------------|
| VPS (OTA server, website) | ~€10 |
| Domain + DNS | ~€2 |
| Code signing cert (amortized) | ~€5 |
| CI/CD (GitHub Actions free tier) | €0 |
| **Total infrastructure** | **~€17/month** |

Founder labor is the dominant cost but valued at €0 during bootstrap. The infrastructure floor is very low -- keeping devices patched costs under €200/year.

**Revenue per unit vs. ongoing cost (ex-VAT):**

| | Per Unit |
|--|---------|
| Gross profit (Scout, ex-VAT) | ~€127 |
| Blended Scout+ revenue (35% attach, ex-VAT) | ~€29 |
| **Total per unit** | **~€156** |

At Year 1 volumes (10-15 units), total gross profit is €1,560-2,340. This covers infrastructure but does not fund a founder salary. At Year 2 volumes (200-300 units), gross profit is €31-47K -- enough for a modest founder salary and reinvestment.

**If sales stall:** The automated on-device pipeline continues working without any server infrastructure -- Suricata rules update daily from ET Open, DNS blocklists update daily from public sources. The OTA server is a ~€10/month VPS. The absolute minimum cost of keeping shipped devices secure is under €200/year. Security patches require founder time but no meaningful capital.

**Break-even for founder salary:** ~25-30 units/month sustained (~€3,900-4,700/month gross profit ex-VAT). Realistic from Month 20-24 based on Year 2 projections.

### Inventory Management (Critical at €35K)

With a tiny budget, inventory management is life or death:

- **Never have more than 1 batch worth of unsold inventory.** If 25 units sit unsold, the business is frozen.
- **Build-to-order where possible.** Accept orders on the website, assemble within 3-5 business days, ship. Customers accept this from a boutique brand.
- **Keep 5-10 units in stock** once cash flow allows, for immediate shipping.
- **Odroid H4+ lead time:** ~1-2 weeks from EU distributors. Order next batch when current batch is 50% sold.
- **Track sell-through rate** obsessively. If units aren't moving, stop buying and figure out why.

### Funding Strategy

**No external funding.** Self-funded with €35,000 runway. Revenue funds growth.

If growth exceeds projections, consider:
- **EU grants** (Horizon Europe, Digital Europe, national innovation funds) -- free money but slow and bureaucratic
- **Revenue-based financing** -- borrow against proven revenue, no equity dilution
- **Angel investment** -- only if a strategic angel (e.g., cybersecurity industry veteran) adds value beyond money
- **Reinvestment** -- keep margins lean, reinvest 80%+ of profit into growth

### Key Financial Risks
- **Runway exhaustion** -- €35K is 9-12 months of survival. The product must be ready to sell within ~12 months or the project stalls. The deferred hardware approach helps -- months 1-8 cost only ~€1K total, preserving runway. Mitigation: keep a day job / freelance income stream until hardware is generating revenue. The business is not self-sustaining until Year 2 volume -- see Post-Launch Sustainability Model above.
- **Unsold inventory** -- 25 unsold units = €7,500 tied up (21% of total budget). Mitigation: build-to-order model, pre-launch email list to gauge demand, don't order the full batch until you have at least 10 pre-orders or strong interest signals.
- **Single-person dependency** -- Founder gets sick, everything stops. Mitigation: automate everything possible (CI/CD, OTA updates, order notifications).
- **CE certification cost** -- Full certification (~€10-25K) can't be paid from initial budget. Mitigation: CE pre-compliance testing (~€3K) in Month 10-11 from contingency, providing basis for self-declaration of conformity before first sale. Full test house certification funded from revenue before Amazon/retail marketplace launch.
- **Returns/RMA** -- Even one return on a 25-unit batch hurts (4% of inventory). Mitigation: thorough QA before shipping (automated test script on every unit), generous troubleshooting support to avoid unnecessary returns.
- **Odroid H4+ supply disruption** -- Board supply is constrained, prices rising. Mitigation: qualify a second platform (Deciso DEC670 as fallback), monitor stock levels at EU distributors.

---

## 11. Product Roadmap

*Roadmap assumes solo founder. Timelines are conservative. Each phase is funded by the previous one. Aligned with PLAN.md development timeline (12 months to v1.0). Hardware is deferred until Month 9-10.*

### Phase 0: Software Development (Months 1-8) -- Budget: ~€500

*Pure software development. No hardware purchase. All testing in Vagrant VMs and host-native (Tier 1 + Tier 2 per PLAN.md).*

- [ ] scoutd skeleton (Go daemon, config schema, REST API, WebSocket)
- [ ] Vagrant VM dev environment (Alpine Linux, all daemons)
- [ ] Stateful firewall + NAT + DHCP/DNS (nftables + dnsmasq + Unbound)
- [ ] Three-phase config commit with rollback
- [ ] VLAN support and zone-based segmentation
- [ ] WireGuard VPN server + peer management + QR codes
- [ ] IDS integration (Suricata NFQUEUE, IDS mode default, IPS opt-in)
- [ ] DNS filtering (malware, phishing, ads via Unbound RPZ)
- [ ] Parental controls (per-device profiles, schedules, SafeSearch, YouTube restricted)
- [ ] Vue.js web dashboard (all views: Dashboard, Firewall, DNS, VPN, Devices, IDS, Parental, Network, System)
- [ ] 7-step setup wizard
- [ ] Device discovery and fingerprinting
- [ ] Integration test suite (11 scripts validated in Vagrant VM)
- [ ] SBOM generation and GPL source archive in CI pipeline
- [ ] **Gate: Alpha (Month 4) -- firewall + DNS + VPN working in VM**
- [ ] **Gate: Beta (Month 8) -- all features functional in VM**

### Phase 1: Image Build + Hardware + First Sales (Months 9-14) -- Budget: ~€12,000

*Hardware purchased Month 9-10. CE pre-compliance Month 10-11 (funded from contingency, €3K). Image build tested in QEMU first, then on hardware. First batch assembled Month 12, first sales Month 13-14 (after CE pre-compliance pass).*

- [ ] SquashFS rootfs build + dm-verity integrity
- [ ] Disk image assembly (GPT, GRUB A/B boot, custom initramfs with OverlayFS)
- [ ] RAUC OTA integration + bundle signing (ECDSA P-256)
- [ ] Test image build in QEMU VM (Month 9)
- [ ] **Purchase Odroid H4+ hardware (Month 9-10, ~€500 for 2 dev boards)**
- [ ] Flash and boot on hardware, validate Intel I226-V NIC driver
- [ ] **Hardware throughput benchmarks (Month 10) -- results define product specs**
- [ ] **CE pre-compliance testing at EU test lab (Month 10-11, ~€3K) -- must pass before first sale**
- [ ] Full QA test suite on hardware (boot time, all services, throughput, thermal)
- [ ] USB recovery mode testing
- [ ] **Gate: Release Candidate (Month 10) -- boots and passes QA on hardware**
- [ ] ScoutUTM.com storefront (bank transfer checkout)
- [ ] Order first sales batch (25x Odroid H4+ kits from EU distributor, ~€7,500)
- [ ] Assemble, flash, test, package all 25 units
- [ ] **Gate: v1.0 + CE pre-compliance pass (Month 12) -- ready for beta testers**
- [ ] Beta program (5-10 units to early testers from Discord community)
- [ ] Quick-start guide + documentation
- [ ] Public launch (Show HN, Reddit, Discord)
- [ ] **First hardware sales Month 13-14 (€499/unit)**
- [ ] EUIPO trademark filing

### Phase 2: General Availability (Months 15-24) -- Budget: funded by first batch revenue

- [ ] GeoIP blocking
- [ ] Advanced parental reporting and analytics (Scout+ tier, 90-day retention)
- [ ] Scout+ tier launch (€99 one-time upgrade)
- [ ] Documentation site + community forum
- [ ] Scout Pro tier launch (Odroid H4 Ultra based, €699)
- [ ] Larger batches (50 units at a time)
- [ ] Investigate Secure Boot + TPM for v1.5 (PLAN.md section 10.10)

### Phase 3: Growth (Months 24-36) -- Budget: funded by sustained hardware revenue

- [ ] Full CE certification (when revenue supports ~€10-25K cost)
- [ ] WEEE registration (DE as first market)
- [ ] OpenVPN support
- [ ] Multi-WAN / failover
- [ ] Multi-site management (Scout+ feature)
- [ ] Amazon.de listing (FBA)
- [ ] YouTube tech reviewer outreach (send 3-5 review units)
- [ ] Batches of 100+ units, better component pricing
- [ ] Consider first part-time contractor (support or development)

### Phase 4: Scale (Months 36+) -- Budget: funded by growth revenue

- [ ] Custom PCB exploration (Eurocircuits + EU assembler) for better margins
- [ ] Native mobile app (iOS + Android)
- [ ] IPv6 dual-stack support
- [ ] QoS / traffic shaping
- [ ] Smart home protocol awareness (Matter, Thread)
- [ ] Home Assistant integration API
- [ ] Broader EU Amazon marketplaces
- [ ] EU retail channel exploration
- [ ] First full-time hire
- [ ] US/UK expansion (FCC/UKCA when budget allows)
- [ ] BSI certification pursuit
- [ ] EU Cyber Resilience Act full compliance (ahead of Dec 2027 deadline)

---

## 12. Success Metrics & KPIs

### Product
- Setup completion rate: >90% within 30 minutes
- Mean time to detect threat: <60 seconds
- False positive rate: <2%

### Business (Year 1 -- bootstrap targets)
- Hardware units sold: 10-15 (partial first batch, sales start Month 12)
- Scout+ attach rate: >30%
- Customer acquisition cost (CAC): ~€0 (organic only)
- Revenue: >€5K
- Burn rate: <€2K/month months 1-8 (infra only), <€3K/month months 9-12 (hardware phase)
- Hardware return rate: <5% (1 return max on first batch)

### Business (Year 2+ -- growth targets)
- Hardware units sold: 200-300+
- Monthly revenue growth: 10-15%
- Customer lifetime value (LTV): >€530 (appliance + Scout+)
- Net Promoter Score (NPS): >60
- Security update response time: <72 hours for Critical/High CVEs
- OTA update adoption: >80% of devices within 7 days of release

### Community
- Discord members: 500+ by end of Year 1
- Active beta testers: 20-50 during MVP phase
- Tech reviewer coverage: 2-3 niche reviews in Year 1, 10+ in Year 2

---

## 13. Team Requirements

### Year 1: Solo Founder

The founder does everything: systems engineering, web development, security configuration, UX design, DevOps, marketing, support, and business operations. This is the reality of a €35K bootstrap.

**Required founder skills:**
- Linux systems engineering (Alpine, kernel, networking, nftables)
- Web development (dashboard UI, API)
- Network security fundamentals (IDS/IPS, firewall policy, VPN)
- Basic UX sense (doesn't need to be a designer, but must care about usability)
- Self-motivation to ship without external accountability

**Post-launch reality (from Month 12):** After v1.0 ships, the founder's time splits between feature development and ongoing security maintenance. Estimated allocation: 8-12 hours/month on security maintenance (CVE triage, patches, OTA releases), 8-12 hours/month on customer support, and remaining time on feature development. **Security maintenance is non-negotiable** -- a UTM appliance that ships known vulnerabilities is worse than one that ships late features. See PLAN.md section 16 for the full operational sustainability plan.

### Year 2: First Help (funded by revenue)

| Role | Priority | When | Notes |
|------|----------|------|-------|
| Part-time support / community | First | ~€50K cumulative revenue | Handle Discord, email support, docs |
| Contract security auditor | Second | ~€100K cumulative revenue | One-time engagement for external audit |
| Part-time developer | Third | ~€200K cumulative revenue | Web UI, mobile app, or backend |

### Year 3: Small Team (funded by sustained revenue)

| Role | Priority | Notes |
|------|----------|-------|
| First full-time hire (generalist engineer) | Critical | OS + web + security -- needs to be a generalist |
| Part-time UX/UI designer | High | Dashboard, mobile app, setup wizard |
| Founder shifts to product + business | | Less coding, more strategy and growth |

---

## 14. Key Decisions Needed

- [x] **Hardware platform?** **DECIDED: Odroid H4+ (South Korea) with EU assembly.** "Designed and assembled in the EU, board manufactured in South Korea." Odroid H4+ offers dual 2.5GbE Intel NICs, Intel N97 with AES-NI, DDR5, at ~€200/board from EU distributors. Assembled, flashed, tested, and shipped from EU by founder. Hardware purchase deferred to Month 9-10 (after software PoC validated in VMs). Fallback: Deciso DEC670 (fully EU-made, but lower margin). Custom PCB at Phase 4 (€500K+ revenue). See Section 16.
- [x] **Sales model?** **DECIDED: Hardware appliance only. No software-only sales.** A new brand needs a physical product customers can see, hold, and trust. Software-only at €149 is uncompetitive against free pfSense/OPNsense and offers no tangible brand experience. Every ScoutUTM sale is a complete, ready-to-use appliance.
- [x] **Open-source strategy?** **DECIDED: Proprietary / closed-source.** ScoutUTM is a commercial product. Source code is not published. Trust is built through independent security audits, EU-origin transparency, and published security architecture docs -- not through source access. This protects IP, enables faster iteration without community governance overhead, and preserves Scout+ value.
- [x] **Base OS?** **DECIDED: Alpine Linux.** Custom build on Alpine Linux. Minimal footprint (~5MB base), musl libc, security-oriented, fast boot, small attack surface. No forking OPNsense/pfSense -- clean break from their UI/UX baggage and licensing entanglements (pfSense's license restricts redistribution). Alpine's apk package manager enables modular feature delivery. Suricata, WireGuard, nftables, Unbound, RAUC all available in Alpine repos. See PLAN.md section 1 for Alpine vs OpenWrt rationale.
- [x] **Crowdfunding vs. bootstrap?** **DECIDED: Bootstrap. No crowdfunding.** Total runway is €35,000. No external funding, no crowdfunding campaigns. This means: build the product, sell hardware appliances direct, reinvest revenue into growth. No custom PCB runs until revenue funds them. Founder does everything. Every euro counts. See Section 10 for budget allocation.
- [x] **Chip platform?** **DECIDED: x86 (Intel N97 / i3-N305).** AES-NI required for VPN performance. x86 needed for full Alpine Linux compatibility and Suricata performance. ARM deferred to potential future "Scout Lite" tier.
- [x] **"Made in EU" definition?** **DECIDED: "Designed and assembled in the European Union. Board manufactured in South Korea."** Full BOM origin breakdown published on website. Transparency is the differentiator, not vague "Made in EU" claims.

### Deferred Until Working Prototype

These decisions require a working product before they make sense to finalize:

- [ ] **Name "ScoutUTM" final?** Trademark search (EUIPO + national offices). No point filing until the product exists. Working name is fine for development. Budget: ~€850 for EUIPO filing.
- [ ] **CE marking scope?** EMC + LVD + RoHS minimum. RED (Radio Equipment Directive) only if Wi-Fi variant. Cannot test without final hardware + software. Budget: €10-30K. Trigger: after first 50-100 units sold direct (pre-marketplace).
- [ ] **EU grant applications?** Not urgent. €35K runway is sufficient to reach first sales. Revisit only after product is shipping and there's bandwidth to handle bureaucracy.
- [ ] **WEEE compliance?** Required before EU hardware sales at scale. Register in each target country or use a compliance scheme. Trigger: before Amazon/retail listing.

---

## 15. Why Now?

1. **WFH is permanent** -- 40%+ of knowledge workers are hybrid/remote. Home networks ARE corporate perimeters now.
2. **IoT explosion** -- Average household has 22+ connected devices (2026). Every one is an attack vector.
3. **Subscription fatigue is real** -- Consumers are revolting against recurring fees. "No subscription" is a competitive weapon.
4. **Cyber insurance requirements** -- Even small businesses face coverage requirements that demand network security.
5. **Privacy regulation** -- GDPR, state privacy laws, COPPA enforcement increasing. Families and businesses need tools to comply.
6. **AI-powered threats** -- Phishing, deepfakes, and automated attacks are more sophisticated. Consumer routers can't keep up.
7. **Smart home adoption** -- Matter protocol and Thread are mainstreaming IoT, but security hasn't kept pace.
8. **EU Cyber Resilience Act** -- Compliance deadlines in 2026-2027 will disrupt the market. Products must prove supply chain security, provide SBOMs, and maintain vulnerability disclosure. ScoutUTM generates CycloneDX SBOMs from the first build, publishes GPL source archives per release, and has a documented CVE response process -- built-in, not bolted on.
9. **Digital sovereignty movement** -- Post-Snowden, post-SolarWinds, EU institutions and citizens increasingly distrust non-EU tech. "Made in EU" is becoming a purchasing factor, not just a label.
10. **EU Chips Act momentum** -- €43B in public/private investment is building EU semiconductor capacity. The ecosystem for EU-origin hardware is maturing rapidly.

---

## 16. EU Hardware Sourcing Strategy

### Policy

**All ScoutUTM products are designed and assembled within the European Union.** Board manufacturing is sourced from trusted partners outside the EU (currently South Korea). Full supply chain is published transparently. This is a core brand commitment, not a nice-to-have. It enables supply chain transparency, regulatory alignment, and market differentiation while keeping costs viable at bootstrap scale.

### What "EU-Origin" Means for ScoutUTM (Honest Definition)

| Layer | EU-Sourced? | Details |
|-------|-------------|---------|
| Product design & engineering | **Yes** | In-house, EU-based founder/team |
| ScoutOS firmware & software | **Yes** | Developed in EU |
| Final assembly, QA & flashing | **Yes** | EU-based (founder workshop, later EU EMS partner) |
| Packaging & fulfillment | **Yes** | EU-based |
| Main board (PCB + SMT) | No | Hardkernel, South Korea |
| CPU (x86) | No | Intel (US/Israel/TSMC) |
| Ethernet NIC chipset | No | Intel I226-V (US) |
| RAM | No | Samsung/SK Hynix (South Korea) |
| SSD | No | Various (South Korea, Japan, US) |
| Enclosure | No | Hardkernel (South Korea) |

**Honest claim: "Designed and assembled in the European Union. Board manufactured in South Korea."** We do not hide the origin of any component. Full BOM origin breakdown published on ScoutUTM.com. Transparency IS the differentiator.

### EU Hardware Partner Options

#### Option A: Deciso B.V. (Netherlands) -- RECOMMENDED for v1

| | Details |
|--|---------|
| **Location** | Netherlands |
| **What they do** | Design and manufacture network security appliances |
| **Existing products** | DEC600-DEC4200 series (desktop and rack) |
| **Relevance** | They ARE the OPNsense hardware partner -- proven network appliance ODM |
| **Engagement model** | White-label/ODM their existing platform, or co-develop custom board |
| **Pros** | Proven, fast time-to-market, Dutch engineering, existing supply chain, CE-certified designs |
| **Cons** | Dependency on single vendor, may have MOQ requirements, their brand is tied to OPNsense, **competitor conflict** (see note below) |
| **Example platform** | DEC750: 8GB DDR4, 256GB SSD, up to 8.5 Gbps firewall throughput |
| **Website** | deciso.com |

**Competitor conflict note:** Deciso manufactures OPNsense appliances and is listed as a direct competitor (Section 3). They may decline to supply a competing product, or impose restrictions. The ODM engagement should only be explored if/when ScoutUTM reaches volumes that make it worthwhile for Deciso (Phase 2+), and only for hardware -- ScoutOS is a completely different software stack. For Phase 1, ScoutUTM uses Odroid H4+ from standard distributors (no ODM relationship needed). If Deciso declines, Kontron or custom PCB (Options B/C) are alternatives.

#### Option B: Kontron AG (Germany) -- Best for Custom Designs

| | Details |
|--|---------|
| **Location** | Germany (S&T Group) |
| **What they do** | Embedded computing, COM modules, industrial PCs, network appliances |
| **Manufacturing** | 20+ plants globally, EU assembly available |
| **Engagement model** | Use their COM (Computer-on-Module) platform + custom carrier board |
| **Pros** | Massive scale, flexible, industrial-grade reliability, German engineering |
| **Cons** | More complex engagement, longer lead time for custom work, enterprise-oriented |
| **Partner** | Congatec (Deggendorf, Germany) for COM modules |
| **Website** | kontron.com |

#### Option C: Custom Build -- EU PCB + EU Assembly

| Layer | Partner | Location |
|-------|---------|----------|
| **PCB fabrication** | Eurocircuits | Belgium (HQ), Germany, Hungary |
| **SMT assembly** | Assel | Gdansk area, Poland |
| **Enclosure** | Local EU metal fabricator | Germany/Poland/Czech Republic |
| **Final assembly + test** | Assel or dedicated EMS | Poland |

| | Details |
|--|---------|
| **Pros** | Full control over design, highest margins at scale, no ODM dependency |
| **Cons** | Longest time-to-market, highest upfront NRE costs, need own hardware engineer |
| **Best for** | Phase 2+ when volume justifies custom design |

#### Option D: SECO (Italy/Germany) -- ARM Platform

| | Details |
|--|---------|
| **Location** | Manufacturing in Germany and Italy |
| **What they do** | ARM and x86 SBCs, Computer-on-Modules |
| **Relevance** | Could provide ARM-based platform for Scout Lite (budget tier) |
| **Pros** | ISO 9001 certified, EU manufacturing, ARM expertise |
| **Cons** | Less network-appliance-specific, would need custom carrier board |
| **Website** | seco.com |

### Recommended Phased Approach (€35K Bootstrap, Hardware-First)

**Phase 0 (development, Months 1-8):** No hardware purchase. ScoutOS developed and tested entirely on host (Go/Vue.js) and in Vagrant VMs (Alpine + all daemons). All features validated in VMs before hardware spend.

**Phase 0.5 (hardware validation, Month 9-10):** Buy 2x **Odroid H4+** kits from EU distributor (Antratek NL or Pollin DE). Total cost: ~€500. Validate NIC drivers, run throughput benchmarks, test production image on real hardware.

**Phase 1 (first sales batch, Month 11-12):** Order 25x Odroid H4+ boards + RAM + SSD from EU distributors. Assemble, flash ScoutOS, run QA, package, and ship from founder's workshop in the EU. Total investment: ~€7,500. Revenue if all sell: ~€13,250.

**Primary hardware platform:**
| Component | Source | Country | Price (approx) | Notes |
|-----------|--------|---------|----------------|-------|
| Odroid H4+ board | Hardkernel (via EU distributor) | South Korea | ~€200 | 2x 2.5GbE Intel I226-V, Intel N97, AES-NI |
| 8GB DDR5 SO-DIMM | Standard component | South Korea | ~€25 | |
| 128GB NVMe SSD | Standard component | Various | ~€15 | |
| Hardkernel Type 1 case | Hardkernel (via EU distributor) | South Korea | ~€15 | Includes fan |
| **Total BOM per unit** | | | **~€255** | |

**Fallback hardware (if H4+ supply disrupted):**
| Device | Manufacturer | Country | Price | Ports | Notes |
|--------|-------------|---------|-------|-------|-------|
| Deciso DEC670 | Deciso | Netherlands | ~€287 | 4x GbE | Fully EU-made, but lower margin |
| Intel N100 4-port | Various (CWWK, Topton) | China | ~€160 | 4x 2.5GbE | Cheapest, but Chinese origin |

**Phase 2 (growth, funded by revenue):** Larger batches (50-100 units). Better pricing from Hardkernel's EU distributors at volume. Add Scout Pro tier (Odroid H4 Ultra). Consider dedicated EU assembly partner (Olimex Bulgaria, Assel Poland) when volumes exceed what founder can handle (~10+ units/week).

**Phase 3 (custom hardware, funded by sustained revenue at €500K+):** Develop custom PCB with **Eurocircuits** (Belgium) for fabrication and **Assel** (Poland) for assembly:
- Higher margins (~10-15pp improvement)
- Hardware tailored to ScoutUTM needs (optimized port layout, branding on PCB, custom case)
- No Hardkernel dependency
- Full control over BOM and supply chain

**Phase 4 (maturity):** Evaluate **SECO** (Italy/Germany) or **Kontron** (Germany) COM-based approach for fully EU-manufactured modular hardware platform.

### EU PCB Fabrication Partners

| Company | Location | Capacity | Specialty | MOQ |
|---------|----------|----------|-----------|-----|
| **Eurocircuits** | Belgium/Germany/Hungary | 110K+ orders/yr | Prototyping through production, 8 SMD lines | From 1 piece |
| **Assel** | Gdansk, Poland | 40+ years experience | Complex PCBA, box build, systems integration | Flexible |
| **Kuttig Electronic** | Germany | Small-medium runs | PCB assembly, 1-10K pieces/lot | 1 piece |
| **AISLER** | Germany | Quick-turn | Rapid prototyping, affordable small runs | 3 pieces |

### EU Regulatory Requirements for Hardware

| Requirement | What It Is | Cost Estimate | Timeline |
|-------------|-----------|---------------|----------|
| **CE marking** | Mandatory for EU market (EMC + LVD + RoHS) | €10-25K | 3-6 months |
| **WEEE** | Waste electronics registration per country | €2-5K/year | 1-2 months |
| **RoHS** | Hazardous substances restriction | Included in CE | Included |
| **REACH** | Chemical safety compliance | €1-3K | 1-2 months |
| **CRA (Cyber Resilience Act)** | Software security, SBOM, vulnerability disclosure | Engineering effort | By Dec 2027 |
| **RED** | Radio Equipment Directive (only if Wi-Fi built in) | €15-30K additional | 4-8 months |
| **BSI certification** | German security certification (optional but valuable) | €30-100K | 6-12 months |
| **UKCA** | UK market access (post-Brexit) | €5-15K | 2-4 months |

### EU vs. Asian Manufacturing: Honest Comparison

| Factor | EU Manufacturing | Asian Manufacturing |
|--------|-----------------|-------------------|
| Unit cost (assembly) | €40-70 per unit | €15-30 per unit |
| PCB fabrication | €25-45 per board | €8-15 per board |
| Minimum order qty | 50-500 units typical | 500-5,000 units typical |
| Lead time (production) | 4-8 weeks | 8-12 weeks |
| Shipping to EU customer | 1-3 days | 4-8 weeks (sea) |
| Communication | Same timezone, no language barrier | Time zone challenges |
| IP protection | Strong EU legal framework | Variable |
| Supply chain transparency | High, auditable | Often opaque |
| Marketing value | "Made in EU" premium | None / negative for some buyers |
| CE compliance | Built into process | Must be managed separately |
| Total landed cost premium | **+20-40%** | Baseline |

### Long-Term EU Silicon Watch List

These won't help in 2026, but are worth tracking for future hardware revisions:

| Initiative | What | When | Relevance |
|-----------|------|------|-----------|
| **SiPearl Rhea** (France) | 80-core ARM HPC processor | Sampling 2026 | Too powerful/expensive for UTM, but proves EU can build CPUs |
| **Kalray MPPA** (France) | Massively parallel processor | Available now | Packet processing potential, but niche |
| **EU RISC-V initiatives** | Open-source CPU architecture | 2027-2030 | Could enable EU-designed network SoC |
| **TSMC Dresden fab** | EU chip fabrication | Operational 2027 | Automotive focus, but expands EU fab capacity |
| **NanoIC Belgium** | Beyond-2nm pilot line | Operational 2026 | R&D only, but builds EU chip ecosystem |
| **EU Chips Act projects** | €43B investment in EU semiconductors | 2024-2030 | Gradual improvement in EU component availability |

---

## 17. EU Compliance & Certification Roadmap

*Aligned with PLAN.md development timeline: hardware available Month 9-10, v1.0 at Month 12.*

| Milestone | Target Date | Dependency |
|-----------|-------------|------------|
| CRA SBOM generation pipeline | Month 1 (from first build) | CI/CD pipeline (syft + CycloneDX in `make bundle`) |
| GPL source archive automation | Month 1 (from first build) | Build pipeline archives Alpine APK sources |
| CRA vulnerability disclosure process | Month 8 (Beta gate) | Security update pipeline documented (PLAN.md section 16) |
| CE pre-compliance testing (prototype) | Month 10-11 | Hardware available, funded from contingency (~€3K). Must pass before first sale. |
| RoHS/REACH documentation | Month 14 | BOM finalized with hardware platform |
| WEEE registration (DE as first market) | Month 15 | Before Amazon/retail listing |
| CE marking achieved | Month 18 | Final hardware design frozen, funded by revenue (~€10-25K) |
| Secure Boot investigation (v1.5) | Month 15-18 | v1.0 stable, bandwidth available |
| BSI certification application | Month 24+ | Stable product, revenue to fund it (~€30-100K) |
| Full CRA compliance | Month 24 | Ahead of Dec 2027 deadline |
| Privacy policy + GDPR documentation | Month 12 (before first sale) | Legal budget (€2K), see GDPR section below |

### GDPR Compliance

**Privacy by design:** ScoutUTM's architecture minimizes personal data processing by ScoutUTM (the company):

| Data | Where Stored | Who Controls It | Leaves the Device? |
|------|-------------|----------------|-------------------|
| DNS query logs (per-device domains) | On-device only (`/var/lib/scout/logs/`) | Device owner (family/business) | No |
| DHCP leases (MAC, IP, hostname) | On-device only | Device owner | No |
| IDS alerts | On-device only | Device owner | No |
| Config (firewall rules, profiles) | On-device only | Device owner | No |
| OTA update checks | Update server | Anonymous -- static file GET, no device ID, no telemetry (see PLAN.md 7.5) | Only an HTTPS request to a static file |

**ScoutUTM is the product manufacturer, not a data processor.** On-device data is controlled by the device owner. ScoutUTM has no access to any data stored on customer devices. There is no cloud backend, no user accounts on ScoutUTM servers, no telemetry.

**What ScoutUTM does process (standard e-commerce):**
- **Order data:** Name, shipping address, email, bank transfer reference (Phase 1 — no card data stored; Stripe added in Phase 2+)
- **Support correspondence:** Email/Discord messages
- **Website analytics:** Privacy-respecting analytics only (e.g., Plausible — no Google Analytics, no cookies requiring consent)

**GDPR obligations (standard e-commerce, not data processor):**
- Privacy policy on ScoutUTM.com (what data is collected, how it's used, retention periods)
- Right to deletion: customer can request deletion of order/support data
- Data breach notification process (72-hour notification to DPA if customer data is breached)
- No Data Processing Agreements needed — ScoutUTM does not process personal data on behalf of customers

**Cost:** Covered within the existing €2K legal/trademark budget (privacy policy drafting is part of the terms of service legal work).

---

*Document version: 0.7 | Created: 2026-02-11 | Updated: 2026-02-12 | Status: Draft -- aligned with PLAN.md v0.9*

*Changelog:*
- *v0.1: Initial GTM draft.*
- *v0.2: EU-origin hardware requirement, sourcing strategy, compliance roadmap.*
- *v0.3: Decided closed-source / proprietary. Decided Alpine Linux.*
- *v0.4: Decided no crowdfunding, €35K bootstrap.*
- *v0.5: Hardware-appliance-only. Odroid H4+ platform. x86/Intel. "Made in EU" definition. Full financial rework.*
- *v0.6: Deferred name, certifications, and registrations until working prototype. All strategic decisions made.*
- *v0.7: Full alignment with PLAN.md v0.9. (1) Removed v2+ features from Core feature list -- only validated v1 features listed. Moved Multi-WAN, QoS, OpenVPN, mobile app, Matter/Thread/Zigbee, WAF, DDoS mitigation, UPnP, Dynamic DNS, network topology visualization to Future. (2) Added IDS-default/IPS-opt-in distinction with throughput caveat. (3) Added benchmark-first rule: no marketing throughput claims until Month 10 hardware benchmarks. (4) Updated platform table: added dm-verity, watchdog, SBOM/compliance, removed grsecurity. (5) Added Alpine vs OpenWrt rationale. (6) Realigned product roadmap: hardware deferred to Month 9-10, first sales Month 12-14, Phase 0 is pure software. (7) Adjusted Year 1-3 financials for later first-sales timing. (8) Updated budget allocation with deferred hardware spend. (9) Added operational sustainability concern to weaknesses (post-launch security maintenance burden). (10) Added musl allocator risk to weaknesses. (11) Realigned EU compliance timeline: SBOM from Month 1, CE pre-compliance deferred to Month 14-15. (12) Added honest throughput and verifiable integrity to strengths. (13) Updated team section with post-launch maintenance reality.*

*Decisions made: 8 of 8 active decisions (Section 14). 4 decisions deferred until working prototype (name, CE, grants, WEEE).*
*Next step: Build the prototype. Start with PLAN.md Month 1 -- scoutd skeleton, Vagrant VM, config schema. No hardware needed until Month 9.*
