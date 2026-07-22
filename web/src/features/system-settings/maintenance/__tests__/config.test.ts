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

import { SIDEBAR_MODULES_DEFAULT, parseSidebarModulesAdmin } from '../config.ts'

describe('admin sidebar module configuration', () => {
  test('enables cluster status by default', () => {
    assert.equal(SIDEBAR_MODULES_DEFAULT.admin.cluster_status, true)
  })

  test('adds cluster status to legacy saved configurations', () => {
    const config = parseSidebarModulesAdmin(
      JSON.stringify({
        admin: {
          enabled: true,
          channel: true,
          models: true,
        },
      })
    )

    assert.equal(config.admin.cluster_status, true)
  })

  test('preserves an explicitly disabled cluster status module', () => {
    const config = parseSidebarModulesAdmin(
      JSON.stringify({
        admin: {
          enabled: true,
          cluster_status: false,
        },
      })
    )

    assert.equal(config.admin.cluster_status, false)
  })
})
