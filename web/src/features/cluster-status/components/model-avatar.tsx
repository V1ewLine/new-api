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
import { getLobeIcon } from '@/lib/lobe-icon'

type ModelAvatarProps = {
  icon?: string
  name: string
  size?: number
}

export function ModelAvatar(props: ModelAvatarProps) {
  const iconKey = props.icon || props.name[0] || 'N'
  const size = props.size ?? 28
  return (
    <div
      className='flex shrink-0 items-center justify-center overflow-hidden rounded-lg'
      style={{ width: size, height: size }}
      aria-hidden='true'
    >
      {getLobeIcon(`${iconKey.split('.')[0]}.Avatar.type={'platform'}`, size)}
    </div>
  )
}
