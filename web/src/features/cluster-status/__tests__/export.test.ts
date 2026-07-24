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

import { buildClusterExportParams, getExportFilename } from '../lib/export.ts'

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
