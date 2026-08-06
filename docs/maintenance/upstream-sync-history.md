# 官方上游同步记录

本文件记录 fork 每次合并 `QuantumNous/new-api` 的基线、冲突判断和验证结果。
后续同步应追加记录，不覆盖历史条目。

## 2026-08-06

### 同步范围

```text
fork 合并前：f825a18cd248fba23bb5337a2cb043b67eea02db
upstream/main：0ab02020603d22e5613bc4cf46bfab06f8567769
共同基线：1721144221ec5c94dd87891a7ae1bee228e7bb63
fork 独有提交：33
upstream 独有提交：49
```

本次上游包含 RelayKit 模块拆分、DeepSeek Responses、渠道 HTTP Transport、
Auto 分组、计费安全修复及前端渠道配置更新，合并范围约 554 个文件。

### 隔离方式

同步在以下独立环境完成：

```text
分支：sync/upstream-20260806
worktree：/tmp/new-api-upstream-sync-20260806
```

原 `main` 工作区中已有的安装文档、压测结果、启动脚本和 `scripts/api/` 未提交
内容没有被 stash、覆盖或加入同步分支。

### 冲突与取舍

虚拟合并和真实合并共发现 14 个文本冲突：

```text
model/channel.go
model/option.go
relay/common/relay_info.go
relaykit/dto/channel_settings.go
relaykit/relayconvert/internal/oai_chat/to_claude_messages_resp.go
web/src/features/channels/components/drawers/channel-mutate-drawer.tsx
web/src/features/channels/types.ts
web/src/i18n/locales/en.json
web/src/i18n/locales/fr.json
web/src/i18n/locales/ja.json
web/src/i18n/locales/ru.json
web/src/i18n/locales/vi.json
web/src/i18n/locales/zh-TW.json
web/src/i18n/locales/zh.json
```

处理结果：

- 渠道保存同时执行 fork 的 Responses 模式校验和 upstream 的 HTTP Transport 校验；
- Options 同时保留集群刷新/遥测保留配置与 upstream 的工具价格、Auto 分组校验；
- fork 的 Responses 能力探测和 Chat 兼容代码迁移到 upstream 新的
  `relaykit/dto`、`relaykit/types`；
- `created_at` 整数小数兼容测试和实现迁移到独立 RelayKit 模块；
- Claude 工具调用流修复迁移到 `convmeta.ClaudeConvertInfo` 和 RelayKit 转换器，
  保留真实 block index、延迟启动工具 block 和逐个关闭已启动 block 的行为；
- DeepSeek 已新增原生 Responses 路由，Chat fallback 测试改为先模拟原生端点 404，
  再验证 `/v1/chat/completions` 回退；
- 渠道编辑界面同时保留 fork 的 `native_text_compat` 和 upstream 的 HTTP 协议、
  HTTP/2 分片及新渠道设置；
- 七种前端语言保留双方所有合法 key，并通过 i18n 同步检查。

### 验证结果

已通过：

```text
go build ./...
go test ./...（relaykit 独立模块）
go test ./relay -run Responses 格式归一化与安全纠偏单元测试
go test ./service/clusterstatus ./service/dashboardexport ./router
前端冲突文件 oxfmt
前端冲突文件 oxlint
i18n sync 与 missing/extras 检查
全部 locale JSON 解析
git diff --check
冲突标记扫描
```

根模块全量 `go test ./...` 已实际启动并验证大量包，其中集群、导出、路由、渠道
适配器和设置包通过；未完成项属于执行环境限制：

- 沙箱禁止测试监听临时 TCP 端口，`httptest`/SMTP 测试无法运行；
- 本机 Go module 缓存缺少既有 `miniredis` 测试依赖且当前无法下载；
- 当前环境没有 Bun，上游新增的 `yace`、`happy-dom` 依赖无法完成安装，因此完整
  前端 typecheck/build 尚需在具备 Bun 和网络的构建环境补跑。

这些限制没有通过删除测试、修改断言或降低校验强度规避。正式部署前应按照
[同步指南](./upstream-sync.md) 在构建容器补齐依赖并执行剩余验证。
