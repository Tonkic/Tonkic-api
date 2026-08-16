/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { render, screen, within } from '@testing-library/react'
import { describe, expect, test } from 'vitest'

import { ModelStatusRow } from '../index'

describe('model hourly status row', () => {
  test('renders one model with exactly 24 ordered hourly status cells', () => {
    const hourlySeries = Array.from({ length: 24 }, (_, index) => ({
      ts: 1_800_000_000 + index * 3600,
      success_rate: index === 0 ? null : 100,
    }))

    render(
      <ModelStatusRow
        model={{
          model_name: 'example-model',
          avg_latency_ms: 250,
          success_rate: 100,
          avg_tps: 10,
          hourly_series: hourlySeries,
        }}
      />
    )

    expect(screen.getByText('example-model')).toBeInTheDocument()
    const timeline = screen.getByLabelText('Recent status')
    const cells = within(timeline).getAllByTitle(/./)
    expect(cells).toHaveLength(24)
    expect(cells[0]).toHaveAttribute('data-status-hour')
    expect(cells[0]).toHaveAttribute(
      'title',
      expect.stringContaining('No data')
    )
    expect(screen.queryByText(/23 requests/)).not.toBeInTheDocument()
    expect(cells[1]).not.toHaveAttribute(
      'title',
      expect.stringContaining('requests')
    )
  })
})
