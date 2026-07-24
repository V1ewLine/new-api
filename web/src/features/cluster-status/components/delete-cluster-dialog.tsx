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
import { Delete02Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
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
import { Spinner } from '@/components/ui/spinner'

import { deleteCluster } from '../api'
import { clusterQueryKeys } from '../query-keys'
import type { Cluster } from '../types'

type DeleteClusterDialogProps = {
  cluster: Pick<Cluster, 'id' | 'name'> | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onDeleted?: () => void
}

export function DeleteClusterDialog(props: DeleteClusterDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const deleteMutation = useMutation({
    mutationFn: async (clusterId: number) => {
      const response = await deleteCluster(clusterId)
      if (!response.success) {
        throw new Error(response.message || t('Delete failed'))
      }
    },
    onSuccess: async () => {
      props.onOpenChange(false)
      toast.success(t('Deleted successfully'))
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: clusterQueryKeys.overviews(),
        }),
        queryClient.invalidateQueries({
          queryKey: clusterQueryKeys.models(),
        }),
      ])
      props.onDeleted?.()
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Delete failed'))
    },
  })

  function handleOpenChange(open: boolean) {
    if (!deleteMutation.isPending) {
      props.onOpenChange(open)
    }
  }

  function handleDelete() {
    if (props.cluster) {
      deleteMutation.mutate(props.cluster.id)
    }
  }

  return (
    <AlertDialog open={props.open} onOpenChange={handleOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t('Confirm delete')}</AlertDialogTitle>
          <AlertDialogDescription>
            {t(
              'Delete cluster "{{name}}" and its stored telemetry? This action cannot be undone.',
              { name: props.cluster?.name ?? '' }
            )}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={deleteMutation.isPending}>
            {t('Cancel')}
          </AlertDialogCancel>
          <AlertDialogAction
            variant='destructive'
            disabled={deleteMutation.isPending || !props.cluster}
            onClick={handleDelete}
          >
            {deleteMutation.isPending ? (
              <Spinner data-icon='inline-start' />
            ) : (
              <HugeiconsIcon
                icon={Delete02Icon}
                strokeWidth={2}
                data-icon='inline-start'
              />
            )}
            {deleteMutation.isPending ? t('Deleting...') : t('Delete')}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
