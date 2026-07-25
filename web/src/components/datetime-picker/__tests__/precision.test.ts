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
  applyDateTimePickerTime,
  dateTimePickerTimeValue,
} from '../../lib/datetime-picker.ts'

describe('date-time picker precision', () => {
  test('keeps seconds when a second-precision time is applied', () => {
    const date = new Date(2026, 6, 25, 0, 0, 0)

    const result = applyDateTimePickerTime(date, '12:34:56')

    assert.equal(result.getHours(), 12)
    assert.equal(result.getMinutes(), 34)
    assert.equal(result.getSeconds(), 56)
    assert.equal(result.getMilliseconds(), 0)
  })

  test('formats existing values without changing minute-precision defaults', () => {
    const date = new Date(2026, 6, 25, 12, 34, 56)

    assert.equal(dateTimePickerTimeValue(date, 'minute'), '12:34')
    assert.equal(dateTimePickerTimeValue(date, 'second'), '12:34:56')
  })
})
