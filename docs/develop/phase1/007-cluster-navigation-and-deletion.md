# 集群导航与删除能力

## 日期

2026-07-24

## 本阶段目标

优化集群详情页的返回入口，并为填写错误或不再使用的集群提供完整、安全的删除能力。

## 实际修改

- 模型集群列表页和单集群详情页的返回入口统一改为“左箭头 + 返回”。
- 模型集群列表的每一行新增删除按钮。
- 单集群详情页操作区新增删除按钮。
- 删除操作使用 `AlertDialog` 二次确认，并在请求期间禁用取消和重复提交。
- 删除成功后刷新集群概览和模型集群列表；从单集群详情删除时返回所属模型的集群列表。
- 后端新增管理员删除集群接口，在事务中同时删除集群配置和最新遥测。
- 删除不存在的集群统一返回 `cluster not found`，避免误报成功。
- 新增删除确认文案的七语言翻译。

## 新增或修改的文件

- `model/cluster.go`
- `service/clusterstatus/service.go`
- `service/clusterstatus/service_test.go`
- `controller/cluster.go`
- `router/api-router.go`
- `router/cluster_router_test.go`
- `web/src/features/cluster-status/api.ts`
- `web/src/features/cluster-status/query-keys.ts`
- `web/src/features/cluster-status/model-detail.tsx`
- `web/src/features/cluster-status/cluster-detail.tsx`
- `web/src/features/cluster-status/components/delete-cluster-dialog.tsx`
- `web/src/features/cluster-status/__tests__/query-invalidation.test.ts`
- `web/src/i18n/locales/{en,zh,zh-TW,fr,ja,ru,vi}.json`

## 数据库与接口变更

新增管理员接口：

```http
DELETE /api/clusters/:clusterId
```

接口继续使用 `middleware.AdminAuth()`。删除过程使用 GORM 事务，按集群 ID 删除：

- `clusters` 中的集群配置；
- `cluster_telemetry_latest` 中对应的最新遥测。

本次不新增数据表或字段，不需要额外迁移。

## 关键设计决策

- 返回入口使用明确的文字和方向图标，不再依赖页面上下文推断链接用途。
- 删除属于不可恢复操作，所有前端入口都必须经过二次确认。
- 删除确认框显示集群名称，并明确最新遥测会一并删除。
- 删除按钮复用项目现有 Hugeicons、Button、AlertDialog、Spinner 和 Sonner 反馈体系。
- 前端仅刷新概览和模型详情查询，避免已删除的单集群详情在跳转前被无意义地重新请求。
- 后端事务保证集群配置与最新遥测不会只删除一部分。

## 遇到的问题

- 集群轮询器可能正在处理即将删除的集群。
- 当前本地执行环境没有可直接调用的 Bun 命令。

## 解决方式

- 删除集群后，正在进行的轮询无法再更新对应集群行；轮询保存事务会因租约行不存在而回滚，不会重新产生孤立的最新遥测。
- i18n 和前端校验使用相同 `package.json` 脚本的 npm 等价命令执行，部署与日常开发仍以 Bun 为首选。

## 测试与验证

新增回归测试覆盖：

- 删除集群时同时删除配置和最新遥测；
- 重复删除返回集群不存在；
- 删除接口已注册且未认证请求被管理员鉴权拒绝。

通过：

```text
Go test:
  go test ./service/clusterstatus ./controller ./router

Go build:
  go build ./...

Frontend typecheck:
  tsgo -b

Frontend lint:
  oxlint（本次涉及的集群状态文件）

Frontend unit tests:
  node --experimental-strip-types --test（10 项通过）

Frontend format and copyright:
  format:check
  copyright:check

Frontend production build:
  rsbuild build

i18n:
  i18n:sync（七种 locale 均无缺失、额外或未翻译项）
```

## 当前限制

- 第一阶段仍只保存最新遥测，因此删除范围只有集群配置和 `cluster_telemetry_latest`。
- 若后续增加遥测历史表，删除事务需要同步扩展历史数据清理。

## 下一步

- 根据后续需求增加集群编辑能力，使 Agent 地址或 Token 变更时不必删除后重建。
- 若增加历史遥测，明确删除、归档和审计保留策略。
