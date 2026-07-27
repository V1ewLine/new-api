# OpenAI Responses 通用 Chat Completions 兼容层

## 1. 开发目标

为只提供 OpenAI Chat Completions 协议的上游渠道补充 `/v1/responses` 兼容能力，使客户端不需要因为模型或渠道变化而切换调用协议。

本轮保持 new-api 对外的 OpenAI Responses 接口不变：

```text
POST /v1/responses
```

当选中的渠道原生支持 Responses 时仍按原逻辑直通；当渠道声明上游只支持 Chat Completions 时，由 new-api 自动完成双向协议转换。

## 2. 调用链

兼容路径如下：

```text
客户端 OpenAI Responses 请求
        ↓
Responses 请求转换为 Chat Completions 请求
        ↓
渠道适配器继续执行模型映射、推理参数和渠道专用参数处理
        ↓
上游 Chat Completions 端点
        ↓
Chat Completions JSON 或 SSE 响应
        ↓
转换为 OpenAI Responses JSON 或 SSE
        ↓
返回客户端并按原有流程结算用量
```

转换过程中会把标准请求路径临时切换为 `/v1/chat/completions`，再由渠道适配器生成实际上游地址；请求完成后恢复原来的 `/v1/responses` 路径和 RelayMode，避免影响重试、日志和后续结算。

## 3. 渠道能力声明

新增可选的渠道能力接口 `OpenAIResponsesViaChatCompletionsAdaptor`。只有明确声明 Chat Completions 兼容能力的渠道才会进入转换层，原生 Responses 渠道不受影响。

本轮接入以下内置渠道：

- DeepSeek；
- Moonshot / Kimi；
- Mistral；
- SiliconFlow；
- 百度千帆 v2；
- 智谱 v4。

新增其他 OpenAI Chat Completions 兼容渠道时，只需声明该能力，即可复用同一套请求、普通响应和流式响应转换，不需要重复实现 Responses 处理器。

## 4. 请求与响应语义

### 4.1 请求转换

复用已有协议转换注册表，将常用 Responses 字段映射到 Chat Completions，包括：

- `input`、`instructions`；
- `max_output_tokens`；
- `temperature`、`top_p`；
- `tools`、`tool_choice`、`parallel_tool_calls`；
- `reasoning.effort`；
- `stream` 和 `stream_options`；
- 多模态输入及工具调用上下文。

转换完成后仍调用原渠道的 `ConvertOpenAIRequest`，因此 DeepSeek V4 的思考后缀、Kimi 模型的温度约束等既有渠道逻辑继续生效。

无法通过无状态 Chat Completions 等价表达的字段会返回明确的 400 错误，不会静默丢弃。目前包括：

- `conversation`；
- `previous_response_id`；
- `prompt`；
- `context_management`。

### 4.2 普通响应

上游 Chat Completions JSON 会被转换为标准 Responses 对象，包含输出文本、工具调用、完成状态和 Token 用量。计费继续使用转换后得到的统一 `dto.Usage`，不会绕过原有预扣与结算流程。

### 4.3 流式响应

上游 Chat Completions SSE 会按顺序转换为 Responses SSE 事件，包括：

- `response.created`；
- `response.output_item.added`；
- `response.output_text.delta`；
- 工具调用参数增量事件；
- `response.output_text.done`；
- `response.completed`。

上游提供 usage 时直接使用；缺少 usage 时沿用现有文本估算逻辑。手动状态码映射和错误处理同样应用于兼容路径。

## 5. 与透传设置的关系

对于已声明使用兼容层的渠道，即使开启全局请求体透传或渠道请求体透传，`/v1/responses` 仍会执行必要的 Responses → Chat 转换。否则 Responses 请求体会被错误地直接发送到 `/v1/chat/completions`。

原生 Responses 渠道继续遵循原来的透传设置。

## 6. 验证

新增端到端回归测试：

- DeepSeek 非流式请求实际发送到 `/v1/chat/completions`，返回标准 Responses JSON；
- Moonshot / Kimi 流式请求实际发送到 `/v1/chat/completions`；
- Kimi 渠道专用温度规则在转换后继续生效；
- Chat SSE 正确转换为 `response.output_text.delta` 和 `response.completed`；
- Token usage、最终上游协议、RelayMode 和请求路径恢复正确。

已执行：

```text
go test ./relay/... -count=1
go test ./service/relayconvert/... -count=1
```

全部通过。

## 7. 部署说明

本轮仅修改 Go 后端，不需要重新构建前端。部署时拉取最新 `main` 后重新编译并重启 new-api：

```bash
git pull origin main
go build -o ./bin/new-api-local .
./bin/new-api-local --port 3000 --log-dir /data/logs
```
