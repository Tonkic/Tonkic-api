import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ChevronDown, ChevronRight, RefreshCw, ShieldAlert } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import { actOnRiskCase, getRiskCases, runRiskScan } from './api'
import type { RiskCase } from './types'

type RiskView = 'active' | 'history'
type RiskAction = 'ignore' | 'ban' | 'acknowledge' | 'revert'

export function RiskAccounts() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [view, setView] = useState<RiskView>('active')
  const [expandedCases, setExpandedCases] = useState<Set<number>>(new Set())
  const [pendingAction, setPendingAction] = useState<{
    id: number
    name: RiskAction
  } | null>(null)

  const query = useQuery({
    queryKey: ['risk-cases', view],
    queryFn: async () => {
      const statuses =
        view === 'active' ? ['open', 'banned'] : ['acknowledged', 'ignored']
      const pages = await Promise.all(
        statuses.map((status) => getRiskCases(status))
      )
      return {
        items: pages
          .flatMap((page) => page.items)
          .sort((a, b) => b.score - a.score || b.last_seen - a.last_seen),
        total: pages.reduce((total, page) => total + page.total, 0),
      }
    },
  })

  const refresh = async () =>
    queryClient.invalidateQueries({ queryKey: ['risk-cases'] })
  const scan = useMutation({
    mutationFn: runRiskScan,
    onSuccess: async () => {
      toast.success(t('Risk scan queued'))
      await refresh()
    },
    onError: () => toast.error(t('Request failed')),
  })
  const action = useMutation({
    mutationFn: ({ id, name }: { id: number; name: RiskAction }) =>
      actOnRiskCase(id, name),
    onSuccess: async () => {
      setPendingAction(null)
      await refresh()
    },
    onError: () => toast.error(t('Request failed')),
  })

  const actionText =
    pendingAction?.name === 'ban'
      ? t('Ban associated accounts')
      : pendingAction?.name === 'acknowledge'
        ? t('Acknowledge ban')
        : pendingAction?.name === 'revert'
          ? t('Revert risk ban')
          : t('Mark as normal')

  const setExpanded = (id: number, open: boolean) => {
    setExpandedCases((current) => {
      const next = new Set(current)
      if (open) next.add(id)
      else next.delete(id)
      return next
    })
  }

  const renderCases = (items: RiskCase[] | undefined) => (
    <div className='space-y-4'>
      {items?.map((item) => {
        const automaticallyBanned =
          item.resolution === 'automatically banned by risk policy'
        const expanded = expandedCases.has(item.id)
        return (
          <Collapsible
            key={item.id}
            open={expanded}
            onOpenChange={(open) => setExpanded(item.id, open)}
          >
            <Card>
              <CardHeader className='p-0'>
                <CollapsibleTrigger
                  render={
                    <button
                      type='button'
                      className='hover:bg-muted/50 focus-visible:ring-ring/50 flex w-full items-center gap-3 rounded-t-xl p-6 text-left outline-none focus-visible:ring-3'
                    />
                  }
                >
                  {expanded ? (
                    <ChevronDown
                      className='size-4 shrink-0'
                      aria-hidden='true'
                    />
                  ) : (
                    <ChevronRight
                      className='size-4 shrink-0'
                      aria-hidden='true'
                    />
                  )}
                  <CardTitle className='flex min-w-0 flex-1 flex-wrap items-center gap-2'>
                    <ShieldAlert aria-hidden='true' />
                    <span>
                      #{item.id} · {t('Risk score')}: {item.score}
                    </span>
                    <Badge
                      variant={
                        item.status === 'banned' ? 'destructive' : 'secondary'
                      }
                    >
                      {t(item.status)}
                    </Badge>
                    {automaticallyBanned && (
                      <Badge variant='destructive'>
                        {t('Automatically banned')}
                      </Badge>
                    )}
                  </CardTitle>
                </CollapsibleTrigger>
              </CardHeader>
              <CollapsibleContent>
                <CardContent className='space-y-3 border-t pt-4'>
                  <p className='text-muted-foreground text-sm'>
                    {item.reason_summary}
                  </p>
                  <div className='flex flex-wrap gap-2'>
                    {item.users.map((user) => (
                      <Badge key={user.user_id} variant='outline'>
                        {user.username || `#${user.user_id}`} ({user.score})
                        {user.banned_by_case &&
                          ` · ${t('Banned by this case')}`}
                      </Badge>
                    ))}
                  </div>
                  <div className='flex flex-wrap gap-2'>
                    {item.status === 'open' && (
                      <>
                        <Button
                          variant='destructive'
                          disabled={action.isPending}
                          onClick={() =>
                            setPendingAction({ id: item.id, name: 'ban' })
                          }
                        >
                          {t('Ban associated accounts')}
                        </Button>
                        <Button
                          variant='outline'
                          disabled={action.isPending}
                          onClick={() =>
                            setPendingAction({ id: item.id, name: 'ignore' })
                          }
                        >
                          {t('Mark as normal')}
                        </Button>
                      </>
                    )}
                    {item.status === 'banned' && (
                      <>
                        <Button
                          variant='outline'
                          disabled={action.isPending}
                          onClick={() =>
                            setPendingAction({
                              id: item.id,
                              name: 'acknowledge',
                            })
                          }
                        >
                          {t('Acknowledge ban')}
                        </Button>
                        <Button
                          variant='outline'
                          disabled={action.isPending}
                          onClick={() =>
                            setPendingAction({ id: item.id, name: 'revert' })
                          }
                        >
                          {t('Revert risk ban')}
                        </Button>
                      </>
                    )}
                    {item.status === 'acknowledged' && (
                      <Button
                        variant='outline'
                        disabled={action.isPending}
                        onClick={() =>
                          setPendingAction({ id: item.id, name: 'revert' })
                        }
                      >
                        {t('Revert risk ban')}
                      </Button>
                    )}
                  </div>
                </CardContent>
              </CollapsibleContent>
            </Card>
          </Collapsible>
        )
      })}
      {!query.isLoading && items?.length === 0 && (
        <p className='text-muted-foreground py-12 text-center'>
          {t('No risk cases')}
        </p>
      )}
    </div>
  )

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Risk accounts')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button onClick={() => scan.mutate()} disabled={scan.isPending}>
          <RefreshCw aria-hidden='true' />
          {t('Run risk scan')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <Tabs
          value={view}
          onValueChange={(value) => {
            setView(value as RiskView)
            setExpandedCases(new Set())
          }}
        >
          <TabsList>
            <TabsTrigger value='active'>{t('Active risk cases')}</TabsTrigger>
            <TabsTrigger value='history'>{t('Risk history')}</TabsTrigger>
          </TabsList>
          <TabsContent value='active'>
            {view === 'active' && renderCases(query.data?.items)}
          </TabsContent>
          <TabsContent value='history'>
            {view === 'history' && renderCases(query.data?.items)}
          </TabsContent>
        </Tabs>
        <ConfirmDialog
          open={pendingAction !== null}
          onOpenChange={(open) => {
            if (!open) setPendingAction(null)
          }}
          title={actionText}
          desc={t('Confirm this risk case action.')}
          destructive={pendingAction?.name === 'ban'}
          isLoading={action.isPending}
          confirmText={actionText}
          handleConfirm={() => {
            if (pendingAction) action.mutate(pendingAction)
          }}
        />
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
