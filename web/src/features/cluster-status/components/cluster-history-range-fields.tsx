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
import { useId, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import dayjs from '@/lib/dayjs'

import { buildClusterHistoryPresetRange } from '../lib/export'

type ClusterHistoryRangeFieldsProps = {
  start: Date
  end: Date
  onChange: (range: { start: Date; end: Date }) => void
  availableFrom?: number
  disabled?: boolean
}

function toInputValue(date: Date): string {
  return dayjs(date).format('YYYY-MM-DDTHH:mm:ss')
}

function fromInputValue(value: string): Date | undefined {
  if (!value) return undefined
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? undefined : date
}

export function ClusterHistoryRangeFields(
  props: ClusterHistoryRangeFieldsProps
) {
  const { t } = useTranslation()
  const startId = useId()
  const endId = useId()
  const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone
  const [selectedPreset, setSelectedPreset] = useState<number | null>(null)

  function applyQuickRange(minutes: number) {
    props.onChange(buildClusterHistoryPresetRange(minutes, props.availableFrom))
    setSelectedPreset(minutes)
  }

  return (
    <FieldGroup className='gap-3'>
      <div className='grid gap-3 sm:grid-cols-2'>
        <Field>
          <FieldLabel htmlFor={startId}>{t('Start time')}</FieldLabel>
          <Input
            id={startId}
            type='datetime-local'
            step={1}
            value={toInputValue(props.start)}
            onChange={(event) => {
              const start = fromInputValue(event.target.value)
              if (start) {
                setSelectedPreset(null)
                props.onChange({ start, end: props.end })
              }
            }}
            disabled={props.disabled}
          />
        </Field>
        <Field>
          <FieldLabel htmlFor={endId}>{t('End time')}</FieldLabel>
          <Input
            id={endId}
            type='datetime-local'
            step={1}
            value={toInputValue(props.end)}
            onChange={(event) => {
              const end = fromInputValue(event.target.value)
              if (end) {
                setSelectedPreset(null)
                props.onChange({ start: props.start, end })
              }
            }}
            disabled={props.disabled}
          />
        </Field>
      </div>

      <div className='flex flex-wrap gap-2'>
        {[
          { label: 'Last 15 minutes', minutes: 15 },
          { label: 'Last hour', minutes: 60 },
          { label: 'Last 6 hours', minutes: 360 },
          { label: 'Last 24 hours', minutes: 1440 },
          { label: 'Last 7 days', minutes: 10080 },
        ].map((preset) => (
          <Button
            key={preset.minutes}
            type='button'
            variant={
              selectedPreset === preset.minutes ? 'secondary' : 'outline'
            }
            size='sm'
            onClick={() => applyQuickRange(preset.minutes)}
            disabled={props.disabled}
            aria-pressed={selectedPreset === preset.minutes}
          >
            {t(preset.label)}
          </Button>
        ))}
      </div>

      <FieldDescription>
        {t('Times use your browser timezone: {{timezone}}', { timezone })}
        {props.availableFrom
          ? ` · ${t('Available from {{time}}', {
              time: dayjs
                .unix(props.availableFrom)
                .format('YYYY-MM-DD HH:mm:ss'),
            })}`
          : null}
      </FieldDescription>
    </FieldGroup>
  )
}
