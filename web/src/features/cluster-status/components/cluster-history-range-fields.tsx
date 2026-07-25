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

import { DateTimePicker } from '@/components/datetime-picker'
import { Button } from '@/components/ui/button'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import dayjs from '@/lib/dayjs'
import { cn } from '@/lib/utils'

import { buildClusterHistoryPresetRange } from '../lib/export'
import { clusterExportDialogLayout } from '../lib/export-dialog-layout'

type ClusterHistoryRangeFieldsProps = {
  start: Date
  end: Date
  onChange: (range: { start: Date; end: Date }) => void
  availableFrom?: number
  disabled?: boolean
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
      <div className={clusterExportDialogLayout.rangeGrid}>
        <Field>
          <FieldLabel htmlFor={startId}>{t('Start time')}</FieldLabel>
          <DateTimePicker
            id={startId}
            value={props.start}
            onChange={(start) => {
              if (start) {
                setSelectedPreset(null)
                props.onChange({ start, end: props.end })
              }
            }}
            disabled={props.disabled}
            clearable={false}
            precision='second'
            placeholder={t('Select start time')}
          />
        </Field>
        <Field>
          <FieldLabel htmlFor={endId}>{t('End time')}</FieldLabel>
          <DateTimePicker
            id={endId}
            value={props.end}
            onChange={(end) => {
              if (end) {
                setSelectedPreset(null)
                props.onChange({ start: props.start, end })
              }
            }}
            disabled={props.disabled}
            clearable={false}
            precision='second'
            placeholder={t('Select end time')}
          />
        </Field>
      </div>

      <div className='grid grid-cols-2 gap-2 sm:flex'>
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
            variant={selectedPreset === preset.minutes ? 'default' : 'outline'}
            size='sm'
            className={cn(
              'flex-1',
              selectedPreset === preset.minutes &&
                'ring-ring ring-2 ring-offset-2'
            )}
            onClick={() => applyQuickRange(preset.minutes)}
            disabled={props.disabled}
            aria-pressed={selectedPreset === preset.minutes}
          >
            {t(preset.label)}
          </Button>
        ))}
      </div>

      <FieldDescription className={clusterExportDialogLayout.rangeMeta}>
        <span>
          {t('Times use your browser timezone: {{timezone}}', { timezone })}
        </span>
        {props.availableFrom ? (
          <span>
            {t('Available from {{time}}', {
              time: dayjs
                .unix(props.availableFrom)
                .format('YYYY-MM-DD HH:mm:ss'),
            })}
          </span>
        ) : null}
      </FieldDescription>
    </FieldGroup>
  )
}
