import { apiFetch } from './client'
import type { FirewallConfig, FirewallRule, PortForward, ForwardingRule, NetworkConfig } from './types'

export async function getFirewall(): Promise<FirewallConfig> {
  return apiFetch<FirewallConfig>('/firewall')
}

export async function updateFirewall(config: FirewallConfig): Promise<FirewallConfig> {
  return apiFetch<FirewallConfig>('/firewall', {
    method: 'PUT',
    body: JSON.stringify(config),
  })
}

export async function getZones(): Promise<NetworkConfig> {
  return apiFetch<NetworkConfig>('/network')
}

export async function addRule(rule: FirewallRule): Promise<FirewallRule> {
  return apiFetch<FirewallRule>('/firewall/rules', {
    method: 'POST',
    body: JSON.stringify(rule),
  })
}

export async function updateRule(index: number, rule: FirewallRule): Promise<FirewallRule> {
  return apiFetch<FirewallRule>(`/firewall/rules/${index}`, {
    method: 'PUT',
    body: JSON.stringify(rule),
  })
}

export async function deleteRule(index: number): Promise<void> {
  return apiFetch(`/firewall/rules/${index}`, { method: 'DELETE' })
}

export async function addPortForward(pf: PortForward): Promise<PortForward> {
  return apiFetch<PortForward>('/firewall/port-forwards', {
    method: 'POST',
    body: JSON.stringify(pf),
  })
}

export async function updatePortForward(index: number, pf: PortForward): Promise<PortForward> {
  return apiFetch<PortForward>(`/firewall/port-forwards/${index}`, {
    method: 'PUT',
    body: JSON.stringify(pf),
  })
}

export async function deletePortForward(index: number): Promise<void> {
  return apiFetch(`/firewall/port-forwards/${index}`, { method: 'DELETE' })
}
