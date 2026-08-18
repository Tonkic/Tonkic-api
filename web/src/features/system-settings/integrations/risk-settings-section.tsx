import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { Form, FormControl, FormDescription, FormField, FormLabel } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { SettingsForm, SettingsSwitchContent, SettingsSwitchItem } from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useResetForm } from '../hooks/use-reset-form'
import { useUpdateOption } from '../hooks/use-update-option'

export type RiskSettings = {
  'risk_setting.enabled': boolean
  'risk_setting.auto_ban_enabled': boolean
  'risk_setting.scan_interval_minutes': number
  'risk_setting.lookback_days': number
  'risk_setting.auto_ban_score': number
  'risk_setting.minimum_categories': number
}

export function RiskSettingsSection(props: { defaultValues: RiskSettings }) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const form = useForm<RiskSettings>({ defaultValues: props.defaultValues })
  useResetForm(form, props.defaultValues)
  const onSubmit = async (values: RiskSettings) => {
    for (const key of Object.keys(values) as Array<keyof RiskSettings>) {
      if (values[key] !== props.defaultValues[key]) await updateOption.mutateAsync({ key, value: values[key] })
    }
  }
	const numericFields = [
	  { name: 'risk_setting.scan_interval_minutes' as const, label: 'Risk scan interval (minutes)' },
	  { name: 'risk_setting.lookback_days' as const, label: 'Risk lookback window (days)' },
	  { name: 'risk_setting.auto_ban_score' as const, label: 'Automatic ban score' },
	  { name: 'risk_setting.minimum_categories' as const, label: 'Minimum evidence categories' },
	]
  return <SettingsSection title={t('Account risk control')}>
    <Form {...form}><SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
      <SettingsPageFormActions onSave={form.handleSubmit(onSubmit)} isSaving={updateOption.isPending} />
      <FormField control={form.control} name='risk_setting.enabled' render={({ field }) => <SettingsSwitchItem><SettingsSwitchContent><FormLabel>{t('Enable account risk detection')}</FormLabel><FormDescription>{t('Analyze recent activity and create risk cases for administrator review.')}</FormDescription></SettingsSwitchContent><FormControl><Switch checked={field.value} onCheckedChange={field.onChange} /></FormControl></SettingsSwitchItem>} />
      <FormField control={form.control} name='risk_setting.auto_ban_enabled' render={({ field }) => <SettingsSwitchItem><SettingsSwitchContent><FormLabel>{t('Automatically ban extreme-risk accounts')}</FormLabel><FormDescription>{t('Only bans accounts meeting multiple evidence categories, with no successful top-up or allowlist protection. Banned users can still submit appeal tickets.')}</FormDescription></SettingsSwitchContent><FormControl><Switch checked={field.value} onCheckedChange={field.onChange} /></FormControl></SettingsSwitchItem>} />
      <div className='grid gap-4 md:grid-cols-2'>
		{numericFields.map((item) => <FormField key={item.name} control={form.control} name={item.name} render={({ field }) => <div className='space-y-2'><FormLabel>{t(item.label)}</FormLabel><FormControl><Input type='number' min={1} value={field.value} onChange={(event) => field.onChange(Number(event.target.value))} /></FormControl></div>} />)}
      </div>
    </SettingsForm></Form>
  </SettingsSection>
}
