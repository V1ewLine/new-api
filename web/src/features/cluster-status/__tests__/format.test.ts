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
  averageGPUUtilization,
  formatBytes,
  formatCompactNumber,
  formatPercent,
  formatWatts,
  getVisiblePages,
  healthPriority,
} from '../lib/format.ts'
import type { NormalizedTelemetry } from '../types.ts'

describe('cluster status formatting', () => {
  test('preserves unavailable values instead of fabricating zero metrics', () => {
    assert.equal(formatCompactNumber(undefined), '—')
    assert.equal(formatPercent(undefined), '—')
    assert.equal(formatBytes(undefined), '—')
    assert.equal(formatWatts(undefined), '—')
  })

  test('formats reported metric values with compact units', () => {
    assert.equal(formatPercent(72.345), '72.3%')
    assert.equal(formatBytes(1_073_741_824), '1.0 GB')
    assert.equal(formatWatts(510), '510.0 W')
    assert.notEqual(formatCompactNumber(12_345), '—')
  })

  test('averages every reported GPU instead of assuming a fixed GPU count', () => {
    const telemetry = {
      machine: {
        gpu: {
          devices: [
            { utilization_percent: 10 },
            { utilization_percent: 30 },
            { utilization_percent: undefined },
            { utilization_percent: 80 },
          ],
        },
      },
    } as NormalizedTelemetry

    assert.equal(averageGPUUtilization(telemetry), 40)
  })
})

describe('cluster status pagination and severity', () => {
  test('keeps the active page inside a bounded five-page window', () => {
    assert.deepEqual(getVisiblePages(1, 12), [1, 2, 3, 4, 5])
    assert.deepEqual(getVisiblePages(7, 12), [5, 6, 7, 8, 9])
    assert.deepEqual(getVisiblePages(12, 12), [8, 9, 10, 11, 12])
  })

  test('orders unhealthy states ahead of healthy states for aggregation', () => {
    assert.ok(healthPriority('offline') > healthPriority('abnormal'))
    assert.ok(healthPriority('abnormal') > healthPriority('partial'))
    assert.ok(healthPriority('partial') > healthPriority('online'))
  })
})
