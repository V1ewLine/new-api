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
import { InformationCircleIcon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

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
import { useDebounce } from '@/hooks/use-debounce'

import { getClusterModelOptions, getClusterOverview } from './api'
import { AddClusterDialog } from './components/add-cluster-dialog'
import { OverviewContent } from './components/overview-content'
import { OverviewToolbar } from './components/overview-toolbar'
import { useClusterRefreshInterval } from './hooks/use-cluster-refresh-interval'
import { clusterQueryKeys } from './query-keys'
import type { ClusterHealthStatus } from './types'

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
  const [search, setSearch] = useState('')
  const debouncedSearch = useDebounce(search, 300)
  const [modelId, setModelId] = useState(0)
  const [status, setStatus] = useState<ClusterHealthStatus | ''>('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [addDialogOpen, setAddDialogOpen] = useState(false)
  const refreshInterval = useClusterRefreshInterval()
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

  function updateSearch(value: string) {
    setSearch(value)
    setPage(1)
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Cluster Status')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='flex flex-col gap-4'>
          <OverviewToolbar
            search={search}
            onSearchChange={updateSearch}
            modelId={modelId}
            onModelIdChange={(value) => {
              setModelId(value)
              setPage(1)
            }}
            status={status}
            onStatusChange={(value) => {
              setStatus(value)
              setPage(1)
            }}
            modelOptions={modelOptionsQuery.data ?? []}
            onAddCluster={() => setAddDialogOpen(true)}
            refreshing={overviewQuery.isFetching && !overviewQuery.isLoading}
          />

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
              onPageChange={setPage}
              onPageSizeChange={(value) => {
                setPageSize(value)
                setPage(1)
              }}
            />
          ) : null}
        </div>

        <AddClusterDialog
          open={addDialogOpen}
          onOpenChange={setAddDialogOpen}
        />
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
