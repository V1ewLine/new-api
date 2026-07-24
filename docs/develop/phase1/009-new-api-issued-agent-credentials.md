# New API 签发并管理 Agent Token

## 日期

2026-07-24

## 本阶段目标

将集群接入流程调整为由 New API 生成 Agent Bearer Token。管理员只需要选择模型、填写集群名称和 Agent 的 IP + 端口，再把 New API 一次性展示的 Token 配置到远端 Agent，不再由管理员预先创建并回填 Token。

同时补齐凭据待配置、连接验证和 Token 轮换流程，使 New API 数据库中的集群配置、凭据状态与实际 Agent 连接状态保持一致。

## 实际修改

- 新建集群时删除“Agent Bearer Token”输入框，只保留模型、集群名称和 Agent IP + 端口。
- New API 使用密码学安全随机数生成 `napi_agent_` 前缀的 32 字节随机 Token。
- 创建成功后只在本次响应和弹窗中展示一次 Token，同时提供：
  - 单独复制 Token；
  - 复制 `AGENT_API_TOKEN='...'` 配置行；
  - 远端 Agent 配置与重启步骤；
  - 连接测试入口。
- Token 明文不会写入普通集群查询响应、日志或浏览器持久化存储。
- 新建集群默认进入“等待配置”状态；只有 Agent 使用该 Token 成功返回遥测后，凭据状态才变为“已激活”。
- 集群详情页增加“测试连接”和“轮换 Token”操作。
- 轮换会立即生成新 Token、替换 New API 保存的旧凭据，并将集群重新置为“等待配置”。
- 关闭一次性 Token 弹窗前增加确认提示，明确关闭后不能再次查看原 Token。
- 对 401、连接超时、不可达、出站策略拦截、遥测结构不兼容和本地密钥不可用提供不同的验证结果提示。
- 尚未完成 Agent 配置的集群不计入异常集群数量，也不生成异常告警。
- 新增七种前端语言的完整翻译。

## 新增或修改的文件

- `model/cluster.go`
- `model/main.go`
- `service/clusterstatus/security.go`
- `service/clusterstatus/security_test.go`
- `service/clusterstatus/service.go`
- `service/clusterstatus/service_test.go`
- `service/clusterstatus/types.go`
- `controller/cluster.go`
- `router/api-router.go`
- `router/cluster_router_test.go`
- `web/src/features/cluster-status/api.ts`
- `web/src/features/cluster-status/types.ts`
- `web/src/features/cluster-status/cluster-detail.tsx`
- `web/src/features/cluster-status/model-detail.tsx`
- `web/src/features/cluster-status/components/add-cluster-dialog.tsx`
- `web/src/features/cluster-status/components/agent-credential-panel.tsx`
- `web/src/features/cluster-status/components/rotate-cluster-credential-dialog.tsx`
- `web/src/features/cluster-status/components/status-badge.tsx`
- `web/src/features/cluster-status/lib/cluster-form.ts`
- `web/src/features/cluster-status/lib/credential.ts`
- `web/src/features/cluster-status/__tests__/credential-flow.test.ts`
- `web/src/i18n/locales/{en,zh,zh-TW,fr,ja,ru,vi}.json`

## 数据库变更

`clusters` 表新增以下字段，由 GORM 自动迁移：

| 字段 | 用途 |
| --- | --- |
| `credential_status` | 凭据状态，值为 `pending` 或 `active` |
| `credential_version` | Token 版本，首次签发为 1，每次轮换加 1 |
| `credential_issued_at` | 当前 Token 的签发时间 |
| `credential_verified_at` | 当前 Token 最近一次验证成功的时间 |

历史集群在迁移时按以下规则初始化：

- `last_success_at > 0`：设置为 `active`，并使用 `last_success_at` 初始化验证时间；
- 从未成功返回遥测：设置为 `pending`，健康状态重置为 `unknown`；
- 版本缺失时设置为 1；
- 签发时间缺失时使用集群创建时间。

初始化过程可重复执行，并使用 GORM 通用表达式，兼容 SQLite、MySQL 和 PostgreSQL。

## 接口变更

创建集群请求不再接收 Token：

```http
POST /api/clusters/
Content-Type: application/json

{
  "model_id": 1,
  "name": "GLM-4.7-Flash",
  "agent_address": "10.251.178.92:31000"
}
```

创建成功响应新增一次性 `bootstrap_token`：

```json
{
  "success": true,
  "data": {
    "cluster": {
      "id": 1,
      "credential_status": "pending",
      "credential_version": 1
    },
    "bootstrap_token": "napi_agent_..."
  }
}
```

新增管理员接口：

```http
POST /api/clusters/:clusterId/credential/verify
POST /api/clusters/:clusterId/credential/rotate
```

- `verify` 立即执行一次受超时和 SSRF 策略保护的 Agent 请求，并返回 `verified`、`error_code` 和最新集群状态。
- `rotate` 返回新的 `bootstrap_token`，New API 随即停止使用旧 Token 发起 Agent 请求。
- 创建和轮换响应设置 `Cache-Control: no-store` 与 `Pragma: no-cache`。
- 所有接口继续使用 `middleware.AdminAuth()`。

## UI 操作流程

1. 管理员选择模型并填写集群名称、Agent IP + 端口。
2. 点击“创建并生成 Token”。
3. 复制 Token 或完整的 `AGENT_API_TOKEN` 配置行。
4. 在远端 `sglang_telemetry_agent` 的 `.env` 或服务环境中更新：

   ```bash
   AGENT_API_TOKEN='napi_agent_...'
   ```

5. 重启远端 Agent，使新环境变量生效。
6. 回到 New API 点击“测试连接”。
7. 验证成功后，集群从“等待配置”切换为正常健康状态。

若 Token 遗失或配置错误，可在集群详情页生成新 Token。生成后必须重新更新并重启 Agent。

## 安全与兼容设计

- Token 使用 `crypto/rand` 生成，不依赖时间、集群 ID 或可预测随机数。
- 数据库仍只保存包含 Token 的 AES-GCM 密文；普通接口不会返回密文或 Token。
- 创建和轮换响应是唯一返回 Token 明文的位置。
- 前端只把明文保存在当前 React 组件状态中，不写入 `localStorage`、`sessionStorage` 或 URL。
- Agent 协议本身不需要修改，仍使用现有的 `Authorization: Bearer <AGENT_API_TOKEN>`。
- Agent 地址创建和轮换时均执行原有出站请求策略校验，实际验证和轮询时再次校验。
- Token 轮换在 New API 侧采用单版本立即替换，不保留旧 Token 的并行宽限期；远端 Agent 仍需通过更新环境变量并重启来完成切换。

如果旧集群已经出现 `CLUSTER_SECRET_INVALID`，New API 无法从旧密文中恢复 Agent 地址和 Token，因此也不能直接轮换；需要删除并重新创建该集群。后续由稳定密钥加密的集群可正常轮换。

## 测试与验证

新增或更新的回归测试覆盖：

- 生成 Token 的前缀、随机长度和唯一性；
- 创建集群无需用户提供 Token；
- 创建响应只展示一次 Token，数据库和普通查询不泄露明文；
- 轮换后 Token 和密文变化、版本递增、状态回到 `pending`；
- 验证失败返回结构化错误码且不泄露 Token；
- 待配置期间连接失败仍保持 `unknown` 健康状态；
- 验证成功将凭据状态切换为 `active`；
- 待配置集群不计入异常统计和告警；
- 历史集群凭据字段迁移及重复执行；
- 新增接口受管理员鉴权保护；
- 前端表单、环境变量格式、待配置状态和验证错误映射。

通过：

```text
Go test:
  go test ./service/clusterstatus ./router

Go build:
  go build ./...

Frontend unit tests:
  node --experimental-strip-types --test（14 项通过）

Frontend typecheck and lint:
  tsgo -b
  oxlint

Frontend format and copyright:
  format:check
  copyright:check

Frontend production build:
  rsbuild build

i18n:
  i18n:sync（七种 locale 的缺失、额外和未翻译项均为 0）
```

本地尝试运行整个 `model` 测试包时，缺失的 `miniredis` 测试依赖因当前网络超时未能下载；相关迁移测试放入已具备 SQLite 测试环境的 `service/clusterstatus` 测试包并已通过，不影响生产构建。

## 当前限制

- Token 只能在创建或轮换成功后查看一次，New API 不提供找回明文功能。
- 轮换后 New API 立即改用新 Token，必须尽快更新并重启 Agent，否则集群连接会保持失败。
- 当前只能轮换 Token，不能直接编辑已保存的 Agent 地址；地址填写错误时可删除集群后重新创建。
- 第一阶段仍只保存最新遥测，凭据版本只记录当前版本号，不保存历史 Token 或轮换审计表。

## 下一步

- 根据部署需要增加 Agent 地址编辑流程，并继续复用一次性 Token 和验证机制。
- 如需无中断轮换，可在后续协议中设计新旧 Token 的短时双版本验证窗口。
- 如需完整审计，可增加凭据签发、验证和轮换事件表，但仍不得保存 Token 明文。
