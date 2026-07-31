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
import { Calendar, Download, Info } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { DateTimePicker } from '@/components/datetime-picker'
import { Dialog } from '@/components/dialog'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Combobox } from '@/components/ui/combobox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Spinner } from '@/components/ui/spinner'
import {
  TIME_GRANULARITY_OPTIONS,
  TIME_RANGE_PRESETS,
} from '@/features/dashboard/constants'
import { buildDefaultDashboardFilters } from '@/features/dashboard/lib'
import {
  alignModelExportRange,
  ALL_MODELS_EXPORT_VALUE,
  getBrowserTimezone,
  saveModelExportFile,
} from '@/features/dashboard/lib/model-export'
import type {
  DashboardChartPreferences,
  DashboardFilters,
} from '@/features/dashboard/types'
import {
  dateToUnixTimestamp,
  formatDateTimeObject,
  getRollingDateRange,
  type TimeGranularity,
} from '@/lib/time'
import { cn } from '@/lib/utils'

import {
  downloadModelAnalyticsExport,
  getModelAnalyticsExportModels,
} from '../../api'

type ModelExportForm = {
  start: Date
  end: Date
  granularity: TimeGranularity
  modelName: string
  username: string
}

type ModelsExportDialogProps = {
  preferences: DashboardChartPreferences
  currentFilters: DashboardFilters
  isAdmin: boolean
}

function buildInitialForm(
  preferences: DashboardChartPreferences,
  filters: DashboardFilters
): ModelExportForm {
  const defaults = buildDefaultDashboardFilters(preferences)
  return {
    start: filters.start_timestamp ?? defaults.start_timestamp ?? new Date(),
    end: filters.end_timestamp ?? defaults.end_timestamp ?? new Date(),
    granularity:
      filters.time_granularity ??
      defaults.time_granularity ??
      preferences.defaultTimeGranularity,
    modelName: ALL_MODELS_EXPORT_VALUE,
    username: filters.username ?? '',
  }
}

function granularityForRangeDays(days: number): TimeGranularity {
  if (days <= 1) return 'hour'
  if (days >= 29) return 'week'
  return 'day'
}

function detectQuickRangeDays(start: Date, end: Date): number | null {
  const days = Math.round((end.getTime() - start.getTime()) / 86_400_000)
  return TIME_RANGE_PRESETS.some((preset) => preset.days === days) ? days : null
}

const SectionDivider = (props: { label: string }) => (
  <div className='relative'>
    <div className='absolute inset-0 flex items-center'>
      <span className='w-full border-t' />
    </div>
    <div className='relative flex justify-center text-xs uppercase'>
      <span className='bg-background text-muted-foreground px-2'>
        {props.label}
      </span>
    </div>
  </div>
)

export function ModelsExportDialog(props: ModelsExportDialogProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [exporting, setExporting] = useState(false)
  const [form, setForm] = useState<ModelExportForm>(() =>
    buildInitialForm(props.preferences, props.currentFilters)
  )
  const [selectedRange, setSelectedRange] = useState<number | null>(() =>
    detectQuickRangeDays(form.start, form.end)
  )
  const timezone = getBrowserTimezone()
  const effectiveRange = useMemo(
    () => alignModelExportRange(form.start, form.end, form.granularity),
    [form.end, form.granularity, form.start]
  )
  const modelQueryParams = useMemo(() => {
    if (effectiveRange.error) return null
    return {
      start_timestamp: dateToUnixTimestamp(effectiveRange.start),
      end_timestamp: dateToUnixTimestamp(effectiveRange.end),
      granularity: form.granularity,
      timezone,
      username:
        props.isAdmin && form.username.trim()
          ? form.username.trim()
          : undefined,
    }
  }, [
    effectiveRange.end,
    effectiveRange.error,
    effectiveRange.start,
    form.granularity,
    form.username,
    props.isAdmin,
    timezone,
  ])
  const modelsQuery = useQuery({
    queryKey: ['dashboard', 'model-export-options', modelQueryParams],
    queryFn: async () => {
      if (!modelQueryParams) return []
      return getModelAnalyticsExportModels(modelQueryParams, props.isAdmin)
    },
    enabled: open && modelQueryParams !== null,
    staleTime: 30_000,
  })
  const modelOptions = useMemo(
    () => [
      { value: ALL_MODELS_EXPORT_VALUE, label: t('All models') },
      ...(modelsQuery.data ?? []).map((modelName) => ({
        value: modelName,
        label: modelName,
      })),
    ],
    [modelsQuery.data, t]
  )
  const selectedModelUnavailable =
    form.modelName !== ALL_MODELS_EXPORT_VALUE &&
    modelsQuery.isSuccess &&
    !modelsQuery.data.includes(form.modelName)
  const noExportableModels =
    modelsQuery.isSuccess && modelsQuery.data.length === 0
  let validationMessage: string | null = null
  if (effectiveRange.error === 'invalid_order') {
    validationMessage = t('Start time must be earlier than end time')
  } else if (effectiveRange.error === 'range_too_large') {
    validationMessage = t(
      'The selected export range is too large for this granularity'
    )
  } else if (modelsQuery.isError) {
    validationMessage = t('Failed to load exportable models')
  } else if (noExportableModels) {
    validationMessage = t('No model data is available in this time range')
  } else if (selectedModelUnavailable) {
    validationMessage = t('The selected model has no data in this time range')
  }
  const exportDisabled =
    exporting ||
    modelsQuery.isLoading ||
    modelsQuery.isError ||
    noExportableModels ||
    selectedModelUnavailable ||
    effectiveRange.error !== null

  function handleOpenChange(nextOpen: boolean) {
    if (exporting && !nextOpen) return
    if (nextOpen) {
      const initial = buildInitialForm(props.preferences, props.currentFilters)
      setForm(initial)
      setSelectedRange(detectQuickRangeDays(initial.start, initial.end))
    }
    setOpen(nextOpen)
  }

  function handleDateChange(field: 'start' | 'end', value?: Date) {
    if (!value) return
    setForm((current) => ({ ...current, [field]: value }))
    setSelectedRange(null)
  }

  function handleQuickRange(days: number) {
    const range = getRollingDateRange(days)
    setForm((current) => ({
      ...current,
      start: range.start,
      end: range.end,
      granularity: granularityForRangeDays(days),
      modelName: ALL_MODELS_EXPORT_VALUE,
    }))
    setSelectedRange(days)
  }

  async function handleExport() {
    if (exportDisabled || !modelQueryParams) return

    setExporting(true)
    const toastId = toast.loading(t('Exporting model analytics...'))
    try {
      const result = await downloadModelAnalyticsExport(
        {
          ...modelQueryParams,
          model_name:
            form.modelName === ALL_MODELS_EXPORT_VALUE
              ? undefined
              : form.modelName,
          format: 'csv',
        },
        props.isAdmin
      )
      saveModelExportFile(result.blob, result.filename)
      toast.success(t('Model analytics exported'), {
        id: toastId,
        description: result.filename,
      })
      setOpen(false)
    } catch (error) {
      let message = t('Failed to export model analytics')
      if (error instanceof Error && error.message) {
        switch (error.message) {
          case 'invalid model analytics export request':
            message = t('invalid model analytics export request')
            break
          case 'model analytics export range is too large':
            message = t('model analytics export range is too large')
            break
          case 'model analytics export exceeds the allowed row count':
            message = t('model analytics export exceeds the allowed row count')
            break
          case 'no model analytics data found':
            message = t('no model analytics data found')
            break
        }
      }
      toast.error(message, { id: toastId })
    } finally {
      setExporting(false)
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={handleOpenChange}
      trigger={
        <Button variant='outline' size='sm'>
          <Download className='mr-2 size-4' aria-hidden='true' />
          {t('Export')}
        </Button>
      }
      title={t('Export Model Analytics')}
      description={t(
        'Export request, token, quota, RPM, and TPM data by model and time bucket.'
      )}
      contentClassName='max-sm:h-dvh max-sm:w-screen max-sm:max-w-none max-sm:rounded-none max-sm:p-4 sm:max-w-lg'
      contentHeight='min(62vh, 620px)'
      footerClassName='grid grid-cols-2 gap-2 sm:flex'
      showCloseButton={!exporting}
      footer={
        <>
          <Button
            type='button'
            variant='outline'
            disabled={exporting}
            onClick={() => handleOpenChange(false)}
          >
            {t('Cancel')}
          </Button>
          <Button
            type='button'
            disabled={exportDisabled}
            onClick={handleExport}
          >
            {exporting ? (
              <Spinner className='mr-2' />
            ) : (
              <Download className='mr-2 size-4' aria-hidden='true' />
            )}
            {t('Export CSV')}
          </Button>
        </>
      }
    >
      <ScrollArea className='h-full pr-3 sm:pr-4'>
        <div className='grid gap-3 py-2'>
          <div className='grid gap-2'>
            <Label htmlFor='model-analytics-export-model'>{t('Model')}</Label>
            <Combobox
              id='model-analytics-export-model'
              options={modelOptions}
              value={form.modelName}
              onValueChange={(value) =>
                setForm((current) => ({
                  ...current,
                  modelName: value ?? ALL_MODELS_EXPORT_VALUE,
                }))
              }
              placeholder={t('Select a model')}
              searchPlaceholder={
                modelsQuery.isLoading
                  ? t('Loading models...')
                  : t('Search models...')
              }
              emptyText={t('No models found')}
              openOnFocus
            />
          </div>

          <div className='grid gap-2'>
            <Label className='flex items-center gap-2'>
              <Calendar className='size-4' aria-hidden='true' />
              {t('Quick Range')}
            </Label>
            <div className='grid grid-cols-2 gap-2 sm:flex'>
              {TIME_RANGE_PRESETS.map((range) => (
                <Button
                  key={range.days}
                  type='button'
                  size='sm'
                  variant={selectedRange === range.days ? 'default' : 'outline'}
                  onClick={() => handleQuickRange(range.days)}
                  className={cn(
                    'flex-1',
                    selectedRange === range.days &&
                      'ring-ring ring-2 ring-offset-2'
                  )}
                >
                  {t(range.label)}
                </Button>
              ))}
            </div>
          </div>

          <SectionDivider label={t('Custom Time Range')} />

          <div className='grid gap-2.5'>
            <div className='grid gap-2'>
              <Label htmlFor='model-analytics-export-start'>
                {t('Start Time')}
              </Label>
              <DateTimePicker
                id='model-analytics-export-start'
                value={form.start}
                onChange={(value) => handleDateChange('start', value)}
                placeholder={t('Select start time')}
              />
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='model-analytics-export-end'>
                {t('End Time')}
              </Label>
              <DateTimePicker
                id='model-analytics-export-end'
                value={form.end}
                onChange={(value) => handleDateChange('end', value)}
                placeholder={t('Select end time')}
              />
            </div>
          </div>

          <SectionDivider label={t('Export Settings')} />

          <div className='grid gap-2'>
            <Label htmlFor='model-analytics-export-granularity'>
              {t('Time Granularity')}
            </Label>
            <Select
              items={TIME_GRANULARITY_OPTIONS.map((option) => ({
                value: option.value,
                label: t(option.label),
              }))}
              value={form.granularity}
              onValueChange={(value) =>
                setForm((current) => ({
                  ...current,
                  granularity: value as TimeGranularity,
                  modelName: ALL_MODELS_EXPORT_VALUE,
                }))
              }
            >
              <SelectTrigger id='model-analytics-export-granularity'>
                <SelectValue placeholder={t('Select time granularity')} />
              </SelectTrigger>
              <SelectContent alignItemWithTrigger={false}>
                <SelectGroup>
                  {TIME_GRANULARITY_OPTIONS.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {t(option.label)}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>

          {props.isAdmin && (
            <div className='grid gap-2'>
              <Label htmlFor='model-analytics-export-username'>
                {t('Username')}
              </Label>
              <Input
                id='model-analytics-export-username'
                placeholder={t('Filter by username')}
                value={form.username}
                onChange={(event) =>
                  setForm((current) => ({
                    ...current,
                    username: event.target.value,
                    modelName: ALL_MODELS_EXPORT_VALUE,
                  }))
                }
              />
            </div>
          )}

          {validationMessage ? (
            <Alert variant='destructive'>
              <AlertDescription>{validationMessage}</AlertDescription>
            </Alert>
          ) : (
            <Alert>
              <Info aria-hidden='true' />
              <AlertDescription>
                {t(
                  'The export uses complete natural time buckets in your browser time zone.'
                )}
                <br />
                {t('Effective range: {{start}} to {{end}} ({{timezone}})', {
                  start: formatDateTimeObject(effectiveRange.start),
                  end: formatDateTimeObject(effectiveRange.end),
                  timezone,
                })}
                <br />
                {t(
                  'Limits: hourly 90 days, daily 2 years, weekly 5 years; up to 200,000 rows.'
                )}
              </AlertDescription>
            </Alert>
          )}
        </div>
      </ScrollArea>
    </Dialog>
  )
}
