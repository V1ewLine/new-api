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
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { Controller, useForm } from 'react-hook-form'
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
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Spinner } from '@/components/ui/spinner'

import {
  createCluster,
  getClusterModelOptions,
  verifyClusterCredential,
} from '../api'
import { clusterFormSchema, type ClusterFormValues } from '../lib/cluster-form'
import { normalizeAgentAddress } from '../lib/connection'
import { clusterQueryKeys } from '../query-keys'
import type {
  CredentialIssueResponse,
  CredentialVerificationResponse,
} from '../types'
import { AgentCredentialPanel } from './agent-credential-panel'
import { ModelSelector } from './model-selector'

type AddClusterDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function AddClusterDialog(props: AddClusterDialogProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [issuedCredential, setIssuedCredential] =
    useState<CredentialIssueResponse>()
  const [verification, setVerification] =
    useState<CredentialVerificationResponse>()
  const [closeConfirmationOpen, setCloseConfirmationOpen] = useState(false)
  const form = useForm<ClusterFormValues>({
    resolver: zodResolver(clusterFormSchema),
    defaultValues: {
      modelId: 0,
      name: '',
      agentAddress: '',
    },
  })
  const optionsQuery = useQuery({
    queryKey: clusterQueryKeys.modelOptions(),
    queryFn: async () => {
      const response = await getClusterModelOptions()
      if (!response.success) {
        throw new Error(response.message || t('Failed to load model options'))
      }
      return response.data ?? []
    },
    enabled: props.open && !issuedCredential,
  })
  const createMutation = useMutation({
    mutationFn: async (values: ClusterFormValues) => {
      const agentAddress = normalizeAgentAddress(values.agentAddress)
      if (!agentAddress) {
        throw new Error(
          t('Agent address must include an IP or hostname and port')
        )
      }
      const response = await createCluster({
        model_id: values.modelId,
        name: values.name.trim(),
        agent_address: agentAddress,
      })
      if (!response.success || !response.data) {
        throw new Error(response.message || t('Failed to add cluster'))
      }
      return response.data
    },
    onSuccess: async (response) => {
      setIssuedCredential(response)
      createMutation.reset()
      await queryClient.invalidateQueries({ queryKey: clusterQueryKeys.all })
      toast.success(t('Cluster created. Configure the Agent to continue.'))
    },
    onError: (error) => {
      toast.error(
        error instanceof Error ? error.message : t('Failed to add cluster')
      )
    },
  })
  const verifyMutation = useMutation({
    mutationFn: async () => {
      if (!issuedCredential) {
        throw new Error(t('No generated Agent credential is available'))
      }
      const response = await verifyClusterCredential(
        issuedCredential.cluster.id
      )
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
  const resetCreateMutation = createMutation.reset
  const resetVerifyMutation = verifyMutation.reset

  useEffect(() => {
    if (!props.open) {
      form.reset()
      setIssuedCredential(undefined)
      setVerification(undefined)
      setCloseConfirmationOpen(false)
      resetCreateMutation()
      resetVerifyMutation()
    }
  }, [form, props.open, resetCreateMutation, resetVerifyMutation])

  function closeDialog() {
    form.reset()
    setIssuedCredential(undefined)
    setVerification(undefined)
    setCloseConfirmationOpen(false)
    createMutation.reset()
    verifyMutation.reset()
    props.onOpenChange(false)
  }

  function requestDialogOpenChange(open: boolean) {
    if (open) {
      props.onOpenChange(true)
      return
    }
    if (issuedCredential && !verification?.verified) {
      setCloseConfirmationOpen(true)
      return
    }
    closeDialog()
  }

  function openClusterDetail() {
    if (!issuedCredential) {
      return
    }
    const clusterID = issuedCredential.cluster.id
    closeDialog()
    navigate({
      to: '/cluster-status/$clusterId',
      params: { clusterId: String(clusterID) },
    })
  }

  let modelErrorMessage: string | undefined
  if (form.formState.errors.modelId?.message) {
    modelErrorMessage = t(form.formState.errors.modelId.message)
  } else if (optionsQuery.isError) {
    modelErrorMessage = t('Failed to load model options')
  }
  let credentialPrimaryLabel = t('Test Connection')
  if (verifyMutation.isPending) {
    credentialPrimaryLabel = t('Testing connection...')
  } else if (verification?.verified) {
    credentialPrimaryLabel = t('View Cluster')
  }

  return (
    <>
      <Dialog open={props.open} onOpenChange={requestDialogOpenChange}>
        <DialogContent
          className={issuedCredential ? 'sm:max-w-2xl' : 'sm:max-w-md'}
          showCloseButton={!issuedCredential}
        >
          <DialogHeader>
            <DialogTitle>
              {issuedCredential ? t('Configure Agent') : t('Add Cluster')}
            </DialogTitle>
            <DialogDescription>
              {issuedCredential
                ? t(
                    'Install the generated credential on the remote Agent and verify the connection.'
                  )
                : t('Connect an SGLang telemetry Agent to an enabled model.')}
            </DialogDescription>
          </DialogHeader>

          {issuedCredential ? (
            <AgentCredentialPanel
              issue={issuedCredential}
              verification={verification}
            />
          ) : (
            <form
              id='add-cluster-form'
              onSubmit={form.handleSubmit((values) =>
                createMutation.mutate(values)
              )}
            >
              <FieldGroup>
                <Field data-invalid={Boolean(form.formState.errors.modelId)}>
                  <FieldLabel htmlFor='cluster-model'>{t('Model')}</FieldLabel>
                  <Controller
                    control={form.control}
                    name='modelId'
                    render={({ field }) => (
                      <ModelSelector
                        id='cluster-model'
                        value={field.value}
                        options={optionsQuery.data ?? []}
                        loading={optionsQuery.isLoading}
                        disabled={
                          optionsQuery.isLoading || createMutation.isPending
                        }
                        invalid={Boolean(form.formState.errors.modelId)}
                        onValueChange={field.onChange}
                      />
                    )}
                  />
                  <FieldError>{modelErrorMessage}</FieldError>
                </Field>

                <Field data-invalid={Boolean(form.formState.errors.name)}>
                  <FieldLabel htmlFor='cluster-name'>
                    {t('Cluster Name')}
                  </FieldLabel>
                  <Input
                    id='cluster-name'
                    placeholder={t('For example: East China A100 Cluster')}
                    aria-invalid={Boolean(form.formState.errors.name)}
                    disabled={createMutation.isPending}
                    {...form.register('name')}
                  />
                  <FieldError>
                    {form.formState.errors.name?.message
                      ? t(form.formState.errors.name.message)
                      : undefined}
                  </FieldError>
                </Field>

                <Field
                  data-invalid={Boolean(form.formState.errors.agentAddress)}
                >
                  <FieldLabel htmlFor='cluster-agent-address'>
                    {t('Agent IP and Port')}
                  </FieldLabel>
                  <Input
                    id='cluster-agent-address'
                    inputMode='url'
                    autoComplete='url'
                    placeholder='10.0.0.8:9100'
                    aria-invalid={Boolean(form.formState.errors.agentAddress)}
                    disabled={createMutation.isPending}
                    {...form.register('agentAddress')}
                  />
                  <FieldDescription>
                    {t(
                      'New API will generate the Agent Bearer Token after the cluster is created.'
                    )}
                  </FieldDescription>
                  <FieldError>
                    {form.formState.errors.agentAddress?.message
                      ? t(form.formState.errors.agentAddress.message)
                      : undefined}
                  </FieldError>
                </Field>
              </FieldGroup>
            </form>
          )}

          <DialogFooter>
            {issuedCredential ? (
              <>
                <Button
                  type='button'
                  variant='outline'
                  onClick={() =>
                    verification?.verified
                      ? closeDialog()
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
                      ? openClusterDetail()
                      : verifyMutation.mutate()
                  }
                  disabled={verifyMutation.isPending}
                >
                  {verifyMutation.isPending ? (
                    <Spinner data-icon='inline-start' />
                  ) : null}
                  {credentialPrimaryLabel}
                </Button>
              </>
            ) : (
              <>
                <Button
                  type='button'
                  variant='outline'
                  onClick={closeDialog}
                  disabled={createMutation.isPending}
                >
                  {t('Cancel')}
                </Button>
                <Button
                  type='submit'
                  form='add-cluster-form'
                  disabled={createMutation.isPending || optionsQuery.isLoading}
                >
                  {createMutation.isPending ? (
                    <Spinner data-icon='inline-start' />
                  ) : null}
                  {createMutation.isPending
                    ? t('Creating...')
                    : t('Create and Generate Token')}
                </Button>
              </>
            )}
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
            <AlertDialogAction onClick={closeDialog}>
              {t('Configure later')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
