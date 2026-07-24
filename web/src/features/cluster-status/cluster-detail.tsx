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
  ArrowLeft01Icon,
  Delete02Icon,
  InformationCircleIcon,
  Key01Icon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import type { TFunction } from 'i18next'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
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
import { Spinner } from '@/components/ui/spinner'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import {
  getClusterDetail,
  refreshCluster,
  verifyClusterCredential,
} from './api'
import { DeleteClusterDialog } from './components/delete-cluster-dialog'
import { RotateClusterCredentialDialog } from './components/rotate-cluster-credential-dialog'
import { ClusterStatusBadge } from './components/status-badge'
import { useClusterRefreshInterval } from './hooks/use-cluster-refresh-interval'
import {
  formatBytes,
  formatCompactNumber,
  formatPercent,
  formatTimestamp,
} from './lib/format'
import { clusterQueryKeys } from './query-keys'
import type { Cluster, NormalizedTelemetry } from './types'

type ClusterDetailProps = {
  clusterId: number
}

function MetricCard(props: {
  title: string
  value: string
  description?: string
}) {
  return (
    <Card size='sm'>
      <CardHeader>
        <CardDescription>{props.title}</CardDescription>
        <CardTitle className='text-xl'>{props.value}</CardTitle>
      </CardHeader>
      {props.description ? (
        <CardContent className='text-muted-foreground text-xs'>
          {props.description}
        </CardContent>
      ) : null}
    </Card>
  )
}

function statusLabel(
  available: boolean | undefined,
  translate: TFunction
): string {
  if (available === undefined) return '—'
  return available ? translate('Online') : translate('Offline')
}

function windowLabel(
  complete: boolean | undefined,
  translate: TFunction
): string {
  if (complete === undefined) return '—'
  return complete ? translate('Complete') : translate('Incomplete')
}

function pollErrorTitle(errorCode: string, translate: TFunction): string {
  if (errorCode === 'AGENT_SCHEMA_UNSUPPORTED') {
    return translate('Unsupported telemetry schema')
  }
  return translate('Latest telemetry poll failed')
}

function credentialSetupDescription(
  errorCode: string | undefined,
  translate: TFunction
): string {
  if (errorCode === 'AGENT_HTTP_401' || errorCode === 'AGENT_HTTP_403') {
    return translate(
      'The Agent rejected the Token. Confirm the environment variable and restart the Agent.'
    )
  }
  if (errorCode === 'AGENT_TIMEOUT') {
    return translate(
      'The Agent connection timed out. Check the address, port, and network.'
    )
  }
  if (errorCode === 'AGENT_UNREACHABLE') {
    return translate(
      'New API cannot reach the Agent. Check the address, port, firewall, and service status.'
    )
  }
  return translate(
    'Install the generated Token on the remote Agent, restart it, and test the connection.'
  )
}

function OverviewTab(props: {
  cluster: Cluster
  telemetry?: NormalizedTelemetry
}) {
  const { t } = useTranslation()
  const telemetry = props.telemetry
  return (
    <div className='flex flex-col gap-4'>
      <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4 2xl:grid-cols-7'>
        <Card size='sm'>
          <CardHeader>
            <CardDescription>{t('Cluster Health')}</CardDescription>
            <CardTitle>
              <ClusterStatusBadge
                status={props.cluster.health_status}
                credentialStatus={props.cluster.credential_status}
              />
            </CardTitle>
          </CardHeader>
        </Card>
        <MetricCard
          title={t('Agent Schema')}
          value={telemetry?.schema_version ?? '—'}
        />
        <MetricCard
          title={t('Last Poll')}
          value={formatTimestamp(props.cluster.last_polled_at)}
        />
        <MetricCard
          title={t('Requests')}
          value={formatCompactNumber(telemetry?.metrics.requests)}
        />
        <MetricCard
          title={t('Tokens')}
          value={formatCompactNumber(telemetry?.metrics.tokens)}
        />
        <MetricCard
          title={t('Engine')}
          value={statusLabel(telemetry?.engine.up, t)}
        />
        <MetricCard
          title={t('Machine')}
          value={statusLabel(telemetry?.machine.up, t)}
        />
      </div>

      <div className='grid gap-4 lg:grid-cols-2'>
        <Card>
          <CardHeader>
            <CardTitle>{t('Agent Identity')}</CardTitle>
            <CardDescription>
              {t('Identity reported by the telemetry Agent')}
            </CardDescription>
          </CardHeader>
          <CardContent className='grid grid-cols-[auto_1fr] gap-x-4 gap-y-3 text-sm'>
            <span className='text-muted-foreground'>{t('Node ID')}</span>
            <code className='truncate'>
              {telemetry?.identity.node_id || '—'}
            </code>
            <span className='text-muted-foreground'>{t('Engine ID')}</span>
            <code className='truncate'>
              {telemetry?.identity.engine_id || '—'}
            </code>
            <span className='text-muted-foreground'>{t('Reported Model')}</span>
            <span className='truncate'>{telemetry?.identity.model || '—'}</span>
            <span className='text-muted-foreground'>{t('Linked Model')}</span>
            <span className='truncate'>{props.cluster.model_name}</span>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>{t('Telemetry Alignment')}</CardTitle>
            <CardDescription>
              {t('Engine and machine sample correlation')}
            </CardDescription>
          </CardHeader>
          <CardContent className='grid grid-cols-[auto_1fr] gap-x-4 gap-y-3 text-sm'>
            <span className='text-muted-foreground'>{t('Method')}</span>
            <span>{telemetry?.alignment.method || '—'}</span>
            <span className='text-muted-foreground'>{t('Quality')}</span>
            <Badge variant='outline'>
              {telemetry?.alignment.quality || t('Unknown')}
            </Badge>
            <span className='text-muted-foreground'>{t('Skew')}</span>
            <span>
              {telemetry?.alignment.skew_ms === undefined
                ? '—'
                : `${telemetry.alignment.skew_ms.toFixed(1)} ms`}
            </span>
            <span className='text-muted-foreground'>{t('Sample Window')}</span>
            <span>{windowLabel(telemetry?.machine.window_complete, t)}</span>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t('Trend History')}</CardTitle>
          <CardDescription>
            {t('Historical charts are not available in phase one.')}
          </CardDescription>
        </CardHeader>
      </Card>
    </div>
  )
}

function EngineTab(props: { telemetry?: NormalizedTelemetry }) {
  const { t } = useTranslation()
  const engine = props.telemetry?.engine
  return (
    <div className='flex flex-col gap-4'>
      <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4'>
        <MetricCard
          title={t('Engine Status')}
          value={statusLabel(engine?.up, t)}
        />
        <MetricCard
          title={t('Engine Version')}
          value={engine?.version || '—'}
        />
        <MetricCard
          title={t('Running Requests')}
          value={formatCompactNumber(engine?.running_requests)}
        />
        <MetricCard
          title={t('Waiting Requests')}
          value={formatCompactNumber(engine?.waiting_requests)}
        />
        <MetricCard
          title={t('Token Usage')}
          value={formatCompactNumber(engine?.token_usage)}
        />
        <MetricCard
          title={t('Throughput')}
          value={formatCompactNumber(engine?.throughput)}
        />
        <MetricCard
          title={t('Cache Usage')}
          value={formatPercent(engine?.cache_usage)}
        />
        <MetricCard
          title={t('Request Duration')}
          value={engine ? `${engine.request_duration_ms.toFixed(1)} ms` : '—'}
        />
      </div>
      <Card>
        <CardHeader>
          <CardTitle>{t('SGLang Load Snapshots')}</CardTitle>
          <CardDescription>
            {t('{{count}} data-parallel snapshots reported', {
              count: engine?.loads.length ?? 0,
            })}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Empty>
            <EmptyHeader>
              <EmptyMedia variant='icon'>
                <HugeiconsIcon icon={InformationCircleIcon} strokeWidth={2} />
              </EmptyMedia>
              <EmptyTitle>{t('Trend history is not enabled')}</EmptyTitle>
              <EmptyDescription>
                {t(
                  'Current values are shown above without fabricated history.'
                )}
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        </CardContent>
      </Card>
    </div>
  )
}

function MachineTab(props: { telemetry?: NormalizedTelemetry }) {
  const { t } = useTranslation()
  const machine = props.telemetry?.machine
  const system = machine?.system
  return (
    <div className='flex flex-col gap-4'>
      <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4'>
        <MetricCard
          title={t('GPU Count')}
          value={machine?.gpu.available ? String(machine.gpu.count) : '—'}
        />
        <MetricCard
          title={t('GPU Board Power')}
          value={
            machine?.gpu.power_total_watts === undefined
              ? '—'
              : `${machine.gpu.power_total_watts.toFixed(1)} W`
          }
        />
        <MetricCard
          title={t('CPU Utilization')}
          value={formatPercent(system?.cpu_utilization_percent)}
        />
        <MetricCard
          title={t('Memory Utilization')}
          value={formatPercent(system?.memory_utilization_percent)}
        />
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t('GPU Devices')}</CardTitle>
          <CardDescription>
            {machine?.gpu.driver_version
              ? t('NVIDIA driver {{version}}', {
                  version: machine.gpu.driver_version,
                })
              : t('Latest NVIDIA telemetry')}
          </CardDescription>
        </CardHeader>
        <CardContent className='overflow-x-auto'>
          {!machine?.gpu.devices.length ? (
            <Empty>
              <EmptyHeader>
                <EmptyTitle>{t('No GPU telemetry')}</EmptyTitle>
                <EmptyDescription>
                  {t('The Agent did not report an available NVIDIA collector.')}
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('GPU')}</TableHead>
                  <TableHead>{t('Utilization')}</TableHead>
                  <TableHead>{t('Memory')}</TableHead>
                  <TableHead>{t('Temperature')}</TableHead>
                  <TableHead>{t('Power')}</TableHead>
                  <TableHead>{t('SM Clock')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {machine.gpu.devices.map((device) => (
                  <TableRow key={device.uuid || device.index}>
                    <TableCell>
                      <div className='font-medium'>
                        {device.name || `GPU ${device.index}`}
                      </div>
                      <code className='text-muted-foreground text-xs'>
                        {device.uuid}
                      </code>
                    </TableCell>
                    <TableCell>
                      {formatPercent(device.utilization_percent)}
                    </TableCell>
                    <TableCell>
                      {formatBytes(device.memory_used_bytes)} /{' '}
                      {formatBytes(device.memory_total_bytes)}
                    </TableCell>
                    <TableCell>
                      {device.temperature_celsius === undefined
                        ? '—'
                        : `${device.temperature_celsius.toFixed(0)} °C`}
                    </TableCell>
                    <TableCell>
                      {device.power_watts === undefined
                        ? '—'
                        : `${device.power_watts.toFixed(1)} W`}
                    </TableCell>
                    <TableCell>
                      {device.sm_clock_mhz === undefined
                        ? '—'
                        : `${device.sm_clock_mhz.toFixed(0)} MHz`}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <div className='grid gap-4 lg:grid-cols-2'>
        <Card>
          <CardHeader>
            <CardTitle>{t('System Resources')}</CardTitle>
          </CardHeader>
          <CardContent className='grid grid-cols-[auto_1fr] gap-x-4 gap-y-3 text-sm'>
            <span className='text-muted-foreground'>{t('CPU Cores')}</span>
            <span>{formatCompactNumber(system?.cpu_count)}</span>
            <span className='text-muted-foreground'>{t('Memory Used')}</span>
            <span>{formatBytes(system?.memory_used_bytes)}</span>
            <span className='text-muted-foreground'>{t('Memory Total')}</span>
            <span>{formatBytes(system?.memory_total_bytes)}</span>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>{t('Load Average')}</CardTitle>
          </CardHeader>
          <CardContent className='grid grid-cols-3 gap-3'>
            {(['1m', '5m', '15m'] as const).map((window) => (
              <div key={window} className='rounded-lg border p-3 text-center'>
                <div className='text-muted-foreground text-xs'>{window}</div>
                <div className='mt-1 font-medium'>
                  {formatCompactNumber(system?.load_average[window])}
                </div>
              </div>
            ))}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

export function ClusterDetail(props: ClusterDetailProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const refreshInterval = useClusterRefreshInterval()
  const clusterQuery = useQuery({
    queryKey: clusterQueryKeys.cluster(props.clusterId),
    queryFn: async () => {
      const response = await getClusterDetail(props.clusterId)
      if (!response.success || !response.data) {
        throw new Error(response.message || t('Failed to load cluster details'))
      }
      return response.data
    },
    refetchInterval: refreshInterval,
  })
  const refreshMutation = useMutation({
    mutationFn: async () => {
      const response = await refreshCluster(props.clusterId)
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
  const verifyMutation = useMutation({
    mutationFn: async () => {
      const response = await verifyClusterCredential(props.clusterId)
      if (!response.success || !response.data) {
        throw new Error(
          response.message || t('Failed to test Agent connection')
        )
      }
      return response.data
    },
    onSuccess: async (response) => {
      await queryClient.invalidateQueries({ queryKey: clusterQueryKeys.all })
      if (response.verified) {
        toast.success(t('Agent connected successfully'))
      } else {
        toast.error(t('Agent connection test failed'))
      }
    },
    onError: (error) => {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to test Agent connection')
      )
    },
  })
  const cluster = clusterQuery.data
  const credentialPending = cluster?.credential_status === 'pending'
  const statusActionPending = credentialPending
    ? verifyMutation.isPending
    : refreshMutation.isPending
  let statusActionLabel = credentialPending
    ? t('Test Connection')
    : t('Refresh')
  if (statusActionPending) {
    statusActionLabel = credentialPending
      ? t('Testing connection...')
      : t('Refreshing...')
  }

  function runStatusAction() {
    if (credentialPending) {
      verifyMutation.mutate()
      return
    }
    refreshMutation.mutate()
  }

  return (
    <>
      <SectionPageLayout>
        <SectionPageLayout.Breadcrumb>
          <Button
            variant='ghost'
            size='sm'
            nativeButton={false}
            render={
              cluster ? (
                <Link
                  to='/cluster-status/models/$modelId'
                  params={{ modelId: String(cluster.model_id) }}
                />
              ) : (
                <Link to='/cluster-status' />
              )
            }
          >
            <HugeiconsIcon
              icon={ArrowLeft01Icon}
              strokeWidth={2}
              data-icon='inline-start'
            />
            {t('Back')}
          </Button>
        </SectionPageLayout.Breadcrumb>
        <SectionPageLayout.Title>
          {cluster?.name ?? t('Cluster Details')}
        </SectionPageLayout.Title>
        <SectionPageLayout.Actions>
          {cluster ? (
            <ClusterStatusBadge
              status={cluster.health_status}
              credentialStatus={cluster.credential_status}
            />
          ) : null}
          <Button
            variant='outline'
            onClick={runStatusAction}
            disabled={statusActionPending || !cluster}
          >
            {statusActionPending ? (
              <Spinner data-icon='inline-start' />
            ) : (
              <HugeiconsIcon
                icon={InformationCircleIcon}
                strokeWidth={2}
                data-icon='inline-start'
              />
            )}
            {statusActionLabel}
          </Button>
          {cluster ? <RotateClusterCredentialDialog cluster={cluster} /> : null}
          <Button
            variant='destructive'
            onClick={() => setDeleteDialogOpen(true)}
            disabled={!cluster}
          >
            <HugeiconsIcon
              icon={Delete02Icon}
              strokeWidth={2}
              data-icon='inline-start'
            />
            {t('Delete')}
          </Button>
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          {clusterQuery.isLoading ? (
            <div className='flex flex-col gap-4'>
              <Skeleton className='h-9 w-72' />
              <Skeleton className='h-96 w-full rounded-xl' />
            </div>
          ) : null}

          {clusterQuery.isError ? (
            <Card>
              <CardContent>
                <Empty>
                  <EmptyHeader>
                    <EmptyMedia variant='icon'>
                      <HugeiconsIcon icon={Alert02Icon} strokeWidth={2} />
                    </EmptyMedia>
                    <EmptyTitle>
                      {t('Failed to load cluster details')}
                    </EmptyTitle>
                    <EmptyDescription>
                      {clusterQuery.error instanceof Error
                        ? clusterQuery.error.message
                        : t('Please try again later.')}
                    </EmptyDescription>
                  </EmptyHeader>
                </Empty>
              </CardContent>
            </Card>
          ) : null}

          {cluster ? (
            <div className='flex flex-col gap-4'>
              {!cluster.model_available ? (
                <Alert variant='destructive'>
                  <HugeiconsIcon icon={Alert02Icon} strokeWidth={2} />
                  <AlertTitle>{t('Linked model is unavailable')}</AlertTitle>
                  <AlertDescription>
                    {t(
                      'The cluster is retained for audit, but the linked model is disabled or deleted.'
                    )}
                  </AlertDescription>
                </Alert>
              ) : null}
              {cluster.telemetry?.model_mismatch ? (
                <Alert>
                  <HugeiconsIcon icon={Alert02Icon} strokeWidth={2} />
                  <AlertTitle>{t('Reported model mismatch')}</AlertTitle>
                  <AlertDescription>
                    {t(
                      'The Agent-reported model does not match the linked New API model.'
                    )}
                  </AlertDescription>
                </Alert>
              ) : null}
              {cluster.credential_status === 'pending' ? (
                <Alert>
                  <HugeiconsIcon icon={Key01Icon} strokeWidth={2} />
                  <AlertTitle>{t('Awaiting Agent configuration')}</AlertTitle>
                  <AlertDescription>
                    {credentialSetupDescription(cluster.last_error_code, t)}
                  </AlertDescription>
                </Alert>
              ) : null}
              {cluster.last_error_code &&
              cluster.credential_status === 'active' ? (
                <Alert variant='destructive'>
                  <HugeiconsIcon icon={Alert02Icon} strokeWidth={2} />
                  <AlertTitle>
                    {pollErrorTitle(cluster.last_error_code, t)}
                  </AlertTitle>
                  <AlertDescription>
                    {cluster.last_error_code === 'AGENT_SCHEMA_UNSUPPORTED'
                      ? t(
                          'The Agent returned an unsupported telemetry schema version.'
                        )
                      : cluster.last_error_code}
                  </AlertDescription>
                </Alert>
              ) : null}
              {!cluster.telemetry && cluster.credential_status === 'active' ? (
                <Alert>
                  <HugeiconsIcon icon={InformationCircleIcon} strokeWidth={2} />
                  <AlertTitle>{t('Waiting for telemetry')}</AlertTitle>
                  <AlertDescription>
                    {t(
                      'The first successful Agent poll has not completed yet.'
                    )}
                  </AlertDescription>
                </Alert>
              ) : null}

              <Tabs defaultValue='overview'>
                <TabsList variant='line'>
                  <TabsTrigger value='overview'>{t('Overview')}</TabsTrigger>
                  <TabsTrigger value='engine'>
                    {t('Engine Metrics')}
                  </TabsTrigger>
                  <TabsTrigger value='machine'>
                    {t('Machine Metrics')}
                  </TabsTrigger>
                </TabsList>
                <TabsContent value='overview'>
                  <OverviewTab
                    cluster={cluster}
                    telemetry={cluster.telemetry}
                  />
                </TabsContent>
                <TabsContent value='engine'>
                  <EngineTab telemetry={cluster.telemetry} />
                </TabsContent>
                <TabsContent value='machine'>
                  <MachineTab telemetry={cluster.telemetry} />
                </TabsContent>
              </Tabs>
            </div>
          ) : null}
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <DeleteClusterDialog
        cluster={cluster ?? null}
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        onDeleted={() => {
          if (!cluster) {
            return
          }
          void navigate({
            to: '/cluster-status/models/$modelId',
            params: { modelId: String(cluster.model_id) },
          })
        }}
      />
    </>
  )
}
