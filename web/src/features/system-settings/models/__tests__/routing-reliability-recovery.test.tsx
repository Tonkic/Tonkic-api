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
import { render, screen } from '@testing-library/react'
import i18next from 'i18next'
import { beforeAll, describe, expect, test } from 'vitest'

import { RoutingReliabilitySection } from '../routing-reliability-section'

describe('routing reliability recovery controls', () => {
  beforeAll(() => {
    i18next.addResourceBundle('en', 'translation', {
      'Re-enable on success': 'Re-enable on success',
    })
  })

  test('keeps the recovery switch after channel tests move to monitoring', () => {
    const queryClient = new QueryClient({
      defaultOptions: { mutations: { retry: false } },
    })

    render(
      <QueryClientProvider client={queryClient}>
        <RoutingReliabilitySection
          defaultValues={{
            RetryTimes: 3,
            ChannelDisableThreshold: '0',
            AutomaticDisableChannelEnabled: true,
            AutomaticEnableChannelEnabled: true,
            AutomaticDisableKeywords: '',
            AutomaticDisableStatusCodes: '401',
            AutomaticRetryStatusCodes: '429,500-599',
          }}
        />
      </QueryClientProvider>
    )

    expect(
      screen.getByRole('switch', { name: 'Re-enable on success' })
    ).toBeChecked()
    expect(
      screen.queryByRole('switch', { name: 'Scheduled channel tests' })
    ).not.toBeInTheDocument()
    queryClient.clear()
  })
})
