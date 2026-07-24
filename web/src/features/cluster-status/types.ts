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
export type ClusterHealthStatus =
  | 'unknown'
  | 'online'
  | 'partial'
  | 'abnormal'
  | 'offline'

export type ClusterCredentialStatus = 'pending' | 'active'

export type ApiResponse<T> = {
  success: boolean
  message?: string
  data?: T
}

export type ClusterStatusSettings = {
  refresh_interval_seconds: number
}

export type ModelOption = {
  id: number
  name: string
  icon?: string
  type: string
  enabled: boolean
}

export type TelemetryGPUDevice = {
  index: number
  uuid: string
  name: string
  power_watts?: number
  temperature_celsius?: number
  utilization_percent?: number
  memory_utilization_percent?: number
  memory_used_bytes?: number
  memory_total_bytes?: number
  sm_clock_mhz?: number
}

export type NormalizedTelemetry = {
  schema_version: string
  collection_id: string
  status: string
  collected_at: string
  model_mismatch: boolean
  identity: {
    node_id: string
    engine_id: string
    model: string
  }
  engine: {
    up: boolean
    version: string
    observed_at: string
    request_duration_ms: number
    loads: Array<Record<string, unknown>>
    running_requests?: number
    waiting_requests?: number
    token_usage?: number
    throughput?: number
    cache_usage?: number
    error_code?: string
  }
  machine: {
    up: boolean
    sampled_at: string
    window_complete: boolean
    gpu: {
      available: boolean
      count: number
      power_total_watts?: number
      driver_version: string
      devices: TelemetryGPUDevice[]
    }
    system: {
      available: boolean
      cpu_utilization_percent?: number
      cpu_count?: number
      memory_used_bytes?: number
      memory_available_bytes?: number
      memory_total_bytes?: number
      memory_utilization_percent?: number
      load_average: Record<string, number | undefined>
    }
  }
  alignment: {
    method: string
    quality: string
    skew_ms?: number
  }
  metrics: {
    requests?: number
    requests_semantics?: 'cumulative' | 'current_inflight' | 'unknown'
    tokens?: number
    tokens_semantics?: 'cumulative' | 'current_usage' | 'unknown'
  }
}

export type Cluster = {
  id: number
  model_id: number
  model_name: string
  model_available: boolean
  name: string
  enabled: boolean
  health_status: ClusterHealthStatus
  credential_status: ClusterCredentialStatus
  credential_version: number
  credential_issued_at: number
  credential_verified_at: number
  has_link_secret: boolean
  last_polled_at: number
  last_success_at: number
  consecutive_failures: number
  last_error_code?: string
  created_at: number
  updated_at: number
  telemetry?: NormalizedTelemetry
}

export type ModelClusterSummary = {
  model_id: number
  model_name: string
  icon?: string
  type: string
  model_available: boolean
  health_status: ClusterHealthStatus
  cluster_count: number
  online_count: number
  abnormal_count: number
  total_requests: number
  total_tokens: number
  requests_available: boolean
  tokens_available: boolean
  gpu_utilization?: number
}

export type ClusterOverview = {
  overview: {
    total_clusters: number
    online_clusters: number
    abnormal_clusters: number
    total_requests: number
    total_tokens: number
    requests_available: boolean
    tokens_available: boolean
  }
  model_groups: Array<{
    type: string
    models: ModelClusterSummary[]
  }>
  alerts: Array<{
    cluster_id: number
    cluster_name: string
    model_name: string
    health_status: ClusterHealthStatus
    error_code?: string
    last_polled_at: number
  }>
  pagination: {
    page: number
    page_size: number
    total: number
  }
}

export type ModelDetail = {
  model: ModelOption
  summary: ModelClusterSummary
  clusters: Cluster[]
}

export type ClusterOverviewParams = {
  search?: string
  model_id?: number
  status?: ClusterHealthStatus
  p?: number
  page_size?: number
}

export type ClusterExportScope = 'models' | 'clusters' | 'cluster' | 'all'

export type ClusterExportFormat = 'csv' | 'zip' | 'json'

export type ClusterExportParams = {
  scope: ClusterExportScope
  format: ClusterExportFormat
  search?: string
  model_id?: number
  cluster_id?: number
  status?: ClusterHealthStatus
}

export type CreateClusterPayload = {
  model_id: number
  name: string
  agent_address: string
}

export type CredentialIssueResponse = {
  cluster: Cluster
  bootstrap_token: string
}

export type CredentialVerificationResponse = {
  verified: boolean
  error_code?: string
  cluster: Cluster
}
