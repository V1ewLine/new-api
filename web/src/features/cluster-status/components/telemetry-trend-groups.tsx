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
import type { TFunction } from 'i18next'

import type { ClusterTelemetryTrendQuery } from '../hooks/use-cluster-telemetry-trends'
import type { TelemetryTrendTimeRange } from '../lib/trend-range'
import type { TelemetryTrendPoint } from '../types'
import {
  TelemetryTrendGrid,
  type TelemetryTrendChartConfig,
  type TelemetryTrendSeries,
} from './telemetry-trend-chart'

type TelemetryTrendGroupProps = {
  clusterId: number
  range: TelemetryTrendTimeRange
  refreshInterval: number
  query: ClusterTelemetryTrendQuery
}

const COLORS = {
  blue: '#2563eb',
  violet: '#7c3aed',
  green: '#16a34a',
  amber: '#d97706',
  red: '#dc2626',
  cyan: '#0891b2',
  pink: '#db2777',
  indigo: '#4f46e5',
} as const

const GPU_COLORS = [
  COLORS.blue,
  COLORS.violet,
  COLORS.green,
  COLORS.amber,
  COLORS.red,
  COLORS.cyan,
  COLORS.pink,
  COLORS.indigo,
]

function oneSeries(
  id: string,
  labelKey: string,
  color: string,
  value: (point: TelemetryTrendPoint) => number | undefined
) {
  return (_points: TelemetryTrendPoint[], translate: TFunction) => [
    { id, label: translate(labelKey), color, value },
  ]
}

const overviewConfigs: TelemetryTrendChartConfig[] = [
  {
    id: 'telemetry-availability',
    titleKey: 'Telemetry Availability',
    descriptionKey:
      'Poll success and component availability in the selected time window.',
    unit: '%',
    percent: true,
    series: (_points, translate) => [
      {
        id: 'poll',
        label: translate('Poll Success'),
        color: COLORS.blue,
        value: (point) => point.poll_success_percent,
      },
      {
        id: 'engine',
        label: translate('Engine'),
        color: COLORS.green,
        value: (point) => point.engine_availability_percent,
      },
      {
        id: 'machine',
        label: translate('Machine'),
        color: COLORS.violet,
        value: (point) => point.machine_availability_percent,
      },
    ],
  },
  {
    id: 'request-pressure',
    titleKey: 'Request Pressure',
    descriptionKey: 'Running and waiting request counts over time.',
    unit: '',
    series: (_points, translate) => [
      {
        id: 'running',
        label: translate('Running Requests'),
        color: COLORS.blue,
        value: (point) => point.running_requests,
      },
      {
        id: 'waiting',
        label: translate('Waiting Requests'),
        color: COLORS.amber,
        value: (point) => point.waiting_requests,
      },
    ],
  },
  {
    id: 'overview-throughput',
    titleKey: 'Throughput',
    descriptionKey: 'Generated token throughput over time.',
    unit: 'tokens/s',
    series: oneSeries(
      'throughput',
      'Throughput',
      COLORS.green,
      (point) => point.throughput
    ),
  },
  {
    id: 'resource-utilization',
    titleKey: 'Resource Utilization',
    descriptionKey: 'CPU, memory, and cache utilization over time.',
    unit: '%',
    percent: true,
    series: (_points, translate) => [
      {
        id: 'cpu',
        label: translate('CPU Utilization'),
        color: COLORS.blue,
        value: (point) => point.cpu_utilization_percent,
      },
      {
        id: 'memory',
        label: translate('Memory Utilization'),
        color: COLORS.violet,
        value: (point) => point.memory_utilization_percent,
      },
      {
        id: 'cache',
        label: translate('Cache Usage'),
        color: COLORS.green,
        value: (point) => point.cache_usage,
      },
    ],
  },
]

const engineConfigs: TelemetryTrendChartConfig[] = [
  {
    id: 'running-requests',
    titleKey: 'Running Requests',
    descriptionKey: 'Requests currently running in the inference engine.',
    unit: '',
    series: oneSeries(
      'running',
      'Running Requests',
      COLORS.blue,
      (point) => point.running_requests
    ),
  },
  {
    id: 'waiting-requests',
    titleKey: 'Waiting Requests',
    descriptionKey: 'Requests waiting for inference engine capacity.',
    unit: '',
    series: oneSeries(
      'waiting',
      'Waiting Requests',
      COLORS.amber,
      (point) => point.waiting_requests
    ),
  },
  {
    id: 'token-usage',
    titleKey: 'Token Usage',
    descriptionKey: 'Tokens currently held by active engine requests.',
    unit: 'tokens',
    series: oneSeries(
      'tokens',
      'Token Usage',
      COLORS.violet,
      (point) => point.token_usage
    ),
  },
  {
    id: 'engine-throughput',
    titleKey: 'Throughput',
    descriptionKey: 'Generated token throughput over time.',
    unit: 'tokens/s',
    series: oneSeries(
      'throughput',
      'Throughput',
      COLORS.green,
      (point) => point.throughput
    ),
  },
  {
    id: 'cache-usage',
    titleKey: 'Cache Usage',
    descriptionKey: 'Inference engine cache utilization over time.',
    unit: '%',
    percent: true,
    series: oneSeries(
      'cache',
      'Cache Usage',
      COLORS.cyan,
      (point) => point.cache_usage
    ),
  },
]

function gpuPowerSeries(
  points: TelemetryTrendPoint[],
  _translate: TFunction
): TelemetryTrendSeries[] {
  const devices = new Map<string, { name: string; index: number }>()
  for (const point of points) {
    for (const gpu of point.gpus) {
      if (!devices.has(gpu.id)) {
        devices.set(gpu.id, { name: gpu.name, index: gpu.index })
      }
    }
  }
  return [...devices.entries()]
    .sort((left, right) => left[1].index - right[1].index)
    .map(([id, gpu], index) => ({
      id,
      label: `${gpu.name} · GPU ${gpu.index}`,
      color: GPU_COLORS[index % GPU_COLORS.length],
      value: (point: TelemetryTrendPoint) =>
        point.gpus.find((device) => device.id === id)?.power_watts,
    }))
}

const machineConfigs: TelemetryTrendChartConfig[] = [
  {
    id: 'gpu-board-power',
    titleKey: 'GPU Board Power',
    descriptionKey: 'Total power reported across all GPU boards.',
    unit: 'W',
    series: oneSeries(
      'gpu-total',
      'GPU Board Power',
      COLORS.blue,
      (point) => point.gpu_board_power_watts
    ),
  },
  {
    id: 'gpu-device-power',
    titleKey: 'Per-GPU Power',
    descriptionKey: 'Power draw for each GPU device over time.',
    unit: 'W',
    series: gpuPowerSeries,
  },
  {
    id: 'cpu-utilization',
    titleKey: 'CPU Utilization',
    descriptionKey: 'Host CPU utilization over time.',
    unit: '%',
    percent: true,
    series: oneSeries(
      'cpu',
      'CPU Utilization',
      COLORS.green,
      (point) => point.cpu_utilization_percent
    ),
  },
  {
    id: 'memory-utilization',
    titleKey: 'Memory Utilization',
    descriptionKey: 'Host memory utilization over time.',
    unit: '%',
    percent: true,
    series: oneSeries(
      'memory',
      'Memory Utilization',
      COLORS.violet,
      (point) => point.memory_utilization_percent
    ),
  },
]

export function OverviewTrendCharts(props: TelemetryTrendGroupProps) {
  return (
    <TelemetryTrendGrid
      clusterId={props.clusterId}
      range={props.range}
      refreshInterval={props.refreshInterval}
      query={props.query}
      configs={overviewConfigs}
    />
  )
}

export function EngineTrendCharts(props: TelemetryTrendGroupProps) {
  return (
    <TelemetryTrendGrid
      clusterId={props.clusterId}
      range={props.range}
      refreshInterval={props.refreshInterval}
      query={props.query}
      configs={engineConfigs}
    />
  )
}

export function MachineTrendCharts(props: TelemetryTrendGroupProps) {
  return (
    <TelemetryTrendGrid
      clusterId={props.clusterId}
      range={props.range}
      refreshInterval={props.refreshInterval}
      query={props.query}
      configs={machineConfigs}
    />
  )
}
