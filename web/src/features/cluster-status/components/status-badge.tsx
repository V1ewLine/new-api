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
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'

import { clusterDisplayStatus } from '../lib/credential'
import type { ClusterCredentialStatus, ClusterHealthStatus } from '../types'

type ClusterStatusBadgeProps = {
  status: ClusterHealthStatus
  credentialStatus?: ClusterCredentialStatus
}

export function ClusterStatusBadge(props: ClusterStatusBadgeProps) {
  const { t } = useTranslation()
  const status = props.credentialStatus
    ? clusterDisplayStatus(props.status, props.credentialStatus)
    : props.status

  if (status === 'pending') {
    return <Badge variant='outline'>{t('Awaiting configuration')}</Badge>
  }
  if (status === 'online') {
    return <Badge variant='default'>{t('Online')}</Badge>
  }
  if (status === 'partial') {
    return <Badge variant='outline'>{t('Partially abnormal')}</Badge>
  }
  if (status === 'abnormal') {
    return <Badge variant='destructive'>{t('Abnormal')}</Badge>
  }
  if (status === 'offline') {
    return <Badge variant='destructive'>{t('Offline')}</Badge>
  }
  return <Badge variant='secondary'>{t('Unknown')}</Badge>
}
