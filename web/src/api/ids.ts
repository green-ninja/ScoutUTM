import { apiFetch } from './client'
import type { IDSConfig, IDSAlert } from './types'

export async function getIDSConfig(): Promise<IDSConfig> {
  return apiFetch<IDSConfig>('/ids')
}

export async function updateIDSConfig(data: Partial<IDSConfig>): Promise<IDSConfig> {
  return apiFetch<IDSConfig>('/ids', {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

export async function getAlerts(): Promise<IDSAlert[]> {
  return apiFetch<IDSAlert[]>('/ids/alerts')
}
