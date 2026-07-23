# 集群二三级 Naive 页面

## 日期

2026-07-23

## 本阶段目标

完成一级页面之后的模型集群列表和单集群详情导航，使后续指标扩展可以复用稳定路由与数据结构。

## 实际修改

- 新增模型集群列表路由，展示模型信息、五项摘要和真实集群表格。
- 新增集群详情路由，只保留总览、引擎指标、机器指标三个 Tab。
- 总览展示健康状态、Agent identity、模型关联、时间对齐和最近采集信息。
- 引擎页展示当前运行/等待请求、Token 使用、吞吐、缓存和响应时间。
- 机器页按 Agent 返回的 GPU 数组动态展示 0、1、2、8 或更多 GPU，并展示 CPU、内存和 Load Average。
- 增加模型不可用、模型 mismatch、等待遥测、刷新中、加载和错误状态。

## 新增或修改的文件

- `web/src/features/cluster-status/model-detail.tsx`
- `web/src/features/cluster-status/cluster-detail.tsx`
- `web/src/routes/_authenticated/cluster-status/models/$modelId.tsx`
- `web/src/routes/_authenticated/cluster-status/$clusterId.tsx`
- `web/src/routeTree.gen.ts`

## 数据库与接口变更

页面使用：

- `GET /api/clusters/models/:modelId`
- `GET /api/clusters/:clusterId`
- `GET /api/clusters/:clusterId/telemetry/latest`
- `GET /api/clusters/:clusterId/telemetry/history`
- `POST /api/clusters/:clusterId/refresh`

历史接口当前返回 `available: false` 与空列表。

## 关键设计决策

- 二三级页面直接消费后端规范化结构，不读取 Agent 原始 JSON。
- 手动刷新与后台采集共用 `Service.PollCluster` 和数据库租约。
- 最后一次成功遥测会在新轮询失败时保留，健康状态和错误码独立更新。
- 模型被禁用或软删除后详情仍可打开，并显示不可用警告。

## 遇到的问题

- 部分 SGLang 版本不会在 `loads` 中提供所有展示指标。
- 当前阶段没有历史仓库，参考图的图表无法从真实数据生成。

## 解决方式

- 所有可选指标缺失时统一显示 `—`，保留原始 `loads` 以便后续 Adapter 扩展。
- 页面明确显示“第一阶段未启用历史趋势”，不生成 Mock 时间序列。

## 测试与验证

- TanStack Router 构建时已生成两条类型安全路由。
- 路由级 `beforeLoad` 校验管理员角色；后端接口再次使用 `AdminAuth`。
- 动态 GPU 适配覆盖 0、1、2、8 张 GPU。

## 当前限制

- 不支持历史图表、导出、集群编辑和删除。
- 引擎字段的实际可用性取决于 SGLang `/v1/loads` 版本。

## 下一步

执行完整测试、构建和文档交接，并记录部署配置与已知限制。
