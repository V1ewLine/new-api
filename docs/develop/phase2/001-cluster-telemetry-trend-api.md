# 集群遥测趋势查询接口

## 1. 目标

第二阶段在现有 `cluster_telemetry_history` 历史采样表上增加面向图表的趋势查询能力。接口需要支持长时间窗口、限制响应点数，并保持 SQLite、MySQL 和 PostgreSQL 三种主数据库兼容。

实现没有新增历史表，也没有复制遥测数据。数据保留周期仍由系统设置 `ClusterTelemetryRetentionDays` 控制。

## 2. 接口

新增管理员只读接口：

```http
GET /api/clusters/:clusterId/telemetry/trends
```

查询参数：

| 参数 | 格式 | 说明 |
| --- | --- | --- |
| `start_at` | RFC3339 | 开始时间，包含该秒 |
| `end_at` | RFC3339 | 结束时间，不包含该秒 |
| `max_points` | 1～2000 | 期望的最大趋势点数；未传时默认 720 |

时间范围采用左闭右开区间：

```text
[start_at, end_at)
```

接口位于现有 `middleware.AdminAuth()` 管理员路由组中，响应设置 `Cache-Control: no-store`。

## 3. 自动分桶

后端根据时间跨度和 `max_points` 自动选择易读的分桶宽度：

```text
1s, 2s, 5s, 10s, 15s, 30s,
1m, 2m, 5m, 10m, 15m, 30m,
1h, 2h, 3h, 6h, 12h,
1d, 2d, 7d
```

如果时间跨度仍然超过这些档位，则按整天的倍数计算分桶。分桶与 Unix 时间边界对齐，因此相同查询条件能够得到稳定的时间轴。

数据库查询先按桶聚合以下信息：

- 采样总数；
- 成功采样数；
- 桶内最后一条成功采样的主键。

随后只读取每个桶最后一条成功采样的标准化 JSON。即使一个时间窗口包含大量 5 秒采样，应用层最多也只反序列化 `max_points` 条载荷，避免把完整历史加载进内存。

## 4. 数据库兼容

分桶表达式按主数据库类型生成：

- MySQL：`FLOOR(collected_at / bucket) * bucket`
- SQLite、PostgreSQL：`(collected_at / bucket) * bucket`

其余查询使用 GORM 的 `Where`、`Group`、`Order`、`Limit` 和 `Find/Scan`。查询复用历史表已有的 `(cluster_id, collected_at, id)` 索引，没有数据库迁移。

## 5. 响应数据

响应包含：

- 实际查询的 UTC 起止时间；
- 当前历史保留天数；
- 当前集群最早可用采样时间；
- 实际分桶秒数和原始采样总数；
- 每个桶的采样成功率；
- 引擎和机器可用率；
- 运行中请求、等待中请求、Token 使用量、吞吐量、缓存使用率；
- GPU 板卡总功耗；
- 每个 GPU 的功耗；
- CPU 和内存利用率。

每个趋势点同时返回桶时间 `timestamp` 和被选中成功采样的实际时间 `sampled_at`。GPU 优先使用 UUID 作为稳定序列标识；UUID 缺失时使用索引和名称生成回退标识。

缺失指标返回空值，不补零、不插值。只有采样成功率可以由成功数和总数直接计算。

## 6. 安全与限制

趋势响应不包含：

- Agent 地址；
- Bearer Token；
- 连接密钥密文；
- 原始 Agent 响应；
- 诊断载荷；
- 完整标准化 JSON。

请求时间跨度不能超过当前保留天数，点数不能超过 2000。无效范围统一返回 `invalid cluster trend request`。

## 7. 主要改动文件

- `model/cluster.go`
- `service/clusterstatus/trends.go`
- `service/clusterstatus/trends_test.go`
- `service/clusterstatus/types.go`
- `controller/cluster.go`
- `router/api-router.go`
- `router/cluster_router_test.go`
