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
import {
  Alert02Icon,
  ArrowRight01Icon,
  ChartLineIcon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { VChart } from '@visactor/react-vchart'
import type { TFunction } from 'i18next'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Skeleton } from '@/components/ui/skeleton'
import { useChartTheme } from '@/lib/use-chart-theme'
import { VCHART_OPTION } from '@/lib/vchart'

import {
  useClusterTelemetryTrends,
  type ClusterTelemetryTrendQuery,
} from '../hooks/use-cluster-telemetry-trends'
import type { TelemetryTrendTimeRange } from '../lib/trend-range'
import type { TelemetryTrendPoint, TelemetryTrendResponse } from '../types'
import { TelemetryTimeRangeControl } from './telemetry-time-range-control'

export type TelemetryTrendSeries = {
  id: string
  label: string
  color: string
  value: (point: TelemetryTrendPoint) => number | undefined
}

export type TelemetryTrendChartConfig = {
  id: string
  titleKey: string
  descriptionKey: string
  unit: string
  percent?: boolean
  series: (
    points: TelemetryTrendPoint[],
    translate: TFunction
  ) => TelemetryTrendSeries[]
}

type TelemetryTrendGridProps = {
  clusterId: number
  range: TelemetryTrendTimeRange
  refreshInterval: number
  query: ClusterTelemetryTrendQuery
  configs: TelemetryTrendChartConfig[]
}

type TelemetryTrendChartCardProps = {
  clusterId: number
  range: TelemetryTrendTimeRange
  refreshInterval: number
  response: TelemetryTrendResponse
  config: TelemetryTrendChartConfig
}

function cloneRange(range: TelemetryTrendTimeRange): TelemetryTrendTimeRange {
  if (range.kind === 'relative') return { ...range }
  return {
    kind: 'custom',
    start: new Date(range.start),
    end: new Date(range.end),
  }
}

function chartThemeTokens(resolvedTheme: string) {
  return {
    text:
      resolvedTheme === 'dark'
        ? 'rgba(255, 255, 255, 0.68)'
        : 'rgba(15, 23, 42, 0.62)',
    grid:
      resolvedTheme === 'dark'
        ? 'rgba(255, 255, 255, 0.12)'
        : 'rgba(15, 23, 42, 0.12)',
  }
}

function axisValue(value: number | string, unit: string): string {
  const numeric = Number(value)
  if (!Number.isFinite(numeric)) return String(value)
  const compact = Intl.NumberFormat(undefined, {
    notation: Math.abs(numeric) >= 1000 ? 'compact' : 'standard',
    maximumFractionDigits: 1,
  }).format(numeric)
  return unit ? `${compact}${unit === '%' ? '' : ' '}${unit}` : compact
}

function TelemetryTrendChartCanvas(props: {
  config: TelemetryTrendChartConfig
  points: TelemetryTrendPoint[]
  expanded?: boolean
}) {
  const { t } = useTranslation()
  const { resolvedTheme, themeReady } = useChartTheme()
  const tokens = chartThemeTokens(resolvedTheme)
  const spec = useMemo(() => {
    const series = props.config.series(props.points, t)
    const firstPoint = props.points.at(0)
    const lastPoint = props.points.at(-1)
    const duration =
      firstPoint && lastPoint ? lastPoint.timestamp - firstPoint.timestamp : 0
    const rows = props.points.flatMap((point) => {
      const date = new Date(point.timestamp * 1000)
      const time =
        duration > 86400
          ? date.toLocaleString(undefined, {
              month: '2-digit',
              day: '2-digit',
              hour: '2-digit',
              minute: '2-digit',
            })
          : date.toLocaleTimeString(undefined, {
              hour: '2-digit',
              minute: '2-digit',
              second: '2-digit',
            })
      const fullTime = date.toLocaleString(undefined, {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
      })
      return series.map((item) => ({
        time,
        fullTime,
        series: item.label,
        value: item.value(point) ?? null,
      }))
    })
    if (!rows.some((row) => typeof row.value === 'number')) {
      return null
    }
    return {
      type: 'line' as const,
      data: [{ id: props.config.id, values: rows }],
      xField: 'time',
      yField: 'value',
      seriesField: 'series',
      color: series.map((item) => item.color),
      invalidType: 'break' as const,
      padding: { top: series.length > 1 ? 38 : 16, right: 16, bottom: 8 },
      line: {
        style: { lineWidth: props.expanded ? 2.5 : 2 },
      },
      point: {
        visible: props.points.length <= 60,
        style: {
          size: props.expanded ? 5 : 4,
          stroke: resolvedTheme === 'dark' ? '#111827' : '#ffffff',
          lineWidth: 1.5,
        },
      },
      legends:
        series.length > 1
          ? {
              visible: true,
              orient: 'top' as const,
              position: 'start' as const,
            }
          : { visible: false },
      tooltip: {
        dimension: {
          title: {
            value: (datum: { fullTime?: string }) => datum.fullTime ?? '',
          },
        },
      },
      axes: [
        {
          orient: 'bottom' as const,
          label: {
            autoRotate: false,
            autoHide: true,
            style: { fill: tokens.text, fontSize: 11 },
          },
          tick: { visible: false },
        },
        {
          orient: 'left' as const,
          min: 0,
          max: props.config.percent ? 100 : undefined,
          label: {
            formatMethod: (value: number | string) =>
              axisValue(value, props.config.unit),
            style: { fill: tokens.text, fontSize: 11 },
          },
          grid: {
            visible: true,
            style: { lineDash: [3, 3], stroke: tokens.grid },
          },
        },
      ],
    }
  }, [
    props.config,
    props.expanded,
    props.points,
    resolvedTheme,
    t,
    tokens.grid,
    tokens.text,
  ])

  if (!spec) {
    return (
      <Empty className={props.expanded ? 'h-[min(62vh,40rem)]' : 'h-56'}>
        <EmptyHeader>
          <EmptyMedia variant='icon'>
            <HugeiconsIcon icon={ChartLineIcon} strokeWidth={2} />
          </EmptyMedia>
          <EmptyTitle>{t('No trend data')}</EmptyTitle>
          <EmptyDescription>
            {t('No successful samples contain this metric in the time window.')}
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    )
  }

  return (
    <div className={props.expanded ? 'h-[min(62vh,40rem)]' : 'h-56'}>
      {themeReady ? (
        <VChart
          key={`${props.config.id}-${resolvedTheme}-${props.expanded ? 'expanded' : 'card'}`}
          spec={{
            ...spec,
            theme: resolvedTheme === 'dark' ? 'dark' : 'light',
            background: 'transparent',
          }}
          option={VCHART_OPTION}
        />
      ) : (
        <Skeleton className='h-full w-full rounded-lg' />
      )}
    </div>
  )
}

function TelemetryTrendChartCard(props: TelemetryTrendChartCardProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [expandedRange, setExpandedRange] = useState(() =>
    cloneRange(props.range)
  )
  const expandedQuery = useClusterTelemetryTrends({
    clusterId: props.clusterId,
    range: expandedRange,
    refreshInterval: props.refreshInterval,
    maxPoints: 1440,
    enabled: open,
  })

  function updateOpen(nextOpen: boolean) {
    if (nextOpen) {
      setExpandedRange(cloneRange(props.range))
    }
    setOpen(nextOpen)
  }

  return (
    <>
      <Card className='h-full'>
        <CardHeader>
          <CardTitle>{t(props.config.titleKey)}</CardTitle>
          <CardDescription>{t(props.config.descriptionKey)}</CardDescription>
        </CardHeader>
        <CardContent className='mt-auto'>
          <TelemetryTrendChartCanvas
            config={props.config}
            points={props.response.points}
          />
        </CardContent>
        <CardFooter className='justify-between'>
          <span className='text-muted-foreground text-xs'>
            {t('{{count}} samples · {{seconds}}s buckets', {
              count: props.response.sample_count,
              seconds: props.response.bucket_seconds,
            })}
          </span>
          <Button
            type='button'
            variant='ghost'
            size='icon-sm'
            aria-label={t('Expand {{chart}}', {
              chart: t(props.config.titleKey),
            })}
            onClick={() => updateOpen(true)}
          >
            <HugeiconsIcon icon={ArrowRight01Icon} strokeWidth={2} />
          </Button>
        </CardFooter>
      </Card>

      <Dialog open={open} onOpenChange={updateOpen}>
        <DialogContent className='sm:max-w-[min(94vw,80rem)]'>
          <DialogHeader className='pr-10'>
            <DialogTitle>{t(props.config.titleKey)}</DialogTitle>
            <DialogDescription>
              {t(props.config.descriptionKey)}
            </DialogDescription>
          </DialogHeader>
          <div className='flex justify-end'>
            <TelemetryTimeRangeControl
              value={expandedRange}
              onChange={setExpandedRange}
              retentionDays={
                expandedQuery.data?.retention_days ??
                props.response.retention_days
              }
              availableFrom={
                expandedQuery.data?.available_from ??
                props.response.available_from
              }
              scope='chart'
            />
          </div>
          {expandedQuery.isLoading ? (
            <Skeleton className='h-[min(62vh,40rem)] w-full rounded-lg' />
          ) : null}
          {expandedQuery.isError ? (
            <ExpandedChartError
              message={
                expandedQuery.error instanceof Error
                  ? expandedQuery.error.message
                  : t('Failed to load telemetry trends')
              }
              onRetry={() => void expandedQuery.refetch()}
            />
          ) : null}
          {expandedQuery.data ? (
            <TelemetryTrendChartCanvas
              config={props.config}
              points={expandedQuery.data.points}
              expanded
            />
          ) : null}
        </DialogContent>
      </Dialog>
    </>
  )
}

function ExpandedChartError(props: { message: string; onRetry: () => void }) {
  const { t } = useTranslation()
  return (
    <Empty className='h-[min(62vh,40rem)]'>
      <EmptyHeader>
        <EmptyMedia variant='icon'>
          <HugeiconsIcon icon={Alert02Icon} strokeWidth={2} />
        </EmptyMedia>
        <EmptyTitle>{t('Failed to load telemetry trends')}</EmptyTitle>
        <EmptyDescription>{props.message}</EmptyDescription>
        <Button type='button' variant='outline' onClick={props.onRetry}>
          {t('Retry')}
        </Button>
      </EmptyHeader>
    </Empty>
  )
}

export function TelemetryTrendGrid(props: TelemetryTrendGridProps) {
  const { t } = useTranslation()
  if (props.query.isLoading && !props.query.data) {
    return (
      <div className='grid gap-4 lg:grid-cols-2'>
        {props.configs.map((config) => (
          <Skeleton key={config.id} className='h-[22rem] w-full rounded-xl' />
        ))}
      </div>
    )
  }
  if (props.query.isError && !props.query.data) {
    return (
      <Card>
        <CardContent>
          <Empty>
            <EmptyHeader>
              <EmptyMedia variant='icon'>
                <HugeiconsIcon icon={Alert02Icon} strokeWidth={2} />
              </EmptyMedia>
              <EmptyTitle>{t('Failed to load telemetry trends')}</EmptyTitle>
              <EmptyDescription>
                {props.query.error instanceof Error
                  ? props.query.error.message
                  : t('Please try again later.')}
              </EmptyDescription>
              <Button
                type='button'
                variant='outline'
                onClick={() => void props.query.refetch()}
              >
                {t('Retry')}
              </Button>
            </EmptyHeader>
          </Empty>
        </CardContent>
      </Card>
    )
  }
  const response = props.query.data
  if (!response?.points.length) {
    return (
      <Card>
        <CardContent>
          <Empty>
            <EmptyHeader>
              <EmptyMedia variant='icon'>
                <HugeiconsIcon icon={ChartLineIcon} strokeWidth={2} />
              </EmptyMedia>
              <EmptyTitle>{t('No telemetry history')}</EmptyTitle>
              <EmptyDescription>
                {t('No samples were stored in the selected time window.')}
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className='grid gap-4 lg:grid-cols-2'>
      {props.configs.map((config) => (
        <TelemetryTrendChartCard
          key={config.id}
          clusterId={props.clusterId}
          range={props.range}
          refreshInterval={props.refreshInterval}
          response={response}
          config={config}
        />
      ))}
    </div>
  )
}
