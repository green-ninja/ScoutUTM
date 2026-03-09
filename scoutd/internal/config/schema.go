package config

import "encoding/json"

// CurrentSchemaVersion is the config schema version this build expects.
const CurrentSchemaVersion = 2

// Config is the top-level configuration for ScoutOS.
type Config struct {
	Version  int            `json:"version"`
	System   SystemConfig   `json:"system"`
	Network  NetworkConfig  `json:"network"`
	Firewall FirewallConfig `json:"firewall"`
	DNS      DNSConfig      `json:"dns"`
	VPN      VPNConfig      `json:"vpn"`
	IDS      IDSConfig      `json:"ids"`
	Parental ParentalConfig `json:"parental"`
	Update   UpdateConfig   `json:"update"`
}

type SystemConfig struct {
	Hostname   string   `json:"hostname"`
	Timezone   string   `json:"timezone"`
	AdminHash  string   `json:"admin_hash"`
	SetupDone  bool     `json:"setup_done"`
	Language   string   `json:"language"`
	NTPServers []string `json:"ntp_servers"`
}

type NetworkConfig struct {
	Interfaces []InterfaceConfig `json:"interfaces"`
	Zones      []ZoneConfig      `json:"zones"`
	VLANs      []VLANConfig      `json:"vlans"`
}

type InterfaceConfig struct {
	Name        string `json:"name"`
	Zone        string `json:"zone"`
	Type        string `json:"type"`
	IPv4Mode    string `json:"ipv4_mode"`
	IPv4Address string `json:"ipv4_address,omitempty"`
}

type ZoneConfig struct {
	Name          string `json:"name"`
	InputPolicy   string `json:"input_policy"`
	ForwardPolicy string `json:"forward_policy"`
	OutputPolicy  string `json:"output_policy"`
	Masquerade    bool   `json:"masquerade"`
}

type VLANConfig struct {
	ID        int       `json:"id"`
	Parent    string    `json:"parent"`
	Zone      string    `json:"zone"`
	Subnet    string    `json:"subnet"`
	DHCPRange DHCPRange `json:"dhcp_range"`
}

type DHCPRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type FirewallConfig struct {
	Forwardings  []ForwardingRule `json:"forwardings"`
	Rules        []FirewallRule   `json:"rules"`
	PortForwards []PortForward    `json:"port_forwards"`
}

type ForwardingRule struct {
	Src  string `json:"src"`
	Dest string `json:"dest"`
}

type FirewallRule struct {
	Name   string `json:"name"`
	Src    string `json:"src"`
	Dest   string `json:"dest"`
	Proto  string `json:"proto"`
	Port   int    `json:"port,omitempty"`
	Action string `json:"action"`
}

type PortForward struct {
	Name     string `json:"name"`
	Proto    string `json:"proto"`
	SrcPort  int    `json:"src_port"`
	DestIP   string `json:"dest_ip"`
	DestPort int    `json:"dest_port"`
}

type DNSConfig struct {
	Upstream     []string `json:"upstream"`
	UseDOT       bool     `json:"use_dot"`
	BlockThreats bool     `json:"block_threats"`
	QueryLogging bool     `json:"query_logging"`
	CustomBlock  []string `json:"custom_block"`
	CustomAllow  []string `json:"custom_allow"`
}

type VPNConfig struct {
	WireGuard WireGuardConfig `json:"wireguard"`
}

type WireGuardConfig struct {
	Enabled          bool            `json:"enabled"`
	ListenPort       int             `json:"listen_port"`
	PrivateKey       string          `json:"private_key,omitempty"`
	KeyCreatedAt     string          `json:"key_created_at,omitempty"`
	Address          string          `json:"address"`
	TunnelSubnet     string          `json:"tunnel_subnet"`
	EndpointOverride string          `json:"endpoint_override,omitempty"`
	Peers            []WireGuardPeer `json:"peers"`
}

type WireGuardPeer struct {
	Name       string   `json:"name"`
	PublicKey  string   `json:"public_key"`
	AllowedIPs []string `json:"allowed_ips"`
	TunnelIP   string   `json:"tunnel_ip"`
}

type IDSConfig struct {
	Enabled    bool     `json:"enabled"`
	Mode       string   `json:"mode"`
	Rulesets   []string `json:"rulesets"`
	HomeNet    []string `json:"home_net"`
	Suppressed []int    `json:"suppressed"`
}

type ParentalConfig struct {
	Profiles []ParentalProfile `json:"profiles"`
	Devices  []ParentalDevice  `json:"devices"`
}

type ParentalProfile struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	BlockCategories   []string `json:"block_categories"`
	SafeSearch        bool     `json:"safe_search"`
	YoutubeRestricted bool     `json:"youtube_restricted"`
	Schedule          Schedule `json:"schedule"`
}

type Schedule struct {
	Enabled  bool   `json:"enabled"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Days     []string `json:"days"`
}

type ParentalDevice struct {
	MAC       string `json:"mac"`
	ProfileID string `json:"profile_id"`
	IP        string `json:"ip,omitempty"`
}

type UpdateConfig struct {
	AutoCheck   bool   `json:"auto_check"`
	AutoInstall bool   `json:"auto_install"`
	Channel     string `json:"channel"`
	ServerURL   string `json:"server_url"`
}

// DeepCopy returns a deep copy of the config via JSON roundtrip.
func (c *Config) DeepCopy() (*Config, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	var cp Config
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, err
	}
	return &cp, nil
}

// DefaultConfig returns a Config with sensible defaults for a fresh install.
func DefaultConfig() *Config {
	return &Config{
		Version: CurrentSchemaVersion,
		System: SystemConfig{
			Hostname:   "scoututm",
			Timezone:   "UTC",
			AdminHash:  "",
			SetupDone:  false,
			Language:   "en",
			NTPServers: []string{"pool.ntp.org"},
		},
		Network: NetworkConfig{
			Interfaces: []InterfaceConfig{
				{Name: "eth0", Zone: "wan", Type: "ethernet", IPv4Mode: "dhcp"},
				{Name: "eth1", Zone: "lan", Type: "ethernet", IPv4Mode: "static", IPv4Address: "192.168.1.1/24"},
			},
			Zones: []ZoneConfig{
				{Name: "wan", InputPolicy: "drop", ForwardPolicy: "drop", OutputPolicy: "accept", Masquerade: true},
				{Name: "lan", InputPolicy: "accept", ForwardPolicy: "accept", OutputPolicy: "accept", Masquerade: false},
			},
			VLANs: []VLANConfig{},
		},
		Firewall: FirewallConfig{
			Forwardings: []ForwardingRule{
				{Src: "lan", Dest: "wan"},
			},
			Rules:        []FirewallRule{},
			PortForwards: []PortForward{},
		},
		DNS: DNSConfig{
			Upstream:      []string{"1.1.1.1", "9.9.9.9"},
			UseDOT:        true,
			BlockThreats:  true,
			QueryLogging:  true,
			CustomBlock:   []string{},
			CustomAllow:   []string{},
		},
		VPN: VPNConfig{
			WireGuard: WireGuardConfig{
				Enabled:      false,
				ListenPort:   51820,
				Address:      "10.0.0.1/24",
				TunnelSubnet: "10.0.0.0/24",
				Peers:        []WireGuardPeer{},
			},
		},
		IDS: IDSConfig{
			Enabled:    true,
			Mode:       "ids",
			Rulesets:   []string{"et-open"},
			HomeNet:    []string{"192.168.1.0/24"},
			Suppressed: []int{},
		},
		Parental: ParentalConfig{
			Profiles: []ParentalProfile{
				{
					ID:              "default-kids",
					Name:            "Kids",
					BlockCategories: []string{"adult", "gambling", "gaming", "social_media", "streaming"},
					SafeSearch:      true,
					YoutubeRestricted: true,
					Schedule: Schedule{
						Enabled:   false,
						StartTime: "",
						EndTime:   "",
						Days:      []string{},
					},
				},
			},
			Devices: []ParentalDevice{},
		},
		Update: UpdateConfig{
			AutoCheck:   true,
			AutoInstall: false,
			Channel:     "stable",
			ServerURL:   "https://update.scoututm.com",
		},
	}
}
