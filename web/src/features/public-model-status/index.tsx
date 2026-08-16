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
import { useQuery } from '@tanstack/react-query'
import { Activity, AlertTriangle, CircleCheck, RefreshCw } from 'lucide-react'
import { memo, useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { getPublicModelStatus } from '@/features/performance-metrics/api'
import {
  formatLatency,
  formatUptimePct,
  getSuccessRateDotClass,
  getSuccessRateLevel,
  getSuccessRateTextClass,
} from '@/features/performance-metrics/lib/format'
import type { PublicModelStatus as PublicModelStatusItem } from '@/features/performance-metrics/types'
import { toIntlLocale } from '@/i18n/languages'
import { cn } from '@/lib/utils'

const SKELETON_IDS = ['one', 'two', 'three', 'four', 'five', 'six']

function statusLabelKey(level: ReturnType<typeof getSuccessRateLevel>): string {
  if (level === 'critical') return 'Outage'
  if (level === 'warning') return 'Degraded performance recently'
  return 'All systems operational'
}

function statusAccentClass(
  level: ReturnType<typeof getSuccessRateLevel>,
  kind: 'ring' | 'text'
): string {
  if (level === 'critical') {
    return kind === 'ring' ? 'ring-red-500/25' : 'text-red-500'
  }
  if (level === 'warning') {
    return kind === 'ring' ? 'ring-amber-500/25' : 'text-amber-500'
  }
  return kind === 'ring' ? 'ring-emerald-500/25' : 'text-emerald-500'
}

export const ModelStatusRow = memo(function ModelStatusRow(props: {
  model: PublicModelStatusItem
}) {
  const { t, i18n } = useTranslation()
  const level = getSuccessRateLevel(props.model.success_rate)
  return (
    <div className='grid min-w-[920px] grid-cols-[minmax(12rem,1fr)_6rem_7rem_minmax(30rem,2fr)] items-center gap-4 border-b px-4 py-3 last:border-b-0'>
      <div className='min-w-0'>
        <div className='truncate font-medium'>{props.model.model_name}</div>
        <div className='text-muted-foreground text-xs'>
          {t('{{count}} requests in the last 24 hours', {
            count: props.model.request_count,
          })}
        </div>
      </div>
      <div
        className={cn(
          'text-right font-mono font-semibold',
          getSuccessRateTextClass(props.model.success_rate)
        )}
      >
        {formatUptimePct(props.model.success_rate)}
      </div>
      <div className='text-muted-foreground text-right font-mono text-sm'>
        {formatLatency(props.model.avg_latency_ms)}
      </div>
      <div
        className='grid grid-cols-[repeat(24,minmax(0,1fr))] gap-1'
        aria-label={t('Recent status')}
      >
        {props.model.hourly_series.map((point) => {
          const hour = new Intl.DateTimeFormat(toIntlLocale(i18n.language), {
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
          }).format(new Date(point.ts * 1000))
          const title =
            point.success_rate === null
              ? `${hour} · ${t('No data')}`
              : `${hour} · ${point.success_rate.toFixed(2)}% · ${t('{{count}} requests', { count: point.request_count })}`
          return (
            <span
              key={point.ts}
              data-status-hour
              className={cn(
                'h-7 rounded-sm',
                point.success_rate === null
                  ? 'bg-muted'
                  : getSuccessRateDotClass(point.success_rate)
              )}
              title={title}
            />
          )
        })}
      </div>
      <span className='sr-only'>{t(statusLabelKey(level))}</span>
    </div>
  )
})

export function PublicModelStatus() {
  const { t, i18n } = useTranslation()
  const query = useQuery({
    queryKey: ['public-model-status'],
    queryFn: getPublicModelStatus,
    refetchInterval: 60_000,
    staleTime: 30_000,
    retry: false,
  })
  const overall = useMemo(() => {
    const models = query.data?.data.models ?? []
    const requests = models.reduce(
      (total, model) => total + model.request_count,
      0
    )
    if (requests === 0) return Number.NaN
    return (
      models.reduce(
        (total, model) => total + model.success_rate * model.request_count,
        0
      ) / requests
    )
  }, [query.data?.data.models])
  const models = query.data?.data.models ?? []
  const level = getSuccessRateLevel(overall)
  const isHealthy = level === 'excellent' || level === 'good'
  const headline = t(statusLabelKey(level))
  const StatusIcon = isHealthy ? CircleCheck : AlertTriangle
  let content: React.ReactNode

  if (query.isLoading) {
    content = (
      <div className='space-y-4'>
        <Skeleton className='h-28 w-full rounded-xl' />
        <div className='space-y-2'>
          {SKELETON_IDS.map((id) => (
            <Skeleton key={id} className='h-16 rounded-xl' />
          ))}
        </div>
      </div>
    )
  } else if (query.isError) {
    content = (
      <Card>
        <CardContent className='py-10 text-center'>
          <AlertTriangle className='text-destructive mx-auto mb-3 size-8' />
          <div className='font-medium'>{t('Unable to load model status')}</div>
          <div className='text-muted-foreground mt-1 text-sm'>
            {t('Please try again later.')}
          </div>
        </CardContent>
      </Card>
    )
  } else if (models.length === 0) {
    content = (
      <Card>
        <CardContent className='py-10 text-center'>
          <Activity className='text-muted-foreground mx-auto mb-3 size-8' />
          <div className='font-medium'>
            {t('No performance data available')}
          </div>
          <div className='text-muted-foreground mt-1 text-sm'>
            {t('Status will appear after API requests are processed.')}
          </div>
        </CardContent>
      </Card>
    )
  } else {
    content = (
      <>
        <Card className={cn(statusAccentClass(level, 'ring'))}>
          <CardContent className='flex flex-col gap-4 py-3 sm:flex-row sm:items-center sm:justify-between'>
            <div className='flex items-center gap-3'>
              <StatusIcon
                className={cn('size-8', statusAccentClass(level, 'text'))}
              />
              <div>
                <div className='text-lg font-semibold'>{headline}</div>
                <div className='text-muted-foreground text-sm'>
                  {t('Overall success rate')}: {formatUptimePct(overall)}
                </div>
              </div>
            </div>
            <div className='text-muted-foreground text-sm'>
              {t('Updated {{time}}', {
                time: new Intl.DateTimeFormat(toIntlLocale(i18n.language), {
                  dateStyle: 'medium',
                  timeStyle: 'short',
                }).format(
                  new Date((query.data?.data.generated_at ?? 0) * 1000)
                ),
              })}
            </div>
          </CardContent>
        </Card>
        <Card className='overflow-x-auto py-0'>
          <div className='min-w-[920px]'>
            <div className='text-muted-foreground grid grid-cols-[minmax(12rem,1fr)_6rem_7rem_minmax(30rem,2fr)] gap-4 border-b px-4 py-2 text-xs font-medium'>
              <div>{t('Model')}</div>
              <div className='text-right'>{t('Success rate')}</div>
              <div className='text-right'>{t('Latency')}</div>
              <div>{t('Recent status')}</div>
            </div>
            {models.map((model) => (
              <ModelStatusRow key={model.model_name} model={model} />
            ))}
          </div>
        </Card>
      </>
    )
  }

  return (
    <section className='mx-auto w-full max-w-7xl space-y-8 overflow-y-auto px-4 py-8 md:px-8 md:py-14'>
      <div className='flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between'>
        <div>
          <div className='text-primary mb-2 flex items-center gap-2 text-sm font-medium'>
            <Activity className='size-4' aria-hidden='true' />
            {t('Service Status')}
          </div>
          <h1 className='text-3xl font-semibold tracking-tight md:text-4xl'>
            {t('Model availability')}
          </h1>
          <p className='text-muted-foreground mt-2 max-w-2xl'>
            {t(
              'Live availability based on real API requests during the last 24 hours.'
            )}
          </p>
        </div>
        <Button
          variant='outline'
          onClick={() => void query.refetch()}
          disabled={query.isFetching}
        >
          <RefreshCw
            className={cn(query.isFetching && 'animate-spin')}
            data-icon='inline-start'
          />
          {t('Refresh')}
        </Button>
      </div>

      {content}
    </section>
  )
}
