export interface FirewallConfig {
  forwardings: ForwardingRule[]
  rules: FirewallRule[]
  port_forwards: PortForward[]
}

export interface ForwardingRule {
  src: string
  dest: string
}

export interface FirewallRule {
  name: string
  src: string
  dest: string
  proto: string
  port?: number
  action: string
}

export interface PortForward {
  name: string
  proto: string
  src_port: number
  dest_ip: string
  dest_port: number
}

export interface DNSConfig {
  upstream: string[]
  use_dot: boolean
  use_doh: boolean
  block_malware: boolean
  block_phishing: boolean
  block_ads: boolean
  query_logging: boolean
  custom_block: string[]
  custom_allow: string[]
}

export interface WireGuardConfig {
  enabled: boolean
  listen_port: number
  public_key?: string
  key_created_at?: string
  address: string
  tunnel_subnet: string
  endpoint_override?: string
  peers: WireGuardPeer[]
}

export interface WireGuardPeer {
  name: string
  public_key: string
  allowed_ips: string[]
  tunnel_ip: string
}

export interface IDSConfig {
  enabled: boolean
  mode: string
  rulesets: string[]
  home_net: string[]
  suppressed: number[]
}

export interface SystemConfig {
  hostname: string
  timezone: string
  setup_done: boolean
  language: string
  ntp_servers: string[]
}

export interface NetworkConfig {
  interfaces: InterfaceConfig[]
  zones: ZoneConfig[]
  vlans: VLANConfig[]
}

export interface InterfaceConfig {
  name: string
  zone: string
  type: string
  ipv4_mode: string
  ipv4_address?: string
}

export interface ZoneConfig {
  name: string
  input_policy: string
  forward_policy: string
  output_policy: string
  masquerade: boolean
}

export interface VLANConfig {
  id: number
  parent: string
  zone: string
  subnet: string
  dhcp_range: { start: string; end: string }
}

export interface StatsSnapshot {
  timestamp: number
  cpu_percent: number
  mem_used_mb: number
  mem_total_mb: number
  uptime_seconds: number
  wan_rx_bps: number
  wan_tx_bps: number
  conn_count: number
  dns_queries: number
  dns_blocked: number
}

export interface HealthResponse {
  healthy: boolean
  uptime: number
  version: string
}
