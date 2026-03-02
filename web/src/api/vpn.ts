import { apiFetch } from './client'
import type { WireGuardConfig, WireGuardPeer } from './types'

export async function getWireGuard(): Promise<WireGuardConfig> {
  return apiFetch<WireGuardConfig>('/vpn/wireguard')
}

export async function updateWireGuard(config: WireGuardConfig): Promise<WireGuardConfig> {
  return apiFetch<WireGuardConfig>('/vpn/wireguard', {
    method: 'PUT',
    body: JSON.stringify(config),
  })
}

export async function addPeer(peer: { name: string; split_tunnel: boolean }): Promise<{ peer: WireGuardPeer; client_config: string; qr_code: string }> {
  return apiFetch('/vpn/wireguard/peers', {
    method: 'POST',
    body: JSON.stringify(peer),
  })
}

export async function deletePeer(name: string): Promise<void> {
  return apiFetch(`/vpn/wireguard/peers/${encodeURIComponent(name)}`, {
    method: 'DELETE',
  })
}

export async function regenerateKeys(): Promise<{ public_key: string }> {
  return apiFetch('/vpn/wireguard/regenerate-keys', { method: 'POST' })
}
