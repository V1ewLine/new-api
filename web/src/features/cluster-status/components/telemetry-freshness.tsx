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

import { formatTimestamp } from '../lib/format'
import { getTelemetryFreshness } from '../lib/freshness'

type TelemetryFreshnessProps = {
  lastSuccessAt: number | undefined
  lastPolledAt?: number
  refreshInterval: number
  showLastAttempt?: boolean
}

export function TelemetryFreshness(props: TelemetryFreshnessProps) {
  const { t } = useTranslation()
  const freshness = getTelemetryFreshness(
    props.lastSuccessAt,
    props.refreshInterval
  )
  let badge = (
    <Badge variant='secondary'>{t('No successful sample')}</Badge>
  )
  if (freshness === 'fresh') {
    badge = <Badge className='bg-success/10 text-success'>{t('Fresh')}</Badge>
  } else if (freshness === 'stale') {
    badge = <Badge variant='destructive'>{t('Stale')}</Badge>
  }

  return (
    <div className='flex min-w-36 flex-col items-start gap-1'>
      {badge}
      <span className='text-muted-foreground text-xs'>
        {t('Last success: {{time}}', {
          time: formatTimestamp(props.lastSuccessAt),
        })}
      </span>
      {props.showLastAttempt ? (
        <span className='text-muted-foreground text-xs'>
          {t('Last attempt: {{time}}', {
            time: formatTimestamp(props.lastPolledAt),
          })}
        </span>
      ) : null}
    </div>
  )
}
