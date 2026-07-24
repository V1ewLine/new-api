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
import { toRFC3339Seconds } from './export.ts'

export type TelemetryTrendTimeRange =
  | {
      kind: 'relative'
      minutes: number
    }
  | {
      kind: 'custom'
      start: Date
      end: Date
    }

export const DEFAULT_TELEMETRY_TREND_RANGE: TelemetryTrendTimeRange = {
  kind: 'relative',
  minutes: 60,
}

export const TELEMETRY_TREND_PRESETS = [
  { minutes: 15, labelKey: 'Last 15 minutes' },
  { minutes: 60, labelKey: 'Last hour' },
  { minutes: 360, labelKey: 'Last 6 hours' },
  { minutes: 1440, labelKey: 'Last 24 hours' },
  { minutes: 10080, labelKey: 'Last 7 days' },
] as const

export function resolveTelemetryTrendRange(
  range: TelemetryTrendTimeRange,
  now = Date.now()
): { start: Date; end: Date } {
  if (range.kind === 'custom') {
    return { start: range.start, end: range.end }
  }
  const end = new Date(Math.floor(now / 1000) * 1000)
  return {
    start: new Date(end.getTime() - range.minutes * 60000),
    end,
  }
}

export function telemetryTrendRangeKey(
  range: TelemetryTrendTimeRange
): readonly (string | number)[] {
  if (range.kind === 'relative') {
    return ['relative', range.minutes]
  }
  return ['custom', range.start.getTime(), range.end.getTime()]
}

export function telemetryTrendRangeParams(
  range: TelemetryTrendTimeRange,
  maxPoints: number,
  now = Date.now()
) {
  const resolved = resolveTelemetryTrendRange(range, now)
  return {
    start_at: toRFC3339Seconds(resolved.start),
    end_at: toRFC3339Seconds(resolved.end),
    max_points: maxPoints,
  }
}

export function validateTelemetryTrendRange(
  range: TelemetryTrendTimeRange,
  retentionDays: number
): 'invalid_order' | 'exceeds_retention' | null {
  if (range.kind === 'relative') {
    return range.minutes > retentionDays * 1440 ? 'exceeds_retention' : null
  }
  if (
    Number.isNaN(range.start.getTime()) ||
    Number.isNaN(range.end.getTime()) ||
    range.start.getTime() >= range.end.getTime()
  ) {
    return 'invalid_order'
  }
  if (range.end.getTime() - range.start.getTime() > retentionDays * 86400000) {
    return 'exceeds_retention'
  }
  return null
}
