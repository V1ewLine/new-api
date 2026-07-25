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
export type DateTimePickerPrecision = 'minute' | 'second'

export function dateTimePickerTimeValue(
  date: Date | undefined,
  precision: DateTimePickerPrecision
): string {
  if (!date) return precision === 'second' ? '00:00:00' : '00:00'
  const hours = date.getHours().toString().padStart(2, '0')
  const minutes = date.getMinutes().toString().padStart(2, '0')
  if (precision === 'minute') return `${hours}:${minutes}`
  const seconds = date.getSeconds().toString().padStart(2, '0')
  return `${hours}:${minutes}:${seconds}`
}

export function applyDateTimePickerTime(date: Date, value: string): Date {
  const [hours = 0, minutes = 0, seconds = 0] = value.split(':').map(Number)
  const result = new Date(date)
  result.setHours(hours, minutes, seconds, 0)
  return result
}
