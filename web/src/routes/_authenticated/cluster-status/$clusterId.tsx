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
import { createFileRoute, redirect } from '@tanstack/react-router'
import z from 'zod'

import { ClusterDetail } from '@/features/cluster-status/cluster-detail'
import {
  CLUSTER_DETAIL_TABS,
  TELEMETRY_TREND_ROUTE_RANGES,
} from '@/features/cluster-status/lib/route-state'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

const clusterDetailSearchSchema = z.object({
  tab: z.enum(CLUSTER_DETAIL_TABS).optional().catch('overview'),
  range: z.enum(TELEMETRY_TREND_ROUTE_RANGES).optional().catch('1h'),
  start: z.string().optional().catch(undefined),
  end: z.string().optional().catch(undefined),
})

export const Route = createFileRoute(
  '/_authenticated/cluster-status/$clusterId'
)({
  beforeLoad: () => {
    const { auth } = useAuthStore.getState()
    if (!auth.user || auth.user.role < ROLE.ADMIN) {
      throw redirect({ to: '/403' })
    }
  },
  validateSearch: clusterDetailSearchSchema,
  component: ClusterDetailRoute,
})

function ClusterDetailRoute() {
  const params = Route.useParams()
  return <ClusterDetail clusterId={Number(params.clusterId)} />
}
