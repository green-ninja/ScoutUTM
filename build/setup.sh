#!/bin/sh
# ================================================================
# ScoutUTM Setup — run this once after booting Alpine from USB
# ================================================================
#
# 1. Boot Alpine from USB on GMKtec
# 2. Mount USB:   mount /dev/sdb1 /media/usb
#    (if unsure:  blkid | grep vfat)
# 3. Run:         sh /media/usb/setup.sh
#
set -e

# Find script directory (where scoutd, templates, defaults live)
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo ""
echo "=== ScoutUTM Setup ==="
echo ""

# -- Preflight --
for f in "$SCRIPT_DIR/scoutd" "$SCRIPT_DIR/defaults/config.json"; do
  if [ ! -f "$f" ]; then
    echo "ERROR: $f not found"
    echo "Make sure scoutd, templates/, and defaults/ are next to this script."
    exit 1
  fi
done
if ! ls "$SCRIPT_DIR/templates/"*.tmpl >/dev/null 2>&1; then
  echo "ERROR: templates/*.tmpl not found next to this script"
  exit 1
fi

# -- Network --
echo "Configuring network..."
echo "  eth0 = WAN (DHCP from router)"
echo "  eth1 = LAN (172.16.0.1/24)"

cat > /etc/network/interfaces <<'EOF'
auto lo
iface lo inet loopback

auto eth0
iface eth0 inet dhcp

auto eth1
iface eth1 inet static
    address 172.16.0.1
    netmask 255.255.255.0
EOF

ip link set eth0 up
ip link set eth1 up
ip addr add 172.16.0.1/24 dev eth1 2>/dev/null || true
udhcpc -i eth0 -q 2>/dev/null || echo "WARNING: DHCP on eth0 failed"

echo 1 > /proc/sys/net/ipv4/ip_forward
mkdir -p /etc/sysctl.d
echo "net.ipv4.ip_forward = 1" > /etc/sysctl.d/99-scout.conf
echo "nameserver 8.8.8.8" > /etc/resolv.conf

echo "OK: network configured"

# -- Keyboard --
setup-keymap hr hr 2>/dev/null || loadkeys hr 2>/dev/null || true

# -- Repos --
ALPINE_VER=$(cat /etc/alpine-release | cut -d. -f1,2)
cat > /etc/apk/repositories <<EOF
https://dl-cdn.alpinelinux.org/alpine/v${ALPINE_VER}/main
https://dl-cdn.alpinelinux.org/alpine/v${ALPINE_VER}/community
EOF

# -- Packages --
echo "Installing packages..."
apk update
apk add \
  nftables suricata unbound dnsmasq wireguard-tools \
  chrony iproute2 conntrack-tools \
  openssl curl jq openssh

# -- SSH --
echo "PermitRootLogin yes" >> /etc/ssh/sshd_config
mkdir -p /root/.ssh && chmod 700 /root/.ssh
echo "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBiKluW/gHAKMSRspHlz0MQ0YalipI2qaY5Lv6i/4cyK dino@DESKTOP-QM0IRVL" >> /root/.ssh/authorized_keys
chmod 600 /root/.ssh/authorized_keys
rc-service sshd start 2>/dev/null || true

# -- ScoutUTM --
echo "Installing ScoutUTM..."
mkdir -p /etc/scout/tls
mkdir -p /var/lib/scout/logs
mkdir -p /var/lib/scout/suricata
mkdir -p /var/lib/scout/blocklists
mkdir -p /var/log/scout
mkdir -p /usr/share/scoutd/templates

install -m 755 "$SCRIPT_DIR/scoutd" /usr/sbin/scoutd
cp "$SCRIPT_DIR/templates/"*.tmpl /usr/share/scoutd/templates/

if [ ! -f /etc/scout/config.json ]; then
  cp "$SCRIPT_DIR/defaults/config.json" /etc/scout/config.json
fi

# -- OpenRC service --
cat > /etc/init.d/scoutd <<'INITEOF'
#!/sbin/openrc-run
name="scoutd"
description="ScoutUTM Management Daemon"
command="/usr/sbin/scoutd"
command_args="--config /etc/scout/config.json --template-dir /usr/share/scoutd/templates --output-dir /etc --tls-dir /etc/scout/tls --addr :443 --http-addr :80"
command_background=true
pidfile="/run/${RC_SVCNAME}.pid"
output_log="/var/log/scout/scoutd.log"
error_log="/var/log/scout/scoutd.log"
depend() {
    need net
    after firewall
}
INITEOF
chmod +x /etc/init.d/scoutd

# -- Enable services --
for svc in nftables dnsmasq unbound suricata chronyd scoutd sshd; do
  rc-update add "$svc" default 2>/dev/null || true
done

# -- Unbound --
mkdir -p /var/lib/unbound
unbound-anchor -a /var/lib/unbound/root.key 2>/dev/null || true
chown -R unbound:unbound /var/lib/unbound
touch /var/lib/scout/logs/dns-query.log
chown unbound:unbound /var/lib/scout/logs 2>/dev/null || true
chown unbound:unbound /var/lib/scout/logs/dns-query.log 2>/dev/null || true

for zone in rpz-malware rpz-phishing rpz-ads rpz-parental-adult rpz-parental-gambling rpz-parental-gaming rpz-parental-social_media rpz-parental-streaming; do
  cat > "/var/lib/scout/blocklists/${zone}.zone" <<RPZEOF
\$SOA 0 0 IN SOA localhost. root.localhost. 1 86400 7200 2592000 86400
\$NS 0 0 IN NS localhost.
RPZEOF
done

# -- Suricata --
mkdir -p /var/run/suricata /var/lib/suricata/rules /var/lib/suricata/update
echo "Downloading Suricata rules..."
suricata-update --no-test --no-reload 2>/dev/null || echo "WARNING: suricata-update failed"

# -- nftables --
mkdir -p /etc/conf.d
echo 'NFTABLES_CONFIG="/etc/nftables.conf"' > /etc/conf.d/nftables

# -- Start --
echo "Starting ScoutUTM..."
rc-service scoutd start
sleep 3

if [ -f /etc/nftables.conf ]; then
  nft -f /etc/nftables.conf 2>/dev/null || echo "WARNING: nft load failed"
  rc-service nftables start 2>/dev/null || true
else
  echo "WARNING: nftables.conf not rendered — check /var/log/scout/scoutd.log"
  tail -20 /var/log/scout/scoutd.log 2>/dev/null
fi

rc-service chronyd start 2>/dev/null || true
rc-service dnsmasq start 2>/dev/null || true
rc-service unbound start || echo "WARNING: unbound failed to start"
rc-service suricata start 2>/dev/null || echo "WARNING: suricata failed to start"
nft -f /etc/nftables.conf 2>/dev/null || true

# -- Done --
echo ""
echo "=== ScoutUTM is running ==="
echo ""
echo "  scoutd:   $(rc-service scoutd status 2>/dev/null || echo 'not running')"
echo "  unbound:  $(rc-service unbound status 2>/dev/null || echo 'not running')"
echo "  suricata: $(rc-service suricata status 2>/dev/null || echo 'not running')"
echo "  chronyd:  $(rc-service chronyd status 2>/dev/null || echo 'not running')"
echo ""
echo "From Windows, set your NIC to:"
echo "  IP:      172.16.0.100"
echo "  Mask:    255.255.255.0"
echo "  Gateway: 172.16.0.1"
echo ""
echo "Then open: https://172.16.0.1"
echo "SSH:       ssh root@172.16.0.1"
echo ""
