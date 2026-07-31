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
import type { TimeGranularity } from '@/lib/time'

const FALLBACK_EXPORT_FILENAME = 'new-api-model-analytics.csv'
const DAY_IN_MILLISECONDS = 86_400_000

export const ALL_MODELS_EXPORT_VALUE = '__all_models__'

export type ModelExportRangeError = 'invalid_order' | 'range_too_large' | null

export type AlignedModelExportRange = {
  start: Date
  end: Date
  error: ModelExportRangeError
}

function floorDate(date: Date, granularity: TimeGranularity): Date {
  const result = new Date(date)
  if (granularity === 'week') {
    result.setHours(0, 0, 0, 0)
    const daysSinceMonday = (result.getDay() + 6) % 7
    result.setDate(result.getDate() - daysSinceMonday)
    return result
  }
  if (granularity === 'day') {
    result.setHours(0, 0, 0, 0)
    return result
  }
  result.setMinutes(0, 0, 0)
  return result
}

function nextBucket(date: Date, granularity: TimeGranularity): Date {
  const result = new Date(date)
  if (granularity === 'week') {
    result.setDate(result.getDate() + 7)
  } else if (granularity === 'day') {
    result.setDate(result.getDate() + 1)
  } else {
    result.setHours(result.getHours() + 1)
  }
  return result
}

function maxRangeMilliseconds(granularity: TimeGranularity): number {
  if (granularity === 'week') return 5 * 366 * DAY_IN_MILLISECONDS
  if (granularity === 'day') return 2 * 366 * DAY_IN_MILLISECONDS
  return 90 * DAY_IN_MILLISECONDS
}

export function alignModelExportRange(
  start: Date,
  end: Date,
  granularity: TimeGranularity
): AlignedModelExportRange {
  if (
    Number.isNaN(start.getTime()) ||
    Number.isNaN(end.getTime()) ||
    start.getTime() >= end.getTime()
  ) {
    return { start, end, error: 'invalid_order' }
  }

  const effectiveStart = floorDate(start, granularity)
  let effectiveEnd = floorDate(end, granularity)
  if (effectiveEnd.getTime() < end.getTime()) {
    effectiveEnd = nextBucket(effectiveEnd, granularity)
  }
  if (
    effectiveEnd.getTime() - effectiveStart.getTime() >
    maxRangeMilliseconds(granularity)
  ) {
    return {
      start: effectiveStart,
      end: effectiveEnd,
      error: 'range_too_large',
    }
  }
  return { start: effectiveStart, end: effectiveEnd, error: null }
}

export function getModelExportFilename(contentDisposition: string): string {
  const encodedMatch = contentDisposition.match(
    /filename\*\s*=\s*UTF-8''([^;]+)/i
  )
  if (encodedMatch?.[1]) {
    try {
      return decodeURIComponent(encodedMatch[1].trim())
    } catch {
      return FALLBACK_EXPORT_FILENAME
    }
  }
  const quotedMatch = contentDisposition.match(/filename\s*=\s*"([^"]+)"/i)
  if (quotedMatch?.[1]) return quotedMatch[1]
  const plainMatch = contentDisposition.match(/filename\s*=\s*([^;]+)/i)
  return plainMatch?.[1]?.trim() || FALLBACK_EXPORT_FILENAME
}

export function saveModelExportFile(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  document.body.append(anchor)
  anchor.click()
  anchor.remove()
  URL.revokeObjectURL(url)
}

export function getBrowserTimezone(): string {
  return Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
}
