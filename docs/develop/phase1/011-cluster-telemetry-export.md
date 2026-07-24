# 集群状态最新遥测数据导出

## 1. 目标

在集群状态一级、二级和三级页面提供与页面粒度一致的数据导出能力，便于离线分析、问题排查和交接。

本阶段只导出 `cluster_telemetry_latest` 中已经持久化的最新快照，不主动请求远端 Agent，也不新增历史表。

## 2. 页面与导出粒度

| 页面 | 导出项 | 格式 | 数据粒度 |
| --- | --- | --- | --- |
| 集群状态总览 | 模型汇总 | CSV | 当前搜索、模型和状态筛选条件下，每个模型一行 |
| 集群状态总览 | 完整快照 | ZIP | 当前筛选条件下的模型、集群、GPU、引擎负载和标准化遥测 |
| 模型集群详情 | 集群列表 | CSV | 当前模型下每个集群一行 |
| 集群详情 | 完整集群快照 | ZIP | 当前集群的集群记录、GPU、引擎负载和标准化遥测 |
| 集群详情 | 标准化快照 | JSON | 当前集群的完整标准化最新快照 |

总览导出不受页面分页影响，会导出所有符合当前筛选条件的记录。

## 3. 后端接口

新增管理员接口：

```http
GET /api/clusters/export/latest
```

查询参数：

| 参数 | 可选值或格式 | 说明 |
| --- | --- | --- |
| `scope` | `models`、`clusters`、`cluster`、`all` | 导出粒度 |
| `format` | `csv`、`zip`、`json` | 文件格式 |
| `search` | 字符串 | 集群或模型搜索条件 |
| `model_id` | 正整数 | 模型筛选 |
| `cluster_id` | 正整数 | 单集群导出时必填 |
| `status` | `unknown`、`online`、`partial`、`abnormal`、`offline` | 健康状态筛选 |

接口位于管理员鉴权路由组中，并在成功生成导出内容后写入 `cluster.export` 管理审计日志。

响应设置 `Content-Disposition`、`Cache-Control: no-store` 和 `X-Content-Type-Options: nosniff`。

## 4. ZIP 文件结构

```text
manifest.json
models.csv
clusters.csv
gpu_devices.csv
engine_loads.csv
normalized_telemetry.jsonl
```

- `manifest.json`：导出版本、时间、范围、筛选条件和文件清单。
- `models.csv`：模型聚合指标。
- `clusters.csv`：集群最新运行状态和标准化指标。
- `gpu_devices.csv`：每块 GPU 一行，支持动态 GPU 数量。
- `engine_loads.csv`：每条 Agent 引擎负载记录一行。
- `normalized_telemetry.jsonl`：每个存在遥测数据的集群一行 JSON。

CSV 使用 UTF-8 BOM，便于 Excel 直接识别；缺失指标输出空单元格，不用 `0` 代替。文本字段对 `= + - @` 开头的公式注入内容增加单引号前缀。

## 5. 指标语义

为请求数和 Token 数新增语义字段：

- `cumulative`：Agent 明确返回累计值。
- `current_inflight`：请求数由当前运行中与等待中请求相加得到。
- `current_usage`：Token 数来自当前负载或占用值。
- `unknown`：旧版已持久化快照包含数值，但当时没有记录语义。
- `unavailable`：CSV 导出中表示该指标没有值。

这可以避免将当前负载误当作累计业务量。

## 6. 安全与容量限制

导出内容不包含：

- Agent Bearer Token；
- Agent 地址或连接密钥密文；
- 原始 Agent 响应；
- `last_failure_payload` 诊断载荷；
- 轮询锁和调度等内部字段。

当前限制：

- 单次最多导出 10,000 个集群；
- GPU 设备行与引擎负载行合计最多 100,000 行。

超过限制时接口返回 `cluster export exceeds the allowed size`。

## 7. 前端交互

- 三级页面均保留原有刷新操作，并新增“导出”按钮。
- 点击后先选择当前页面允许的导出内容，再由浏览器下载后端生成的文件。
- 下载文件名从 `Content-Disposition` 读取，解析失败时使用稳定的兜底文件名。
- 成功和失败状态通过 Toast 提示。
- 本阶段完成英文和简体中文文案；其他语言键保留英文回退值，避免运行时缺键。

## 8. 主要改动文件

- `service/clusterstatus/export.go`
- `service/clusterstatus/types.go`
- `service/clusterstatus/schema_adapter.go`
- `controller/cluster.go`
- `controller/audit.go`
- `router/api-router.go`
- `web/src/features/cluster-status/api.ts`
- `web/src/features/cluster-status/types.ts`
- `web/src/features/cluster-status/lib/export.ts`
- `web/src/features/cluster-status/components/cluster-export-dialog.tsx`
- `web/src/features/usage-logs/lib/format.ts`
- 集群状态三级页面及中英文国际化文件

## 9. 回归测试

- 后端覆盖筛选条件、全量而非分页导出、空值、CSV 公式注入、ZIP 文件结构、动态 GPU 行、敏感信息排除和旧数据指标语义。
- 路由测试覆盖导出接口管理员鉴权。
- 前端测试覆盖 `Content-Disposition` 文件名解析和异常回退。
