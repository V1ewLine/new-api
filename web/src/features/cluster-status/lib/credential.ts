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
  Cluster,
  ClusterCredentialStatus,
  ClusterHealthStatus,
} from '../types'

export type ClusterDisplayStatus = ClusterHealthStatus | 'pending'

export type CredentialVerificationOutcome =
  | 'unauthorized'
  | 'timeout'
  | 'unreachable'
  | 'blocked'
  | 'schema'
  | 'invalid-secret'
  | 'unknown'

export function agentTokenEnvLine(token: string): string {
  return `AGENT_API_TOKEN='${token}'`
}

export function clusterDisplayStatus(
  healthStatus: ClusterHealthStatus,
  credentialStatus: ClusterCredentialStatus
): ClusterDisplayStatus {
  if (credentialStatus === 'pending') {
    return 'pending'
  }
  return healthStatus
}

export function clusterDisplayStatusFromCluster(
  cluster: Pick<Cluster, 'health_status' | 'credential_status'>
): ClusterDisplayStatus {
  return clusterDisplayStatus(cluster.health_status, cluster.credential_status)
}

export function credentialVerificationOutcome(
  errorCode: string | undefined
): CredentialVerificationOutcome {
  if (errorCode === 'AGENT_HTTP_401' || errorCode === 'AGENT_HTTP_403') {
    return 'unauthorized'
  }
  if (errorCode === 'AGENT_TIMEOUT') return 'timeout'
  if (errorCode === 'AGENT_UNREACHABLE') return 'unreachable'
  if (errorCode === 'AGENT_ADDRESS_BLOCKED') return 'blocked'
  if (errorCode?.startsWith('AGENT_SCHEMA_')) return 'schema'
  if (
    errorCode === 'CLUSTER_SECRET_INVALID' ||
    errorCode === 'CLUSTER_LINK_INVALID'
  ) {
    return 'invalid-secret'
  }
  return 'unknown'
}
