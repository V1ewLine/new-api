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

import { clusterExportDialogLayout } from '../lib/export-dialog-layout.ts'

describe('cluster export dialog layout', () => {
  test('keeps long content readable in a wider scroll-safe dialog', () => {
    assert.match(clusterExportDialogLayout.content, /sm:max-w-2xl/)
    assert.match(
      clusterExportDialogLayout.content,
      /max-h-\[calc\(100dvh-2rem\)\]/
    )
    assert.match(clusterExportDialogLayout.content, /overflow-y-auto/)
  })

  test('stacks date-time pickers like the dashboard filter dialog', () => {
    assert.match(clusterExportDialogLayout.rangeGrid, /grid/)
    assert.match(clusterExportDialogLayout.rangeGrid, /gap-2\.5/)
    assert.doesNotMatch(clusterExportDialogLayout.rangeGrid, /grid-cols-2/)
  })

  test('separates timezone and history availability into distinct lines', () => {
    assert.match(clusterExportDialogLayout.rangeMeta, /flex-col/)
    assert.match(clusterExportDialogLayout.rangeMeta, /gap-1/)
  })
})
