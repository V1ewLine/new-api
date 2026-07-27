# Responses 上游协议自动探测

## 1. 开发目标

在已有 Responses → Chat Completions 通用转换层之上，移除按渠道类型维护的固定名单。new-api 现在根据“具体渠道 + 映射后的上游模型 + 是否流式”记录真实能力，并自动选择以下转发方式：

- 原生 Responses：请求 `/v1/responses`；
- Chat Completions 兼容：将请求转换后访问 `/v1/chat/completions`，再把响应转换回 Responses；
- 未确定：保持原生 Responses 行为，不根据网络抖动、鉴权失败或限流结果冒险切换协议。

管理员仍可在渠道编辑页强制指定协议，用于上游不允许探测或自动判断结果不适合的场景。

## 2. 管理员配置

渠道编辑页的“高级设置 → 路由策略”增加“Responses 上游协议”：

| 模式 | 行为 |
| --- | --- |
| 自动探测 | 优先读取已保存能力；没有记录时执行一次最小探测 |
| 原生 Responses | 始终按 `/v1/responses` 转发，不进行探测 |
| Chat Completions 兼容 | 始终经过通用双向转换层 |

自动模式下会显示最近一次探测模型，以及“非流式”和“流式”两个独立结果。具备渠道操作权限的管理员可以点击“重新探测”。

保存渠道时，如果已经设置测试模型，前端会在保存成功后异步触发一次探测；未设置测试模型时，不阻止渠道保存，第一次真实 Responses 请求会按需探测。

## 3. 自动判定流程

```text
收到 /v1/responses
        ↓
读取渠道手动策略
        ↓ auto
读取 1 分钟进程缓存
        ↓ miss
读取数据库能力记录
        ↓ unknown / miss
singleflight 合并同一渠道、模型、流式类型的并发探测
        ↓
先探测原生 Responses
        ↓ 明确不支持
再探测 Chat Completions
        ↓
保存结果并处理当前真实请求
```

探测请求使用映射后的真实上游模型，最大输出为 16 Token，提示内容只要求回复 `OK`。普通与流式能力分开探测、分开缓存，因为部分上游只支持其中一种形式。

探测是独立的小请求，不会在真实用户请求失败后把同一个请求重新发送到另一端点，因此不会导致用户请求被重复执行。

## 4. 安全判定规则

只有同时满足以下条件，自动模式才会选择 Chat Completions 兼容：

1. Chat Completions 探测成功并返回有效协议数据；
2. 原生 Responses 被明确判定为端点不支持。

可视为“明确不支持”的情况包括：

- HTTP `404`、`405`、`501`；
- HTTP `400` 且错误明确表示未知端点、无效 URL、路由不存在或不支持流式；
- 当前适配器没有实现 Responses 请求转换；
- 适配器把 Responses 和 Chat Completions 解析到同一个 Chat 路由；
- 上游返回成功状态，但响应内容不是有效 Responses 协议。

以下情况不会触发自动切换：

- 超时和网络错误；
- `401`、`403` 等鉴权错误；
- `429` 限流；
- `5xx` 临时服务错误；
- 无法明确归类的请求参数错误。

判断不明确时保存为 `unknown`，运行时继续使用原生 Responses。`unknown` 结果在 6 小时内不会反复探测，避免上游故障期间产生探测风暴。

## 5. 数据模型与失效规则

新增 `channel_responses_capabilities` 表，由 GORM 自动迁移，兼容 SQLite、MySQL 和 PostgreSQL。主要字段：

- `channel_id`；
- 完整上游模型名及其 SHA-256 索引值；
- 非流式模式、探测时间和最后错误；
- 流式模式、探测时间和最后错误；
- 更新时间。

唯一键使用 `channel_id + model_hash`，避免 MySQL 对长 `utf8mb4` 模型名建立联合索引时超过索引长度限制；完整模型名仍单独保存用于管理界面展示。

以下渠道配置发生变化时会删除旧能力并清理进程缓存：

- 渠道类型、密钥或 Base URL；
- 模型列表、测试模型或模型映射；
- 渠道附加参数与协议设置；
- 请求参数覆盖或 Header 覆盖。

删除渠道时会在同一事务中清理能力记录。多实例部署共享数据库结果，每个实例只保留 1 分钟的内存读缓存。

## 6. 管理接口

新增管理员接口：

```text
GET  /api/channel/:id/responses-capabilities
POST /api/channel/:id/responses-capabilities/detect
```

- 查询接口需要渠道读取权限；
- 探测接口需要渠道操作权限；
- 手动探测按“请求指定模型 → 测试模型 → 渠道模型列表第一个模型”的顺序选择模型；
- 探测前继续应用渠道选择上下文、模型映射、代理、请求覆盖和 Header 覆盖。

新增渠道接口的成功响应会包含 `data.channel_ids`，供前端在批量或多密钥创建完成后异步触发对应渠道的探测。

## 7. 通用转换层调整

进入 Chat Completions 兼容路径后，在调用渠道适配器前会同时切换：

- `RelayMode` 为 Chat Completions；
- `RelayFormat` 为 OpenAI Chat Completions；
- 请求路径为 `/v1/chat/completions`。

请求完成后恢复原值。这样依赖 `RelayFormat` 选择请求转换、路由或响应处理逻辑的渠道也能复用兼容层，不再要求为 DeepSeek、Kimi 等模型分别编写 Responses 实现。

`/v1/responses/compact` 仍只走原生 Responses，不进入 Chat Completions 兼容层，因为 Chat Completions 没有与 compact 等价的无状态语义。

## 8. 国际化

渠道编辑界面新增文案已同步到：

- English；
- 简体中文；
- 繁体中文；
- Français；
- 日本語；
- Русский；
- Tiếng Việt。

## 9. 验证记录

已通过：

```bash
go test ./dto ./relay ./controller ./router -count=1
go build ./...

cd web
./node_modules/.bin/oxlint -c .oxlintrc.json \
  src/features/channels/api.ts \
  src/features/channels/components/drawers/channel-mutate-drawer.tsx \
  src/features/channels/hooks/use-channel-mutate-form.ts \
  src/features/channels/lib/channel-form-errors.ts \
  src/features/channels/lib/channel-form.ts \
  src/features/channels/types.ts
npm run typecheck
npm run build
node scripts/sync-i18n.mjs
```

新增回归覆盖包括：

- 配置模式校验与默认值归一化；
- 原生 Responses 与 Chat Completions 的实际协议探测；
- 流式与非流式能力独立判断；
- 能力记录的独立字段更新；
- 管理接口权限；
- 删除渠道时清理能力记录。

当前环境无法从官方 Go 模块代理下载既有 `model` 测试所依赖的 `miniredis`，因此 `go test ./model` 未能进入编译阶段；错误是依赖下载超时，与本次代码断言无关。包含模型代码的后端构建、控制器 SQLite 测试及其他相关包测试均已通过。

## 10. 部署说明

拉取最新 `main`，重新构建前后端并启动 new-api。首次启动时 GORM 会自动创建能力表，不需要执行手工 SQL：

```bash
cd web
bun install
bun run build
cd ..

go build -o ./bin/new-api-local .
./bin/new-api-local --port 3000 --log-dir /data/logs
```

探测会产生极小的真实上游模型调用，可能计入上游供应商用量。若上游禁止探测或对探测成本敏感，可把渠道模式手动设置为“原生 Responses”或“Chat Completions 兼容”。
