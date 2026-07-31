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
  alignModelExportRange,
  getModelExportFilename,
} from '../lib/model-export.ts'

describe('model analytics export range', () => {
  test('aligns hourly exports to complete natural-hour buckets', () => {
    const result = alignModelExportRange(
      new Date(2026, 6, 25, 8, 15, 20),
      new Date(2026, 6, 25, 10, 0, 0),
      'hour'
    )

    assert.equal(result.error, null)
    assert.deepEqual(result.start, new Date(2026, 6, 25, 8, 0, 0))
    assert.deepEqual(result.end, new Date(2026, 6, 25, 10, 0, 0))
  })

  test('aligns weekly exports to Monday and the next exclusive boundary', () => {
    const result = alignModelExportRange(
      new Date(2026, 6, 29, 8, 0, 0),
      new Date(2026, 7, 4, 12, 0, 0),
      'week'
    )

    assert.equal(result.error, null)
    assert.deepEqual(result.start, new Date(2026, 6, 27, 0, 0, 0))
    assert.deepEqual(result.end, new Date(2026, 7, 10, 0, 0, 0))
  })

  test('rejects reversed and oversized hourly ranges', () => {
    assert.equal(
      alignModelExportRange(
        new Date('2026-07-25T10:00:00Z'),
        new Date('2026-07-25T09:00:00Z'),
        'hour'
      ).error,
      'invalid_order'
    )
    assert.equal(
      alignModelExportRange(
        new Date('2026-01-01T00:00:00Z'),
        new Date('2026-07-25T00:00:00Z'),
        'hour'
      ).error,
      'range_too_large'
    )
  })
})

describe('model analytics export filename', () => {
  test('parses quoted and UTF-8 Content-Disposition filenames', () => {
    assert.equal(
      getModelExportFilename(
        'attachment; filename="new-api-model-analytics.csv"'
      ),
      'new-api-model-analytics.csv'
    )
    assert.equal(
      getModelExportFilename(
        "attachment; filename*=UTF-8''new-api-%E6%A8%A1%E5%9E%8B.csv"
      ),
      'new-api-模型.csv'
    )
  })

  test('uses a CSV fallback for malformed headers', () => {
    assert.equal(
      getModelExportFilename("attachment; filename*=UTF-8''%E0%A4%A"),
      'new-api-model-analytics.csv'
    )
  })
})
