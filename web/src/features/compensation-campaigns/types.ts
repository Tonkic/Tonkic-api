/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
export type CompensationCampaign = {
  id: number
  code: string
  name: string
  description: string
  quota: number
  enabled: boolean
  created_time: number
  expires_time: number
  claim_count: number
  claimed: boolean
}

export type CompensationCampaignInput = {
  code?: string
  name: string
  description: string
  quota: number
  enabled: boolean
  expires_time: number
}

export type ApiResponse<T> = {
  success: boolean
  message?: string
  data?: T
}
