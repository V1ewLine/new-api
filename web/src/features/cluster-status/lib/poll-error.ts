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
export type ClusterPollErrorPresentation = {
  title: string
  description: string
  retryable: boolean
}

export function getClusterPollErrorPresentation(
  errorCode: string | undefined
): ClusterPollErrorPresentation {
  if (errorCode === 'AGENT_TIMEOUT') {
    return {
      title: 'Agent request timed out',
      description:
        'New API did not receive a response in time. Check Agent load and network latency, then retry.',
      retryable: true,
    }
  }
  if (errorCode === 'AGENT_UNREACHABLE') {
    return {
      title: 'Agent is unreachable',
      description:
        'Check the Agent address, port, firewall, and service status, then retry.',
      retryable: true,
    }
  }
  if (errorCode === 'AGENT_HTTP_401' || errorCode === 'AGENT_HTTP_403') {
    return {
      title: 'Agent rejected the Token',
      description:
        'Generate a new Agent Token, update the remote Agent, and test the connection again.',
      retryable: false,
    }
  }
  if (errorCode === 'AGENT_ADDRESS_BLOCKED') {
    return {
      title: 'Agent address is blocked',
      description:
        'The outbound request policy blocks this address. Review the cluster Agent allowlist settings.',
      retryable: false,
    }
  }
  if (errorCode?.startsWith('AGENT_SCHEMA_')) {
    return {
      title: 'Unsupported telemetry schema',
      description:
        'The Agent response is incompatible with this New API version. Check the Agent version and schema.',
      retryable: false,
    }
  }
  if (
    errorCode === 'CLUSTER_SECRET_INVALID' ||
    errorCode === 'CLUSTER_LINK_INVALID'
  ) {
    return {
      title: 'Cluster credential is unavailable',
      description:
        'Generate a new Agent Token and update the remote Agent configuration.',
      retryable: false,
    }
  }
  return {
    title: 'Latest telemetry poll failed',
    description:
      'Open cluster details to review the failure, or retry the collection now.',
    retryable: true,
  }
}
