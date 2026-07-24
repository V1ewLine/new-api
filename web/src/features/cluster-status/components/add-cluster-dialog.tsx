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
import { useEffect } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { z } from 'zod'

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

import { createCluster, getClusterModelOptions } from '../api'
import { normalizeAgentAddress } from '../lib/connection'
import { clusterQueryKeys } from '../query-keys'
import { ModelSelector } from './model-selector'

const clusterFormSchema = z.object({
  modelId: z.number().int().positive('Please select a model'),
  name: z
    .string()
    .trim()
    .min(1, 'Cluster name is required')
    .max(128, 'Cluster name must be 128 characters or fewer'),
  agentAddress: z
    .string()
    .trim()
    .min(1, 'Agent address is required')
    .max(2048, 'Agent address is too long')
    .refine(
      (value) => normalizeAgentAddress(value) !== null,
      'Agent address must include an IP or hostname and port'
    ),
  agentBearerToken: z
    .string()
    .trim()
    .min(1, 'Agent Bearer Token is required')
    .max(8192, 'Agent Bearer Token is too long'),
})

type ClusterFormValues = z.infer<typeof clusterFormSchema>

type AddClusterDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function AddClusterDialog(props: AddClusterDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const form = useForm<ClusterFormValues>({
    resolver: zodResolver(clusterFormSchema),
    defaultValues: {
      modelId: 0,
      name: '',
      agentAddress: '',
      agentBearerToken: '',
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
    enabled: props.open,
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
        agent_bearer_token: values.agentBearerToken.trim(),
      })
      if (!response.success) {
        throw new Error(response.message || t('Failed to add cluster'))
      }
      return response.data
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: clusterQueryKeys.all })
      toast.success(t('Cluster added successfully'))
      props.onOpenChange(false)
    },
    onError: (error) => {
      toast.error(
        error instanceof Error ? error.message : t('Failed to add cluster')
      )
    },
  })

  useEffect(() => {
    if (!props.open) {
      form.reset()
    }
  }, [form, props.open])

  let modelErrorMessage: string | undefined
  if (form.formState.errors.modelId?.message) {
    modelErrorMessage = t(form.formState.errors.modelId.message)
  } else if (optionsQuery.isError) {
    modelErrorMessage = t('Failed to load model options')
  }

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='sm:max-w-md'>
        <DialogHeader>
          <DialogTitle>{t('Add Cluster')}</DialogTitle>
          <DialogDescription>
            {t('Connect an SGLang telemetry Agent to an enabled model.')}
          </DialogDescription>
        </DialogHeader>

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

            <Field data-invalid={Boolean(form.formState.errors.agentAddress)}>
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
              <FieldError>
                {form.formState.errors.agentAddress?.message
                  ? t(form.formState.errors.agentAddress.message)
                  : undefined}
              </FieldError>
            </Field>

            <Field
              data-invalid={Boolean(form.formState.errors.agentBearerToken)}
            >
              <FieldLabel htmlFor='cluster-agent-bearer-token'>
                {t('Agent Bearer Token')}
              </FieldLabel>
              <Input
                id='cluster-agent-bearer-token'
                type='password'
                autoComplete='new-password'
                placeholder={t('Enter the Agent Bearer Token')}
                aria-invalid={Boolean(form.formState.errors.agentBearerToken)}
                disabled={createMutation.isPending}
                {...form.register('agentBearerToken')}
              />
              <FieldDescription>
                {t(
                  'The Agent Bearer Token is encrypted after submission and is never shown again.'
                )}
              </FieldDescription>
              <FieldError>
                {form.formState.errors.agentBearerToken?.message
                  ? t(form.formState.errors.agentBearerToken.message)
                  : undefined}
              </FieldError>
            </Field>
          </FieldGroup>
        </form>

        <DialogFooter>
          <Button
            type='button'
            variant='outline'
            onClick={() => props.onOpenChange(false)}
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
            {createMutation.isPending ? t('Adding...') : t('Add Cluster')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
