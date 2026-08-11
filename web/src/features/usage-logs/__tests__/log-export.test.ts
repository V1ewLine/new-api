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
  buildUsageLogExportParams,
  getUsageLogExportFilename,
  hasValidUsageLogExportRange,
} from '../lib/log-export.ts'

describe('usage log export parameters', () => {
  test('uses the visible time range and keeps administrator filters', () => {
    const params = buildUsageLogExportParams(
      {
        startTime: 1_786_410_000_000,
        endTime: 1_786_413_600_000,
        type: ['2'],
        model: 'deepseek-v4-flash',
        username: 'alice',
        channel: '3',
      },
      true,
      'Asia/Shanghai'
    )

    assert.equal(params.start_timestamp, 1_786_410_000)
    assert.equal(params.end_timestamp, 1_786_413_600)
    assert.equal(params.type, 2)
    assert.equal(params.model_name, 'deepseek-v4-flash')
    assert.equal(params.username, 'alice')
    assert.equal(params.channel, 3)
    assert.equal(params.timezone, 'Asia/Shanghai')
    assert.equal(hasValidUsageLogExportRange(params), true)
  })

  test('drops administrator-only filters from a self export', () => {
    const params = buildUsageLogExportParams(
      {
        startTime: 1_786_410_000_000,
        endTime: 1_786_413_600_000,
        username: 'another-user',
        channel: '9',
      },
      false,
      'UTC'
    )

    assert.equal(params.username, undefined)
    assert.equal(params.channel, undefined)
  })
})

describe('usage log export filename', () => {
  test('parses quoted and UTF-8 filenames', () => {
    assert.equal(
      getUsageLogExportFilename(
        'attachment; filename="new-api-usage-logs.csv"'
      ),
      'new-api-usage-logs.csv'
    )
    assert.equal(
      getUsageLogExportFilename(
        "attachment; filename*=UTF-8''new-api-%E6%97%A5%E5%BF%97.csv"
      ),
      'new-api-日志.csv'
    )
  })
})
