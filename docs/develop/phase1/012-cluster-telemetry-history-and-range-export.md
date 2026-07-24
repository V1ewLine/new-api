# 集群遥测历史保留与时间范围导出

## 1. 目标

在保留现有最新快照展示和导出的基础上，将集群轮询结果按时间持续写入主数据库，并允许管理员配置历史数据保留天数、按精确到秒的时间窗口导出历史数据。

本阶段解决以下问题：

- `cluster_telemetry_latest` 只能保存最后一次成功采样，无法还原时间段内的变化；
- Agent 离线时只有集群当前错误状态，无法在历史导出中识别故障区间；
- 历史数据保留周期无法由管理员控制；
- 原导出弹窗只能导出最新瞬时快照。

## 2. 数据模型

新增主数据库表：

```text
cluster_telemetry_history
```

每次轮询完成后追加一条记录：

| 字段 | 说明 |
| --- | --- |
| `id` | GORM 生成的主键 |
| `cluster_id` | 集群 ID |
| `collection_id` | Agent 成功采样 ID；失败采样为空 |
| `status` | `success` 或 `error` |
| `health_status` | 本次采样计算出的集群健康状态 |
| `schema_version` | 成功采样的遥测协议版本 |
| `normalized_payload` | 成功采样的标准化遥测 JSON |
| `error_code` | 失败采样的脱敏错误码 |
| `collected_at` | 采样时间，Unix 秒 |
| `created_at` | 数据库写入时间，Unix 秒 |

历史表不保存 Agent 地址、Bearer Token、连接密钥密文、原始 Agent 响应或诊断载荷。

索引：

- `(cluster_id, collected_at, id)`：集群时间范围查询；
- `(collected_at, id)`：过期数据清理；
- `(cluster_id, collection_id)`：成功采样去重。失败采样的 `collection_id` 为 `NULL`。

## 3. 轮询写入

成功轮询在同一数据库事务中：

1. 追加 `success` 历史记录；
2. 更新 `cluster_telemetry_latest`；
3. 更新 `clusters` 的健康状态、凭据状态和下一次轮询时间；
4. 释放轮询租约。

失败轮询在同一数据库事务中：

1. 追加 `error` 历史记录，只保存安全错误码；
2. 更新连续失败次数、健康状态和下一次退避时间；
3. 释放轮询租约。

删除集群时，同时删除该集群的最新快照和全部历史记录。

## 4. 保留天数设置

新增系统选项：

```text
ClusterTelemetryRetentionDays
```

位置：

```text
系统设置 → 控制台内容 → 数据仪表盘
```

规则：

- 默认值：7 天；
- 最小值：1 天；
- 最大值：365 天；
- 后端和前端同时校验整数范围；
- 设置保存在主数据库 `options` 表，服务重启后继续生效。

以 5 秒轮询为例，每个集群每天最多产生 17,280 条记录，7 天最多产生 120,960 条记录。实际数量会受到失败退避、手动刷新和轮询间隔调整影响。

## 5. 过期清理

集群轮询主节点同时运行历史清理协程：

- 服务启动后立即清理一次；
- 此后每小时清理一次；
- 管理员修改保留天数后立即唤醒清理任务；
- 截止时间为 `当前时间 - 保留天数`；
- 每批查询并删除最多 5,000 个主键，避免长事务；
- 不使用数据库专属的 `DELETE LIMIT`，兼容 SQLite、MySQL 和 PostgreSQL。

缩短保留天数会删除超期数据；增加保留天数不会恢复已经删除的数据。

## 6. 历史导出接口

新增管理员接口：

```http
GET /api/clusters/export/history
```

参数：

| 参数 | 说明 |
| --- | --- |
| `scope` | `all` 或 `cluster` |
| `start_at` | RFC3339 开始时间，包含该秒 |
| `end_at` | RFC3339 结束时间，不包含该秒 |
| `search` | 总览页搜索条件 |
| `model_id` | 模型筛选 |
| `cluster_id` | 单集群导出时必填 |
| `status` | 集群当前健康状态筛选 |

时间范围采用左闭右开区间：

```text
[start_at, end_at)
```

这样连续导出相邻时间段时不会重复计算边界采样。

单次导出限制：

- 最多 10,000 个集群；
- 最多 1,000,000 个历史采样点；
- 时间跨度不能超过当前配置的历史保留天数。

历史导出按 `(collected_at, id)` 游标每批读取 2,000 条，逐个 ZIP 文件流式写出，不在服务端内存中构造完整历史数组。客户端取消请求后，数据库查询随请求上下文结束。

## 7. ZIP 文件结构

```text
manifest.json
clusters.csv
telemetry_history.csv
gpu_device_history.csv
engine_load_history.csv
normalized_telemetry_history.jsonl
```

- `manifest.json`：导出版本、UTC 时间窗口、左闭右开语义、筛选条件、保留天数和记录数；
- `clusters.csv`：导出时的集群元数据；
- `telemetry_history.csv`：每个采样点一行，包含成功和失败采样；
- `gpu_device_history.csv`：成功采样中的 GPU 设备明细；
- `engine_load_history.csv`：成功采样中的引擎负载明细；
- `normalized_telemetry_history.jsonl`：完整标准化历史，每行一个采样点。

请求数和 Token 数保留 Agent 声明的 `cumulative`、`current_inflight`、`current_usage` 或 `unknown` 语义，不跨采样点直接求和。

## 8. 前端交互

原导出弹窗新增数据来源切换：

- 最新快照：保留原有 CSV、ZIP、JSON 导出；
- 时间范围：导出历史 ZIP。

时间范围支持：

- 年、月、日、时、分、秒；
- 最近 15 分钟、1 小时、6 小时、24 小时、7 天快捷范围；
- 浏览器本地时区展示；
- 请求时转换为精确到秒的 UTC RFC3339；
- 显示数据库中最早可用的历史时间；
- 校验开始时间、结束时间、可用范围和保留天数。

本阶段提供英文和简体中文文案；其他语言保留英文回退值，并确保所有语言文件不存在缺失键。

## 9. 兼容性与迁移

- `cluster_telemetry_history` 通过 GORM `AutoMigrate` 创建；
- 载荷使用 `TEXT`，不依赖 PostgreSQL `JSONB`；
- 查询和清理使用 GORM，兼容 SQLite、MySQL 5.7.8+、PostgreSQL 9.6+；
- 升级后从首次轮询开始积累历史，无法从最新快照反向补齐部署前数据；
- 历史数据位于 PostgreSQL 等主数据库，不写入 Redis。

## 10. 测试与验证

后端覆盖：

- 保留天数边界校验；
- 成功和失败轮询写入；
- 删除集群级联清理历史；
- 时间范围左闭右开；
- 同一秒多记录的稳定游标分页；
- 分批过期清理；
- ZIP 文件结构、时间边界和敏感信息排除；
- 超过保留期限的导出拒绝；
- 历史导出路由管理员鉴权。

前端覆盖：

- 秒级 RFC3339 参数生成；
- 总览、模型和集群页面范围映射；
- 反向时间、超保留时间和早于可用历史的校验；
- 类型检查、受影响文件 lint 和生产构建。
