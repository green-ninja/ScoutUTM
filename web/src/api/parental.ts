import { apiFetch } from './client'
import type { ParentalConfig, ParentalProfile, ParentalDevice } from './types'

export async function getParental(): Promise<ParentalConfig> {
  return apiFetch<ParentalConfig>('/parental')
}

export async function getProfiles(): Promise<ParentalProfile[]> {
  return apiFetch<ParentalProfile[]>('/parental/profiles')
}

export async function createProfile(profile: Omit<ParentalProfile, 'id'>): Promise<ParentalProfile> {
  return apiFetch<ParentalProfile>('/parental/profiles', {
    method: 'POST',
    body: JSON.stringify(profile),
  })
}

export async function updateProfile(id: string, profile: Omit<ParentalProfile, 'id'>): Promise<ParentalProfile> {
  return apiFetch<ParentalProfile>(`/parental/profiles/${id}`, {
    method: 'PUT',
    body: JSON.stringify(profile),
  })
}

export async function deleteProfile(id: string): Promise<void> {
  return apiFetch<void>(`/parental/profiles/${id}`, { method: 'DELETE' })
}

export async function assignDevices(assignments: ParentalDevice[]): Promise<ParentalConfig> {
  return apiFetch<ParentalConfig>('/parental/devices', {
    method: 'PUT',
    body: JSON.stringify(assignments),
  })
}
