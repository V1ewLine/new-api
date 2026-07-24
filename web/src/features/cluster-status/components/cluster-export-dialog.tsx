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
import { useQuery } from '@tanstack/react-query'
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
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldTitle,
} from '@/components/ui/field'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Spinner } from '@/components/ui/spinner'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'

import {
  downloadClusterExport,
  downloadClusterHistoryExport,
  getClusterStatusSettings,
} from '../api'
import {
  buildClusterExportParams,
  buildClusterHistoryExportParams,
  saveExportFile,
  type ClusterExportContext,
  validateClusterHistoryRange,
} from '../lib/export'
import { clusterQueryKeys } from '../query-keys'
import type {
  ClusterExportFormat,
  ClusterExportScope,
  ClusterHealthStatus,
} from '../types'
import { ClusterHistoryRangeFields } from './cluster-history-range-fields'

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

type ExportSource = 'latest' | 'history'

function defaultHistoryRange(availableFrom?: number) {
  const end = new Date(Math.floor(Date.now() / 1000) * 1000)
  const requestedStart = end.getTime() - 60 * 60 * 1000
  const availableStart = availableFrom ? availableFrom * 1000 : requestedStart
  return {
    start: new Date(
      Math.min(Math.max(requestedStart, availableStart), end.getTime() - 1000)
    ),
    end,
  }
}

export function ClusterExportDialog(props: ClusterExportDialogProps) {
  const { t } = useTranslation()
  const selectId = useId()
  const [source, setSource] = useState<ExportSource>('latest')
  const [selection, setSelection] = useState(0)
  const [exporting, setExporting] = useState(false)
  const [historyRange, setHistoryRange] = useState(defaultHistoryRange)
  const settingsQuery = useQuery({
    queryKey: clusterQueryKeys.settings(),
    queryFn: async () => {
      const response = await getClusterStatusSettings()
      if (!response.success || !response.data) {
        throw new Error(response.message || 'Failed to load cluster settings')
      }
      return response.data
    },
    staleTime: Number.POSITIVE_INFINITY,
  })

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
    setExporting(true)
    try {
      let result: { blob: Blob; filename: string }
      if (source === 'history') {
        const retentionDays = settingsQuery.data?.retention_days ?? 7
        const rangeError = validateClusterHistoryRange(
          historyRange.start,
          historyRange.end,
          retentionDays,
          settingsQuery.data?.history_available_from
        )
        if (rangeError === 'invalid_order') {
          throw new Error(t('Start time must be earlier than end time'))
        }
        if (rangeError === 'exceeds_retention') {
          throw new Error(
            t('The selected range exceeds the {{days}}-day retention period', {
              days: retentionDays,
            })
          )
        }
        if (rangeError === 'before_available') {
          throw new Error(t('The start time is earlier than available history'))
        }
        result = await downloadClusterHistoryExport(
          buildClusterHistoryExportParams({
            context: props.context,
            start: historyRange.start,
            end: historyRange.end,
            search: props.search,
            modelId: props.modelId,
            clusterId: props.clusterId,
            status: props.status,
          })
        )
      } else {
        if (!selected) return
        result = await downloadClusterExport(
          buildClusterExportParams({
            context: props.context,
            scope: selected.scope,
            format: selected.format,
            search: props.search,
            modelId: props.modelId,
            clusterId: props.clusterId,
            status: props.status,
          })
        )
      }
      saveExportFile(result.blob, result.filename)
      toast.success(t('Cluster data exported'))
      closeDialog()
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

  function closeDialog() {
    setSource('latest')
    setSelection(0)
    setHistoryRange(
      defaultHistoryRange(settingsQuery.data?.history_available_from)
    )
    props.onOpenChange(false)
  }

  function handleOpenChange(open: boolean) {
    if (open) {
      props.onOpenChange(true)
      return
    }
    closeDialog()
  }

  return (
    <Dialog open={props.open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('Export Cluster Data')}</DialogTitle>
          <DialogDescription>
            {source === 'latest'
              ? t(
                  'Exports the latest stored snapshot. Agent addresses, Bearer Tokens, and diagnostic payloads are excluded.'
                )
              : t(
                  'Exports stored samples in the selected time window. Agent addresses, Bearer Tokens, raw responses, and diagnostic payloads are excluded.'
                )}
          </DialogDescription>
        </DialogHeader>

        <FieldGroup>
          <Field>
            <FieldTitle id='cluster-export-source'>
              {t('Data source')}
            </FieldTitle>
            <ToggleGroup
              aria-labelledby='cluster-export-source'
              value={[source]}
              onValueChange={(value) => {
                const next = value.find((item) => item !== source)
                if (next === 'latest' || next === 'history') {
                  setSource(next)
                  if (next === 'history') {
                    setHistoryRange(
                      defaultHistoryRange(
                        settingsQuery.data?.history_available_from
                      )
                    )
                  }
                }
              }}
              variant='outline'
              spacing={2}
              className='grid w-full grid-cols-2'
              disabled={exporting}
            >
              <ToggleGroupItem value='latest' className='w-full'>
                {t('Latest snapshot')}
              </ToggleGroupItem>
              <ToggleGroupItem value='history' className='w-full'>
                {t('Time range')}
              </ToggleGroupItem>
            </ToggleGroup>
          </Field>

          {source === 'latest' ? (
            <Field>
              <FieldTitle>
                <label htmlFor={selectId}>{t('Export content')}</label>
              </FieldTitle>
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
              <FieldDescription>{selected?.description}</FieldDescription>
            </Field>
          ) : (
            <Field>
              <FieldTitle>{t('Export time window')}</FieldTitle>
              <ClusterHistoryRangeFields
                start={historyRange.start}
                end={historyRange.end}
                onChange={setHistoryRange}
                availableFrom={settingsQuery.data?.history_available_from}
                disabled={exporting}
              />
              <FieldDescription>
                {t(
                  'The start time is included and the end time is excluded. The ZIP contains telemetry, GPU, engine load, and normalized JSONL history.'
                )}
              </FieldDescription>
            </Field>
          )}
        </FieldGroup>

        <DialogFooter>
          <Button variant='outline' onClick={closeDialog} disabled={exporting}>
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
