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

import { getClusterPollErrorPresentation } from '../lib/poll-error.ts'

describe('cluster poll error presentation', () => {
  test('offers an immediate retry for transient Agent failures', () => {
    assert.equal(
      getClusterPollErrorPresentation('AGENT_TIMEOUT').retryable,
      true
    )
    assert.equal(
      getClusterPollErrorPresentation('AGENT_UNREACHABLE').retryable,
      true
    )
  })

  test('directs credential and compatibility errors to configuration', () => {
    assert.equal(
      getClusterPollErrorPresentation('AGENT_HTTP_401').retryable,
      false
    )
    assert.equal(
      getClusterPollErrorPresentation('AGENT_SCHEMA_UNSUPPORTED').retryable,
      false
    )
  })

  test('keeps unknown failures actionable', () => {
    const presentation = getClusterPollErrorPresentation('AGENT_UNKNOWN')

    assert.equal(presentation.title, 'Latest telemetry poll failed')
    assert.equal(presentation.retryable, true)
  })
})
