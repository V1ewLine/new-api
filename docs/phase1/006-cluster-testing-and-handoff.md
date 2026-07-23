# 集群模块测试与交接

## 日期

2026-07-23

## 本阶段目标

完成第一阶段最终验证，记录数据库、API、配置、测试结果、限制和后续扩展入口。

## 实际修改

- 补充后端 Secret、Agent Client、Schema Adapter、健康状态、动态 GPU、模型联动、筛选分页和连续失败测试。
- 补充前端指标格式、GPU 平均值、分页窗口和状态优先级测试。
- 完成七种前端语言同步、版权头、格式、lint、typecheck、生产构建和 Go 构建检查。
- 使用全新 SQLite 数据库启动最终二进制，验证迁移、首页、管理员 API 权限和优雅停止。

## 新增或修改的文件

后端与生命周期：

- `main.go`
- `model/main.go`
- `model/cluster.go`
- `service/clusterstatus/types.go`
- `service/clusterstatus/security.go`
- `service/clusterstatus/agent_client.go`
- `service/clusterstatus/schema_adapter.go`
- `service/clusterstatus/health.go`
- `service/clusterstatus/service.go`
- `service/clusterstatus/poller.go`
- `controller/cluster.go`
- `router/api-router.go`

后端测试：

- `service/clusterstatus/security_test.go`
- `service/clusterstatus/agent_client_test.go`
- `service/clusterstatus/schema_adapter_test.go`
- `service/clusterstatus/service_test.go`

前端：

- `web/src/features/cluster-status/`
- `web/src/routes/_authenticated/cluster-status/`
- `web/src/hooks/use-sidebar-data.ts`
- `web/src/routeTree.gen.ts`
- `web/src/i18n/locales/en.json`
- `web/src/i18n/locales/zh.json`
- `web/src/i18n/locales/zh-TW.json`
- `web/src/i18n/locales/fr.json`
- `web/src/i18n/locales/ja.json`
- `web/src/i18n/locales/ru.json`
- `web/src/i18n/locales/vi.json`

开发日志：

- `docs/phase1/001-cluster-module-plan.md`
- `docs/phase1/002-cluster-backend-and-storage.md`
- `docs/phase1/003-cluster-telemetry-poller.md`
- `docs/phase1/004-cluster-status-page.md`
- `docs/phase1/005-cluster-naive-detail-pages.md`
- `docs/phase1/006-cluster-testing-and-handoff.md`

## 数据库与接口变更

全新 SQLite 冒烟数据库已确认创建：

- `clusters`
- `cluster_telemetry_latest`

两张表通过现有 GORM `AutoMigrate` 加入普通和快速迁移路径。模型使用稳定 `models.id` 关联，不设置级联删除。

管理员 API：

- `GET /api/clusters/overview`
- `GET /api/clusters/model-options`
- `POST /api/clusters/`
- `GET /api/clusters/models/:modelId`
- `GET /api/clusters/:clusterId`
- `GET /api/clusters/:clusterId/telemetry/latest`
- `GET /api/clusters/:clusterId/telemetry/history`
- `POST /api/clusters/:clusterId/refresh`

全部路由使用 `middleware.AdminAuth()`。未登录冒烟请求返回 HTTP 401 和 `AUTH_UNAUTHORIZED`。

轮询配置：

- `CLUSTER_TELEMETRY_POLL_INTERVAL_SECONDS`，默认 `5`
- `CLUSTER_TELEMETRY_REQUEST_TIMEOUT_SECONDS`，默认 `3`
- `CLUSTER_TELEMETRY_MAX_CONCURRENCY`，默认 `8`
- `CLUSTER_TELEMETRY_FAILURE_THRESHOLD`，默认 `3`
- `CLUSTER_TELEMETRY_MAX_BODY_BYTES`，默认 `2097152`
- `CLUSTER_TELEMETRY_LEASE_TTL_SECONDS`，默认请求超时加 `5`
- `CLUSTER_TELEMETRY_MAX_BACKOFF_SECONDS`，默认 `300`

Secret 加密使用项目现有 `CRYPTO_SECRET`。修改 `CRYPTO_SECRET` 会导致已有集群连接密钥无法解密，部署时必须保持稳定。

## 关键设计决策

- 临时连接密钥格式是 `sgta1.<base64url-json>`，JSON 只包含 `base_url` 和 `bearer_token`；协议仅存在于 `TemporaryLinkResolver`，后续可替换为 Agent Enrollment Key。
- 管理员可以用下面的本地命令生成临时密钥，Token 通过隐藏输入读取，不会写入命令行历史：

```bash
python3 - <<'PY'
import base64
import getpass
import json

base_url = input("Agent base URL: ").strip()
bearer_token = getpass.getpass("Agent bearer token: ").strip()
payload = json.dumps(
    {"base_url": base_url, "bearer_token": bearer_token},
    separators=(",", ":"),
).encode()
print("sgta1." + base64.urlsafe_b64encode(payload).rstrip(b"=").decode())
PY
```

- 密钥使用 AES-GCM 加密保存，数据库列和诊断字段均不参与 API JSON 序列化。
- Agent 失败保留最后一次成功遥测；schema 失败的受限原始响应只用于内部诊断。
- 轮询器使用数据库租约和进程内去重，手动刷新与后台轮询共用同一路径。
- 前端按规范化遥测动态渲染 GPU，不依赖固定 GPU 数量。
- shadcn/Base UI 技能约束使页面复用现有组合组件和语义主题；React 最佳实践使一级页使用单个聚合查询并保留后台刷新旧数据；i18n 流程同步了七种 locale。

## 遇到的问题

- 当前 Codex 运行环境没有暴露用户 zsh 中的 Bun 可执行文件。
- 当前会话没有提供内置浏览器控制接口，无法执行交互式截图检查。
- 普通沙箱不允许现有部分 Go 测试和本地服务打开回环端口。

## 解决方式

- 前端使用同一 `package.json` 脚本的 npm 等价命令完成 typecheck、lint、测试和 Rsbuild 生产构建；项目交付命令仍以 Bun 为准。
- 使用本地二进制、curl 和 SQLite 完成非浏览器集成验证。
- 仅对测试和临时本地服务使用允许回环端口的受控执行环境，验证结束后发送中断信号，轮询器和 HTTP 服务均优雅停止。

## 测试与验证

通过：

```text
Go test:
  go test ./service/clusterstatus ./model ./controller ./router

Go build:
  go build ./...

Frontend typecheck:
  tsgo -b

Frontend lint:
  oxlint（集群模块、路由和关联侧边栏文件）

Frontend tests:
  node --test ...
  8 tests passed

Frontend build:
  rsbuild build

i18n:
  sync-i18n.mjs

Format and copyright:
  format:check
  copyright:check
```

本地启动冒烟结果：

```text
GET /                              -> 200
GET /api/clusters/overview         -> 401（未登录，符合管理员权限）
SQLite clusters                    -> 存在
SQLite cluster_telemetry_latest    -> 存在
SIGINT                             -> 轮询器与服务优雅停止
```

常用运行命令：

```bash
# 前端
cd web
bun install
bun run typecheck
bun run lint
bun run build
cd ..

# 后端
go test ./service/clusterstatus ./model ./controller ./router
go build -o ./bin/new-api-local .
./bin/new-api-local --port 3000 --log-dir /data/logs
```

## 当前限制

- 只支持 `sglang_telemetry_agent` schema `1.0`。
- 只持久化最后一次成功遥测，不提供历史时序、长期保留和历史告警。
- 二三级页面是 Naive 版本，不含真实趋势图和导出。
- 不提供集群编辑、删除、密钥替换或连接测试接口。
- 临时 `sgta1.` 格式应在 Agent 提供正式 Enrollment Key 后替换。
- SQLite 已完成实际迁移冒烟；MySQL 和 PostgreSQL 本次未连接真实实例执行迁移。
- 当前会话未完成浏览器交互式视觉验收，已完成类型、构建、路由生成和 HTTP 冒烟验证。

## 下一步

- 新 schema：在 `service/clusterstatus/` 增加新的 `TelemetrySchemaAdapter`。
- 新遥测来源：实现 `TelemetryAgentClient` 或新的 Resolver。
- 历史数据：替换 `EmptyHistoryRepository` 并启用历史接口。
- 新健康规则：替换 `ClusterHealthEvaluator`，页面状态组件无需读取 Agent 原始结构。
- 完整二三级页面：复用 `GET /api/clusters/models/:modelId`、`GET /api/clusters/:clusterId` 和规范化 telemetry 类型。
- 正式连接密钥：替换 `TemporaryLinkResolver`，无需修改 Controller、数据库响应或前端表单。
