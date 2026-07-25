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
import { SearchIcon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useTranslation } from 'react-i18next'

import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
} from '@/components/ui/input-group'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'

import type { ClusterHealthStatus, ModelOption } from '../types'

type OverviewToolbarProps = {
  search: string
  onSearchChange: (value: string) => void
  modelId: number
  onModelIdChange: (value: number) => void
  status: ClusterHealthStatus | ''
  onStatusChange: (value: ClusterHealthStatus | '') => void
  modelOptions: ModelOption[]
}

export function OverviewToolbar(props: OverviewToolbarProps) {
  const { t } = useTranslation()

  return (
    <div className='flex flex-col gap-2'>
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
          onChange={(event) =>
            props.onModelIdChange(Number(event.target.value))
          }
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
          <NativeSelectOption value='offline'>
            {t('Offline')}
          </NativeSelectOption>
          <NativeSelectOption value='unknown'>
            {t('Unknown')}
          </NativeSelectOption>
        </NativeSelect>
      </div>
      <p className='text-muted-foreground text-xs'>
        {t(
          'Search and filters apply only to the model list below; summary metrics and trends remain global.'
        )}
      </p>
    </div>
  )
}
