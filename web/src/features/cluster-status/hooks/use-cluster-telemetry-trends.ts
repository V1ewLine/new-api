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
import { useTranslation } from 'react-i18next'

import {
  getAggregateClusterTelemetryTrends,
  getClusterTelemetryTrends,
} from '../api'
import {
  telemetryTrendRangeKey,
  telemetryTrendRangeParams,
  type TelemetryTrendTimeRange,
} from '../lib/trend-range'
import { clusterQueryKeys } from '../query-keys'

type UseClusterTelemetryTrendsOptions = {
  clusterId: number
  range: TelemetryTrendTimeRange
  refreshInterval: number
  maxPoints?: number
  enabled?: boolean
}

export type TelemetryTrendScope =
  | { kind: 'cluster'; clusterId: number }
  | { kind: 'overview' }
  | { kind: 'model'; modelId: number }

type UseTelemetryTrendsOptions = {
  scope: TelemetryTrendScope
  range: TelemetryTrendTimeRange
  refreshInterval: number
  maxPoints?: number
  enabled?: boolean
}

export function useTelemetryTrends(options: UseTelemetryTrendsOptions) {
  const { t } = useTranslation()
  const maxPoints = options.maxPoints ?? 720
  const rangeKey = telemetryTrendRangeKey(options.range)
  const queryKey =
    options.scope.kind === 'cluster'
      ? clusterQueryKeys.trends(options.scope.clusterId, rangeKey, maxPoints)
      : clusterQueryKeys.aggregateTrends(
          options.scope.kind === 'model' ? options.scope.modelId : undefined,
          rangeKey,
          maxPoints
        )
  return useQuery({
    queryKey,
    queryFn: async () => {
      const params = telemetryTrendRangeParams(options.range, maxPoints)
      const response =
        options.scope.kind === 'cluster'
          ? await getClusterTelemetryTrends(options.scope.clusterId, params)
          : await getAggregateClusterTelemetryTrends(
              options.scope.kind === 'model'
                ? options.scope.modelId
                : undefined,
              params
            )
      if (!response.success || !response.data) {
        throw new Error(
          response.message || t('Failed to load telemetry trends')
        )
      }
      return response.data
    },
    enabled: options.enabled ?? true,
    refetchInterval:
      options.range.kind === 'relative' ? options.refreshInterval : false,
  })
}

export function useClusterTelemetryTrends(
  options: UseClusterTelemetryTrendsOptions
) {
  return useTelemetryTrends({
    scope: { kind: 'cluster', clusterId: options.clusterId },
    range: options.range,
    refreshInterval: options.refreshInterval,
    maxPoints: options.maxPoints,
    enabled: options.enabled,
  })
}

export type TelemetryTrendQuery = ReturnType<typeof useTelemetryTrends>
export type ClusterTelemetryTrendQuery = TelemetryTrendQuery
