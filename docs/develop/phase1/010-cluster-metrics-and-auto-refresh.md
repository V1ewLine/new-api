# 集群指标展示与自动刷新设置

## 日期

2026-07-24

## 本阶段目标

修复集群状态一级页面、模型集群二级页面和集群详情页中的请求数、Token 数长期显示为 `—` 的问题，并将集群状态自动刷新间隔纳入“系统设置 → 控制台内容 → 数据仪表板”统一管理。

## 问题原因

实际运行的 SGLang Agent `/v1/telemetry` 响应没有提供原适配器优先读取的 `total_requests` 和 `total_tokens` 字段，而是提供以下当前负载字段：

- `num_running_reqs`
- `num_waiting_reqs`
- `num_total_tokens`
- `num_used_tokens`

原适配器因此把请求数和 Token 数标记为不可用，一级汇总、二级列表和详情页最终都显示 `—`。

此外，前端页面虽然存在固定的 5 秒 React Query 轮询，但刷新间隔写死在三个页面中；后端 Agent 采集间隔也只读取启动环境变量，无法从系统设置中统一调整。

## 实际修改

### 请求数与 Token 数

- 继续优先使用 Agent 明确提供的累计字段：
  - 请求数：`total_requests`、`request_count`、`requests`
  - Token 数：`total_tokens`、`token_count`、`tokens`
- 累计字段缺失时，按 SGLang 当前负载字段回退：
  - 请求数为所有 data-parallel rank 的 `num_running_reqs + num_waiting_reqs` 之和；
  - Token 数优先汇总所有 rank 的 `num_total_tokens`，缺失时再读取 `num_used_tokens`、`token_usage` 或 `used_tokens`。
- 即使当前值为 0，也保留“指标可用”状态，因此页面会显示 `0`，不会再显示 `—`。
- 一级页面继续汇总所有最新集群快照；二级页面和集群详情页展示各自最新快照。

### 自动刷新

- 新增系统选项：

  ```text
  ClusterStatusRefreshIntervalSeconds
  ```

- 默认值为 5 秒，可设置范围为 1～300 秒。
- 设置位置：

  ```text
  系统设置 → 控制台内容 → 数据仪表板
  ```

- 同一个设置同时控制：
  - New API 后端请求 Agent 的遥测采集间隔；
  - 集群状态一级页面、模型集群二级页面和集群详情页的数据查询间隔。
- 保存新间隔后无需重启 New API：
  - 后端会立即把已启用集群重新加入待采集队列；
  - 前端会使刷新设置缓存失效并读取新值。
- 集群详情页原有“刷新”按钮保留，可继续触发一次立即采集。
- `CLUSTER_TELEMETRY_POLL_INTERVAL_SECONDS` 继续作为未保存数据库选项时的启动默认值；数据库中的系统设置优先。

## 接口变更

新增管理员只读接口：

```http
GET /api/clusters/settings
```

响应示例：

```json
{
  "success": true,
  "data": {
    "refresh_interval_seconds": 5
  }
}
```

接口使用现有 `middleware.AdminAuth()`，只返回集群页面所需的非敏感刷新配置，管理员页面无需读取 Root 权限下的完整系统选项列表。

系统选项更新仍使用：

```http
PUT /api/option/
```

后端会在控制器和模型写入层校验刷新间隔，非法值不会写入数据库或运行时配置。

## 新增或修改的文件

- `common/constants.go`
- `common/cluster_status.go`
- `common/cluster_status_test.go`
- `model/cluster.go`
- `model/option.go`
- `controller/cluster.go`
- `controller/option.go`
- `router/api-router.go`
- `router/cluster_router_test.go`
- `service/clusterstatus/poller.go`
- `service/clusterstatus/poller_test.go`
- `service/clusterstatus/service.go`
- `service/clusterstatus/types.go`
- `service/clusterstatus/schema_adapter.go`
- `service/clusterstatus/schema_adapter_test.go`
- `web/src/features/cluster-status/api.ts`
- `web/src/features/cluster-status/types.ts`
- `web/src/features/cluster-status/query-keys.ts`
- `web/src/features/cluster-status/index.tsx`
- `web/src/features/cluster-status/model-detail.tsx`
- `web/src/features/cluster-status/cluster-detail.tsx`
- `web/src/features/cluster-status/hooks/use-cluster-refresh-interval.ts`
- `web/src/features/cluster-status/lib/refresh-interval.ts`
- `web/src/features/cluster-status/__tests__/refresh-interval.test.ts`
- `web/src/features/system-settings/types.ts`
- `web/src/features/system-settings/content/index.tsx`
- `web/src/features/system-settings/content/section-registry.tsx`
- `web/src/features/system-settings/content/dashboard-section.tsx`
- `web/src/features/system-settings/hooks/use-update-option.ts`
- `web/src/i18n/locales/{en,zh,zh-TW,fr,ja,ru,vi}.json`

## 数据库变更

没有新增表或字段。

`ClusterStatusRefreshIntervalSeconds` 使用现有 `options` 表保存，兼容 SQLite、MySQL 和 PostgreSQL。

## 测试与验证

覆盖以下回归场景：

- Agent 只有 SGLang 当前负载字段时，请求数和 Token 数能够正常映射；
- 多个 data-parallel rank 的请求和 Token 正确求和；
- Agent 提供累计字段时仍优先使用累计字段；
- 刷新间隔只接受 1～300 的整数；
- 环境变量默认值经过边界校验；
- 后端轮询配置可以在服务不重启时读取新间隔；
- 新设置接口受管理员鉴权保护；
- 前端秒数正确转换为 React Query 使用的毫秒数；
- 缺失或非法前端配置安全回退到 5 秒；
- 七种前端语言的新增文案完整同步。

验证结果：

```text
Go 定向测试：
  go test ./common -run '^Test(Parse|Default)ClusterStatusRefreshInterval'
  go test ./service/clusterstatus ./router

Go build：
  go build ./...

Frontend：
  typecheck 通过
  集群状态单元测试 16 项通过
  变更文件定向 lint 无错误
  format:check 通过
  copyright:check 通过
  生产构建通过

i18n：
  七种 locale 的 missing、extras、untranslated 均为 0
```

仓库当前全量前端 lint 仍包含本次修改范围之外的历史规则错误；本次变更文件没有新增 lint 错误。`common` 包全量测试中的邮件服务器用例需要监听本机端口，在当前沙箱中被系统策略阻止；本次新增的 common 定向测试以及集群服务、路由测试均已通过。

## 当前语义

Agent 明确提供累计指标时，页面展示累计值。当前部署的 SGLang Agent 只提供即时负载，因此回退值表示最新采样时刻的当前请求和当前 Token 占用；一级页面的“总数”表示所有最新集群快照之和，不是由 New API 虚构的历史累计量。
