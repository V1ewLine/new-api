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
  Alert02Icon,
  ArrowRight01Icon,
  CheckmarkCircle02Icon,
  InformationCircleIcon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { Link } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Progress } from '@/components/ui/progress'
import { Spinner } from '@/components/ui/spinner'

import { formatCompactNumber, getVisiblePages } from '../lib/format'
import { getClusterPollErrorPresentation } from '../lib/poll-error'
import type { ClusterOverview, ModelClusterSummary } from '../types'
import { ModelAvatar } from './model-avatar'
import { ClusterStatusBadge } from './status-badge'
import { TelemetryFreshness } from './telemetry-freshness'

type OverviewContentProps = {
  data: ClusterOverview
  refreshInterval: number
  retryingClusterId: number | null
  onRetryCluster: (clusterId: number) => void
  onAddCluster: () => void
  onPageChange: (page: number) => void
  onPageSizeChange: (pageSize: number) => void
}

function SummaryCard(props: {
  title: string
  value: string
  description: string
  icon: ReactNode
}) {
  return (
    <Card size='sm'>
      <CardHeader>
        <CardTitle className='text-muted-foreground text-xs font-medium'>
          {props.title}
        </CardTitle>
        <CardAction>{props.icon}</CardAction>
        <CardDescription className='text-foreground text-2xl font-semibold'>
          {props.value}
        </CardDescription>
      </CardHeader>
      <CardContent className='text-muted-foreground text-xs'>
        {props.description}
      </CardContent>
    </Card>
  )
}

function ModelSummaryCard(props: { model: ModelClusterSummary }) {
  const { t } = useTranslation()
  const onlineRate =
    props.model.cluster_count === 0
      ? 0
      : (props.model.online_count / props.model.cluster_count) * 100
  return (
    <Card
      size='sm'
      className='[contain-intrinsic-size:auto_220px] [content-visibility:auto]'
    >
      <CardHeader>
        <div className='flex min-w-0 items-center gap-2'>
          <ModelAvatar icon={props.model.icon} name={props.model.model_name} />
          <div className='min-w-0'>
            <CardTitle className='truncate'>{props.model.model_name}</CardTitle>
            <CardDescription className='mt-1 flex items-center gap-2'>
              <Badge variant='secondary'>
                {groupTitle(props.model.type, t)}
              </Badge>
              <span>
                {t('{{count}} clusters', { count: props.model.cluster_count })}
              </span>
            </CardDescription>
          </div>
        </div>
        <CardAction>
          {props.model.model_available ? (
            <ClusterStatusBadge status={props.model.health_status} />
          ) : (
            <Badge variant='destructive'>{t('Model unavailable')}</Badge>
          )}
        </CardAction>
      </CardHeader>
      <CardContent className='grid grid-cols-2 gap-3 text-xs'>
        <div>
          <div className='text-muted-foreground'>{t('Online Clusters')}</div>
          <div className='mt-1 font-medium'>{props.model.online_count}</div>
        </div>
        <div>
          <div className='text-muted-foreground'>{t('Abnormal Clusters')}</div>
          <div className='mt-1 font-medium'>{props.model.abnormal_count}</div>
        </div>
        <div>
          <div className='text-muted-foreground'>{t('Requests')}</div>
          <div className='mt-1 font-medium'>
            {formatCompactNumber(
              props.model.requests_available
                ? props.model.total_requests
                : undefined
            )}
          </div>
        </div>
        <div>
          <div className='text-muted-foreground'>{t('Tokens')}</div>
          <div className='mt-1 font-medium'>
            {formatCompactNumber(
              props.model.tokens_available
                ? props.model.total_tokens
                : undefined
            )}
          </div>
        </div>
      </CardContent>
      <CardContent>
        <div className='mb-1 flex items-center justify-between text-xs'>
          <span className='text-muted-foreground'>{t('Success rate')}</span>
          <span className='font-medium'>{onlineRate.toFixed(1)}%</span>
        </div>
        <Progress
          value={onlineRate}
          aria-label={t('Success rate')}
          className={
            onlineRate < 100
              ? '[&_[data-slot=progress-indicator]]:bg-warning'
              : '[&_[data-slot=progress-indicator]]:bg-success'
          }
        />
      </CardContent>
      <CardFooter className='justify-end'>
        <Button
          size='sm'
          variant='ghost'
          nativeButton={false}
          render={
            <Link
              to='/cluster-status/models/$modelId'
              params={{ modelId: String(props.model.model_id) }}
            />
          }
        >
          {t('View Model Clusters')}
          <HugeiconsIcon
            icon={ArrowRight01Icon}
            strokeWidth={2}
            data-icon='inline-end'
          />
        </Button>
      </CardFooter>
    </Card>
  )
}

function groupTitle(type: string, translate: (key: string) => string) {
  if (type === 'language') return translate('Language Models')
  if (type === 'embedding') return translate('Embedding Models')
  if (type === 'multimodal') return translate('Multimodal Models')
  return translate('Other Models')
}

export function OverviewContent(props: OverviewContentProps) {
  const { t } = useTranslation()
  const pageCount = Math.max(
    1,
    Math.ceil(props.data.pagination.total / props.data.pagination.page_size)
  )
  const visiblePages = getVisiblePages(props.data.pagination.page, pageCount)
  const overview = props.data.overview

  return (
    <div className='flex flex-col gap-5'>
      <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-5'>
        <SummaryCard
          title={t('Total Clusters')}
          value={formatCompactNumber(overview.total_clusters)}
          description={t('Configured clusters')}
          icon={
            <HugeiconsIcon
              icon={InformationCircleIcon}
              strokeWidth={2}
              aria-hidden='true'
            />
          }
        />
        <SummaryCard
          title={t('Online Clusters')}
          value={formatCompactNumber(overview.online_clusters)}
          description={t('Healthy and reporting')}
          icon={
            <HugeiconsIcon
              icon={CheckmarkCircle02Icon}
              strokeWidth={2}
              aria-hidden='true'
            />
          }
        />
        <SummaryCard
          title={t('Abnormal Clusters')}
          value={formatCompactNumber(overview.abnormal_clusters)}
          description={t('Require attention')}
          icon={
            <HugeiconsIcon
              icon={Alert02Icon}
              strokeWidth={2}
              aria-hidden='true'
            />
          }
        />
        <SummaryCard
          title={t('Total Requests')}
          value={formatCompactNumber(
            overview.requests_available ? overview.total_requests : undefined
          )}
          description={t('Reported by connected Agents')}
          icon={
            <HugeiconsIcon
              icon={InformationCircleIcon}
              strokeWidth={2}
              aria-hidden='true'
            />
          }
        />
        <SummaryCard
          title={t('Total Tokens')}
          value={formatCompactNumber(
            overview.tokens_available ? overview.total_tokens : undefined
          )}
          description={t('Reported by connected Agents')}
          icon={
            <HugeiconsIcon
              icon={InformationCircleIcon}
              strokeWidth={2}
              aria-hidden='true'
            />
          }
        />
      </div>

      <div className='grid min-w-0 gap-4 xl:grid-cols-[minmax(0,1fr)_20rem]'>
        <div className='flex min-w-0 flex-col gap-5'>
          {props.data.model_groups.length === 0 ? (
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
                    <EmptyTitle>{t('No clusters found')}</EmptyTitle>
                    <EmptyDescription>
                      {t('Add a cluster or adjust the current filters.')}
                    </EmptyDescription>
                  </EmptyHeader>
                  <EmptyContent>
                    <Button onClick={props.onAddCluster}>
                      {t('Add Cluster')}
                    </Button>
                  </EmptyContent>
                </Empty>
              </CardContent>
            </Card>
          ) : (
            props.data.model_groups.map((group) => (
              <section key={group.type} className='flex flex-col gap-3'>
                <div className='flex items-center gap-2'>
                  <h3 className='font-semibold'>{groupTitle(group.type, t)}</h3>
                  <Badge variant='secondary'>{group.models.length}</Badge>
                </div>
                <div className='grid gap-3 md:grid-cols-2 2xl:grid-cols-3'>
                  {group.models.map((model) => (
                    <ModelSummaryCard key={model.model_id} model={model} />
                  ))}
                </div>
              </section>
            ))
          )}
        </div>

        <aside>
          <Card>
            <CardHeader>
              <CardTitle>{t('Cluster Alerts')}</CardTitle>
              <CardDescription>
                {t('Latest polling warnings and failures')}
              </CardDescription>
            </CardHeader>
            <CardContent className='flex flex-col gap-3'>
              {props.data.alerts.length === 0 ? (
                <Empty>
                  <EmptyHeader>
                    <EmptyMedia variant='icon'>
                      <HugeiconsIcon
                        icon={CheckmarkCircle02Icon}
                        strokeWidth={2}
                      />
                    </EmptyMedia>
                    <EmptyTitle>{t('No active alerts')}</EmptyTitle>
                  </EmptyHeader>
                </Empty>
              ) : (
                props.data.alerts.map((alert) => {
                  const error = getClusterPollErrorPresentation(
                    alert.error_code
                  )
                  const retrying = props.retryingClusterId === alert.cluster_id
                  return (
                    <div
                      key={alert.cluster_id}
                      className='flex flex-col gap-3 rounded-lg border p-3'
                    >
                      <div className='flex items-center justify-between gap-2'>
                        <span className='truncate font-medium'>
                          {alert.cluster_name}
                        </span>
                        <ClusterStatusBadge status={alert.health_status} />
                      </div>
                      <div>
                        <div className='text-sm font-medium'>
                          {t(error.title)}
                        </div>
                        <div className='text-muted-foreground mt-1 text-xs'>
                          {t(error.description)}
                        </div>
                      </div>
                      <div className='text-muted-foreground truncate text-xs'>
                        {alert.model_name}
                      </div>
                      {alert.error_code ? (
                        <code className='text-destructive truncate text-xs'>
                          {alert.error_code}
                        </code>
                      ) : null}
                      {alert.consecutive_failures > 0 ? (
                        <div className='text-muted-foreground text-xs'>
                          {t('Failed {{count}} times in a row', {
                            count: alert.consecutive_failures,
                          })}
                        </div>
                      ) : null}
                      <TelemetryFreshness
                        lastSuccessAt={alert.last_success_at}
                        lastPolledAt={alert.last_polled_at}
                        refreshInterval={props.refreshInterval}
                        showLastAttempt
                      />
                      <div className='flex flex-wrap gap-2'>
                        <Button
                          size='sm'
                          variant='outline'
                          nativeButton={false}
                          render={
                            <Link
                              to='/cluster-status/$clusterId'
                              params={{
                                clusterId: String(alert.cluster_id),
                              }}
                            />
                          }
                        >
                          {t('View Details')}
                        </Button>
                        {error.retryable ? (
                          <Button
                            size='sm'
                            onClick={() =>
                              props.onRetryCluster(alert.cluster_id)
                            }
                            disabled={props.retryingClusterId !== null}
                          >
                            {retrying ? (
                              <Spinner data-icon='inline-start' />
                            ) : null}
                            {retrying ? t('Retrying...') : t('Retry now')}
                          </Button>
                        ) : null}
                      </div>
                    </div>
                  )
                })
              )}
            </CardContent>
          </Card>
        </aside>
      </div>

      {props.data.pagination.total > 0 ? (
        <div className='flex flex-wrap items-center justify-center gap-2'>
          <Button
            variant='outline'
            size='icon-sm'
            aria-label={t('Previous')}
            onClick={() =>
              props.onPageChange(Math.max(1, props.data.pagination.page - 1))
            }
            disabled={props.data.pagination.page <= 1}
          >
            <span aria-hidden='true'>‹</span>
          </Button>
          {visiblePages.map((visiblePage) => (
            <Button
              key={visiblePage}
              variant={
                visiblePage === props.data.pagination.page ? 'outline' : 'ghost'
              }
              size='icon-sm'
              aria-current={
                visiblePage === props.data.pagination.page ? 'page' : undefined
              }
              onClick={() => props.onPageChange(visiblePage)}
            >
              {visiblePage}
            </Button>
          ))}
          <Button
            variant='outline'
            size='icon-sm'
            aria-label={t('Next')}
            onClick={() =>
              props.onPageChange(
                Math.min(pageCount, props.data.pagination.page + 1)
              )
            }
            disabled={props.data.pagination.page >= pageCount}
          >
            <span aria-hidden='true'>›</span>
          </Button>
          <span className='text-muted-foreground ml-2 text-sm'>
            {t('Rows per page')}
          </span>
          <NativeSelect
            value={String(props.data.pagination.page_size)}
            aria-label={t('Rows per page')}
            onChange={(event) =>
              props.onPageSizeChange(Number(event.target.value))
            }
          >
            {[10, 20, 50].map((pageSize) => (
              <NativeSelectOption key={pageSize} value={pageSize}>
                {pageSize}
              </NativeSelectOption>
            ))}
          </NativeSelect>
        </div>
      ) : null}
    </div>
  )
}
