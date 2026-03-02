package netmon

import "strings"

// LookupOUI returns the manufacturer name for a MAC address based on OUI prefix.
// Returns empty string if the vendor is unknown.
func LookupOUI(mac string) string {
	// Normalize MAC: remove separators and uppercase.
	mac = strings.ToUpper(mac)
	mac = strings.NewReplacer(":", "", "-", "", ".", "").Replace(mac)

	if len(mac) < 6 {
		return ""
	}

	prefix := mac[:6]
	return ouiDB[prefix]
}
