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
import type { ClusterHealthStatus, NormalizedTelemetry } from '../types'

export function formatCompactNumber(value: number | undefined): string {
  if (value === undefined || !Number.isFinite(value)) return '—'
  return Intl.NumberFormat(undefined, {
    notation: 'compact',
    maximumFractionDigits: 2,
  }).format(value)
}

export function formatPercent(value: number | undefined): string {
  if (value === undefined || !Number.isFinite(value)) return '—'
  return `${value.toFixed(1)}%`
}

export function formatBytes(value: number | undefined): string {
  if (value === undefined || !Number.isFinite(value) || value < 0) return '—'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let scaled = value
  let unitIndex = 0
  while (scaled >= 1024 && unitIndex < units.length - 1) {
    scaled /= 1024
    unitIndex++
  }
  return `${scaled.toFixed(unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`
}

export function formatWatts(value: number | undefined): string {
  if (value === undefined || !Number.isFinite(value) || value < 0) return '—'
  return `${value.toFixed(1)} W`
}

export function formatTimestamp(value: number | undefined): string {
  if (!value) return '—'
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value * 1000))
}

export function averageGPUUtilization(
  telemetry: NormalizedTelemetry | undefined
): number | undefined {
  const values =
    telemetry?.machine.gpu.devices
      .map((device) => device.utilization_percent)
      .filter((value): value is number => value !== undefined) ?? []
  if (values.length === 0) return undefined
  return values.reduce((sum, value) => sum + value, 0) / values.length
}

export function getVisiblePages(
  currentPage: number,
  pageCount: number
): number[] {
  if (pageCount <= 5) {
    return Array.from(
      { length: Math.max(0, pageCount) },
      (_, index) => index + 1
    )
  }
  const start = Math.min(
    Math.max(1, currentPage - 2),
    Math.max(1, pageCount - 4)
  )
  return Array.from({ length: 5 }, (_, index) => start + index)
}

export function healthPriority(status: ClusterHealthStatus): number {
  switch (status) {
    case 'offline':
      return 4
    case 'abnormal':
      return 3
    case 'partial':
      return 2
    case 'online':
      return 1
    default:
      return 0
  }
}
