# Claude 交错工具调用流修复

## 1. 开发目标

修复 Claude Code 通过 `/v1/messages` 调用 OpenAI Chat Completions 兼容模型时，偶发出现以下错误的问题：

```text
API Error: Content block not found
```

问题主要出现在模型连续生成多个工具调用，并在工具调用之间插入文本的流式响应中。纯文本请求和普通单工具请求通常不会触发。

## 2. 抓包结论

本轮使用同一次失败请求的网络抓包，对照检查了以下两段明文 HTTP 流：

- DeepSeek/SGLang 返回给 new-api 的 OpenAI Chat Completions SSE；
- new-api 返回给 Claude Code 的 Anthropic Messages SSE。

对应 new-api 请求 ID：

```text
202607300602431051335018268d9d6mxTtOVD9
```

SGLang 返回的主要内容顺序为：

```text
文本
tool_call index=0
文本
tool_call index=1
文本
finish_reason=tool_calls
[DONE]
```

原始 OpenAI SSE 完整，工具名称、工具 ID、参数分片、结束原因和 `[DONE]` 均存在。

转换后的 Anthropic SSE 出现了以下非法顺序：

```text
content_block_start index=4
content_block_stop  index=3
content_block_stop  index=4
```

其中 `index=3` 从未发送过 `content_block_start`，Claude Code 因此在处理 `content_block_stop index=3` 时抛出 `Content block not found`。

抓包中包含认证信息、请求正文和工具输入，只作为本地诊断材料使用，不纳入代码仓库。

## 3. 根因

旧转换状态只记录：

- 当前工具块的 Anthropic 基址；
- 上游工具调用的最大 index。

当响应从工具切换到文本、再切换回工具时，转换器会重新设置 Anthropic 基址，但仍把 OpenAI 的全局工具 index 当作当前工具段的局部偏移。

在抓包场景中：

```text
当前 Anthropic 基址 = 3
第二个 OpenAI 工具 index = 1
错误计算结果 = 3 + 1 = 4
```

关闭工具块时，旧逻辑又按 `0..最大偏移` 生成停止事件，因此同时关闭了从未打开的 block 3 和实际打开的 block 4。

## 4. 修改内容

### 4.1 显式映射工具块

`ClaudeConvertInfo` 不再使用“基址 + 最大偏移”推算工具块，而是记录：

```text
OpenAI tool_call index → Anthropic content block index
```

每个新工具调用按当前 Anthropic 消息中的下一个连续 block index 分配，不再直接把上游 index 加到 Anthropic 基址。

因此抓包场景会被转换为：

```text
text block 0
tool block 1
text block 2
tool block 3
text block 4
```

### 4.2 只关闭实际打开的工具块

结束工具段时，从映射中取得实际创建过的 Anthropic block index，排序后逐个发送 `content_block_stop`。

不再根据最大偏移补齐中间位置，避免生成没有对应 `content_block_start` 的停止事件。

### 4.3 延迟处理缺少工具名称的分片

部分 OpenAI 兼容上游可能先发送工具参数分片，后发送工具名称。转换器会暂存尚不能创建 `tool_use` block 的工具分片，获得名称后再发送：

```text
content_block_start
content_block_delta
```

避免产生 delta 先于 start 的另一类非法 Anthropic SSE。

### 4.4 统一首帧工具处理

首个 OpenAI 流帧和后续流帧现在复用相同的工具块分配逻辑。首帧包含多个工具调用时，也会为每个工具创建独立且连续的 Anthropic block。

## 5. 回归测试

新增回归场景：

```text
文本
tool_call index=0 的名称
tool_call index=0 的参数
文本
tool_call index=1 的名称
tool_call index=1 的参数
文本
结束
```

测试逐项验证所有 `content_block_start`、`content_block_delta` 和 `content_block_stop` 的 index，确保输出严格为连续的 `0..4`，且每个 stop 都有对应 start。

已执行：

```bash
GOCACHE=/tmp/new-api-go-cache \
  go test ./service/relayconvert/internal/oai_chat -count=1

GOCACHE=/tmp/new-api-go-cache \
  go test ./service/relayconvert/... ./relay/common -count=1

GOCACHE=/tmp/new-api-go-cache \
  go test ./relay/... -count=1

GOCACHE=/tmp/new-api-go-cache \
  go build ./...
```

全部通过。

## 6. 影响范围

本轮仅修改 OpenAI Chat Completions SSE 转换为 Anthropic Messages SSE 的工具调用状态管理：

- 不修改数据库表结构；
- 不影响 PostgreSQL、Redis 和已有业务数据；
- 不修改非流式响应转换；
- 不修改直接使用 `/v1/chat/completions` 的客户端；
- 不需要重新构建前端。

DeepSeek/SGLang 是本次问题的实际触发来源，但修复不按模型名称定制。所有产生交错文本和多工具调用分片的 OpenAI 兼容模型都会使用同一套修复逻辑。

## 7. 部署说明

拉取代码后重新编译并重启 new-api：

```bash
go build -o ./bin/new-api-local .
./bin/new-api-local --port 3000 --log-dir /data/logs
```

部署后建议使用 Claude Code 连续执行读取文件、Shell 命令和多轮工具调用，确认不再出现 `Content block not found`。
