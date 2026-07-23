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
import { ArrowDown01Icon, Tick02Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { cn } from '@/lib/utils'

import type { ModelOption } from '../types'
import { ModelAvatar } from './model-avatar'

type ModelSelectorProps = {
  id: string
  value: number
  options: ModelOption[]
  loading: boolean
  disabled: boolean
  invalid: boolean
  onValueChange: (value: number) => void
}

export function ModelSelector(props: ModelSelectorProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const selected = props.options.find((option) => option.id === props.value)

  function selectModel(modelId: number) {
    props.onValueChange(modelId)
    setOpen(false)
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button
            id={props.id}
            type='button'
            variant='outline'
            role='combobox'
            aria-expanded={open}
            aria-invalid={props.invalid}
            disabled={props.disabled}
            className='w-full justify-between font-normal'
          />
        }
      >
        <span className='flex min-w-0 items-center gap-2'>
          {selected ? (
            <ModelAvatar icon={selected.icon} name={selected.name} size={20} />
          ) : null}
          <span
            className={cn('truncate', !selected && 'text-muted-foreground')}
          >
            {props.loading
              ? t('Loading models...')
              : selected?.name || t('Select an enabled model')}
          </span>
        </span>
        <HugeiconsIcon
          icon={ArrowDown01Icon}
          strokeWidth={2}
          className='text-muted-foreground'
        />
      </PopoverTrigger>
      <PopoverContent
        align='start'
        className='w-(--anchor-width) p-0'
        onWheel={(event) => event.stopPropagation()}
      >
        <Command>
          <CommandInput placeholder={t('Search models...')} />
          <CommandList>
            <CommandEmpty>{t('No models found.')}</CommandEmpty>
            <CommandGroup>
              {props.options.map((option) => (
                <CommandItem
                  key={option.id}
                  value={`${option.name} ${option.id}`}
                  disabled={!option.enabled}
                  onSelect={() => selectModel(option.id)}
                >
                  <ModelAvatar
                    icon={option.icon}
                    name={option.name}
                    size={20}
                  />
                  <span className='min-w-0 flex-1 truncate'>{option.name}</span>
                  <HugeiconsIcon
                    icon={Tick02Icon}
                    strokeWidth={2}
                    className={cn(
                      'shrink-0',
                      option.id === props.value ? 'opacity-100' : 'opacity-0'
                    )}
                  />
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  )
}
