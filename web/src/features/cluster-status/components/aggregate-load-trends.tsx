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
import { useTranslation } from 'react-i18next'

import type {
  TelemetryTrendQuery,
  TelemetryTrendScope,
} from '../hooks/use-cluster-telemetry-trends'
import type { TelemetryTrendTimeRange } from '../lib/trend-range'
import type { TelemetryTrendPoint } from '../types'
import { TelemetryTimeRangeControl } from './telemetry-time-range-control'
import {
  TelemetryTrendGrid,
  type TelemetryTrendChartConfig,
} from './telemetry-trend-chart'

const aggregateLoadConfigs: TelemetryTrendChartConfig[] = [
  {
    id: 'current-requests',
    titleKey: 'Current Requests Trend',
    descriptionKey:
      'Sum of running and waiting requests across reporting clusters.',
    unit: '',
    series: (_points, translate) => [
      {
        id: 'current-requests',
        label: translate('Current Requests'),
        color: '#2563eb',
        value: (point: TelemetryTrendPoint) => point.current_requests,
      },
    ],
    coverage: (point) => ({
      reporting: point.requests_reporting_clusters ?? 0,
      monitored: point.monitored_clusters ?? 0,
    }),
  },
  {
    id: 'current-token-usage',
    titleKey: 'Current Token Usage Trend',
    descriptionKey: 'Sum of current token usage across reporting clusters.',
    unit: 'tokens',
    series: (_points, translate) => [
      {
        id: 'current-token-usage',
        label: translate('Current Token Usage'),
        color: '#7c3aed',
        value: (point: TelemetryTrendPoint) => point.current_token_usage,
      },
    ],
    coverage: (point) => ({
      reporting: point.tokens_reporting_clusters ?? 0,
      monitored: point.monitored_clusters ?? 0,
    }),
  },
]

type AggregateLoadTrendsProps = {
  title: string
  description: string
  scope: TelemetryTrendScope
  range: TelemetryTrendTimeRange
  onRangeChange: (range: TelemetryTrendTimeRange) => void
  refreshInterval: number
  query: TelemetryTrendQuery
}

export function AggregateLoadTrends(props: AggregateLoadTrendsProps) {
  const { t } = useTranslation()
  return (
    <section className='flex flex-col gap-3'>
      <div className='flex flex-col justify-between gap-3 sm:flex-row sm:items-end'>
        <div>
          <h2 className='text-lg font-semibold'>{props.title}</h2>
          <p className='text-muted-foreground mt-1 text-sm'>
            {props.description}
          </p>
        </div>
        <TelemetryTimeRangeControl
          value={props.range}
          onChange={props.onRangeChange}
          retentionDays={props.query.data?.retention_days ?? 7}
          availableFrom={props.query.data?.available_from ?? 0}
          scope='page'
        />
      </div>
      <TelemetryTrendGrid
        scope={props.scope}
        range={props.range}
        refreshInterval={props.refreshInterval}
        query={props.query}
        configs={aggregateLoadConfigs}
      />
      <p className='text-muted-foreground text-xs'>
        {t(
          'Trend points sum only valid samples in each bucket; missing data is shown as a gap.'
        )}
      </p>
    </section>
  )
}
