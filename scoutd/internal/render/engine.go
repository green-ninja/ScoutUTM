package render

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/scoututm/scoutd/internal/config"
)

// Engine renders daemon config files from templates.
type Engine struct {
	templateDir string
	outputDir   string
	devMode     bool
}

// NewEngine creates a template rendering engine.
func NewEngine(templateDir, outputDir string, devMode bool) *Engine {
	return &Engine{
		templateDir: templateDir,
		outputDir:   outputDir,
		devMode:     devMode,
	}
}

// DaemonRenderer is a function that renders a specific daemon's config.
type DaemonRenderer func(cfg *config.Config) error

// renderers maps daemon names to their render functions.
func (e *Engine) renderers() map[string]DaemonRenderer {
	return map[string]DaemonRenderer{
		"nftables":  e.RenderNftables,
		"dnsmasq":   e.RenderDnsmasq,
		"chrony":    e.RenderChrony,
		"unbound":   e.RenderUnbound,
		"wireguard": e.RenderWireGuard,
	}
}

// DaemonNames returns the names of all daemons that have registered renderers.
func (e *Engine) DaemonNames() []string {
	var names []string
	for name := range e.renderers() {
		names = append(names, name)
	}
	return names
}

// RenderAll renders all daemon config files from the given config.
func (e *Engine) RenderAll(cfg *config.Config) error {
	for name, render := range e.renderers() {
		if err := render(cfg); err != nil {
			return fmt.Errorf("rendering %s: %w", name, err)
		}
	}
	return nil
}

// RenderDaemons renders config files only for the specified daemons.
func (e *Engine) RenderDaemons(cfg *config.Config, daemons []string) error {
	renderers := e.renderers()
	for _, name := range daemons {
		render, ok := renderers[name]
		if !ok {
			continue // Skip daemons without templates (e.g., suricata, wireguard stubs).
		}
		if err := render(cfg); err != nil {
			return fmt.Errorf("rendering %s: %w", name, err)
		}
	}
	return nil
}

// renderTemplate parses a template file, executes it with data, and writes atomically.
func (e *Engine) renderTemplate(tmplName, outputPath string, data interface{}) error {
	tmplPath := filepath.Join(e.templateDir, tmplName)
	funcMap := templateFuncMap()

	tmpl, err := template.New(tmplName).Funcs(funcMap).ParseFiles(tmplPath)
	if err != nil {
		return fmt.Errorf("parsing template %s: %w", tmplName, err)
	}

	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".render-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()

	if err := tmpl.ExecuteTemplate(tmp, tmplName, data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("executing template %s: %w", tmplName, err)
	}

	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("syncing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("closing temp file: %w", err)
	}
	if err := os.Rename(tmpName, outputPath); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("renaming temp file: %w", err)
	}
	return nil
}

// outputPath returns the full output path for a config file.
func (e *Engine) outputPath(filename string) string {
	return filepath.Join(e.outputDir, filename)
}

// localSubnets returns the CIDR subnets from non-WAN interfaces and VLANs.
// Used by chrony (NTP allow) and unbound (access-control) to determine
// which local networks should be allowed to query local services.
func localSubnets(cfg *config.Config) []string {
	var nets []string
	for _, iface := range cfg.Network.Interfaces {
		if iface.Zone == "wan" {
			continue
		}
		if iface.IPv4Mode == "static" && iface.IPv4Address != "" {
			_, ipnet, err := net.ParseCIDR(iface.IPv4Address)
			if err == nil {
				nets = append(nets, ipnet.String())
			}
		}
	}
	for _, vlan := range cfg.Network.VLANs {
		if vlan.Subnet != "" {
			_, ipnet, err := net.ParseCIDR(vlan.Subnet)
			if err == nil {
				nets = append(nets, ipnet.String())
			}
		}
	}
	return nets
}

// templateFuncMap returns the shared template helper functions.
func templateFuncMap() template.FuncMap {
	return template.FuncMap{
		"join": strings.Join,
		"cidrToIP": func(cidr string) string {
			ip, _, err := net.ParseCIDR(cidr)
			if err != nil {
				return cidr
			}
			return ip.String()
		},
		"cidrToNet": func(cidr string) string {
			_, ipnet, err := net.ParseCIDR(cidr)
			if err != nil {
				return cidr
			}
			return ipnet.String()
		},
		"cidrToMask": func(cidr string) string {
			_, ipnet, err := net.ParseCIDR(cidr)
			if err != nil {
				return ""
			}
			mask := ipnet.Mask
			return fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])
		},
		"dhcpStart": func(cidr string) string {
			ip, _, err := net.ParseCIDR(cidr)
			if err != nil {
				return ""
			}
			ip4 := ip.To4()
			if ip4 == nil {
				return ""
			}
			return fmt.Sprintf("%d.%d.%d.100", ip4[0], ip4[1], ip4[2])
		},
		"dhcpEnd": func(cidr string) string {
			ip, _, err := net.ParseCIDR(cidr)
			if err != nil {
				return ""
			}
			ip4 := ip.To4()
			if ip4 == nil {
				return ""
			}
			return fmt.Sprintf("%d.%d.%d.253", ip4[0], ip4[1], ip4[2])
		},
	}
}
