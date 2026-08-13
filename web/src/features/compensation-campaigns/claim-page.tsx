/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { formatQuota } from '@/lib/format'

import { claimCompensationCampaign, getCompensationCampaign } from './api'
import type { CompensationCampaign } from './types'

export function CompensationClaimPage(props: { code: string }) {
  const { t } = useTranslation()
  const [campaign, setCampaign] = useState<CompensationCampaign | null>(null)
  const [loading, setLoading] = useState(true)
  const [claiming, setClaiming] = useState(false)

  const load = useCallback(async () => {
    const result = await getCompensationCampaign(props.code)
    setCampaign(result.success ? (result.data ?? null) : null)
    setLoading(false)
  }, [props.code])

  useEffect(() => {
    void load()
  }, [load])

  const claim = async () => {
    setClaiming(true)
    try {
      const result = await claimCompensationCampaign(props.code)
      if (!result.success) {
        toast.error(result.message ?? t('Unable to claim this compensation'))
        return
      }
      toast.success(t('Compensation claimed successfully'))
      await load()
    } finally {
      setClaiming(false)
    }
  }

  if (loading) return <div className='p-8 text-center'>{t('Loading...')}</div>
  if (!campaign) return <div className='p-8 text-center'>{t('Compensation campaign not found')}</div>

  const expired = campaign.expires_time < Math.floor(Date.now() / 1000)
  let buttonLabel = t('Claim compensation')
  if (campaign.claimed) {
    buttonLabel = t('Already claimed')
  } else if (expired) {
    buttonLabel = t('Expired')
  } else if (claiming) {
    buttonLabel = t('Claiming...')
  }
  return (
    <div className='mx-auto flex min-h-[60vh] max-w-xl items-center px-4'>
      <Card className='w-full'>
        <CardHeader>
          <CardTitle>{campaign.name}</CardTitle>
          <CardDescription>{campaign.description}</CardDescription>
        </CardHeader>
        <CardContent className='space-y-5'>
          <div className='text-3xl font-semibold'>{formatQuota(campaign.quota)}</div>
          <div className='text-muted-foreground text-sm'>
            {t('Available until {{time}}', { time: new Date(campaign.expires_time * 1000).toLocaleString() })}
          </div>
          <Button className='w-full' disabled={campaign.claimed || expired || !campaign.enabled || claiming} onClick={() => void claim()}>
            {buttonLabel}
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
