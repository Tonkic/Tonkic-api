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
import i18next from 'i18next'
import type { ReactNode } from 'react'
import { beforeAll, describe, expect, test, vi } from 'vitest'

import { updateSystemOption } from '../../api'
import { MonitoringSettingsSection } from '../monitoring-settings-section'

vi.mock('../../api', () => ({
  updateSystemOption: vi.fn().mockResolvedValue({ success: true }),
}))

const defaultValues = {
  QuotaRemindThreshold: '',
  'perf_metrics_setting.enabled': true,
  'perf_metrics_setting.flush_interval': 5,
  'perf_metrics_setting.bucket_time': 'hour' as const,
  'perf_metrics_setting.retention_days': 0,
  'monitor_setting.auto_test_channel_enabled': false,
  'monitor_setting.auto_test_channel_minutes': 15,
  'monitor_setting.channel_test_mode': 'scheduled_all' as const,
  'monitor_setting.excluded_auto_test_models': '["gpt-image-*"]',
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

describe('monitoring settings', () => {
  beforeAll(() => {
    i18next.addResourceBundle('en', 'translation', {
      'Scheduled channel tests': 'Scheduled channel tests',
      'Channel test mode': 'Channel test mode',
      'Test interval (minutes)': 'Test interval (minutes)',
      'Model performance metrics': 'Model performance metrics',
      'Channel health checks': 'Channel health checks',
      'Enable model performance metrics': 'Enable model performance metrics',
      'Excluded models from scheduled tests':
        'Excluded models from scheduled tests',
    })
  })

  test('disables probe mode and interval until scheduled tests are enabled', () => {
    const { queryClient } = renderWithQueryClient(
      <MonitoringSettingsSection defaultValues={defaultValues} />
    )

    const enableSwitch = screen.getByRole('switch', {
      name: 'Scheduled channel tests',
    })
    const modeSelect = screen.getByRole('combobox', {
      name: 'Channel test mode',
    })
    const intervalInput = screen.getByRole('spinbutton', {
      name: 'Test interval (minutes)',
    })

    expect(modeSelect).toBeDisabled()
    expect(intervalInput).toBeDisabled()

    fireEvent.click(enableSwitch)

    expect(modeSelect).toBeEnabled()
    expect(intervalInput).toBeEnabled()
    queryClient.clear()
  })

  test('submits probe settings with the shared option update path', async () => {
    vi.mocked(updateSystemOption).mockClear()
    const { container, queryClient } = renderWithQueryClient(
      <MonitoringSettingsSection defaultValues={defaultValues} />
    )
    const enableSwitch = screen.getByRole('switch', {
      name: 'Scheduled channel tests',
    })
    const intervalInput = screen.getByRole('spinbutton', {
      name: 'Test interval (minutes)',
    })

    fireEvent.click(enableSwitch)
    fireEvent.change(intervalInput, { target: { value: '20' } })
    fireEvent.input(
      screen.getByRole('textbox', {
        name: 'Excluded models from scheduled tests',
      }),
      { target: { value: '["gpt-image-*","dall-e-*"]' } }
    )
    const form = container.querySelector('form')
    expect(form).not.toBeNull()
    if (!form) return
    fireEvent.submit(form)

    await waitFor(() => {
      expect(updateSystemOption).toHaveBeenCalledWith({
        key: 'monitor_setting.auto_test_channel_enabled',
        value: true,
      })
      expect(updateSystemOption).toHaveBeenCalledWith({
        key: 'monitor_setting.auto_test_channel_minutes',
        value: 20,
      })
      expect(updateSystemOption).toHaveBeenCalledWith({
        key: 'monitor_setting.excluded_auto_test_models',
        value: '["gpt-image-*","dall-e-*"]',
      })
    })
    queryClient.clear()
  })

  test('keeps each heading with its related settings', () => {
    const { queryClient } = renderWithQueryClient(
      <MonitoringSettingsSection
        defaultValues={{
          ...defaultValues,
          'monitor_setting.auto_test_channel_enabled': true,
        }}
      />
    )

    const metricsHeading = screen.getByRole('heading', {
      name: 'Model performance metrics',
    })
    const healthHeading = screen.getByRole('heading', {
      name: 'Channel health checks',
    })
    const metricsGroup = metricsHeading.parentElement?.parentElement
    const healthGroup = healthHeading.parentElement?.parentElement

    expect(metricsGroup).toContainElement(
      screen.getByRole('switch', { name: 'Enable model performance metrics' })
    )
    expect(healthGroup).toContainElement(
      screen.getByRole('switch', { name: 'Scheduled channel tests' })
    )
    expect(healthGroup).toContainElement(
      screen.getByRole('textbox', {
        name: 'Excluded models from scheduled tests',
      })
    )
    queryClient.clear()
  })
})
