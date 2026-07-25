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
import type { TelemetryTrendTimeRange } from './trend-range.ts'

export const CLUSTER_DETAIL_TABS = ['overview', 'engine', 'machine'] as const
export type ClusterDetailTab = (typeof CLUSTER_DETAIL_TABS)[number]

export const TELEMETRY_TREND_ROUTE_RANGES = [
  '15m',
  '1h',
  '6h',
  '24h',
  '7d',
  'custom',
] as const
export type TelemetryTrendRouteRange =
  (typeof TELEMETRY_TREND_ROUTE_RANGES)[number]

const RELATIVE_MINUTES_BY_ROUTE_RANGE: Record<
  Exclude<TelemetryTrendRouteRange, 'custom'>,
  number
> = {
  '15m': 15,
  '1h': 60,
  '6h': 360,
  '24h': 1440,
  '7d': 10080,
}

export type ClusterDetailRouteSearch = {
  tab?: ClusterDetailTab
  range?: TelemetryTrendRouteRange
  start?: string
  end?: string
}

export function trendRangeFromRouteSearch(
  search: ClusterDetailRouteSearch
): TelemetryTrendTimeRange {
  if (search.range !== 'custom') {
    return {
      kind: 'relative',
      minutes: search.range
        ? RELATIVE_MINUTES_BY_ROUTE_RANGE[search.range]
        : 60,
    }
  }

  const start = new Date(search.start ?? '')
  const end = new Date(search.end ?? '')
  if (
    Number.isNaN(start.getTime()) ||
    Number.isNaN(end.getTime()) ||
    start.getTime() >= end.getTime()
  ) {
    return { kind: 'relative', minutes: 60 }
  }
  return { kind: 'custom', start, end }
}

export function trendRangeToRouteSearch(
  range: TelemetryTrendTimeRange
): Pick<ClusterDetailRouteSearch, 'range' | 'start' | 'end'> {
  if (range.kind === 'custom') {
    return {
      range: 'custom',
      start: range.start.toISOString(),
      end: range.end.toISOString(),
    }
  }

  const routeRange = Object.entries(RELATIVE_MINUTES_BY_ROUTE_RANGE).find(
    ([, minutes]) => minutes === range.minutes
  )?.[0] as Exclude<TelemetryTrendRouteRange, 'custom'> | undefined
  return {
    range: routeRange ?? '1h',
    start: undefined,
    end: undefined,
  }
}
