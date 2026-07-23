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
  ArrowRight01Icon,
  InformationCircleIcon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

import { getClusterModelDetail } from './api'
import { ModelAvatar } from './components/model-avatar'
import { ClusterStatusBadge } from './components/status-badge'
import { formatCompactNumber, formatTimestamp, formatWatts } from './lib/format'
import { clusterQueryKeys } from './query-keys'

type ClusterModelDetailProps = {
  modelId: number
}

function MetricCard(props: {
  title: string
  value: string
  description: string
}) {
  return (
    <Card size='sm'>
      <CardHeader>
        <CardDescription>{props.title}</CardDescription>
        <CardTitle className='text-2xl'>{props.value}</CardTitle>
      </CardHeader>
      <CardContent className='text-muted-foreground text-xs'>
        {props.description}
      </CardContent>
    </Card>
  )
}

export function ClusterModelDetail(props: ClusterModelDetailProps) {
  const { t } = useTranslation()
  const detailQuery = useQuery({
    queryKey: clusterQueryKeys.model(props.modelId),
    queryFn: async () => {
      const response = await getClusterModelDetail(props.modelId)
      if (!response.success || !response.data) {
        throw new Error(response.message || t('Failed to load model clusters'))
      }
      return response.data
    },
    refetchInterval: 5000,
  })

  const detail = detailQuery.data
  return (
    <SectionPageLayout>
      <SectionPageLayout.Breadcrumb>
        <Button
          variant='link'
          size='sm'
          nativeButton={false}
          render={<Link to='/cluster-status' />}
        >
          {t('Cluster Status')}
        </Button>
      </SectionPageLayout.Breadcrumb>
      <SectionPageLayout.Title>
        <div className='flex items-center gap-2'>
          {detail ? (
            <ModelAvatar
              icon={detail.model.icon}
              name={detail.model.name}
              size={32}
            />
          ) : null}
          <span>{detail?.model.name ?? t('Model Clusters')}</span>
        </div>
      </SectionPageLayout.Title>
      <SectionPageLayout.Content>
        {detailQuery.isLoading ? (
          <div className='flex flex-col gap-4'>
            <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-5'>
              {Array.from({ length: 5 }, (_, index) => (
                <Skeleton key={index} className='h-28 w-full rounded-xl' />
              ))}
            </div>
            <Skeleton className='h-80 w-full rounded-xl' />
          </div>
        ) : null}

        {detailQuery.isError ? (
          <Card>
            <CardContent>
              <Empty>
                <EmptyHeader>
                  <EmptyMedia variant='icon'>
                    <HugeiconsIcon
                      icon={InformationCircleIcon}
                      strokeWidth={2}
                    />
                  </EmptyMedia>
                  <EmptyTitle>{t('Failed to load model clusters')}</EmptyTitle>
                  <EmptyDescription>
                    {detailQuery.error instanceof Error
                      ? detailQuery.error.message
                      : t('Please try again later.')}
                  </EmptyDescription>
                </EmptyHeader>
              </Empty>
            </CardContent>
          </Card>
        ) : null}

        {detail ? (
          <div className='flex flex-col gap-5'>
            <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-5'>
              <MetricCard
                title={t('Total Clusters')}
                value={String(detail.summary.cluster_count)}
                description={t('Clusters linked to this model')}
              />
              <MetricCard
                title={t('Online Clusters')}
                value={String(detail.summary.online_count)}
                description={t('Healthy and reporting')}
              />
              <MetricCard
                title={t('Abnormal Clusters')}
                value={String(detail.summary.abnormal_count)}
                description={t('Require attention')}
              />
              <MetricCard
                title={t('Total Requests')}
                value={formatCompactNumber(
                  detail.summary.requests_available
                    ? detail.summary.total_requests
                    : undefined
                )}
                description={t('Reported by connected Agents')}
              />
              <MetricCard
                title={t('Total Tokens')}
                value={formatCompactNumber(
                  detail.summary.tokens_available
                    ? detail.summary.total_tokens
                    : undefined
                )}
                description={t('Reported by connected Agents')}
              />
            </div>

            <Card>
              <CardHeader>
                <CardTitle>{t('Cluster List')}</CardTitle>
                <CardDescription>
                  {t('Latest status for every cluster linked to this model')}
                </CardDescription>
              </CardHeader>
              <CardContent className='overflow-x-auto'>
                {detail.clusters.length === 0 ? (
                  <Empty>
                    <EmptyHeader>
                      <EmptyTitle>{t('No clusters found')}</EmptyTitle>
                      <EmptyDescription>
                        {t('Add a cluster from the cluster status page.')}
                      </EmptyDescription>
                    </EmptyHeader>
                  </Empty>
                ) : (
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{t('Cluster Name')}</TableHead>
                        <TableHead>{t('Status')}</TableHead>
                        <TableHead>{t('Requests')}</TableHead>
                        <TableHead>{t('Tokens')}</TableHead>
                        <TableHead>{t('GPU Board Power')}</TableHead>
                        <TableHead>{t('Last Poll')}</TableHead>
                        <TableHead className='text-right'>
                          {t('Actions')}
                        </TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {detail.clusters.map((cluster) => (
                        <TableRow key={cluster.id}>
                          <TableCell className='font-medium'>
                            {cluster.name}
                          </TableCell>
                          <TableCell>
                            <ClusterStatusBadge
                              status={cluster.health_status}
                            />
                          </TableCell>
                          <TableCell>
                            {formatCompactNumber(
                              cluster.telemetry?.metrics.requests
                            )}
                          </TableCell>
                          <TableCell>
                            {formatCompactNumber(
                              cluster.telemetry?.metrics.tokens
                            )}
                          </TableCell>
                          <TableCell>
                            {formatWatts(
                              cluster.telemetry?.machine.gpu.power_total_watts
                            )}
                          </TableCell>
                          <TableCell>
                            {formatTimestamp(cluster.last_polled_at)}
                          </TableCell>
                          <TableCell className='text-right'>
                            <Button
                              size='sm'
                              variant='ghost'
                              nativeButton={false}
                              render={
                                <Link
                                  to='/cluster-status/$clusterId'
                                  params={{
                                    clusterId: String(cluster.id),
                                  }}
                                />
                              }
                            >
                              {t('View Details')}
                              <HugeiconsIcon
                                icon={ArrowRight01Icon}
                                strokeWidth={2}
                                data-icon='inline-end'
                              />
                            </Button>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                )}
              </CardContent>
            </Card>
          </div>
        ) : null}
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
