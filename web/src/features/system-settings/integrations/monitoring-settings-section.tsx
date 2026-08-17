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
import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useMemo, useRef } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import { JsonCodeEditor } from '@/components/json-code-editor'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'

import {
  SettingsControlGroup,
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useResetForm } from '../hooks/use-reset-form'
import { useUpdateOption } from '../hooks/use-update-option'
import { safeNumberFieldProps } from '../utils/numeric-field'

const numericString = z.string().refine((value) => {
  const trimmed = value.trim()
  if (!trimmed) return true
  return !Number.isNaN(Number(trimmed)) && Number(trimmed) >= 0
}, 'Enter a non-negative number or leave empty')

const modelPatternList = z.string().superRefine((value, ctx) => {
  try {
    const parsed: unknown = JSON.parse(value)
    if (
      !Array.isArray(parsed) ||
      parsed.some((item) => typeof item !== 'string')
    ) {
      ctx.addIssue({
        code: 'custom',
        message: 'Enter a JSON array of model patterns',
      })
    }
  } catch {
    ctx.addIssue({ code: 'custom', message: 'Enter valid JSON' })
  }
})

const monitoringSchema = z.object({
  QuotaRemindThreshold: numericString,
  perf_metrics_setting: z.object({
    enabled: z.boolean(),
    flush_interval: z.coerce.number().min(1),
    bucket_time: z.enum(['minute', '5min', 'hour']),
    retention_days: z.coerce.number().min(0),
  }),
  monitor_setting: z.object({
    auto_test_channel_enabled: z.boolean(),
    auto_test_channel_minutes: z.coerce
      .number()
      .int()
      .min(1, 'Interval must be at least 1 minute'),
    channel_test_mode: z.enum([
      'scheduled_all',
      'auto_ban_only',
      'passive_recovery',
    ]),
    excluded_auto_test_models: modelPatternList,
  }),
})

type MonitoringFormInput = z.input<typeof monitoringSchema>
type MonitoringFormValues = z.output<typeof monitoringSchema>

type FlatMonitoringDefaults = {
  QuotaRemindThreshold: string
  'perf_metrics_setting.enabled': boolean
  'perf_metrics_setting.flush_interval': number
  'perf_metrics_setting.bucket_time': 'minute' | '5min' | 'hour'
  'perf_metrics_setting.retention_days': number
  'monitor_setting.auto_test_channel_enabled': boolean
  'monitor_setting.auto_test_channel_minutes': number
  'monitor_setting.channel_test_mode':
    | 'scheduled_all'
    | 'auto_ban_only'
    | 'passive_recovery'
  'monitor_setting.excluded_auto_test_models': string
}

type MonitoringSettingsSectionProps = {
  defaultValues: FlatMonitoringDefaults
}

const buildFormDefaults = (
  defaults: MonitoringSettingsSectionProps['defaultValues']
): MonitoringFormInput => ({
  QuotaRemindThreshold: defaults.QuotaRemindThreshold ?? '',
  perf_metrics_setting: {
    enabled: defaults['perf_metrics_setting.enabled'],
    flush_interval: defaults['perf_metrics_setting.flush_interval'],
    bucket_time: defaults['perf_metrics_setting.bucket_time'],
    retention_days: defaults['perf_metrics_setting.retention_days'],
  },
  monitor_setting: {
    auto_test_channel_enabled:
      defaults['monitor_setting.auto_test_channel_enabled'],
    auto_test_channel_minutes:
      defaults['monitor_setting.auto_test_channel_minutes'],
    channel_test_mode: defaults['monitor_setting.channel_test_mode'],
    excluded_auto_test_models:
      defaults['monitor_setting.excluded_auto_test_models'],
  },
})

const normalizeDefaults = (
  defaults: MonitoringSettingsSectionProps['defaultValues']
): FlatMonitoringDefaults => ({
  QuotaRemindThreshold: (defaults.QuotaRemindThreshold ?? '').trim(),
  'perf_metrics_setting.enabled': defaults['perf_metrics_setting.enabled'],
  'perf_metrics_setting.flush_interval':
    defaults['perf_metrics_setting.flush_interval'],
  'perf_metrics_setting.bucket_time':
    defaults['perf_metrics_setting.bucket_time'],
  'perf_metrics_setting.retention_days':
    defaults['perf_metrics_setting.retention_days'],
  'monitor_setting.auto_test_channel_enabled':
    defaults['monitor_setting.auto_test_channel_enabled'],
  'monitor_setting.auto_test_channel_minutes':
    defaults['monitor_setting.auto_test_channel_minutes'],
  'monitor_setting.channel_test_mode':
    defaults['monitor_setting.channel_test_mode'],
  'monitor_setting.excluded_auto_test_models':
    defaults['monitor_setting.excluded_auto_test_models'],
})

const normalizeFormValues = (
  values: MonitoringFormValues
): FlatMonitoringDefaults => ({
  QuotaRemindThreshold: values.QuotaRemindThreshold.trim(),
  'perf_metrics_setting.enabled': values.perf_metrics_setting.enabled,
  'perf_metrics_setting.flush_interval':
    values.perf_metrics_setting.flush_interval,
  'perf_metrics_setting.bucket_time': values.perf_metrics_setting.bucket_time,
  'perf_metrics_setting.retention_days':
    values.perf_metrics_setting.retention_days,
  'monitor_setting.auto_test_channel_enabled':
    values.monitor_setting.auto_test_channel_enabled,
  'monitor_setting.auto_test_channel_minutes':
    values.monitor_setting.auto_test_channel_minutes,
  'monitor_setting.channel_test_mode': values.monitor_setting.channel_test_mode,
  'monitor_setting.excluded_auto_test_models':
    values.monitor_setting.excluded_auto_test_models,
})

export function MonitoringSettingsSection({
  defaultValues,
}: MonitoringSettingsSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const baselineRef = useRef<FlatMonitoringDefaults>(
    normalizeDefaults(defaultValues)
  )
  const baselineSerializedRef = useRef<string>(
    JSON.stringify(normalizeDefaults(defaultValues))
  )

  const formDefaults = useMemo(
    () => buildFormDefaults(defaultValues),
    [defaultValues]
  )

  const form = useForm<MonitoringFormInput, unknown, MonitoringFormValues>({
    resolver: zodResolver(monitoringSchema),
    defaultValues: formDefaults,
  })

  useResetForm(form, formDefaults)

  useEffect(() => {
    const normalized = normalizeDefaults(defaultValues)
    const serialized = JSON.stringify(normalized)
    if (serialized === baselineSerializedRef.current) return
    baselineRef.current = normalized
    baselineSerializedRef.current = serialized
  }, [defaultValues])

  const perfMetricsEnabled = form.watch('perf_metrics_setting.enabled')
  const channelTestsEnabled = form.watch(
    'monitor_setting.auto_test_channel_enabled'
  )
  const channelTestMode = form.watch('monitor_setting.channel_test_mode')

  const onSubmit = async (values: MonitoringFormValues) => {
    const normalized = normalizeFormValues(values)
    const updates = (
      Object.keys(normalized) as Array<keyof FlatMonitoringDefaults>
    ).filter((key) => normalized[key] !== baselineRef.current[key])

    if (updates.length === 0) {
      toast.info(t('No changes to save'))
      return
    }

    for (const key of updates) {
      await updateOption.mutateAsync({
        key,
        value: normalized[key],
      })
    }

    baselineRef.current = normalized
    baselineSerializedRef.current = JSON.stringify(normalized)
  }

  return (
    <SettingsSection title={t('Monitoring & Alerts')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
          />
          <FormField
            control={form.control}
            name='QuotaRemindThreshold'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Quota reminder (tokens)')}</FormLabel>
                <FormControl>
                  <Input
                    type='number'
                    min={0}
                    step={1}
                    value={field.value}
                    onChange={(event) => field.onChange(event.target.value)}
                  />
                </FormControl>
                <FormDescription>
                  {t('Send email alerts when a user falls below this quota')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <SettingsControlGroup>
            <div>
              <h4 className='font-medium'>{t('Model performance metrics')}</h4>
              <p className='text-muted-foreground mt-1 text-xs'>
                {t(
                  'Collect relay latency and success-rate metrics for the model square and service status page.'
                )}
              </p>
            </div>
            <div className='grid grid-cols-1 gap-4 md:grid-cols-4'>
              <FormField
                control={form.control}
                name='perf_metrics_setting.enabled'
                render={({ field }) => (
                  <SettingsSwitchItem>
                    <SettingsSwitchContent>
                      <FormLabel>
                        {t('Enable model performance metrics')}
                      </FormLabel>
                    </SettingsSwitchContent>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                  </SettingsSwitchItem>
                )}
              />
              <FormField
                control={form.control}
                name='perf_metrics_setting.flush_interval'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Flush interval (minutes)')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={1}
                        step={1}
                        {...safeNumberFieldProps(field)}
                        disabled={!perfMetricsEnabled}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='perf_metrics_setting.bucket_time'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Aggregation bucket')}</FormLabel>
                    <Select
                      items={[
                        { value: 'minute', label: t('1 minute') },
                        { value: '5min', label: t('5 minutes') },
                        { value: 'hour', label: t('1 hour') },
                      ]}
                      value={field.value}
                      onValueChange={field.onChange}
                      disabled={!perfMetricsEnabled}
                    >
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent alignItemWithTrigger={false}>
                        <SelectGroup>
                          <SelectItem value='minute'>
                            {t('1 minute')}
                          </SelectItem>
                          <SelectItem value='5min'>{t('5 minutes')}</SelectItem>
                          <SelectItem value='hour'>{t('1 hour')}</SelectItem>
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='perf_metrics_setting.retention_days'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Retention days')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={0}
                        step={1}
                        {...safeNumberFieldProps(field)}
                        disabled={!perfMetricsEnabled}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('0 means data is kept permanently')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
          </SettingsControlGroup>

          <SettingsControlGroup>
            <div>
              <h4 className='font-medium'>{t('Channel health checks')}</h4>
              <p className='text-muted-foreground mt-1 text-xs'>
                {t(
                  'Scheduled channel tests feed availability data into the same model performance metrics.'
                )}
              </p>
            </div>
            <div className='grid grid-cols-1 gap-4 md:grid-cols-3'>
              <FormField
                control={form.control}
                name='monitor_setting.auto_test_channel_enabled'
                render={({ field }) => (
                  <SettingsSwitchItem>
                    <SettingsSwitchContent>
                      <FormLabel>{t('Scheduled channel tests')}</FormLabel>
                      <FormDescription>
                        {t(
                          'Automatically probe all channels in the background'
                        )}
                      </FormDescription>
                    </SettingsSwitchContent>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                  </SettingsSwitchItem>
                )}
              />
              <FormField
                control={form.control}
                name='monitor_setting.channel_test_mode'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Channel test mode')}</FormLabel>
                    <Select
                      items={[
                        {
                          value: 'scheduled_all',
                          label: t('Actively check all channels'),
                        },
                        {
                          value: 'auto_ban_only',
                          label: t(
                            'Actively check auto-disable-enabled channels'
                          ),
                        },
                        {
                          value: 'passive_recovery',
                          label: t('Check channels awaiting recovery only'),
                        },
                      ]}
                      value={field.value}
                      onValueChange={field.onChange}
                      disabled={!channelTestsEnabled}
                    >
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent alignItemWithTrigger={false}>
                        <SelectGroup>
                          <SelectItem value='scheduled_all'>
                            {t('Actively check all channels')}
                          </SelectItem>
                          <SelectItem value='auto_ban_only'>
                            {t('Actively check auto-disable-enabled channels')}
                          </SelectItem>
                          <SelectItem value='passive_recovery'>
                            {t('Check channels awaiting recovery only')}
                          </SelectItem>
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='monitor_setting.auto_test_channel_minutes'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Test interval (minutes)')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={1}
                        step={1}
                        {...safeNumberFieldProps(field)}
                        disabled={!channelTestsEnabled}
                      />
                    </FormControl>
                    <FormDescription>
                      {channelTestMode === 'passive_recovery'
                        ? t(
                            'How frequently the system checks auto-disabled channels for recovery'
                          )
                        : t('How frequently the system tests all channels')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
            <FormField
              control={form.control}
              name='monitor_setting.excluded_auto_test_models'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('Excluded models from scheduled tests')}
                  </FormLabel>
                  <FormControl>
                    <JsonCodeEditor
                      value={field.value}
                      onChange={field.onChange}
                      onBlur={field.onBlur}
                      name={field.name}
                      disabled={!channelTestsEnabled}
                      heightClassName='h-36 min-h-36 max-h-36'
                      placeholder={'[\n  "gpt-image-*",\n  "dall-e-*"\n]'}
                      ariaLabel={t('Excluded models from scheduled tests')}
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'JSON array of exact model names or * wildcard patterns. This only affects scheduled tests.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </SettingsControlGroup>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
