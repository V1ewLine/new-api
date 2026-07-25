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

import { getTelemetryFreshness } from '../lib/freshness.ts'

describe('cluster telemetry freshness', () => {
  test('reports missing until the first successful sample', () => {
    assert.equal(getTelemetryFreshness(0, 5_000, 100_000), 'missing')
  })

  test('allows three missed refreshes with a minimum 30-second window', () => {
    const now = 1_700_000_100_000

    assert.equal(getTelemetryFreshness(now / 1000 - 30, 5_000, now), 'fresh')
    assert.equal(getTelemetryFreshness(now / 1000 - 31, 5_000, now), 'stale')
    assert.equal(getTelemetryFreshness(now / 1000 - 60, 20_000, now), 'fresh')
    assert.equal(getTelemetryFreshness(now / 1000 - 61, 20_000, now), 'stale')
  })
})
