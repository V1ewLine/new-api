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
import { Key01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Spinner } from '@/components/ui/spinner'

import { rotateClusterCredential, verifyClusterCredential } from '../api'
import { clusterQueryKeys } from '../query-keys'
import type {
  Cluster,
  CredentialIssueResponse,
  CredentialVerificationResponse,
} from '../types'
import { AgentCredentialPanel } from './agent-credential-panel'

type RotateClusterCredentialDialogProps = {
  cluster: Cluster
}

export function RotateClusterCredentialDialog(
  props: RotateClusterCredentialDialogProps
) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [credentialOpen, setCredentialOpen] = useState(false)
  const [closeConfirmationOpen, setCloseConfirmationOpen] = useState(false)
  const [issuedCredential, setIssuedCredential] =
    useState<CredentialIssueResponse>()
  const [verification, setVerification] =
    useState<CredentialVerificationResponse>()

  const rotateMutation = useMutation({
    mutationFn: async () => {
      const response = await rotateClusterCredential(props.cluster.id)
      if (!response.success || !response.data) {
        throw new Error(response.message || t('Failed to generate a new Token'))
      }
      return response.data
    },
    onSuccess: async (response) => {
      setIssuedCredential(response)
      setVerification(undefined)
      setConfirmOpen(false)
      setCredentialOpen(true)
      rotateMutation.reset()
      await queryClient.invalidateQueries({ queryKey: clusterQueryKeys.all })
      toast.success(t('A new Agent Token has been generated'))
    },
    onError: (error) => {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to generate a new Token')
      )
    },
  })
  const verifyMutation = useMutation({
    mutationFn: async () => {
      const response = await verifyClusterCredential(props.cluster.id)
      if (!response.success || !response.data) {
        throw new Error(
          response.message || t('Failed to test Agent connection')
        )
      }
      return response.data
    },
    onSuccess: async (response) => {
      setVerification(response)
      setIssuedCredential((current) =>
        current ? { ...current, cluster: response.cluster } : current
      )
      await queryClient.invalidateQueries({ queryKey: clusterQueryKeys.all })
      if (response.verified) {
        toast.success(t('Agent connected successfully'))
      }
    },
    onError: (error) => {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to test Agent connection')
      )
    },
  })

  function closeCredentialDialog() {
    setCredentialOpen(false)
    setCloseConfirmationOpen(false)
    setIssuedCredential(undefined)
    setVerification(undefined)
    rotateMutation.reset()
    verifyMutation.reset()
  }

  function requestCredentialOpenChange(open: boolean) {
    if (open) {
      setCredentialOpen(true)
      return
    }
    if (issuedCredential && !verification?.verified) {
      setCloseConfirmationOpen(true)
      return
    }
    closeCredentialDialog()
  }

  let primaryLabel = t('Test Connection')
  if (verifyMutation.isPending) {
    primaryLabel = t('Testing connection...')
  } else if (verification?.verified) {
    primaryLabel = t('Done')
  }

  return (
    <>
      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogTrigger
          render={
            <Button
              variant={
                props.cluster.credential_status === 'pending'
                  ? 'default'
                  : 'outline'
              }
            />
          }
        >
          <HugeiconsIcon
            icon={Key01Icon}
            strokeWidth={2}
            data-icon='inline-start'
          />
          {props.cluster.credential_status === 'pending'
            ? t('Generate New Token')
            : t('Rotate Token')}
        </AlertDialogTrigger>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {t('Generate a new Agent Token?')}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                'The current Token will stop working immediately. Update AGENT_API_TOKEN on the remote Agent and restart it.'
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={rotateMutation.isPending}>
              {t('Cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              variant='destructive'
              onClick={() => rotateMutation.mutate()}
              disabled={rotateMutation.isPending}
            >
              {rotateMutation.isPending ? (
                <Spinner data-icon='inline-start' />
              ) : null}
              {rotateMutation.isPending
                ? t('Generating...')
                : t('Generate Token')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <Dialog open={credentialOpen} onOpenChange={requestCredentialOpenChange}>
        <DialogContent className='sm:max-w-2xl' showCloseButton={false}>
          <DialogHeader>
            <DialogTitle>{t('Configure Agent')}</DialogTitle>
            <DialogDescription>
              {t(
                'Install the generated credential on the remote Agent and verify the connection.'
              )}
            </DialogDescription>
          </DialogHeader>
          {issuedCredential ? (
            <AgentCredentialPanel
              issue={issuedCredential}
              verification={verification}
            />
          ) : null}
          <DialogFooter>
            <Button
              type='button'
              variant='outline'
              onClick={() =>
                verification?.verified
                  ? closeCredentialDialog()
                  : setCloseConfirmationOpen(true)
              }
              disabled={verifyMutation.isPending}
            >
              {verification?.verified ? t('Close') : t('Configure later')}
            </Button>
            <Button
              type='button'
              onClick={() =>
                verification?.verified
                  ? closeCredentialDialog()
                  : verifyMutation.mutate()
              }
              disabled={verifyMutation.isPending}
            >
              {verifyMutation.isPending ? (
                <Spinner data-icon='inline-start' />
              ) : null}
              {primaryLabel}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={closeConfirmationOpen}
        onOpenChange={setCloseConfirmationOpen}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {t('Close without verifying the Agent?')}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                'The cluster will remain in Awaiting configuration status. This Token cannot be shown again; generate a new one if you lose it.'
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('Continue configuring')}</AlertDialogCancel>
            <AlertDialogAction onClick={closeCredentialDialog}>
              {t('Configure later')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
