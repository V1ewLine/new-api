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

import { clusterQueryKeys } from '../query-keys.ts'

describe('cluster deletion query invalidation', () => {
  test('overview and model detail keys remain under their deletion refresh scopes', () => {
    const overviewScope = clusterQueryKeys.overviews()
    const modelScope = clusterQueryKeys.models()
    const overviewKey = clusterQueryKeys.overview({ p: 1, page_size: 10 })
    const modelKey = clusterQueryKeys.model(42)

    assert.deepEqual(overviewKey.slice(0, overviewScope.length), overviewScope)
    assert.deepEqual(modelKey.slice(0, modelScope.length), modelScope)
  })

  test('single-cluster detail stays outside list refresh scopes before redirect', () => {
    const clusterKey = clusterQueryKeys.cluster(7)

    assert.notDeepEqual(
      clusterKey.slice(0, clusterQueryKeys.overviews().length),
      clusterQueryKeys.overviews()
    )
    assert.notDeepEqual(
      clusterKey.slice(0, clusterQueryKeys.models().length),
      clusterQueryKeys.models()
    )
  })
})
