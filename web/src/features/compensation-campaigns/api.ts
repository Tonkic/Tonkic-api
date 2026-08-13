/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { api } from '@/lib/api'

import type {
  ApiResponse,
  CompensationCampaign,
  CompensationCampaignInput,
} from './types'

export async function listCompensationCampaigns(): Promise<
  ApiResponse<{ items: CompensationCampaign[]; total: number }>
> {
  const response = await api.get('/api/compensation/admin?page_size=100')
  return response.data
}

export async function createCompensationCampaign(
  input: CompensationCampaignInput
): Promise<ApiResponse<CompensationCampaign>> {
  const response = await api.post('/api/compensation/admin', input)
  return response.data
}

export async function updateCompensationCampaign(
  id: number,
  input: CompensationCampaignInput
): Promise<ApiResponse<null>> {
  const response = await api.patch(`/api/compensation/admin/${id}`, input)
  return response.data
}

export async function getCompensationCampaign(
  code: string
): Promise<ApiResponse<CompensationCampaign>> {
  const response = await api.get(`/api/compensation/${code}`)
  return response.data
}

export async function claimCompensationCampaign(
  code: string
): Promise<ApiResponse<{ quota: number }>> {
  const response = await api.post(`/api/compensation/${code}/claim`)
  return response.data
}
