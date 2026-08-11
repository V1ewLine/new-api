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
import type { UsageLogExportParams } from '../types'

const FALLBACK_EXPORT_FILENAME = 'new-api-usage-logs.csv'

export function buildUsageLogExportParams(
  searchParams: Record<string, unknown>,
  isAdmin: boolean,
  timezone = Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
): UsageLogExportParams {
  const typeValue = Array.isArray(searchParams.type)
    ? searchParams.type[0]
    : searchParams.type
  const parsedType = Number(typeValue)
  const startTime = Number(searchParams.startTime)
  const endTime = Number(searchParams.endTime)

  return {
    type: Number.isFinite(parsedType) ? parsedType : undefined,
    username:
      isAdmin && searchParams.username
        ? String(searchParams.username)
        : undefined,
    token_name: searchParams.token ? String(searchParams.token) : undefined,
    model_name: searchParams.model ? String(searchParams.model) : undefined,
    start_timestamp: Number.isFinite(startTime)
      ? Math.floor(startTime / 1000)
      : undefined,
    end_timestamp: Number.isFinite(endTime)
      ? Math.floor(endTime / 1000)
      : undefined,
    channel:
      isAdmin && searchParams.channel
        ? Number(searchParams.channel) || undefined
        : undefined,
    group: searchParams.group ? String(searchParams.group) : undefined,
    request_id: searchParams.requestId
      ? String(searchParams.requestId)
      : undefined,
    upstream_request_id: searchParams.upstreamRequestId
      ? String(searchParams.upstreamRequestId)
      : undefined,
    timezone,
    format: 'csv',
  }
}

export function hasValidUsageLogExportRange(
  params: UsageLogExportParams
): boolean {
  return (
    typeof params.start_timestamp === 'number' &&
    typeof params.end_timestamp === 'number' &&
    params.start_timestamp > 0 &&
    params.end_timestamp >= params.start_timestamp
  )
}

export function getUsageLogExportFilename(contentDisposition: string): string {
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

export function saveUsageLogExportFile(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  document.body.append(anchor)
  anchor.click()
  anchor.remove()
  URL.revokeObjectURL(url)
}
