import { apiFetch } from './client'
import type { DNSConfig } from './types'

export async function getDns(): Promise<DNSConfig> {
  return apiFetch<DNSConfig>('/dns')
}

export async function updateDns(config: DNSConfig): Promise<DNSConfig> {
  return apiFetch<DNSConfig>('/dns', {
    method: 'PUT',
    body: JSON.stringify(config),
  })
}
