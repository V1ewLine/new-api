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
import { api } from '@/lib/api'

import { getExportFilename } from './lib/export'
import type {
  ApiResponse,
  Cluster,
  ClusterExportParams,
  ClusterOverview,
  ClusterOverviewParams,
  ClusterStatusSettings,
  CreateClusterPayload,
  CredentialIssueResponse,
  CredentialVerificationResponse,
  ModelDetail,
  ModelOption,
  NormalizedTelemetry,
} from './types'

export async function getClusterStatusSettings() {
  const response = await api.get<ApiResponse<ClusterStatusSettings>>(
    '/api/clusters/settings'
  )
  return response.data
}

export async function getClusterOverview(params: ClusterOverviewParams) {
  const response = await api.get<ApiResponse<ClusterOverview>>(
    '/api/clusters/overview',
    { params }
  )
  return response.data
}

export async function getClusterModelOptions() {
  const response = await api.get<ApiResponse<ModelOption[]>>(
    '/api/clusters/model-options'
  )
  return response.data
}

export async function downloadClusterExport(params: ClusterExportParams) {
  const response = await api.get<Blob>('/api/clusters/export/latest', {
    params,
    responseType: 'blob',
    skipBusinessError: true,
    disableDuplicate: true,
  })
  const disposition = response.headers['content-disposition']
  if (!disposition) {
    const payload = await response.data.text()
    let errorResponse: ApiResponse<never>
    try {
      errorResponse = JSON.parse(payload) as ApiResponse<never>
    } catch {
      throw new Error('Cluster export failed')
    }
    throw new Error(errorResponse.message || 'Cluster export failed')
  }
  return {
    blob: response.data,
    filename: getExportFilename(disposition),
  }
}

export async function createCluster(payload: CreateClusterPayload) {
  const response = await api.post<ApiResponse<CredentialIssueResponse>>(
    '/api/clusters/',
    payload
  )
  return response.data
}

export async function deleteCluster(clusterId: number) {
  const response = await api.delete<ApiResponse<null>>(
    `/api/clusters/${clusterId}`
  )
  return response.data
}

export async function getClusterModelDetail(modelId: number) {
  const response = await api.get<ApiResponse<ModelDetail>>(
    `/api/clusters/models/${modelId}`
  )
  return response.data
}

export async function getClusterDetail(clusterId: number) {
  const response = await api.get<ApiResponse<Cluster>>(
    `/api/clusters/${clusterId}`
  )
  return response.data
}

export async function getLatestClusterTelemetry(clusterId: number) {
  const response = await api.get<ApiResponse<NormalizedTelemetry | null>>(
    `/api/clusters/${clusterId}/telemetry/latest`
  )
  return response.data
}

export async function refreshCluster(clusterId: number) {
  const response = await api.post<ApiResponse<Cluster>>(
    `/api/clusters/${clusterId}/refresh`
  )
  return response.data
}

export async function rotateClusterCredential(clusterId: number) {
  const response = await api.post<ApiResponse<CredentialIssueResponse>>(
    `/api/clusters/${clusterId}/credential/rotate`
  )
  return response.data
}

export async function verifyClusterCredential(clusterId: number) {
  const response = await api.post<ApiResponse<CredentialVerificationResponse>>(
    `/api/clusters/${clusterId}/credential/verify`
  )
  return response.data
}
