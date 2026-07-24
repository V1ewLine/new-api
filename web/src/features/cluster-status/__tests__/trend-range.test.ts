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
  resolveTelemetryTrendRange,
  telemetryTrendRangeKey,
  telemetryTrendRangeParams,
  validateTelemetryTrendRange,
} from '../lib/trend-range.ts'

describe('cluster telemetry trend ranges', () => {
  test('relative ranges move with the current time while keeping a stable query key', () => {
    const range = { kind: 'relative' as const, minutes: 60 }
    const first = resolveTelemetryTrendRange(range, 1_700_000_000_999)
    const second = resolveTelemetryTrendRange(range, 1_700_000_005_999)

    assert.equal(first.end.getMilliseconds(), 0)
    assert.equal(first.end.getTime() - first.start.getTime(), 3600000)
    assert.equal(second.end.getTime() - first.end.getTime(), 5000)
    assert.deepEqual(telemetryTrendRangeKey(range), ['relative', 60])
  })

  test('custom ranges preserve exact seconds in API parameters and query keys', () => {
    const range = {
      kind: 'custom' as const,
      start: new Date('2026-07-25T00:39:47+08:00'),
      end: new Date('2026-07-25T00:42:13+08:00'),
    }

    assert.deepEqual(telemetryTrendRangeParams(range, 1440), {
      start_at: '2026-07-24T16:39:47Z',
      end_at: '2026-07-24T16:42:13Z',
      max_points: 1440,
    })
    assert.deepEqual(telemetryTrendRangeKey(range), [
      'custom',
      range.start.getTime(),
      range.end.getTime(),
    ])
  })

  test('rejects reversed and over-retention custom ranges', () => {
    assert.equal(
      validateTelemetryTrendRange(
        {
          kind: 'custom',
          start: new Date('2026-07-25T01:00:00Z'),
          end: new Date('2026-07-25T00:00:00Z'),
        },
        7
      ),
      'invalid_order'
    )
    assert.equal(
      validateTelemetryTrendRange(
        {
          kind: 'custom',
          start: new Date('2026-07-01T00:00:00Z'),
          end: new Date('2026-07-25T00:00:00Z'),
        },
        7
      ),
      'exceeds_retention'
    )
  })
})
