/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { render, screen, within } from '@testing-library/react'
import { describe, expect, test } from 'vitest'

import type { PublicStatusPoint } from '@/features/performance-metrics/types'

import { groupPublicModelStatuses, ModelStatusRow } from '../index'

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
          group: 'gpt_sale',
          group_display_name: '0.1倍率分组',
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

  test('groups model rows by billing group and uses display names', () => {
    const hourlySeries: PublicStatusPoint[] = []
    const groups = groupPublicModelStatuses([
      {
        model_name: 'gpt-5.6-luna',
        group: 'gpt_sale',
        group_display_name: '0.1倍率分组',
        avg_latency_ms: 100,
        success_rate: 100,
        avg_tps: 1,
        hourly_series: hourlySeries,
      },
      {
        model_name: 'gpt-5.6-sol',
        group: '0.05羊毛',
        group_display_name: '0.05倍率分组',
        avg_latency_ms: 100,
        success_rate: 100,
        avg_tps: 1,
        hourly_series: hourlySeries,
      },
      {
        model_name: 'gpt-5.6-sol',
        group: 'gpt_sale',
        group_display_name: '0.1倍率分组',
        avg_latency_ms: 100,
        success_rate: 100,
        avg_tps: 1,
        hourly_series: hourlySeries,
      },
    ])

    expect(groups.map((group) => group.group)).toEqual([
      '0.05羊毛',
      'gpt_sale',
    ])
    expect(groups[1].displayName).toBe('0.1倍率分组')
    expect(groups[1].models.map((model) => model.model_name)).toEqual([
      'gpt-5.6-luna',
      'gpt-5.6-sol',
    ])
  })
})
