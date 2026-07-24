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
import {
  Alert02Icon,
  CheckmarkCircle02Icon,
  Copy01Icon,
  InformationCircleIcon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from '@/components/ui/input-group'

import {
  agentTokenEnvLine,
  credentialVerificationOutcome,
} from '../lib/credential'
import type {
  CredentialIssueResponse,
  CredentialVerificationResponse,
} from '../types'

type AgentCredentialPanelProps = {
  issue: CredentialIssueResponse
  verification?: CredentialVerificationResponse
}

function CredentialVerificationAlert(props: {
  verification?: CredentialVerificationResponse
}) {
  const { t } = useTranslation()
  if (!props.verification) {
    return (
      <Alert>
        <HugeiconsIcon icon={InformationCircleIcon} strokeWidth={2} />
        <AlertTitle>{t('Waiting for Agent configuration')}</AlertTitle>
        <AlertDescription>
          {t(
            'Configure and restart the Agent, then test the connection from this window.'
          )}
        </AlertDescription>
      </Alert>
    )
  }
  if (props.verification.verified) {
    return (
      <Alert>
        <HugeiconsIcon icon={CheckmarkCircle02Icon} strokeWidth={2} />
        <AlertTitle>{t('Agent connected successfully')}</AlertTitle>
        <AlertDescription>
          {t('The credential is active and telemetry is available.')}
        </AlertDescription>
      </Alert>
    )
  }

  const outcome = credentialVerificationOutcome(props.verification.error_code)
  let description = t(
    'The connection test failed. Check the Agent configuration and try again.'
  )
  if (outcome === 'unauthorized') {
    description = t(
      'The Agent rejected the Token. Confirm the environment variable and restart the Agent.'
    )
  } else if (outcome === 'timeout') {
    description = t(
      'The Agent connection timed out. Check the address, port, and network.'
    )
  } else if (outcome === 'unreachable') {
    description = t(
      'New API cannot reach the Agent. Check the address, port, firewall, and service status.'
    )
  } else if (outcome === 'blocked') {
    description = t(
      'The Agent address is blocked by the outbound request policy.'
    )
  } else if (outcome === 'schema') {
    description = t(
      'The Agent responded, but its telemetry schema is not supported.'
    )
  } else if (outcome === 'invalid-secret') {
    description = t(
      'The stored credential is unavailable. Generate a new Token or recreate the cluster.'
    )
  }
  return (
    <Alert variant='destructive'>
      <HugeiconsIcon icon={Alert02Icon} strokeWidth={2} />
      <AlertTitle>{t('Agent connection test failed')}</AlertTitle>
      <AlertDescription>{description}</AlertDescription>
    </Alert>
  )
}

export function AgentCredentialPanel(props: AgentCredentialPanelProps) {
  const { t } = useTranslation()
  const [copiedValue, setCopiedValue] = useState<'token' | 'env'>()
  const envLine = agentTokenEnvLine(props.issue.bootstrap_token)

  async function copyValue(value: string, type: 'token' | 'env') {
    try {
      await navigator.clipboard.writeText(value)
      setCopiedValue(type)
      toast.success(t('Copied to clipboard'))
    } catch {
      toast.error(t('Failed to copy to clipboard'))
    }
  }

  return (
    <div className='flex flex-col gap-4'>
      <Alert>
        <HugeiconsIcon icon={Alert02Icon} strokeWidth={2} />
        <AlertTitle>{t('This Token is shown only once')}</AlertTitle>
        <AlertDescription>
          {t(
            'Copy it before closing this window. New API will not return the plaintext Token again.'
          )}
        </AlertDescription>
      </Alert>

      <FieldGroup>
        <Field>
          <FieldLabel htmlFor='cluster-generated-token'>
            {t('Agent Bearer Token')}
          </FieldLabel>
          <InputGroup>
            <InputGroupInput
              id='cluster-generated-token'
              className='font-mono'
              value={props.issue.bootstrap_token}
              readOnly
              autoComplete='off'
            />
            <InputGroupAddon align='inline-end'>
              <InputGroupButton
                onClick={() => copyValue(props.issue.bootstrap_token, 'token')}
              >
                <HugeiconsIcon
                  icon={
                    copiedValue === 'token' ? CheckmarkCircle02Icon : Copy01Icon
                  }
                  strokeWidth={2}
                  data-icon='inline-start'
                />
                {copiedValue === 'token' ? t('Copied') : t('Copy Token')}
              </InputGroupButton>
            </InputGroupAddon>
          </InputGroup>
        </Field>

        <Field>
          <FieldLabel htmlFor='cluster-agent-env'>
            {t('Agent environment variable')}
          </FieldLabel>
          <InputGroup>
            <InputGroupInput
              id='cluster-agent-env'
              className='font-mono'
              value={envLine}
              readOnly
              autoComplete='off'
            />
            <InputGroupAddon align='inline-end'>
              <InputGroupButton onClick={() => copyValue(envLine, 'env')}>
                <HugeiconsIcon
                  icon={
                    copiedValue === 'env' ? CheckmarkCircle02Icon : Copy01Icon
                  }
                  strokeWidth={2}
                  data-icon='inline-start'
                />
                {copiedValue === 'env' ? t('Copied') : t('Copy configuration')}
              </InputGroupButton>
            </InputGroupAddon>
          </InputGroup>
          <FieldDescription>
            {t(
              'Set this value as AGENT_API_TOKEN in the Agent .env file or service environment.'
            )}
          </FieldDescription>
        </Field>
      </FieldGroup>

      <ol className='text-muted-foreground list-decimal space-y-1 pl-5 text-sm'>
        <li>{t('Connect to the server running the Agent.')}</li>
        <li>{t('Update AGENT_API_TOKEN with the generated value.')}</li>
        <li>{t('Restart the Agent service to load the new Token.')}</li>
        <li>{t('Return here and test the connection.')}</li>
      </ol>

      <CredentialVerificationAlert verification={props.verification} />
    </div>
  )
}
