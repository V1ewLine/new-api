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

import { clusterFormSchema } from '../lib/cluster-form.ts'
import {
  agentTokenEnvLine,
  clusterDisplayStatus,
  credentialVerificationOutcome,
} from '../lib/credential.ts'

describe('cluster credential setup flow', () => {
  test('cluster creation accepts model, name, and Agent address without a user-provided Token', () => {
    const result = clusterFormSchema.safeParse({
      modelId: 7,
      name: 'GLM cluster',
      agentAddress: '10.0.0.8:31000',
    })

    assert.equal(result.success, true)
  })

  test('generated Token is formatted as the Agent environment variable', () => {
    assert.equal(
      agentTokenEnvLine('napi_agent_example'),
      "AGENT_API_TOKEN='napi_agent_example'"
    )
  })

  test('pending credential status takes precedence over operational health', () => {
    assert.equal(clusterDisplayStatus('offline', 'pending'), 'pending')
    assert.equal(clusterDisplayStatus('online', 'active'), 'online')
  })

  test('verification errors map to actionable setup outcomes', () => {
    assert.equal(
      credentialVerificationOutcome('AGENT_HTTP_401'),
      'unauthorized'
    )
    assert.equal(
      credentialVerificationOutcome('AGENT_HTTP_403'),
      'unauthorized'
    )
    assert.equal(
      credentialVerificationOutcome('AGENT_UNREACHABLE'),
      'unreachable'
    )
    assert.equal(
      credentialVerificationOutcome('AGENT_SCHEMA_UNSUPPORTED'),
      'schema'
    )
  })
})
