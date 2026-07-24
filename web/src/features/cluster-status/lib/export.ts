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
import type {
  ClusterExportFormat,
  ClusterExportParams,
  ClusterExportScope,
  ClusterHealthStatus,
} from '../types'

const FALLBACK_EXPORT_FILENAME = 'cluster-status-export'

export type ClusterExportContext = 'overview' | 'model' | 'cluster'

type BuildClusterExportParamsInput = {
  context: ClusterExportContext
  scope: ClusterExportScope
  format: ClusterExportFormat
  search?: string
  modelId?: number
  clusterId?: number
  status?: ClusterHealthStatus
}

export function buildClusterExportParams(
  input: BuildClusterExportParamsInput
): ClusterExportParams {
  const params: ClusterExportParams = {
    scope: input.scope,
    format: input.format,
  }
  if (input.context === 'overview') {
    params.search = input.search || undefined
    params.model_id = input.modelId || undefined
    params.status = input.status || undefined
  } else if (input.context === 'model') {
    params.model_id = input.modelId
  } else {
    params.cluster_id = input.clusterId
  }
  return params
}

export function getExportFilename(contentDisposition: string): string {
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
  if (quotedMatch?.[1]) {
    return quotedMatch[1]
  }
  const plainMatch = contentDisposition.match(/filename\s*=\s*([^;]+)/i)
  return plainMatch?.[1]?.trim() || FALLBACK_EXPORT_FILENAME
}

export function saveExportFile(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  document.body.append(anchor)
  anchor.click()
  anchor.remove()
  URL.revokeObjectURL(url)
}
