import { apiFetch } from './client'
import type { SystemConfig, HealthResponse, SystemAbout, ThroughputSample, Connection } from './types'

export async function getSystem(): Promise<SystemConfig> {
  return apiFetch<SystemConfig>('/system')
}

export async function updateSystem(patch: Partial<SystemConfig>): Promise<SystemConfig> {
  return apiFetch<SystemConfig>('/system', {
    method: 'PATCH',
    body: JSON.stringify(patch),
  })
}

export async function getHealth(): Promise<HealthResponse> {
  return apiFetch<HealthResponse>('/health')
}

export async function reboot(): Promise<void> {
  return apiFetch<void>('/system/reboot', { method: 'POST' })
}

export async function downloadBackup(): Promise<Blob> {
  const token = sessionStorage.getItem('scout_token')
  const res = await fetch('/api/v1/system/backup', {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })
  if (!res.ok) throw new Error('Backup download failed')
  return res.blob()
}

export async function uploadRestore(file: File): Promise<void> {
  const formData = new FormData()
  formData.append('config', file)
  const token = sessionStorage.getItem('scout_token')
  const headers: Record<string, string> = {}
  if (token) headers['Authorization'] = `Bearer ${token}`
  const res = await fetch('/api/v1/system/restore', {
    method: 'POST',
    headers,
    body: formData,
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: 'Restore failed' }))
    throw new Error(body.error || 'Restore failed')
  }
}

export async function regenerateCert(): Promise<void> {
  return apiFetch<void>('/system/regenerate-cert', { method: 'POST' })
}

export async function getAbout(): Promise<SystemAbout> {
  return apiFetch<SystemAbout>('/system/about')
}

export async function changePassword(current: string, newPassword: string): Promise<void> {
  return apiFetch<void>('/system/password', {
    method: 'POST',
    body: JSON.stringify({ current, new_password: newPassword }),
  })
}

export async function getStatsHistory(range_: string): Promise<ThroughputSample[]> {
  return apiFetch<ThroughputSample[]>(`/stats/history?range=${range_}`)
}

export async function getConnections(): Promise<Connection[]> {
  return apiFetch<Connection[]>('/stats/connections')
}
