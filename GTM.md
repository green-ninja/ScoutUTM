# ScoutUTM - Go-To-Market Strategy

## Executive Summary

ScoutUTM is a unified threat management appliance built for families and family-run businesses who need enterprise-grade network security without enterprise complexity or subscription fatigue. It fills the gap between overly technical open-source firewalls (pfSense, OPNsense) and consumer devices that lack depth (Firewalla), delivering powerful protection through a modern, approachable interface at a one-time cost.

**All hardware is designed and assembled in the European Union**, giving ScoutUTM a verifiable, trustworthy supply chain -- a strategic differentiator as EU regulations (Cyber Resilience Act, NIS2) increasingly demand supply chain transparency and digital sovereignty.

---

## 1. Product Vision

**One-liner:** Enterprise security for the rest of us.

**Problem:** Home networks now host 20-50+ connected devices -- laptops, phones, smart TVs, IoT sensors, security cameras, work-from-home setups, and kids' devices. The attack surface rivals small businesses, but the security tools available are either too complex (pfSense), too expensive with recurring fees (Fortinet, SonicWall), or too limited (consumer routers).

**Solution:** A purpose-built appliance that combines:
- Deep packet inspection and IDS/IPS at multi-gigabit speeds
- Intuitive, mobile-first management
- Smart home-aware network segmentation
- Parental controls that actually work
- No subscription fees for core security features
- EU-designed and EU-assembled hardware with a transparent, auditable supply chain

---

## 2. Target Audience

### Primary: Tech-Comfortable Families (TAM: ~35M EU + ~18M US households)
- Households with 15-50+ connected devices
- At least one technically inclined member (not necessarily IT professional)
- Privacy-conscious, aware of cyber threats (GDPR-aware in EU)
- Willing to invest 300-700 EUR/USD once for peace of mind
- Tired of configuring consumer routers that don't deliver
- Often have kids who need age-appropriate internet access
- EU customers who prefer locally manufactured, trustworthy hardware

### Secondary: Family-Run Small Businesses (TAM: ~23M EU + ~6M US businesses)
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
| **Firewalla** | Limited IDS/IPS depth, email-only support, no WAF, weaker business features |
| **pfSense/OPNsense** | Intimidating UI, no guided setup, no parental controls, no mobile app |
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
| Kernel | Hardened Linux, custom config, grsecurity patches where feasible |
| Root filesystem | Read-only (squashfs), overlay for config persistence |
| Init system | OpenRC (lightweight, no systemd) |
| Package delivery | OTA firmware images (not individual packages) |
| Footprint | ~50-80MB installed OS (vs. ~500MB+ for pfSense/OPNsense) |
| Boot time | Target <15 seconds to operational |
| Update model | A/B partition scheme for safe rollback |
| Licensing | Proprietary. Not open-source. |

**Why Alpine over Debian/Ubuntu:** 10x smaller attack surface, faster boot, no systemd complexity, musl libc is more security-auditable than glibc, and Alpine is already the standard base for security-focused container images.

**Why not fork OPNsense/pfSense:** Their UIs are deeply coupled to FreeBSD, their codebases carry decades of legacy, and pfSense's license restricts redistribution. Starting clean on Alpine lets us build the UX we want without inheriting technical debt.

### Core (Free Forever -- included with hardware)

**Network Security**
- Stateful firewall with zone-based policies
- IDS/IPS (Suricata-based) with curated rulesets
- DNS filtering and sinkholing (malware, phishing, ads)
- GeoIP blocking
- DDoS mitigation
- Automatic threat intelligence updates

**Network Management**
- VLAN support with guided segmentation wizard ("Work", "Kids", "IoT", "Guest")
- Multi-WAN / failover support
- DHCP and DNS server
- QoS / traffic shaping with application awareness
- UPnP with security guardrails
- Dynamic DNS

**VPN**
- WireGuard and OpenVPN server/client
- Site-to-site VPN for multi-location families/businesses
- Split tunneling
- VPN policy routing (route specific devices through commercial VPN)

**Family Controls**
- Per-device and per-user content filtering
- Age-based profiles (child, teen, adult)
- Screen time schedules and bedtime enforcement
- SafeSearch enforcement
- YouTube restricted mode
- Activity reporting (privacy-respecting -- summaries, not surveillance)

**Smart Home Aware**
- Automatic IoT device fingerprinting and profiling
- Matter, Thread, Zigbee, Z-Wave traffic awareness
- Micro-segmentation that doesn't break smart home automations
- IoT-specific anomaly detection (is your camera suddenly uploading to a foreign IP?)

**Interface**
- Modern web dashboard (responsive, dark/light mode)
- Mobile app (iOS + Android) for management and alerts
- Guided setup wizard (15-minute onboarding)
- Plain-language security alerts ("Your smart TV tried to contact a known malware server")
- Network topology visualization

### Scout+ (Optional -- one-time $99 upgrade, NOT a subscription)

- Advanced application-layer inspection
- Reverse proxy with automatic SSL (Let's Encrypt)
- Web Application Firewall (WAF) for self-hosted services
- Advanced reporting and analytics (90-day retention)
- Multi-site management (manage parents' ScoutUTM remotely)
- Priority email and chat support
- Early access to new features

### Future Considerations (v2+)
- Built-in Wi-Fi 7 AP option (ScoutUTM AX variant)
- PoE ports for cameras/APs
- Cellular failover (4G/5G modem slot)
- Home Assistant / smart home platform integration
- AI-driven behavioral analysis and anomaly detection

---

## 5. Strengths (Why ScoutUTM Wins)

1. **No subscription fees** -- Core security is a one-time purchase. Period. This alone differentiates from Fortinet, SonicWall, Untangle, and Ubiquiti's CyberSecure.

2. **The "prosumer sweet spot"** -- Enterprise features (IDS/IPS, VLANs, WAF) wrapped in a consumer-friendly UI. Nobody else nails both.

3. **Smart home native** -- First UTM to understand Matter/Thread/Zigbee protocols and segment IoT without breaking automations. Competitors either ignore IoT or break it.

4. **Family-first design** -- Parental controls are a first-class feature, not an afterthought bolted onto a business firewall.

5. **Transparent, honest pricing** -- No hidden costs, no "call for quote," no artificial feature gating on hardware you already own.

6. **Modern UX** -- Mobile-first, guided wizards, plain-language alerts. Designed in 2026, not 2006.

7. **Privacy-respecting** -- Local processing, no cloud dependency for core functions, no telemetry without consent.

8. **Dual-use design** -- Handles family AND small business needs on the same box with proper segmentation.

9. **Made in the EU** -- Designed, assembled, and supported within the European Union. Verifiable supply chain, no black-box offshore manufacturing. Aligns with EU Cyber Resilience Act and NIS2. Among consumer-facing UTMs, almost nobody else can claim this.

10. **Regulatory tailwind** -- EU Cyber Resilience Act (CRA) compliance deadlines in 2026-2027 will force all products with digital elements to prove supply chain security. ScoutUTM is built for this from day one, while competitors scramble to retrofit.

---

## 6. Weaknesses & Risks (Honest Assessment)

1. **New brand, zero trust** -- Firewalla has years of community trust, pfSense has decades. We start from zero. Mitigation: transparent security audits (independent third-party), community engagement, "Made in EU" trust signal, and published security architecture documentation.

2. **EU manufacturing cost premium** -- EU assembly is 20-40% more expensive than Asian manufacturing, squeezing margins. Mitigation: position the premium as a feature ("you're paying for trust, not just a box"), use cost-effective EU regions (Poland, Czech Republic, Hungary) for assembly, and offset with Scout+ software revenue.

3. **Component dependency on non-EU silicon** -- CPUs (Intel/AMD), NICs, and memory are not made in the EU. "EU-origin" means EU-designed and EU-assembled, not EU-fabbed silicon. Mitigation: be transparent about what "Made in EU" means; pursue EU PCB fabrication and final assembly; monitor EU Chips Act progress for future opportunities.

4. **Performance vs. price tension** -- Multi-gigabit IDS/IPS requires capable hardware, which drives up BOM cost -- compounded by EU assembly premium. Mitigation: tiered hardware (Scout 1G, Scout 2.5G), hardware acceleration, and partner with EU ODMs who already have production lines (Deciso, Kontron).

4. **Support burden** -- Home users expect more hand-holding than enterprise customers. Mitigation: excellent documentation, community forums, video tutorials, AI-assisted troubleshooting in the app.

6. **Feature scope creep** -- Trying to be everything to everyone. Mitigation: ruthless prioritization, clear "not for us" boundaries (no enterprise features like SIEM integration in v1).

7. **Open-source competition** -- pfSense and OPNsense are free and open-source. Mitigation: compete on experience, polish, and time saved -- not raw feature count. Our value is curation, UX, EU trust, and "it just works." Open-source users are rarely our target buyer anyway; they enjoy the tinkering.

8. **Market education** -- Many families don't know they need a UTM. Mitigation: content marketing focused on real-world threat scenarios, not technical jargon.

9. **Regulatory/compliance** -- If targeting businesses handling health/financial data, may need certifications. CE marking mandatory for EU hardware sales. Mitigation: design for compliance from day one, pursue BSI/CE certifications early, leverage EU-origin as compliance shortcut.

10. **EU ODM partner risk** -- Fewer EU hardware partners than Asia means less negotiating leverage and potential single-vendor dependency. Mitigation: qualify at least two EU assembly partners, maintain designs that are portable across manufacturers.

---

## 7. Pricing Strategy

### Hardware Tiers

ScoutUTM is a physical product. Every unit is assembled, tested, and shipped from the EU. There is no software-only option -- customers buy a complete, ready-to-use appliance. This is non-negotiable for building brand trust as a new entrant.

All prices include EU assembly. Prices shown in EUR (EU primary market) and USD equivalent.

| Model | Platform | Specs | EUR | USD | Target |
|-------|----------|-------|-----|-----|--------|
| **Scout** | Odroid H4+ | 2x 2.5GbE, Intel N97, 8GB DDR5, 128GB NVMe | €499 | $529 | Families, home offices |
| **Scout Pro** | Odroid H4 Ultra | 2x 2.5GbE, Intel i3-N305 8C, 16GB DDR5, 256GB NVMe | €699 | $739 | Power users, family businesses |

*"Designed and assembled in the European Union. Board manufactured in South Korea by Hardkernel."*
*Pricing reflects EU assembly, QA, branding, packaging, and support. Transparent supply chain published on ScoutUTM.com.*

### Software

| Tier | Price | Includes |
|------|-------|---------|
| **Core** | Included | All security features, parental controls, VPN, IoT awareness |
| **Scout+** | €99 / $99 one-time | WAF, reverse proxy, advanced analytics, multi-site, priority support |

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

---

## 8. Go-To-Market Channels

### Phase 1: Build + First Hardware Sales (Months 1-12) -- €0 marketing budget

**Launch Strategy: Build in Public (free channels only)**
- Personal blog / dev log (static site, free hosting)
- GitHub for community engagement (bug reports, feature requests, security disclosures)
- Reddit presence (r/homelab, r/homeautomation, r/selfhosted, r/netsec, r/smarthome, r/europrivacy)
- Discord community (free tier)
- Hacker News Show HN posts at key milestones
- No paid advertising, no influencer campaigns -- earn attention through the product

**Sales Model: Hardware appliance, direct only**
- ScoutUTM.com (simple storefront -- WooCommerce or Stripe checkout)
- Sell complete, ready-to-use ScoutUTM appliances (assembled and tested in EU)
- First batch: ~25 units from €35K budget (see Section 10)
- Ship from home / workshop -- founder does assembly, QA, and fulfillment
- EU VAT via OSS (One-Stop Shop) scheme to avoid per-country registration

### Phase 2: Growth + Marketplace (Months 12-24) -- funded by Phase 1 revenue

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

---

## 10. Financial Goals & Business Viability

### Runway

**Total available budget: €35,000.** No external funding. No crowdfunding. Every euro must be justified.

### Budget Allocation (€35,000)

| Category | Amount | Notes |
|----------|--------|-------|
| **Development hardware** | €800 | 2x Odroid H4+ boards + RAM + SSD for dev/test |
| **Domain + hosting + infra** | €500 | Domain, VPS for OTA updates, static site hosting |
| **Legal / trademark** | €2,000 | EUIPO trademark filing (~€850), terms of service, privacy policy |
| **CE pre-compliance testing** | €3,000 | EMC/safety pre-scan at EU test lab (not full cert yet) |
| **First sales batch (25 units)** | €7,500 | 25x Odroid H4+ kits (board + RAM + SSD + case) @ ~€300 each |
| **Packaging + branding** | €1,500 | Branded boxes, printed quick-start guide, labels |
| **Software licenses + tools** | €500 | Code signing cert, CI/CD, build infra |
| **Contingency / runway** | €19,200 | Living expenses + emergency buffer for 6-9 months |
| **Total** | **€35,000** | |

*This budget assumes the founder is the sole worker for the first 6-12 months, doing development, assembly, QA, support, marketing, and fulfillment. No salaries, no contractors, no paid marketing.*

### Revenue Model

ScoutUTM is a hardware appliance business. Every euro of revenue comes from selling a physical product. There is no software-only option.

- **Primary (85%+ of revenue):** ScoutUTM appliance sales (€499 / €699)
- **Secondary (10-15%):** Scout+ one-time upgrade (€99)
- **No software-only sales.** No licenses. No subscriptions. You buy a box, it works.

### Unit Economics (Odroid H4+ platform, EU-assembled)

| | **Scout** | **Scout Pro** |
|--|----------|--------------|
| Retail price | €499 | €699 |
| Odroid H4+ / H4 Ultra (EU distributor) | ~€200 | ~€300 |
| 8GB / 16GB DDR5 SO-DIMM | ~€25 | ~€50 |
| 128GB / 256GB NVMe SSD | ~€15 | ~€25 |
| Case (Hardkernel Type 1) | ~€15 | ~€15 |
| Assembly labor (flash, test, burn-in) | ~€15 | ~€15 |
| Packaging + quick-start guide | ~€10 | ~€10 |
| Shipping materials | ~€5 | ~€5 |
| **Total landed cost** | **~€285** | **~€420** |
| **Gross margin** | **~43%** | **~40%** |
| **Gross profit per unit** | **~€214** | **~€279** |

Scout+ attachment rate target: 35% of customers at €99 = ~€34.65 blended revenue per unit (near 100% margin)

### Year 1-3 Financial Targets (EUR) -- Hardware-First Bootstrap

| Metric | Year 1 | Year 2 | Year 3 |
|--------|--------|--------|--------|
| Units sold | 25 | 300 | 1,500 |
| Avg selling price | €530 | €550 | €560 |
| Hardware revenue | €13.3K | €165K | €840K |
| Scout+ revenue (35% attach) | €0.9K | €10.4K | €52.5K |
| **Total revenue** | **€14.2K** | **€175K** | **€893K** |
| COGS | €7.9K | €93K | €465K |
| **Gross profit** | **€6.3K** | **€82K** | **€428K** |
| Operating expenses (founder only) | €4K | €25K | €80K |
| **Net income** | **€2.3K** | **€57K** | **€348K** |

*Year 1 is build + prove: finish the product, sell the first 25 units, validate demand. Months 1-8 are pure development with zero revenue. First sales happen ~Month 9-10. Year 2 is the inflection point: reinvest profits into larger batches, reach Amazon, get reviews. Year 3 is where it becomes a real business.*

### The First 25 Units: Make-or-Break

The entire business hinges on selling the first batch of 25 units:

| | Amount |
|--|--------|
| Cost to build 25 units | ~€7,500 |
| Revenue if all 25 sell (avg €530) | ~€13,250 |
| Gross profit from first batch | ~€5,750 |
| **This funds:** | Next batch of ~20 units + CE pre-compliance |

If the first 25 sell, the business is self-sustaining. Each batch funds the next, slightly larger batch. If they don't sell, the product needs rethinking before spending more.

### Milestones That Unlock Growth

| Milestone | Units Sold | What It Enables |
|-----------|-----------|-----------------|
| First 25 units sold | 25 | Validates product-market fit, funds next batch |
| 50 cumulative units | 50 | CE pre-compliance testing, EUIPO trademark |
| 100 cumulative units | 100 | Amazon.de listing (FBA), first reviewer units |
| €50K cumulative revenue | ~95 units | Consider first part-time contractor |
| €200K cumulative revenue | ~370 units | Full CE certification, WEEE registration |
| €500K cumulative revenue | ~930 units | First full-time hire, custom PCB exploration |

### Breakeven
- Every unit sold is profitable from unit #1 (~43% gross margin)
- Initial investment (€7,500 first batch) recovered after selling ~16 units
- Company breakeven (covering all startup costs + operating expenses): ~unit 30-35
- Founder salary possible: ~Month 15-18 (when cumulative profit allows it)

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
- **Runway exhaustion** -- €35K is 6-9 months of survival. The product must be ready to sell within ~8 months or the project stalls. Mitigation: keep a day job / freelance income stream until hardware is generating revenue.
- **Unsold inventory** -- 25 unsold units = €7,500 tied up (21% of total budget). Mitigation: build-to-order model, pre-launch email list to gauge demand, don't order the full batch until you have at least 10 pre-orders or strong interest signals.
- **Single-person dependency** -- Founder gets sick, everything stops. Mitigation: automate everything possible (CI/CD, OTA updates, payment processing, order notifications).
- **CE certification cost** -- Full certification (~€10-25K) can't be paid from initial budget. Mitigation: start with pre-compliance testing (~€3K), fund full cert from first ~100 units of revenue. Selling small quantities direct-to-consumer carries lower enforcement risk than retail/Amazon, but plan for full cert before marketplace launch.
- **Returns/RMA** -- Even one return on a 25-unit batch hurts (4% of inventory). Mitigation: thorough QA before shipping (automated test script on every unit), generous troubleshooting support to avoid unnecessary returns.
- **Odroid H4+ supply disruption** -- Board supply is constrained, prices rising. Mitigation: qualify a second platform (Deciso DEC670 as fallback), monitor stock levels at EU distributors.

---

## 11. Product Roadmap

*Roadmap assumes solo founder. Timelines are conservative. Each phase is funded by the previous one.*

### Phase 0: Foundation (Months 1-4) -- Budget: ~€800

- [ ] Purchase 2x Odroid H4+ dev kits from EU distributor (~€400 each with RAM/SSD)
- [ ] Core OS build (Alpine Linux, hardened kernel, musl libc, read-only rootfs)
- [ ] Stateful firewall + NAT + DHCP/DNS (nftables)
- [ ] Basic web UI (setup wizard, dashboard, device list)
- [ ] WireGuard VPN server
- [ ] OTA update mechanism (A/B partition, signed images)
- [ ] Internal testing on dev hardware
- [ ] Automated QA test script (boot, network throughput, VPN, DNS -- runs on every unit before shipping)

### Phase 1: MVP + First Sales (Months 5-9) -- Budget: ~€9,000

- [ ] IDS/IPS integration (Suricata with curated home/SMB rulesets)
- [ ] DNS filtering (malware, phishing, ads)
- [ ] VLAN support with guided segmentation wizard
- [ ] Parental controls (DNS-based filtering, schedules)
- [ ] Web-based responsive UI (mobile-friendly -- defer native app to save time)
- [ ] ScoutUTM.com storefront (WooCommerce or Stripe checkout)
- [ ] Order first sales batch (25x Odroid H4+ kits from EU distributor)
- [ ] Assemble, flash, test, package all 25 units
- [ ] Beta program (5-10 units to early testers from Discord community)
- [ ] Public launch (Show HN, Reddit, Discord)
- [ ] **First hardware sales (€499/unit)**
- [ ] EUIPO trademark filing

### Phase 2: General Availability (Months 10-15) -- Budget: funded by first batch revenue

- [ ] CE pre-compliance testing (~€3K, funded by first ~15 unit sales)
- [ ] OpenVPN support
- [ ] GeoIP blocking
- [ ] Multi-WAN / failover
- [ ] QoS with application awareness
- [ ] Advanced parental controls (SafeSearch, YouTube restricted, per-user profiles)
- [ ] IoT device fingerprinting and auto-profiling
- [ ] Scout+ tier launch (€99 one-time: WAF, reverse proxy, advanced analytics)
- [ ] Documentation site + community forum
- [ ] Scout Pro tier launch (Odroid H4 Ultra based, €699)
- [ ] Larger batches (50 units at a time)

### Phase 3: Growth (Months 16-24) -- Budget: funded by sustained hardware revenue

- [ ] Full CE certification (when revenue supports ~€10-25K cost)
- [ ] WEEE registration (DE as first market)
- [ ] Smart home protocol awareness (Matter, Thread)
- [ ] Multi-site management
- [ ] Amazon.de listing (FBA)
- [ ] YouTube tech reviewer outreach (send 3-5 review units)
- [ ] Batches of 100+ units, better component pricing
- [ ] Consider first part-time contractor (support or development)

### Phase 4: Scale (Months 24-36) -- Budget: funded by growth revenue

- [ ] Custom PCB exploration (Eurocircuits + EU assembler) for better margins
- [ ] Mobile app (iOS + Android -- native)
- [ ] API for integrations (Home Assistant, etc.)
- [ ] Broader EU Amazon marketplaces
- [ ] EU retail channel exploration
- [ ] First full-time hire
- [ ] US/UK expansion (FCC/UKCA when budget allows)
- [ ] BSI certification pursuit
- [ ] EU Cyber Resilience Act compliance (ahead of Dec 2027 deadline)

---

## 12. Success Metrics & KPIs

### Product
- Setup completion rate: >90% within 30 minutes
- Mean time to detect threat: <60 seconds
- False positive rate: <2%

### Business (Year 1 -- bootstrap targets)
- Hardware units sold: 25 (first batch)
- Scout+ attach rate: >30%
- Customer acquisition cost (CAC): ~€0 (organic only)
- Revenue: >€13K
- Burn rate: <€3K/month (infrastructure only, no salary)
- Hardware return rate: <5% (1 return max on 25-unit batch)

### Business (Year 2+ -- growth targets)
- Hardware units sold: 300+
- Monthly revenue growth: 10-15%
- Customer lifetime value (LTV): >€530 (appliance + Scout+)
- Net Promoter Score (NPS): >60

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

- [x] **Hardware platform?** **DECIDED: Odroid H4+ (South Korea) with EU assembly.** "Designed and assembled in the EU, board manufactured in South Korea." Odroid H4+ offers dual 2.5GbE Intel NICs, Intel N97 with AES-NI, DDR5, at ~€200/board from EU distributors. Assembled, flashed, tested, and shipped from EU by founder. Fallback: Deciso DEC670 (fully EU-made, but lower margin). Custom PCB at Phase 3 (€500K+ revenue). See Section 16.
- [x] **Sales model?** **DECIDED: Hardware appliance only. No software-only sales.** A new brand needs a physical product customers can see, hold, and trust. Software-only at €149 is uncompetitive against free pfSense/OPNsense and offers no tangible brand experience. Every ScoutUTM sale is a complete, ready-to-use appliance.
- [x] **Open-source strategy?** **DECIDED: Proprietary / closed-source.** ScoutUTM is a commercial product. Source code is not published. Trust is built through independent security audits, EU-origin transparency, and published security architecture docs -- not through source access. This protects IP, enables faster iteration without community governance overhead, and preserves Scout+ value.
- [x] **Base OS?** **DECIDED: Alpine Linux.** Custom build on Alpine Linux. Minimal footprint (~5MB base), musl libc, security-oriented, fast boot, small attack surface. No forking OPNsense/pfSense -- clean break from their UI/UX baggage and licensing entanglements (pfSense's license restricts redistribution). Alpine's apk package manager enables modular feature delivery. Suricata, WireGuard, OpenVPN, nftables all available in Alpine repos.
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
8. **EU Cyber Resilience Act** -- Compliance deadlines in 2026-2027 will disrupt the market. Products must prove supply chain security, provide SBOMs, and maintain vulnerability disclosure. EU-origin products have a head start.
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
| **Cons** | Dependency on single vendor, may have MOQ requirements, their brand is tied to OPNsense |
| **Example platform** | DEC750: 8GB DDR4, 256GB SSD, up to 8.5 Gbps firewall throughput |
| **Website** | deciso.com |

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

**Phase 0 (development):** Buy 2x **Odroid H4+** kits from EU distributor (Antratek NL or Pollin DE). Total cost: ~€800. Develop ScoutOS on these boards.

**Phase 1 (first sales batch):** Order 25x Odroid H4+ boards + RAM + SSD from EU distributors. Assemble, flash ScoutOS, run QA, package, and ship from founder's workshop in the EU. Total investment: ~€7,500. Revenue if all sell: ~€13,250.

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

| Milestone | Target Date | Dependency |
|-----------|-------------|------------|
| CE pre-compliance testing (prototype) | Month 4 | Hardware prototype ready |
| CE marking achieved | Month 8 | Final hardware design frozen |
| WEEE registration (DE, NL, FR) | Month 9 | Before first public sale |
| RoHS/REACH documentation | Month 8 | BOM finalized |
| CRA vulnerability disclosure process | Month 10 | Software architecture defined |
| CRA SBOM generation pipeline | Month 12 | CI/CD pipeline mature |
| BSI certification application | Month 18 | Stable product, revenue to fund it |
| Full CRA compliance | Month 22 | Ahead of Dec 2027 deadline |

---

*Document version: 0.6 | Created: 2026-02-11 | Updated: 2026-02-11 | Status: Draft -- ready for prototype phase*

*Changelog:*
- *v0.1: Initial GTM draft.*
- *v0.2: EU-origin hardware requirement, sourcing strategy, compliance roadmap.*
- *v0.3: Decided closed-source / proprietary. Decided Alpine Linux.*
- *v0.4: Decided no crowdfunding, €35K bootstrap.*
- *v0.5: Hardware-appliance-only. Odroid H4+ platform. x86/Intel. "Made in EU" definition. Full financial rework.*
- *v0.6: Deferred name, certifications, and registrations until working prototype. All strategic decisions made. GTM is ready for execution -- next step is building the prototype.*

*Decisions made: 8 of 8 active decisions (Section 14). 4 decisions deferred until working prototype (name, CE, grants, WEEE).*
*Next step: Build the prototype. Get ScoutOS running on Odroid H4+.*
