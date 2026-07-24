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
import { Download04Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Spinner } from '@/components/ui/spinner'

import { downloadClusterExport } from '../api'
import {
  buildClusterExportParams,
  saveExportFile,
  type ClusterExportContext,
} from '../lib/export'
import type {
  ClusterExportFormat,
  ClusterExportScope,
  ClusterHealthStatus,
} from '../types'

type ExportOption = {
  scope: ClusterExportScope
  format: ClusterExportFormat
  label: string
  description: string
}

type ClusterExportDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  context: ClusterExportContext
  search?: string
  modelId?: number
  clusterId?: number
  status?: ClusterHealthStatus
}

export function ClusterExportDialog(props: ClusterExportDialogProps) {
  const { t } = useTranslation()
  const selectId = useId()
  const [selection, setSelection] = useState(0)
  const [exporting, setExporting] = useState(false)

  let options: ExportOption[]
  if (props.context === 'overview') {
    options = [
      {
        scope: 'models',
        format: 'csv',
        label: t('Model summary (CSV)'),
        description: t(
          'Exports one row per model using all records that match the current filters.'
        ),
      },
      {
        scope: 'all',
        format: 'zip',
        label: t('Complete snapshot (ZIP)'),
        description: t(
          'Includes model, cluster, GPU device, engine load, and normalized telemetry files.'
        ),
      },
    ]
  } else if (props.context === 'model') {
    options = [
      {
        scope: 'clusters',
        format: 'csv',
        label: t('Cluster list (CSV)'),
        description: t(
          'Exports one row for every cluster linked to this model.'
        ),
      },
    ]
  } else {
    options = [
      {
        scope: 'cluster',
        format: 'zip',
        label: t('Complete cluster snapshot (ZIP)'),
        description: t(
          'Includes the cluster row, GPU devices, engine loads, and normalized telemetry.'
        ),
      },
      {
        scope: 'cluster',
        format: 'json',
        label: t('Normalized snapshot (JSON)'),
        description: t(
          'Exports the complete normalized latest snapshot as a JSON file.'
        ),
      },
    ]
  }
  const selected = options[selection] ?? options[0]

  async function runExport() {
    if (!selected) return
    const params = buildClusterExportParams({
      context: props.context,
      scope: selected.scope,
      format: selected.format,
      search: props.search,
      modelId: props.modelId,
      clusterId: props.clusterId,
      status: props.status,
    })

    setExporting(true)
    try {
      const result = await downloadClusterExport(params)
      saveExportFile(result.blob, result.filename)
      toast.success(t('Cluster data exported'))
      props.onOpenChange(false)
    } catch (error) {
      const message =
        error instanceof Error && error.message !== 'Cluster export failed'
          ? error.message
          : t('Failed to export cluster data')
      toast.error(message)
    } finally {
      setExporting(false)
    }
  }

  function handleOpenChange(open: boolean) {
    if (open) {
      setSelection(0)
    }
    props.onOpenChange(open)
  }

  return (
    <Dialog open={props.open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('Export Cluster Data')}</DialogTitle>
          <DialogDescription>
            {t(
              'Exports the latest stored snapshot. Agent addresses, Bearer Tokens, and diagnostic payloads are excluded.'
            )}
          </DialogDescription>
        </DialogHeader>

        <div className='flex flex-col gap-2'>
          <label htmlFor={selectId} className='text-sm font-medium'>
            {t('Export content')}
          </label>
          <NativeSelect
            id={selectId}
            className='w-full'
            value={String(selection)}
            onChange={(event) => setSelection(Number(event.target.value))}
            disabled={exporting}
          >
            {options.map((option, index) => (
              <NativeSelectOption
                key={`${option.scope}-${option.format}`}
                value={index}
              >
                {option.label}
              </NativeSelectOption>
            ))}
          </NativeSelect>
          <p className='text-muted-foreground text-xs'>
            {selected?.description}
          </p>
        </div>

        <DialogFooter>
          <Button
            variant='outline'
            onClick={() => props.onOpenChange(false)}
            disabled={exporting}
          >
            {t('Cancel')}
          </Button>
          <Button onClick={runExport} disabled={exporting}>
            {exporting ? (
              <Spinner data-icon='inline-start' />
            ) : (
              <HugeiconsIcon
                icon={Download04Icon}
                strokeWidth={2}
                data-icon='inline-start'
              />
            )}
            {exporting ? t('Exporting...') : t('Export')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
