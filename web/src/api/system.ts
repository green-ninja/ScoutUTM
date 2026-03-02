import { apiFetch } from './client'
import type { SystemConfig, HealthResponse } from './types'

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
