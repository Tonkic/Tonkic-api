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
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { PublicLayout } from '@/components/layout/components/public-layout'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
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

function ModelStatusCard(props: { model: PublicModelStatusItem }) {
  const { t } = useTranslation()
  const level = getSuccessRateLevel(props.model.success_rate)
  return (
    <Card size='sm'>
      <CardHeader>
        <CardTitle className='break-all'>{props.model.model_name}</CardTitle>
        <CardDescription>
          {t('{{count}} requests in the last 24 hours', {
            count: props.model.request_count,
          })}
        </CardDescription>
      </CardHeader>
      <CardContent className='space-y-3'>
        {props.model.groups.map((group) => {
          const groupRates = group.recent_success_rates?.slice(-3) ?? []
          const visibleRates =
            groupRates.length > 0 ? groupRates : [group.success_rate]
          return (
            <div
              key={group.group}
              className='bg-muted/40 grid grid-cols-[minmax(0,1fr)_auto_auto] items-center gap-3 rounded-lg px-3 py-2.5'
            >
              <div className='min-w-0'>
                <div className='truncate font-medium'>{group.group}</div>
                <div className='text-muted-foreground text-xs'>
                  {t('{{count}} requests', { count: group.request_count })}
                </div>
              </div>
              <div className='text-right'>
                <div
                  className={cn(
                    'font-mono font-semibold',
                    getSuccessRateTextClass(group.success_rate)
                  )}
                >
                  {formatUptimePct(group.success_rate)}
                </div>
                <div className='text-muted-foreground font-mono text-xs'>
                  {formatLatency(group.avg_latency_ms)}
                </div>
              </div>
              <div
                className='flex w-12 items-center gap-0.5'
                aria-label={t('Recent status')}
              >
                {visibleRates.map((rate, index) => (
                  <span
                    key={`${rate}-${visibleRates.slice(0, index).filter((item) => item === rate).length}`}
                    className={cn(
                      'h-3 flex-1 rounded-full',
                      getSuccessRateDotClass(rate)
                    )}
                    title={`${rate.toFixed(2)}%`}
                  />
                ))}
              </div>
            </div>
          )
        })}
        <span className='sr-only'>{t(statusLabelKey(level))}</span>
      </CardContent>
    </Card>
  )
}

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
        <div className='grid gap-4 md:grid-cols-2 lg:grid-cols-3'>
          {SKELETON_IDS.map((id) => (
            <Skeleton key={id} className='h-40 rounded-xl' />
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
                time: new Intl.DateTimeFormat(i18n.language, {
                  dateStyle: 'medium',
                  timeStyle: 'short',
                }).format(
                  new Date((query.data?.data.generated_at ?? 0) * 1000)
                ),
              })}
            </div>
          </CardContent>
        </Card>
        <div className='grid gap-4 md:grid-cols-2 lg:grid-cols-3'>
          {models.map((model) => (
            <ModelStatusCard key={model.model_name} model={model} />
          ))}
        </div>
      </>
    )
  }

  return (
    <PublicLayout>
      <section className='mx-auto max-w-6xl space-y-8 py-8 md:py-14'>
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
    </PublicLayout>
  )
}
