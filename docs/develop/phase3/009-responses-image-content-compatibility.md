# Responses 图片内容块兼容修复

## 1. 问题背景

部分 SGLang 版本虽然提供 `/v1/responses` 路由，但请求内容块会进入内部 `ChatCompletionRequest` 校验链。

标准 Responses 图片块使用：

```json
{
  "type": "input_image",
  "image_url": "data:image/png;base64,...",
  "detail": "low"
}
```

Chat Completions 内容块使用：

```json
{
  "type": "image_url",
  "image_url": {
    "url": "data:image/png;base64,...",
    "detail": "low"
  }
}
```

原有 `native_text_compat` 模式只会把 `input_text/output_text` 改写为 `text`，`input_image` 会被原样发送。SGLang 因此返回大量 `ChatCompletionRequest` 联合类型校验错误，其中包含：

```text
Input should be 'image_url'
input_value=... type='input_image'
```

## 2. 根因

问题由三个条件共同触发：

1. 能力探针只使用纯文本，文本探测结果会被同一渠道、模型和流式维度的图片请求复用。
2. 原生兼容标准化器仅处理文本块，没有处理 `input_image`。
3. 运行时纠偏只识别 `input_text` 错误，图片格式错误会直接返回客户端。

## 3. 修复方案

### 3.1 扩展原生内容块兼容

保留现有数据库模式值 `native_text_compat`，避免已保存能力记录失效，但将其标准化范围扩展到 Responses 内容块：

- `input_text/output_text → text`；
- `input_image → image_url`；
- 普通 URL 和 `data:` URL 原样保留；
- 将顶层 `detail` 移入 `image_url` 对象；
- 已有对象中的 `detail` 优先，不被顶层值覆盖；
- 仅修改明确的 Responses 内容块位置，不递归改写工具输出中的用户 JSON；
- 只有 `file_id` 或非法 `image_url` 类型时不伪造 URL，不静默丢弃原始字段。

该路径继续请求 SGLang `/v1/responses`，因此可保留原生 Responses JSON/SSE 和状态字段语义。

### 3.2 运行时纠偏

自动模式下，当真实请求收到明确的：

```text
ChatCompletionRequest + input_image + expected image_url
```

格式校验错误时，使用以下有界纠偏：

```text
native
  → native_text_compat（修正文本和图片内容块后重试）

native_text_compat
  → chat_completions（兼容请求仍被同类校验拒绝时的最后兜底）
```

仅 `400/422` 且同时命中 `ChatCompletionRequest` 和明确的类型期望时才纠偏。“模型不支持图片”等业务错误、鉴权、限流、超时和 `5xx` 不会被误判为协议错误。

### 3.3 完整 Responses → Chat 路径

同步修正完整 Chat 兼容转换器：

- 字符串 `image_url` 包装为标准 `{ "url": ... }` 对象；
- 保留 `data:` URL；
- 保留并正确安置 `detail`；
- 对已经是对象的输入进行复制，不改写请求原值。

因此管理员手工选择“Chat Completions 兼容”或原生兼容二次失败后自动兜底时，图片格式也保持正确。

## 4. 兼容性与数据影响

- 不新增或修改数据库表和字段；
- 旧的 `native_text_compat` 能力记录可继续使用；
- 管理界面暂继续显示历史名称“原生 Responses（文本兼容）”，底层模式现已同时覆盖图片内容块；
- 不修改客户端原始请求，只修改发往上游的深拷贝；
- 本次转换不解码、重新编码或新增专门的 Base64 持久化，现有请求体调试日志策略保持不变；
- 不需要前端或数据库迁移。

## 5. 验证

回归测试覆盖：

- 普通图片 URL 和 `data:` URL；
- `detail` 字段迁移与内层值优先级；
- 仅 `file_id` 和非法 URL 类型不被误转换；
- 工具输出中任意嵌套载荷不被递归改写；
- `native → native_text_compat → chat_completions` 有界纠偏；
- 模型视觉能力错误不被误判；
- 完整 Responses → Chat 转换输出标准图片对象。

执行命令：

```bash
GOCACHE=/tmp/new-api-go-cache go test ./relay -count=1
GOCACHE=/tmp/new-api-go-cache go test ./service/relayconvert/internal/oai_responses -count=1
GOCACHE=/tmp/new-api-go-cache go build ./...
```
