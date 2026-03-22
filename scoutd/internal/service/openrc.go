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
		return runCmd("rc-service", "unbound", "restart")
	case Suricata:
		return runCmd("rc-service", "suricata", "restart")
	case WireGuard:
		// Create wg0 if it doesn't exist, then apply config.
		_ = runCmd("ip", "link", "add", "dev", "wg0", "type", "wireguard")
		if err := runCmd("wg", "setconf", "wg0", "/etc/wireguard.conf"); err != nil {
			return err
		}
		// Re-apply address and bring up (idempotent).
		_ = runCmd("ip", "link", "set", "wg0", "up")
		return nil
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
