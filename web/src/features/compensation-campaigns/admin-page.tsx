/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { formatQuota, parseQuotaFromDollars } from '@/lib/format'

import {
  createCompensationCampaign,
  listCompensationCampaigns,
  updateCompensationCampaign,
} from './api'
import type { CompensationCampaign } from './types'

const defaultExpiry = () => {
  const date = new Date()
  date.setDate(date.getDate() + 2)
  date.setHours(23, 59, 0, 0)
  return date.toISOString().slice(0, 16)
}

export function CompensationCampaignsAdminPage() {
  const { t } = useTranslation()
  const [campaigns, setCampaigns] = useState<CompensationCampaign[]>([])
  const [name, setName] = useState('')
  const [code, setCode] = useState('')
  const [description, setDescription] = useState('')
  const [amount, setAmount] = useState(5)
  const [expiresAt, setExpiresAt] = useState(defaultExpiry)
  const [saving, setSaving] = useState(false)

  const load = async () => {
    const result = await listCompensationCampaigns()
    if (result.success) setCampaigns(result.data?.items ?? [])
  }

  useEffect(() => {
    void load()
  }, [])

  const create = async () => {
    setSaving(true)
    try {
      const result = await createCompensationCampaign({
        code,
        name,
        description,
        quota: parseQuotaFromDollars(amount),
        enabled: true,
        expires_time: Math.floor(new Date(expiresAt).getTime() / 1000),
      })
      if (!result.success) {
        toast.error(result.message ?? t('Unable to create campaign'))
        return
      }
      toast.success(t('Compensation campaign created'))
      setName('')
      setCode('')
      setDescription('')
      await load()
    } finally {
      setSaving(false)
    }
  }

  const toggle = async (campaign: CompensationCampaign) => {
    const result = await updateCompensationCampaign(campaign.id, {
      name: campaign.name,
      description: campaign.description,
      quota: campaign.quota,
      enabled: !campaign.enabled,
      expires_time: campaign.expires_time,
    })
    if (result.success) await load()
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('Compensation Campaigns')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='space-y-6'>
          <Card>
          <CardHeader>
            <CardTitle>{t('Create Compensation Campaign')}</CardTitle>
          </CardHeader>
          <CardContent className='grid gap-4 md:grid-cols-2'>
            <div className='space-y-2'>
              <Label htmlFor='campaign-name'>{t('Name')}</Label>
              <Input id='campaign-name' value={name} onChange={(event) => setName(event.target.value)} />
            </div>
            <div className='space-y-2'>
              <Label htmlFor='campaign-code'>{t('Link code')}</Label>
              <Input id='campaign-code' value={code} placeholder='august-compensation' onChange={(event) => setCode(event.target.value.toLowerCase())} />
            </div>
            <div className='space-y-2'>
              <Label htmlFor='campaign-amount'>{t('Amount (USD)')}</Label>
              <Input id='campaign-amount' type='number' min='0.01' step='0.01' value={amount} onChange={(event) => setAmount(Number(event.target.value))} />
            </div>
            <div className='space-y-2'>
              <Label htmlFor='campaign-expiry'>{t('Expiration Time')}</Label>
              <Input id='campaign-expiry' type='datetime-local' value={expiresAt} onChange={(event) => setExpiresAt(event.target.value)} />
            </div>
            <div className='space-y-2 md:col-span-2'>
              <Label htmlFor='campaign-description'>{t('Description')}</Label>
              <Input id='campaign-description' value={description} onChange={(event) => setDescription(event.target.value)} />
            </div>
            <Button className='md:w-fit' disabled={saving || !name || !code || !expiresAt || amount <= 0} onClick={() => void create()}>
              {saving ? t('Saving...') : t('Create')}
            </Button>
          </CardContent>
          </Card>

          <div className='grid gap-4'>
            {campaigns.map((campaign) => (
              <Card key={campaign.id}>
              <CardContent className='flex flex-col gap-3 pt-6 md:flex-row md:items-center md:justify-between'>
                <div>
                  <div className='font-semibold'>{campaign.name}</div>
                  <div className='text-muted-foreground text-sm'>
                    {window.location.origin}/claim/{campaign.code}
                  </div>
                  <div className='text-muted-foreground text-sm'>
                    {formatQuota(campaign.quota)} · {t('{{count}} claims', { count: campaign.claim_count })} · {new Date(campaign.expires_time * 1000).toLocaleString()}
                  </div>
                </div>
                <Button variant={campaign.enabled ? 'destructive' : 'default'} onClick={() => void toggle(campaign)}>
                  {campaign.enabled ? t('Disable') : t('Enable')}
                </Button>
              </CardContent>
              </Card>
            ))}
          </div>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
