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
export const DEFAULT_CLUSTER_STATUS_REFRESH_INTERVAL_MS = 5000

const MIN_CLUSTER_STATUS_REFRESH_INTERVAL_SECONDS = 1
const MAX_CLUSTER_STATUS_REFRESH_INTERVAL_SECONDS = 300

export function clusterStatusRefreshIntervalMs(seconds: unknown): number {
  if (
    typeof seconds !== 'number' ||
    !Number.isInteger(seconds) ||
    seconds < MIN_CLUSTER_STATUS_REFRESH_INTERVAL_SECONDS ||
    seconds > MAX_CLUSTER_STATUS_REFRESH_INTERVAL_SECONDS
  ) {
    return DEFAULT_CLUSTER_STATUS_REFRESH_INTERVAL_MS
  }

  return seconds * 1000
}
