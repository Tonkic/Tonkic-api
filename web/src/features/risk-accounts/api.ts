import { api } from '@/lib/api'
import type { RiskCasePage } from './types'

export async function getRiskCases(status: string): Promise<RiskCasePage> {
  const response = await api.get('/api/risk/cases', { params: { status, page_size: 100 } })
  return response.data.data
}

export async function runRiskScan(): Promise<void> { await api.post('/api/risk/scan') }
export async function actOnRiskCase(id: number, action: 'ignore' | 'ban' | 'revert'): Promise<void> {
  await api.post(`/api/risk/cases/${id}/${action}`, {})
}
