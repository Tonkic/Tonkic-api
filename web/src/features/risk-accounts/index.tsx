import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { RefreshCw, ShieldAlert } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { actOnRiskCase, getRiskCases, runRiskScan } from './api'

export function RiskAccounts() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
	const [pendingAction, setPendingAction] = useState<{ id: number; name: 'ignore' | 'ban' | 'revert' } | null>(null)
  const query = useQuery({ queryKey: ['risk-cases'], queryFn: () => getRiskCases('') })
  const refresh = async () => queryClient.invalidateQueries({ queryKey: ['risk-cases'] })
	const scan = useMutation({ mutationFn: runRiskScan, onSuccess: async () => { toast.success(t('Risk scan queued')); await refresh() }, onError: () => toast.error(t('Request failed')) })
	const action = useMutation({ mutationFn: ({ id, name }: { id: number; name: 'ignore' | 'ban' | 'revert' }) => actOnRiskCase(id, name), onSuccess: async () => { setPendingAction(null); await refresh() }, onError: () => toast.error(t('Request failed')) })
	let actionText = t('Mark as normal')
	if (pendingAction?.name === 'ban') actionText = t('Ban associated accounts')
	if (pendingAction?.name === 'revert') actionText = t('Revert risk ban')

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Risk accounts')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button onClick={() => scan.mutate()} disabled={scan.isPending}><RefreshCw aria-hidden='true' />{t('Run risk scan')}</Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='space-y-4'>
          {query.data?.items.map((item) => (
            <Card key={item.id}>
              <CardHeader className='flex-row items-center justify-between'>
                <CardTitle className='flex items-center gap-2'><ShieldAlert aria-hidden='true' />#{item.id} · {t('Risk score')}: {item.score}</CardTitle>
                <Badge variant={item.status === 'banned' ? 'destructive' : 'secondary'}>{t(item.status)}</Badge>
              </CardHeader>
              <CardContent className='space-y-3'>
                <p className='text-muted-foreground text-sm'>{item.reason_summary}</p>
                <div className='flex flex-wrap gap-2'>{item.users.map((user) => <Badge key={user.user_id} variant='outline'>{user.username || `#${user.user_id}`} ({user.score})</Badge>)}</div>
                <div className='flex gap-2'>
				  {item.status === 'open' && <><Button variant='destructive' disabled={action.isPending} onClick={() => setPendingAction({ id: item.id, name: 'ban' })}>{t('Ban associated accounts')}</Button><Button variant='outline' disabled={action.isPending} onClick={() => setPendingAction({ id: item.id, name: 'ignore' })}>{t('Mark as normal')}</Button></>}
				  {item.status === 'banned' && <Button variant='outline' disabled={action.isPending} onClick={() => setPendingAction({ id: item.id, name: 'revert' })}>{t('Revert risk ban')}</Button>}
                </div>
              </CardContent>
            </Card>
          ))}
          {!query.isLoading && query.data?.items.length === 0 && <p className='text-muted-foreground py-12 text-center'>{t('No risk cases')}</p>}
        </div>
		<ConfirmDialog open={pendingAction !== null} onOpenChange={(open) => { if (!open) setPendingAction(null) }} title={actionText} desc={t('Confirm this risk case action.')} destructive={pendingAction?.name === 'ban'} isLoading={action.isPending} confirmText={actionText} handleConfirm={() => { if (pendingAction) action.mutate(pendingAction) }} />
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
