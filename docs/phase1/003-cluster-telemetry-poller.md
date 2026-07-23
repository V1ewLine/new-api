# 集群遥测轮询器

## 日期

2026-07-23

## 本阶段目标

完成 New API 到 `sglang_telemetry_agent` 的主动采集链路。

## 实际修改

- 新增带 Bearer 认证、Content-Type 校验、超时和响应体上限的 Agent HTTP Client。
- 新增 Agent schema `1.0` 适配器。
- 新增统一健康状态评估器。
- 新增支持最大并发、数据库租约、失败退避、抖动、手动刷新和优雅停止的轮询器。
- 在 New API 启动和关闭流程中注册轮询器生命周期。

## 新增或修改的文件

- `service/clusterstatus/agent_client.go`
- `service/clusterstatus/schema_adapter.go`
- `service/clusterstatus/health.go`
- `service/clusterstatus/poller.go`
- `main.go`

## 数据库与接口变更

`clusters` 表中的 `next_poll_at`、`poll_locked_by` 和 `poll_locked_until` 用于跨实例互斥。手动刷新使用：

```http
POST /api/clusters/:clusterId/refresh
```

## 关键设计决策

- 轮询间隔、请求超时、并发、失败阈值、响应体上限、租约和最大退避均从统一环境配置读取。
- 后台轮询和手动刷新共用 `Service.PollCluster`。
- HTTP 和安全错误只保存稳定错误码；schema 解析失败会额外保存受大小限制的 Agent 原始载荷供管理员侧诊断，但不会记录或返回 Secret。
- `identity.model` 不一致只产生 mismatch 和异常状态，不修改模型关联。
- 私网、域名、端口、重定向和 DNS 重绑定策略复用 New API 的 Fetch Setting 与 SSRF Protected Client。

## 遇到的问题

Agent 的 `engine.loads` 字段由 SGLang 版本决定，部分计数和吞吐指标可能不存在。

## 解决方式

适配器保留完整 loads，并仅对已知且实际存在的字段生成可选规范化指标；缺失数据保持为空，不伪造零值。

## 测试与验证

已补充 Bearer Header、超时、超大响应、未知 schema、partial、mismatch、动态 GPU 数量、健康状态和连续失败退避测试。

## 当前限制

- 只支持 Agent schema `1.0`。
- 不保存历史趋势。

## 下一步

实现一级、二级和三级前端页面及添加集群 Dialog。
