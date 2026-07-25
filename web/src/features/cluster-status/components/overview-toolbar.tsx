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
  Add01Icon,
  Download04Icon,
  Refresh01Icon,
  SearchIcon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
} from '@/components/ui/input-group'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Spinner } from '@/components/ui/spinner'

import type { ClusterHealthStatus, ModelOption } from '../types'

type OverviewToolbarProps = {
  search: string
  onSearchChange: (value: string) => void
  modelId: number
  onModelIdChange: (value: number) => void
  status: ClusterHealthStatus | ''
  onStatusChange: (value: ClusterHealthStatus | '') => void
  modelOptions: ModelOption[]
  onExport: () => void
  onAddCluster: () => void
  onRefresh: () => void
  refreshing: boolean
}

export function OverviewToolbar(props: OverviewToolbarProps) {
  const { t } = useTranslation()

  return (
    <div className='flex flex-wrap items-center gap-2'>
      <InputGroup className='w-full sm:w-56'>
        <InputGroupAddon>
          <HugeiconsIcon icon={SearchIcon} strokeWidth={2} />
        </InputGroupAddon>
        <InputGroupInput
          value={props.search}
          onChange={(event) => props.onSearchChange(event.target.value)}
          placeholder={t('Search clusters or models')}
          aria-label={t('Search clusters or models')}
        />
      </InputGroup>

      <NativeSelect
        value={String(props.modelId)}
        onChange={(event) => props.onModelIdChange(Number(event.target.value))}
        aria-label={t('Filter by model')}
      >
        <NativeSelectOption value='0'>{t('All Models')}</NativeSelectOption>
        {props.modelOptions.map((option) => (
          <NativeSelectOption key={option.id} value={option.id}>
            {option.name}
          </NativeSelectOption>
        ))}
      </NativeSelect>

      <NativeSelect
        value={props.status}
        onChange={(event) =>
          props.onStatusChange(event.target.value as ClusterHealthStatus | '')
        }
        aria-label={t('Filter by status')}
      >
        <NativeSelectOption value=''>{t('All Statuses')}</NativeSelectOption>
        <NativeSelectOption value='online'>{t('Online')}</NativeSelectOption>
        <NativeSelectOption value='partial'>
          {t('Partially abnormal')}
        </NativeSelectOption>
        <NativeSelectOption value='abnormal'>
          {t('Abnormal')}
        </NativeSelectOption>
        <NativeSelectOption value='offline'>{t('Offline')}</NativeSelectOption>
        <NativeSelectOption value='unknown'>{t('Unknown')}</NativeSelectOption>
      </NativeSelect>

      <Button variant='outline' onClick={props.onExport}>
        <HugeiconsIcon
          icon={Download04Icon}
          strokeWidth={2}
          data-icon='inline-start'
        />
        {t('Export')}
      </Button>

      <Button
        variant='outline'
        onClick={props.onRefresh}
        disabled={props.refreshing}
      >
        {props.refreshing ? (
          <Spinner data-icon='inline-start' />
        ) : (
          <HugeiconsIcon
            icon={Refresh01Icon}
            strokeWidth={2}
            data-icon='inline-start'
          />
        )}
        {props.refreshing ? t('Refreshing...') : t('Refresh')}
      </Button>

      <Button onClick={props.onAddCluster}>
        <HugeiconsIcon
          icon={Add01Icon}
          strokeWidth={2}
          data-icon='inline-start'
        />
        {t('Add Cluster')}
      </Button>
    </div>
  )
}
