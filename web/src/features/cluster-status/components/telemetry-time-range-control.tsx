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
import { Clock01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useId, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { DateTimePicker } from '@/components/datetime-picker'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import {
  Popover,
  PopoverContent,
  PopoverDescription,
  PopoverHeader,
  PopoverTitle,
  PopoverTrigger,
} from '@/components/ui/popover'
import dayjs from '@/lib/dayjs'
import { cn } from '@/lib/utils'

import {
  TELEMETRY_TREND_PRESETS,
  validateTelemetryTrendRange,
  type TelemetryTrendTimeRange,
} from '../lib/trend-range'

type TelemetryTimeRangeControlProps = {
  value: TelemetryTrendTimeRange
  onChange: (range: TelemetryTrendTimeRange) => void
  retentionDays?: number
  availableFrom?: number
  scope?: 'page' | 'chart'
}

function cloneRange(range: TelemetryTrendTimeRange): TelemetryTrendTimeRange {
  if (range.kind === 'relative') {
    return { ...range }
  }
  return {
    kind: 'custom',
    start: new Date(range.start),
    end: new Date(range.end),
  }
}

function draftDates(range: TelemetryTrendTimeRange): {
  start: Date | undefined
  end: Date | undefined
} {
  if (range.kind === 'custom') {
    return {
      start: new Date(range.start),
      end: new Date(range.end),
    }
  }
  const end = new Date(Math.floor(Date.now() / 1000) * 1000)
  return {
    start: new Date(end.getTime() - range.minutes * 60000),
    end,
  }
}

export function TelemetryTimeRangeControl(
  props: TelemetryTimeRangeControlProps
) {
  const { t } = useTranslation()
  const startId = useId()
  const endId = useId()
  const [open, setOpen] = useState(false)
  const [draft, setDraft] = useState(() => draftDates(props.value))
  const retentionDays = props.retentionDays ?? 7

  let currentLabel = t('Custom range')
  if (props.value.kind === 'relative') {
    const minutes = props.value.minutes
    const preset = TELEMETRY_TREND_PRESETS.find(
      (item) => item.minutes === minutes
    )
    currentLabel = preset ? t(preset.labelKey) : t('Relative range')
  }

  const draftRange =
    draft.start && draft.end
      ? ({ kind: 'custom', start: draft.start, end: draft.end } as const)
      : undefined
  const rangeError = draftRange
    ? validateTelemetryTrendRange(draftRange, retentionDays)
    : 'invalid_order'
  const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone

  function applyCustomRange() {
    if (!draftRange || rangeError) return
    props.onChange(cloneRange(draftRange))
    setOpen(false)
  }

  return (
    <Popover
      open={open}
      onOpenChange={(nextOpen) => {
        if (nextOpen) setDraft(draftDates(props.value))
        setOpen(nextOpen)
      }}
    >
      <PopoverTrigger
        render={
          <Button
            variant='outline'
            aria-label={t('Select chart time window')}
          />
        }
      >
        <HugeiconsIcon
          icon={Clock01Icon}
          strokeWidth={2}
          data-icon='inline-start'
        />
        {currentLabel}
      </PopoverTrigger>
      <PopoverContent
        align='end'
        className='w-[min(calc(100vw-2rem),34rem)] gap-4 p-4'
      >
        <PopoverHeader>
          <PopoverTitle>{t('Chart time window')}</PopoverTitle>
          <PopoverDescription>
            {props.scope === 'chart'
              ? t('This time window applies only to the expanded chart.')
              : t(
                  'This time window applies to every chart on the current page.'
                )}
          </PopoverDescription>
        </PopoverHeader>

        <div className='grid grid-cols-2 gap-2 sm:flex'>
          {TELEMETRY_TREND_PRESETS.map((preset) => {
            const selected =
              props.value.kind === 'relative' &&
              props.value.minutes === preset.minutes
            return (
              <Button
                key={preset.minutes}
                type='button'
                variant={selected ? 'default' : 'outline'}
                size='sm'
                className={cn(
                  'flex-1',
                  selected && 'ring-ring ring-2 ring-offset-2'
                )}
                aria-pressed={selected}
                disabled={preset.minutes > retentionDays * 1440}
                onClick={() => {
                  props.onChange({
                    kind: 'relative',
                    minutes: preset.minutes,
                  })
                  setOpen(false)
                }}
              >
                {t(preset.labelKey)}
              </Button>
            )
          })}
        </div>

        <div className='grid gap-2.5'>
          <div className='grid gap-2'>
            <Label htmlFor={startId}>{t('Start time')}</Label>
            <DateTimePicker
              id={startId}
              value={draft.start}
              onChange={(date) =>
                setDraft((current) => ({
                  ...current,
                  start: date,
                }))
              }
              placeholder={t('Select start time')}
              precision='second'
            />
          </div>
          <div className='grid gap-2'>
            <Label htmlFor={endId}>{t('End time')}</Label>
            <DateTimePicker
              id={endId}
              value={draft.end}
              onChange={(date) =>
                setDraft((current) => ({
                  ...current,
                  end: date,
                }))
              }
              placeholder={t('Select end time')}
              precision='second'
            />
          </div>
        </div>

        {rangeError ? (
          <p className='text-destructive text-xs' role='alert'>
            {rangeError === 'exceeds_retention'
              ? t('The selected range exceeds the {{days}}-day retention.', {
                  days: retentionDays,
                })
              : t('The start time must be earlier than the end time.')}
          </p>
        ) : null}

        <div className='text-muted-foreground text-xs'>
          {t('Times use your browser timezone: {{timezone}}', { timezone })}
          {props.availableFrom
            ? ` · ${t('Available from {{time}}', {
                time: dayjs
                  .unix(props.availableFrom)
                  .format('YYYY-MM-DD HH:mm:ss'),
              })}`
            : null}
        </div>

        <div className='flex justify-end'>
          <Button
            type='button'
            size='sm'
            onClick={applyCustomRange}
            disabled={Boolean(rangeError)}
          >
            {t('Apply custom range')}
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  )
}
