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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  DEFAULT_CLUSTER_STATUS_REFRESH_INTERVAL_MS,
  clusterStatusRefreshIntervalMs,
} from '../lib/refresh-interval.ts'

describe('cluster status refresh interval', () => {
  test('converts valid second values to milliseconds', () => {
    assert.equal(clusterStatusRefreshIntervalMs(1), 1000)
    assert.equal(clusterStatusRefreshIntervalMs(5), 5000)
    assert.equal(clusterStatusRefreshIntervalMs(300), 300000)
  })

  test('falls back to five seconds for invalid values', () => {
    for (const value of [undefined, null, '5', 0, 301, 1.5, Number.NaN]) {
      assert.equal(
        clusterStatusRefreshIntervalMs(value),
        DEFAULT_CLUSTER_STATUS_REFRESH_INTERVAL_MS
      )
    }
  })
})
