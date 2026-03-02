import { apiFetch } from './client'
import type { Device } from './types'

export async function getDevices(): Promise<Device[]> {
  return apiFetch<Device[]>('/devices')
}

export async function getDevice(mac: string): Promise<Device> {
  return apiFetch<Device>(`/devices/${encodeURIComponent(mac)}`)
}

export async function updateDevice(mac: string, data: { name?: string; profile_id?: string }): Promise<Device> {
  return apiFetch<Device>(`/devices/${encodeURIComponent(mac)}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}
