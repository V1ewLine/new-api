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

import { normalizeAgentAddress } from '../lib/connection.ts'

describe('cluster Agent address normalization', () => {
  test('adds HTTP when an IP and port are entered without a scheme', () => {
    assert.equal(normalizeAgentAddress('10.0.0.8:9100'), 'http://10.0.0.8:9100')
  })

  test('preserves HTTPS and supports bracketed IPv6 addresses', () => {
    assert.equal(
      normalizeAgentAddress('https://agent.example:9443/'),
      'https://agent.example:9443'
    )
    assert.equal(normalizeAgentAddress('[::1]:9100'), 'http://[::1]:9100')
  })

  test('rejects addresses without a port or with unsafe URL components', () => {
    const invalidAddresses = [
      '10.0.0.8',
      'https://user:pass@agent.example:9443',
      'https://agent.example:9443/internal',
      'https://agent.example:9443?token=leak',
      'ftp://agent.example:21',
      'agent.example:99999',
    ]

    for (const address of invalidAddresses) {
      assert.equal(normalizeAgentAddress(address), null)
    }
  })
})
