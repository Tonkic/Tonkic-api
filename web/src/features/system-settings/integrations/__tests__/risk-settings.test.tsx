/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { beforeEach, describe, expect, test, vi } from 'vitest'

import { updateSystemOption } from '../../api'
import { RiskSettingsSection } from '../risk-settings-section'

vi.mock('../../api', () => ({
  updateSystemOption: vi.fn().mockResolvedValue({ success: true }),
}))

const defaultValues = {
  'risk_setting.enabled': false,
  'risk_setting.auto_ban_enabled': false,
  'risk_setting.scan_interval_minutes': 5,
  'risk_setting.lookback_days': 7,
  'risk_setting.auto_ban_score': 100,
  'risk_setting.minimum_categories': 2,
}

function renderWithQueryClient(children: ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: { mutations: { retry: false } },
  })
  const result = render(
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  )
  return { ...result, queryClient }
}

describe('risk settings', () => {
  beforeEach(() => {
    vi.mocked(updateSystemOption).mockClear()
  })

  test('submits changed fields as dotted option keys', async () => {
    const { container, queryClient } = renderWithQueryClient(
      <RiskSettingsSection defaultValues={defaultValues} />
    )

    fireEvent.click(
      screen.getByRole('switch', { name: 'Enable account risk detection' })
    )
    const interval = screen.getAllByRole('spinbutton')[0]
    fireEvent.change(interval, { target: { value: '30' } })
    const form = container.querySelector('form')
    expect(form).not.toBeNull()
    if (!form) return
    fireEvent.submit(form)

    await waitFor(() => {
      expect(updateSystemOption).toHaveBeenCalledWith({
        key: 'risk_setting.enabled',
        value: true,
      })
      expect(updateSystemOption).toHaveBeenCalledWith({
        key: 'risk_setting.scan_interval_minutes',
        value: 30,
      })
    })
    expect(updateSystemOption).not.toHaveBeenCalledWith(
      expect.objectContaining({ key: 'risk_setting' })
    )
    queryClient.clear()
  })
})
