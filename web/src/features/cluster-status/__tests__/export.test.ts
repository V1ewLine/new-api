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
  buildClusterExportParams,
  buildClusterHistoryExportParams,
  getExportFilename,
  validateClusterHistoryRange,
} from '../lib/export.ts'

describe('cluster export filename parsing', () => {
  test('reads quoted and UTF-8 Content-Disposition filenames', () => {
    assert.equal(
      getExportFilename(
        'attachment; filename="cluster-status-models-20260724.csv"'
      ),
      'cluster-status-models-20260724.csv'
    )
    assert.equal(
      getExportFilename(
        "attachment; filename*=UTF-8''cluster-status-%E9%9B%86%E7%BE%A4.zip"
      ),
      'cluster-status-集群.zip'
    )
  })

  test('uses a stable fallback for malformed headers', () => {
    assert.equal(getExportFilename('attachment'), 'cluster-status-export')
    assert.equal(
      getExportFilename("attachment; filename*=UTF-8''%E0%A4%A"),
      'cluster-status-export'
    )
  })
})

describe('cluster export page granularity', () => {
  test('keeps overview filters and ignores pagination', () => {
    assert.deepEqual(
      buildClusterExportParams({
        context: 'overview',
        scope: 'all',
        format: 'zip',
        search: 'east',
        modelId: 12,
        status: 'online',
      }),
      {
        scope: 'all',
        format: 'zip',
        search: 'east',
        model_id: 12,
        status: 'online',
      }
    )
  })

  test('maps model and cluster detail exports to their resource IDs', () => {
    assert.deepEqual(
      buildClusterExportParams({
        context: 'model',
        scope: 'clusters',
        format: 'csv',
        modelId: 12,
      }),
      {
        scope: 'clusters',
        format: 'csv',
        model_id: 12,
      }
    )
    assert.deepEqual(
      buildClusterExportParams({
        context: 'cluster',
        scope: 'cluster',
        format: 'json',
        clusterId: 34,
      }),
      {
        scope: 'cluster',
        format: 'json',
        cluster_id: 34,
      }
    )
  })
})

describe('cluster history export time window', () => {
  test('preserves exact seconds and maps the current page scope', () => {
    assert.deepEqual(
      buildClusterHistoryExportParams({
        context: 'model',
        modelId: 12,
        start: new Date('2026-07-24T04:30:25.900Z'),
        end: new Date('2026-07-24T05:31:26.100Z'),
      }),
      {
        scope: 'all',
        model_id: 12,
        start_at: '2026-07-24T04:30:25Z',
        end_at: '2026-07-24T05:31:26Z',
      }
    )
    assert.deepEqual(
      buildClusterHistoryExportParams({
        context: 'cluster',
        clusterId: 34,
        start: new Date('2026-07-24T04:30:25Z'),
        end: new Date('2026-07-24T05:30:25Z'),
      }),
      {
        scope: 'cluster',
        cluster_id: 34,
        start_at: '2026-07-24T04:30:25Z',
        end_at: '2026-07-24T05:30:25Z',
      }
    )
  })

  test('rejects reversed, over-retention, and unavailable ranges', () => {
    const start = new Date('2026-07-20T00:00:00Z')
    const end = new Date('2026-07-25T00:00:00Z')

    assert.equal(validateClusterHistoryRange(end, start, 7), 'invalid_order')
    assert.equal(
      validateClusterHistoryRange(start, end, 3),
      'exceeds_retention'
    )
    assert.equal(
      validateClusterHistoryRange(
        start,
        end,
        7,
        new Date('2026-07-21T00:00:00Z').getTime() / 1000
      ),
      'before_available'
    )
    assert.equal(validateClusterHistoryRange(start, end, 7), null)
  })
})
