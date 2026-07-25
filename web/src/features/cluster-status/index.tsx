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
  InformationCircleIcon,
  Refresh01Icon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { getRouteApi } from '@tanstack/react-router'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import { useDebounce } from '@/hooks/use-debounce'

import {
  getClusterModelOptions,
  getClusterOverview,
  refreshCluster,
} from './api'
import { AddClusterDialog } from './components/add-cluster-dialog'
import { AggregateLoadTrends } from './components/aggregate-load-trends'
import { ClusterExportDialog } from './components/cluster-export-dialog'
import { OverviewContent } from './components/overview-content'
import { OverviewToolbar } from './components/overview-toolbar'
import { useClusterRefreshInterval } from './hooks/use-cluster-refresh-interval'
import { useTelemetryTrends } from './hooks/use-cluster-telemetry-trends'
import {
  trendRangeFromRouteSearch,
  trendRangeToRouteSearch,
} from './lib/route-state'
import { clusterQueryKeys } from './query-keys'
import type { ClusterHealthStatus } from './types'

const route = getRouteApi('/_authenticated/cluster-status/')

function ClusterOverviewSkeleton() {
  return (
    <div className='flex flex-col gap-5' aria-busy='true'>
      <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-5'>
        {Array.from({ length: 5 }, (_, index) => (
          <Card key={index} size='sm'>
            <CardHeader>
              <Skeleton className='h-3 w-24' />
              <Skeleton className='h-8 w-20' />
            </CardHeader>
            <CardContent>
              <Skeleton className='h-3 w-32' />
            </CardContent>
          </Card>
        ))}
      </div>
      <div className='grid gap-3 md:grid-cols-2 xl:grid-cols-3'>
        {Array.from({ length: 6 }, (_, index) => (
          <Card key={index}>
            <CardHeader>
              <Skeleton className='h-6 w-36' />
              <Skeleton className='h-4 w-24' />
            </CardHeader>
            <CardContent>
              <Skeleton className='h-24 w-full' />
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}

export function ClusterStatus() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const routeSearch = route.useSearch()
  const navigate = route.useNavigate()
  const search = routeSearch.q ?? ''
  const debouncedSearch = useDebounce(search, 300)
  const modelId = routeSearch.model ?? 0
  const status = (routeSearch.status ?? '') as ClusterHealthStatus | ''
  const page = routeSearch.page ?? 1
  const pageSize = routeSearch.pageSize ?? 10
  const [addDialogOpen, setAddDialogOpen] = useState(false)
  const [exportDialogOpen, setExportDialogOpen] = useState(false)
  const refreshInterval = useClusterRefreshInterval()
  const trendRange = trendRangeFromRouteSearch(routeSearch)
  const params = {
    search: debouncedSearch || undefined,
    model_id: modelId || undefined,
    status: status || undefined,
    p: page,
    page_size: pageSize,
  }
  const overviewQuery = useQuery({
    queryKey: clusterQueryKeys.overview(params),
    queryFn: async () => {
      const response = await getClusterOverview(params)
      if (!response.success || !response.data) {
        throw new Error(response.message || t('Failed to load cluster status'))
      }
      return response.data
    },
    refetchInterval: refreshInterval,
  })
  const modelOptionsQuery = useQuery({
    queryKey: clusterQueryKeys.modelOptions(),
    queryFn: async () => {
      const response = await getClusterModelOptions()
      if (!response.success) {
        throw new Error(response.message || t('Failed to load model options'))
      }
      return response.data ?? []
    },
  })
  const trendQuery = useTelemetryTrends({
    scope: { kind: 'overview' },
    range: trendRange,
    refreshInterval,
  })
  const refreshMutation = useMutation({
    mutationFn: async (clusterId: number) => {
      const response = await refreshCluster(clusterId)
      if (!response.success || !response.data) {
        throw new Error(response.message || t('Failed to refresh cluster'))
      }
      return response.data
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: clusterQueryKeys.all })
      toast.success(t('Cluster refreshed successfully'))
    },
    onError: (error) => {
      toast.error(
        error instanceof Error ? error.message : t('Failed to refresh cluster')
      )
    },
  })

  function updateSearch(value: string) {
    navigate({
      replace: true,
      search: (previous) => ({
        ...previous,
        q: value || undefined,
        page: undefined,
      }),
    })
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Cluster Status')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button variant='outline' onClick={() => setExportDialogOpen(true)}>
          <HugeiconsIcon
            icon={Download04Icon}
            strokeWidth={2}
            data-icon='inline-start'
          />
          {t('Export')}
        </Button>
        <Button
          variant='outline'
          onClick={() => {
            void Promise.all([
              overviewQuery.refetch(),
              modelOptionsQuery.refetch(),
              trendQuery.refetch(),
            ])
          }}
          disabled={
            overviewQuery.isFetching ||
            modelOptionsQuery.isFetching ||
            trendQuery.isFetching
          }
        >
          {overviewQuery.isFetching ||
          modelOptionsQuery.isFetching ||
          trendQuery.isFetching ? (
            <Spinner data-icon='inline-start' />
          ) : (
            <HugeiconsIcon
              icon={Refresh01Icon}
              strokeWidth={2}
              data-icon='inline-start'
            />
          )}
          {overviewQuery.isFetching ||
          modelOptionsQuery.isFetching ||
          trendQuery.isFetching
            ? t('Refreshing...')
            : t('Refresh')}
        </Button>
        <Button onClick={() => setAddDialogOpen(true)}>
          <HugeiconsIcon
            icon={Add01Icon}
            strokeWidth={2}
            data-icon='inline-start'
          />
          {t('Add Cluster')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='flex flex-col gap-4'>
          {overviewQuery.isLoading ? <ClusterOverviewSkeleton /> : null}
          {overviewQuery.isError ? (
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
                    <EmptyTitle>
                      {t('Failed to load cluster status')}
                    </EmptyTitle>
                    <EmptyDescription>
                      {overviewQuery.error instanceof Error
                        ? overviewQuery.error.message
                        : t('Please try again later.')}
                    </EmptyDescription>
                  </EmptyHeader>
                  <EmptyContent>
                    <Button
                      variant='outline'
                      onClick={() => overviewQuery.refetch()}
                    >
                      {t('Retry')}
                    </Button>
                  </EmptyContent>
                </Empty>
              </CardContent>
            </Card>
          ) : null}
          {overviewQuery.data ? (
            <OverviewContent
              data={overviewQuery.data}
              refreshInterval={refreshInterval}
              retryingClusterId={
                refreshMutation.isPending
                  ? (refreshMutation.variables ?? null)
                  : null
              }
              onRetryCluster={(clusterId) => refreshMutation.mutate(clusterId)}
              onAddCluster={() => setAddDialogOpen(true)}
              onPageChange={(value) =>
                navigate({
                  search: (previous) => ({
                    ...previous,
                    page: value === 1 ? undefined : value,
                  }),
                })
              }
              onPageSizeChange={(value) => {
                navigate({
                  search: (previous) => ({
                    ...previous,
                    pageSize: value,
                    page: undefined,
                  }),
                })
              }}
              afterSummary={
                <AggregateLoadTrends
                  title={t('Current Load Trends')}
                  description={t(
                    'Global current request and token load over the selected time window.'
                  )}
                  scope={{ kind: 'overview' }}
                  range={trendRange}
                  onRangeChange={(value) => {
                    navigate({
                      replace: true,
                      search: (previous) => ({
                        ...previous,
                        ...trendRangeToRouteSearch(value),
                      }),
                    })
                  }}
                  refreshInterval={refreshInterval}
                  query={trendQuery}
                />
              }
              toolbar={
                <OverviewToolbar
                  search={search}
                  onSearchChange={updateSearch}
                  modelId={modelId}
                  onModelIdChange={(value) => {
                    navigate({
                      search: (previous) => ({
                        ...previous,
                        model: value || undefined,
                        page: undefined,
                      }),
                    })
                  }}
                  status={status}
                  onStatusChange={(value) => {
                    navigate({
                      search: (previous) => ({
                        ...previous,
                        status: value || undefined,
                        page: undefined,
                      }),
                    })
                  }}
                  modelOptions={modelOptionsQuery.data ?? []}
                />
              }
            />
          ) : null}
        </div>

        <AddClusterDialog
          open={addDialogOpen}
          onOpenChange={setAddDialogOpen}
        />
        <ClusterExportDialog
          open={exportDialogOpen}
          onOpenChange={setExportDialogOpen}
          context='overview'
          search={debouncedSearch || undefined}
          modelId={modelId || undefined}
          status={status || undefined}
        />
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
