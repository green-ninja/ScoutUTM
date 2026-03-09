package render

import (
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/scoututm/scoutd/internal/blocklist"
	"github.com/scoututm/scoutd/internal/config"
	"github.com/scoututm/scoutd/internal/parental"
)

// UnboundData holds template data for unbound.conf.
type UnboundData struct {
	ListenAddr   string
	Port         int
	AccessNets   []string
	Upstreams    []UnboundUpstream
	UseTLS       bool
	DNSSEC       bool
	QueryLogging bool
	HasRPZ       bool // True when any RPZ zones exist (requires respip module).
	RPZZones     []UnboundRPZZone
	CustomBlock    []string
	CustomAllow    []string
	CustomAllowRPZ *UnboundRPZZone // RPZ passthru zone for custom allow (overrides blocklists).

	// Parental controls.
	TagDefinition      string // Space-separated list of profile tag names for define-tag.
	DeviceTags         []UnboundDeviceTag
	ParentalRPZZones   []UnboundParentalRPZ
	SafeSearchOverrides []UnboundSafeSearch
	ScheduleBlockedIPs []string
}

// UnboundUpstream represents a DNS forwarder.
type UnboundUpstream struct {
	IP       string
	Port     int
	AuthName string // TLS auth name for DoT certificate verification.
}

// UnboundRPZZone represents a response policy zone for blocklists.
type UnboundRPZZone struct {
	Name     string
	ZoneFile string
}

// UnboundDeviceTag maps a device IP to its profile tag.
type UnboundDeviceTag struct {
	IP  string
	Tag string
}

// UnboundParentalRPZ is a per-category RPZ zone restricted to specific tags.
type UnboundParentalRPZ struct {
	Name     string
	ZoneFile string
	Tags     string // Space-separated tag names this zone applies to.
}

// UnboundSafeSearch is a DNS override for safe search enforcement.
type UnboundSafeSearch struct {
	Domain string
	Target string
	Tags   string // Space-separated tag names.
}

// RenderUnbound renders unbound.conf from the config, including any
// currently active schedule-based DNS deny rules. This ensures that
// config applies never wipe schedule blocking.
func (e *Engine) RenderUnbound(cfg *config.Config) error {
	blocked := parental.BlockedDeviceIPs(cfg, time.Now())
	return e.RenderUnboundWithSchedule(cfg, blocked)
}

// RenderUnboundWithSchedule renders unbound.conf with schedule-blocked device IPs.
func (e *Engine) RenderUnboundWithSchedule(cfg *config.Config, blockedIPs []string) error {
	data := buildUnboundData(cfg, blockedIPs)
	e.ensureParentalZoneFiles(cfg)
	e.ensureThreatsZoneFile(cfg)
	e.writeCustomAllowRPZ(cfg)
	return e.renderTemplate("unbound.conf.tmpl", e.outputPath("unbound/unbound.conf"), data)
}

// ensureThreatsZoneFile creates an empty but valid RPZ zone file for the
// threats blocklist if it doesn't already exist. The actual domains are
// populated by blocklist.FetchThreatsBlocklist on startup. This stub
// prevents Unbound from failing to load when BlockThreats is toggled on
// before the next reboot/fetch.
func (e *Engine) ensureThreatsZoneFile(cfg *config.Config) {
	if !cfg.DNS.BlockThreats {
		return
	}

	dir := "/var/lib/scout/blocklists"
	if _, err := os.Stat("/var/lib/scout"); os.IsNotExist(err) {
		dir = e.outputPath("scout/blocklists")
	}

	path := dir + "/rpz-threats.zone"
	if _, err := os.Stat(path); err == nil {
		return // Already exists (populated by FetchThreatsBlocklist).
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("warning: creating blocklist dir for threats stub: %v", err)
		return
	}
	if err := blocklist.WriteZoneFile(path, "_empty"); err != nil {
		log.Printf("warning: writing threats zone stub: %v", err)
	}
}

// writeCustomAllowRPZ writes an RPZ passthru zone file for custom allow
// domains. This zone must be placed BEFORE threats/blocklist RPZ zones in the
// Unbound config so that allowed domains bypass blocking. local-zone
// transparent does NOT override RPZ; only RPZ passthru entries do.
func (e *Engine) writeCustomAllowRPZ(cfg *config.Config) {
	if len(cfg.DNS.CustomAllow) == 0 {
		return
	}

	// Only needed when there are blocking RPZ zones to override.
	if !cfg.DNS.BlockThreats && len(parentalCategories(cfg)) == 0 {
		return
	}

	dir := "/var/lib/scout/blocklists"
	if _, err := os.Stat("/var/lib/scout"); os.IsNotExist(err) {
		dir = e.outputPath("scout/blocklists")
	}

	path := dir + "/rpz-custom-allow.zone"
	if err := blocklist.WritePassthruZone(path, cfg.DNS.CustomAllow); err != nil {
		log.Printf("warning: writing custom allow RPZ zone: %v", err)
	}
}

// ensureParentalZoneFiles writes RPZ zone files for any parental categories
// referenced in the config. Without these files, Unbound cannot load the RPZ
// zones and blocking will silently fail. Always writes (not just ensures) so
// that empty stubs from provisioning are replaced with real domain lists.
func (e *Engine) ensureParentalZoneFiles(cfg *config.Config) {
	cats := parentalCategories(cfg)
	if len(cats) == 0 {
		return
	}
	// The Unbound template hardcodes /var/lib/scout/blocklists/ as the
	// absolute path. Zone files must go there in production and in the
	// Vagrant VM. Only use a local path when outputDir is not the real
	// system path (i.e., unit tests with a temp dir).
	dir := "/var/lib/scout/blocklists"
	if _, err := os.Stat("/var/lib/scout"); os.IsNotExist(err) {
		dir = e.outputPath("scout/blocklists")
	}
	if err := blocklist.WriteParentalZones(cats, dir); err != nil {
		log.Printf("warning: failed to write parental zone files: %v", err)
	}
}

// parentalCategories returns the deduplicated set of blocked categories
// across all parental profiles.
func parentalCategories(cfg *config.Config) []string {
	seen := make(map[string]bool)
	for _, p := range cfg.Parental.Profiles {
		for _, cat := range p.BlockCategories {
			seen[cat] = true
		}
	}
	cats := make([]string, 0, len(seen))
	for cat := range seen {
		cats = append(cats, cat)
	}
	return cats
}

// dotAuthNames maps well-known public DNS resolver IPs to their TLS
// authentication names for DoT certificate verification.
var dotAuthNames = map[string]string{
	// Cloudflare
	"1.1.1.1":             "cloudflare-dns.com",
	"1.0.0.1":             "cloudflare-dns.com",
	"2606:4700:4700::1111": "cloudflare-dns.com",
	"2606:4700:4700::1001": "cloudflare-dns.com",
	// Google
	"8.8.8.8":     "dns.google",
	"8.8.4.4":     "dns.google",
	"2001:4860:4860::8888": "dns.google",
	"2001:4860:4860::8844": "dns.google",
	// Quad9
	"9.9.9.9":     "dns.quad9.net",
	"149.112.112.112": "dns.quad9.net",
	"2620:fe::fe": "dns.quad9.net",
	"2620:fe::9":  "dns.quad9.net",
	// OpenDNS
	"208.67.222.222": "dns.opendns.com",
	"208.67.220.220": "dns.opendns.com",
	// Mullvad
	"194.242.2.2": "dns.mullvad.net",
	// AdGuard
	"94.140.14.14":  "dns.adguard-dns.com",
	"94.140.15.15":  "dns.adguard-dns.com",
}

// buildUnboundData constructs the template data from config.
func buildUnboundData(cfg *config.Config, blockedIPs []string) UnboundData {
	port := 53
	if cfg.DNS.UseDOT {
		port = 853
	}

	var upstreams []UnboundUpstream
	for _, ip := range cfg.DNS.Upstream {
		u := UnboundUpstream{
			IP:   ip,
			Port: port,
		}
		if cfg.DNS.UseDOT {
			u.AuthName = dotAuthNames[ip]
		}
		upstreams = append(upstreams, u)
	}

	var rpzZones []UnboundRPZZone
	if cfg.DNS.BlockThreats {
		rpzZones = append(rpzZones, UnboundRPZZone{
			Name:     "rpz-threats",
			ZoneFile: "/var/lib/scout/blocklists/rpz-threats.zone",
		})
	}
	data := UnboundData{
		ListenAddr:         "0.0.0.0",
		Port:               53,
		AccessNets:         localSubnets(cfg),
		Upstreams:          upstreams,
		UseTLS:             cfg.DNS.UseDOT,
		DNSSEC:             true,
		QueryLogging:       cfg.DNS.QueryLogging,
		RPZZones:           rpzZones,
		CustomBlock:        cfg.DNS.CustomBlock,
		CustomAllow:        cfg.DNS.CustomAllow,
		ScheduleBlockedIPs: blockedIPs,
	}

	buildParentalTags(cfg, &data)

	// Custom allow RPZ passthru zone — placed between parental and threats RPZ
	// zones so that allowed domains bypass global blocklists but parental
	// per-device controls still take priority.
	hasBlockingRPZ := len(data.RPZZones) > 0 || len(data.ParentalRPZZones) > 0
	if len(cfg.DNS.CustomAllow) > 0 && hasBlockingRPZ {
		data.CustomAllowRPZ = &UnboundRPZZone{
			Name:     "rpz-custom-allow",
			ZoneFile: "/var/lib/scout/blocklists/rpz-custom-allow.zone",
		}
	}

	data.HasRPZ = hasBlockingRPZ || data.CustomAllowRPZ != nil

	return data
}

// buildParentalTags populates parental control fields in UnboundData.
func buildParentalTags(cfg *config.Config, data *UnboundData) {
	if len(cfg.Parental.Devices) == 0 {
		return
	}

	// Build profile ID→tag name map and profile ID→profile map.
	profileTags := make(map[string]string)
	profiles := make(map[string]config.ParentalProfile)
	var tagNames []string
	for _, p := range cfg.Parental.Profiles {
		tag := sanitizeTag(p.Name)
		profileTags[p.ID] = tag
		profiles[p.ID] = p
		tagNames = append(tagNames, tag)
	}

	if len(tagNames) == 0 {
		return
	}

	data.TagDefinition = strings.Join(tagNames, " ")

	// Map device IPs to profile tags.
	for _, d := range cfg.Parental.Devices {
		if d.IP == "" || d.MAC == "" {
			continue
		}
		tag, ok := profileTags[d.ProfileID]
		if !ok {
			continue
		}
		data.DeviceTags = append(data.DeviceTags, UnboundDeviceTag{
			IP:  d.IP,
			Tag: tag,
		})
	}

	// Build per-category RPZ zones with tag restrictions.
	// Collect which tags need each category.
	categoryTags := make(map[string][]string)
	for _, p := range cfg.Parental.Profiles {
		tag := profileTags[p.ID]
		for _, cat := range p.BlockCategories {
			categoryTags[cat] = append(categoryTags[cat], tag)
		}
	}

	// Sort categories for deterministic output.
	cats := make([]string, 0, len(categoryTags))
	for cat := range categoryTags {
		cats = append(cats, cat)
	}
	sort.Strings(cats)

	for _, cat := range cats {
		data.ParentalRPZZones = append(data.ParentalRPZZones, UnboundParentalRPZ{
			Name:     "rpz-parental-" + cat,
			ZoneFile: "/var/lib/scout/blocklists/rpz-parental-" + cat + ".zone",
			Tags:     strings.Join(categoryTags[cat], " "),
		})
	}

	// Build safe search and YouTube restricted mode overrides.
	safeSearchDomains := []struct {
		Domain string
		Target string
	}{
		{"www.google.com", "forcesafesearch.google.com"},
		{"www.bing.com", "strict.bing.com"},
		{"duckduckgo.com", "safe.duckduckgo.com"},
	}

	youtubeDomains := []struct {
		Domain string
		Target string
	}{
		{"www.youtube.com", "restrict.youtube.com"},
		{"m.youtube.com", "restrict.youtube.com"},
		{"youtube.googleapis.com", "restrict.youtube.com"},
	}

	// Collect tags that need safe search or YouTube restricted.
	var safeSearchTags []string
	var youtubeTags []string
	for _, p := range cfg.Parental.Profiles {
		tag := profileTags[p.ID]
		if p.SafeSearch {
			safeSearchTags = append(safeSearchTags, tag)
		}
		if p.YoutubeRestricted {
			youtubeTags = append(youtubeTags, tag)
		}
	}

	if len(safeSearchTags) > 0 {
		tagStr := strings.Join(safeSearchTags, " ")
		for _, ss := range safeSearchDomains {
			data.SafeSearchOverrides = append(data.SafeSearchOverrides, UnboundSafeSearch{
				Domain: ss.Domain,
				Target: ss.Target,
				Tags:   tagStr,
			})
		}
	}

	if len(youtubeTags) > 0 {
		tagStr := strings.Join(youtubeTags, " ")
		for _, yt := range youtubeDomains {
			data.SafeSearchOverrides = append(data.SafeSearchOverrides, UnboundSafeSearch{
				Domain: yt.Domain,
				Target: yt.Target,
				Tags:   tagStr,
			})
		}
	}
}

var tagSanitizeRe = regexp.MustCompile(`[^a-z0-9_]`)

// sanitizeTag converts a profile name to a safe Unbound tag name.
func sanitizeTag(name string) string {
	tag := strings.ToLower(strings.TrimSpace(name))
	tag = strings.ReplaceAll(tag, " ", "_")
	tag = tagSanitizeRe.ReplaceAllString(tag, "")
	if tag == "" {
		tag = "profile"
	}
	return tag
}

