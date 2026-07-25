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
import { ArrowDown01Icon, Cancel01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import * as React from 'react'
import { enUS, fr, ja, ru, vi, zhCN } from 'react-day-picker/locale'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Calendar } from '@/components/ui/calendar'
import { Input } from '@/components/ui/input'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import dayjs from '@/lib/dayjs'
import { cn } from '@/lib/utils'

import {
  applyDateTimePickerTime,
  dateTimePickerTimeValue,
  type DateTimePickerPrecision,
} from './lib/datetime-picker'

const calendarLocales = {
  en: enUS,
  zh: zhCN,
  fr,
  ru,
  ja,
  vi,
} as const

interface DateTimePickerProps {
  value?: Date
  onChange?: (date: Date | undefined) => void
  placeholder?: string
  className?: string
  id?: string
  precision?: DateTimePickerPrecision
  clearable?: boolean
  disabled?: boolean
}

export function DateTimePicker(props: DateTimePickerProps) {
  const { t, i18n } = useTranslation()
  const precision = props.precision ?? 'minute'
  const placeholderText = props.placeholder ?? t('Select date')
  const calendarLocale =
    calendarLocales[i18n.language as keyof typeof calendarLocales] ?? enUS
  const currentYear = new Date().getFullYear()
  const [open, setOpen] = React.useState(false)
  const [date, setDate] = React.useState<Date | undefined>(props.value)
  const [month, setMonth] = React.useState<Date | undefined>(props.value)
  const [time, setTime] = React.useState<string>(() =>
    dateTimePickerTimeValue(props.value, precision)
  )

  React.useEffect(() => {
    setDate(props.value)
    setMonth(props.value)
    if (props.value) {
      setTime(dateTimePickerTimeValue(props.value, precision))
    }
  }, [precision, props.value])

  const handleDateSelect = (selectedDate: Date | undefined) => {
    if (selectedDate) {
      const newDate = applyDateTimePickerTime(selectedDate, time)
      setDate(newDate)
      setMonth(newDate)
      props.onChange?.(newDate)
      setOpen(false)
    } else {
      setDate(undefined)
      setMonth(undefined)
      props.onChange?.(undefined)
    }
  }

  const handleTimeChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newTime = e.target.value
    setTime(newTime)

    if (date) {
      const newDate = applyDateTimePickerTime(date, newTime)
      setDate(newDate)
      props.onChange?.(newDate)
    }
  }

  const handleClear = () => {
    setDate(undefined)
    setMonth(undefined)
    setTime(dateTimePickerTimeValue(undefined, precision))
    props.onChange?.(undefined)
  }

  return (
    <div className={cn('flex gap-2', props.className)}>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger
          render={
            <Button
              variant='outline'
              className={cn(
                'flex-1 justify-between font-normal',
                !date && 'text-muted-foreground'
              )}
              id={props.id}
              disabled={props.disabled}
            />
          }
        >
          {date ? dayjs(date).format('YYYY-MM-DD') : placeholderText}
          <HugeiconsIcon
            icon={ArrowDown01Icon}
            strokeWidth={2}
            data-icon='inline-end'
            className='opacity-50'
          />
        </PopoverTrigger>
        <PopoverContent className='w-auto overflow-hidden p-0' align='start'>
          <Calendar
            mode='single'
            selected={date}
            month={month}
            onMonthChange={setMonth}
            captionLayout='dropdown'
            onSelect={handleDateSelect}
            locale={calendarLocale}
            startMonth={new Date(currentYear - 100, 0)}
            endMonth={new Date(currentYear + 100, 11)}
          />
        </PopoverContent>
      </Popover>
      <Input
        type='time'
        step={precision === 'second' ? 1 : undefined}
        value={time}
        onChange={handleTimeChange}
        className='w-32 appearance-none [&::-webkit-calendar-picker-indicator]:hidden [&::-webkit-calendar-picker-indicator]:appearance-none'
        disabled={!date || props.disabled}
      />
      {date && (props.clearable ?? true) ? (
        <Button
          type='button'
          variant='outline'
          size='icon'
          onClick={handleClear}
          className='shrink-0'
          aria-label={t('Clear')}
          disabled={props.disabled}
        >
          <HugeiconsIcon icon={Cancel01Icon} strokeWidth={2} />
        </Button>
      ) : null}
    </div>
  )
}
