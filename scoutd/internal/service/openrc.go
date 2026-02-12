package service

import (
	"fmt"
	"os/exec"
)

// reloadDaemon executes the appropriate reload command for each daemon.
func reloadDaemon(name string) error {
	switch name {
	case Nftables:
		return runCmd("nft", "-f", "/etc/nftables.conf")
	case Dnsmasq:
		return runCmd("rc-service", "dnsmasq", "restart")
	case Chrony:
		return runCmd("rc-service", "chronyd", "restart")
	case Unbound:
		// Stub — will be implemented in Month 3.
		return runCmd("unbound-control", "reload")
	case Suricata:
		// Stub — will be implemented in Month 5.
		return runCmd("suricatasc", "-c", "reload-rules")
	case WireGuard:
		// Stub — will be implemented in Month 4.
		return runCmd("wg", "syncconf", "wg0", "/etc/wireguard/wg0.conf")
	default:
		return fmt.Errorf("unknown daemon: %s", name)
	}
}

// checkDaemon verifies a daemon is running after reload.
func checkDaemon(name string) error {
	switch name {
	case Nftables:
		// nftables has no persistent daemon; verify the ruleset loaded.
		return runCmd("nft", "list", "table", "inet", "scout")
	case Dnsmasq:
		return runCmd("rc-service", "dnsmasq", "status")
	case Chrony:
		return runCmd("rc-service", "chronyd", "status")
	case Unbound:
		return runCmd("rc-service", "unbound", "status")
	case Suricata:
		return runCmd("rc-service", "suricata", "status")
	case WireGuard:
		// WireGuard is a kernel interface; check it exists.
		return runCmd("ip", "link", "show", "wg0")
	default:
		return nil
	}
}

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s: %w", name, string(output), err)
	}
	return nil
}
