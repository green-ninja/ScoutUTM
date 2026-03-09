package blocklist

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const blocklistDir = "/var/lib/scout/blocklists"

// rpzHeader is the SOA+NS header required for a valid RPZ zone file.
const rpzHeader = `$SOA 0 0 IN SOA localhost. root.localhost. 1 86400 7200 2592000 86400
$NS 0 0 IN NS localhost.
`

// WriteParentalZones writes RPZ zone files for the given categories,
// always overwriting any existing files. This ensures that empty stubs
// from provisioning are replaced with real domain lists.
func WriteParentalZones(categories []string, dir string) error {
	if dir == "" {
		dir = blocklistDir
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating blocklist dir: %w", err)
	}

	for _, cat := range categories {
		path := filepath.Join(dir, "rpz-parental-"+cat+".zone")
		if err := WriteZoneFile(path, cat); err != nil {
			return fmt.Errorf("writing zone %s: %w", cat, err)
		}
	}
	return nil
}

// WriteZoneFile writes (or overwrites) an RPZ zone file for the given category.
func WriteZoneFile(path, category string) error {
	domains, ok := categoryDomains[category]
	if !ok {
		// Unknown category — write an empty but valid zone.
		domains = nil
	}

	var b strings.Builder
	b.WriteString(rpzHeader)
	for _, d := range domains {
		// RPZ CNAME . means "return NXDOMAIN" for this domain.
		fmt.Fprintf(&b, "%s CNAME .\n", d)
		fmt.Fprintf(&b, "*.%s CNAME .\n", d)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0644)
}

// WritePassthruZone writes an RPZ zone file with passthru entries for the
// given domains. CNAME rpz-passthru. tells Unbound to allow the response
// through without applying subsequent RPZ block rules.
func WritePassthruZone(path string, domains []string) error {
	var b strings.Builder
	b.WriteString(rpzHeader)
	for _, d := range domains {
		fmt.Fprintf(&b, "%s CNAME rpz-passthru.\n", d)
		fmt.Fprintf(&b, "*.%s CNAME rpz-passthru.\n", d)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0644)
}

// Categories returns all supported parental block category names.
func Categories() []string {
	return []string{"adult", "gambling", "gaming", "social_media", "streaming"}
}

// categoryDomains maps each parental category to its blocked domains.
// These are well-known domains for each category. The lists are intentionally
// focused on the most popular sites — comprehensive blocking requires
// downloading external blocklists (future enhancement).
var categoryDomains = map[string][]string{
	"adult": {
		"pornhub.com",
		"xvideos.com",
		"xnxx.com",
		"xhamster.com",
		"redtube.com",
		"youporn.com",
		"tube8.com",
		"spankbang.com",
		"eporner.com",
		"beeg.com",
		"porn.com",
		"brazzers.com",
		"naughtyamerica.com",
		"realitykings.com",
		"bangbros.com",
		"chaturbate.com",
		"stripchat.com",
		"bongacams.com",
		"livejasmin.com",
		"cam4.com",
		"myfreecams.com",
		"onlyfans.com",
		"fansly.com",
		"manyvids.com",
		"clips4sale.com",
		"literotica.com",
		"imagefap.com",
		"motherless.com",
		"efukt.com",
		"4chan.org",
		"hentaihaven.xxx",
		"nhentai.net",
		"rule34.xxx",
		"e621.net",
		"gelbooru.com",
		"danbooru.donmai.us",
		"tnaflix.com",
		"drtuber.com",
		"txxx.com",
		"hqporner.com",
		"daftsex.com",
		"fuq.com",
		"youjizz.com",
		"porntrex.com",
		"3movs.com",
		"fapello.com",
		"coomer.su",
		"kemono.su",
		"sex.com",
		"thumbzilla.com",
	},
	"gambling": {
		"bet365.com",
		"draftkings.com",
		"fanduel.com",
		"betmgm.com",
		"caesars.com",
		"williamhill.com",
		"paddypower.com",
		"betfair.com",
		"ladbrokes.com",
		"coral.co.uk",
		"888poker.com",
		"888casino.com",
		"pokerstars.com",
		"partypoker.com",
		"bovada.lv",
		"betway.com",
		"unibet.com",
		"bwin.com",
		"betfred.com",
		"sportingbet.com",
		"pinnacle.com",
		"stake.com",
		"roobet.com",
		"rollbit.com",
		"casinocom.com",
		"jackpotcity.com",
		"spinpalace.com",
		"royalvegas.com",
		"mrgreen.com",
		"leovegas.com",
	},
	"gaming": {
		"store.steampowered.com",
		"steampowered.com",
		"steamcommunity.com",
		"epicgames.com",
		"ea.com",
		"origin.com",
		"battle.net",
		"blizzard.com",
		"riotgames.com",
		"leagueoflegends.com",
		"valorant.com",
		"roblox.com",
		"minecraft.net",
		"fortnite.com",
		"twitch.tv",
		"xbox.com",
		"playstation.com",
		"nintendo.com",
		"gog.com",
		"itch.io",
		"humblebundle.com",
		"g2a.com",
		"kinguin.net",
		"mmoga.com",
		"curseforge.com",
		"nexusmods.com",
		"moddb.com",
		"gamebanana.com",
		"kongregate.com",
		"poki.com",
	},
	"social_media": {
		"facebook.com",
		"instagram.com",
		"twitter.com",
		"x.com",
		"tiktok.com",
		"snapchat.com",
		"reddit.com",
		"tumblr.com",
		"pinterest.com",
		"linkedin.com",
		"discord.com",
		"discordapp.com",
		"whatsapp.com",
		"telegram.org",
		"signal.org",
		"mastodon.social",
		"threads.net",
		"bsky.app",
		"bluesky.social",
		"wechat.com",
		"line.me",
		"viber.com",
		"kik.com",
		"meetme.com",
		"tagged.com",
		"ask.fm",
		"quora.com",
		"vk.com",
		"weibo.com",
		"flickr.com",
	},
	"streaming": {
		"netflix.com",
		"hulu.com",
		"disneyplus.com",
		"hbomax.com",
		"max.com",
		"peacocktv.com",
		"paramountplus.com",
		"primevideo.com",
		"crunchyroll.com",
		"funimation.com",
		"appletv.apple.com",
		"tv.apple.com",
		"vudu.com",
		"pluto.tv",
		"tubitv.com",
		"roku.com",
		"sling.com",
		"fubo.tv",
		"philo.com",
		"dazn.com",
		"espnplus.com",
		"curiositystream.com",
		"discoveryplus.com",
		"britbox.com",
		"acorn.tv",
		"shudder.com",
		"twitch.tv",
		"kick.com",
		"dailymotion.com",
		"vimeo.com",
	},
}
