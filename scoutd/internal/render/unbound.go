package render

import (
	"log"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/scoututm/scoutd/internal/blocklist"
	"github.com/scoututm/scoutd/internal/config"
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
	CustomBlock  []string
	CustomAllow  []string

	// Parental controls.
	TagDefinition      string // Space-separated list of profile tag names for define-tag.
	DeviceTags         []UnboundDeviceTag
	ParentalRPZZones   []UnboundParentalRPZ
	SafeSearchOverrides []UnboundSafeSearch
	ScheduleBlockedIPs []string
}

// UnboundUpstream represents a DNS forwarder.
type UnboundUpstream struct {
	IP   string
	Port int
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

// RenderUnbound renders unbound.conf from the config.
func (e *Engine) RenderUnbound(cfg *config.Config) error {
	data := buildUnboundData(cfg, nil)
	e.ensureParentalZoneFiles(cfg)
	return e.renderTemplate("unbound.conf.tmpl", e.outputPath("unbound/unbound.conf"), data)
}

// RenderUnboundWithSchedule renders unbound.conf with schedule-blocked device IPs.
func (e *Engine) RenderUnboundWithSchedule(cfg *config.Config, blockedIPs []string) error {
	data := buildUnboundData(cfg, blockedIPs)
	e.ensureParentalZoneFiles(cfg)
	return e.renderTemplate("unbound.conf.tmpl", e.outputPath("unbound/unbound.conf"), data)
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

// buildUnboundData constructs the template data from config.
func buildUnboundData(cfg *config.Config, blockedIPs []string) UnboundData {
	port := 53
	if cfg.DNS.UseDOT {
		port = 853
	}

	var upstreams []UnboundUpstream
	for _, ip := range cfg.DNS.Upstream {
		upstreams = append(upstreams, UnboundUpstream{
			IP:   ip,
			Port: port,
		})
	}

	var rpzZones []UnboundRPZZone
	if cfg.DNS.BlockMalware {
		rpzZones = append(rpzZones, UnboundRPZZone{
			Name:     "rpz-malware",
			ZoneFile: "/var/lib/scout/blocklists/rpz-malware.zone",
		})
	}
	if cfg.DNS.BlockPhishing {
		rpzZones = append(rpzZones, UnboundRPZZone{
			Name:     "rpz-phishing",
			ZoneFile: "/var/lib/scout/blocklists/rpz-phishing.zone",
		})
	}
	if cfg.DNS.BlockAds {
		rpzZones = append(rpzZones, UnboundRPZZone{
			Name:     "rpz-ads",
			ZoneFile: "/var/lib/scout/blocklists/rpz-ads.zone",
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

	data.HasRPZ = len(data.RPZZones) > 0 || len(data.ParentalRPZZones) > 0

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

