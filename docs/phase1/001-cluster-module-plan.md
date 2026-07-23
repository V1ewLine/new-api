# 集群状态模块实施计划

## 日期

2026-07-23

## 本阶段目标

在编码前确认 New API 的技术栈、权限、数据源、迁移方式和 Agent 协议，确定第一阶段的实现边界。

## 实际修改

完成项目审计并确定实现方案，尚未在本阶段写入业务代码。

## 新增或修改的文件

- `docs/phase1/001-cluster-module-plan.md`

## 数据库与接口变更

计划新增 `clusters` 与 `cluster_telemetry_latest` 两张表。

计划新增管理员接口：

- `GET /api/clusters/overview`
- `GET /api/clusters/model-options`
- `POST /api/clusters/`
- `GET /api/clusters/models/:modelId`
- `GET /api/clusters/:clusterId`
- `GET /api/clusters/:clusterId/telemetry/latest`
- `GET /api/clusters/:clusterId/telemetry/history`
- `POST /api/clusters/:clusterId/refresh`

## 关键设计决策

- 后端沿用 Gin、GORM 和 `middleware.AdminAuth()`。
- 前端沿用 React 19、TanStack Router、React Query、Base UI、Tailwind CSS 和现有主题变量。
- 模型复用 `models` 表的稳定 `id`，新建集群仅允许选择启用模型，同时保存模型名称快照。
- 数据库迁移沿用 `model/main.go` 中的 GORM `AutoMigrate`，兼容 SQLite、MySQL 和 PostgreSQL。
- Agent 当前没有 Enrollment Key，第一阶段由独立 `ClusterLinkResolver` 解析临时 `sgta1.` 不透明连接密钥。
- 密钥由独立 `SecretProtector` 使用 `CRYPTO_SECRET` 加密，API 永不返回原文。
- 现有 SystemTask 调度最小周期为 15 秒，不适合默认 5 秒采集，因此使用独立且支持优雅停止的轮询器。

## 遇到的问题

- Agent 仅定义 Agent URL 和 Bearer Token，没有统一连接密钥。
- 项目没有现成的可逆 Secret 加密组件。
- 当前迁移机制没有 down migration。

## 解决方式

- 将临时连接格式限制在 Resolver 内，前端和 Controller 均不解析。
- 新建隔离的 AES-GCM SecretProtector，密钥派生自已有 `CRYPTO_SECRET`。
- 迁移只新增表，不破坏现有结构；旧版本回滚后新表保留但不会被使用。

## 测试与验证

计划覆盖密钥保护、Agent Client、Schema Adapter、健康状态、轮询互斥、模型联动、API 权限和前端主要状态。

## 当前限制

- 第一阶段只保存最新遥测。
- 历史接口只保留稳定边界。
- 不实现完整告警历史、通知和时序分析。

## 下一步

实现数据库模型、Agent 接入、轮询调度和管理员 API。
