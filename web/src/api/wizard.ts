import { apiFetch } from './client'
import type { WizardStepResponse } from './types'

export async function wizardStep(step: number, data: object = {}): Promise<WizardStepResponse> {
  return apiFetch<WizardStepResponse>(`/wizard/step/${step}`, {
    method: 'POST',
    body: JSON.stringify(data),
  })
}
