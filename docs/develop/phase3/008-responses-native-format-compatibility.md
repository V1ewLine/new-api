# Responses 原生格式兼容探测与运行时纠偏

## 1. 开发背景

部分 SGLang 版本已经提供 `/v1/responses` 路由，但原生 Responses 请求进入其内部 `ChatCompletionRequest` 处理链后，没有把标准内容块：

```json
{"type":"input_text","text":"hello"}
```

标准化为内部 Chat 格式：

```json
{"type":"text","text":"hello"}
```

因此上游会返回类似错误：

```text
validation errors for ChatCompletionRequest
Input should be 'text'
input_value='input_text'
```

旧版能力探测只能区分“原生 Responses”和“Chat Completions 兼容”。上述场景既不是端点不存在，也不是标准 Responses 完全可用，因此会被记为 `unknown`，运行时继续按原生标准格式发送并重复失败。

本次开发在保持客户端继续使用 `/v1/responses` 的前提下，增加 SGLang 原生文本兼容模式，并为缓存过期或上游版本变化提供一次安全纠偏。

## 2. 能力模式

能力记录继续按照以下维度保存：

```text
渠道 ID + 映射后的上游模型 + 是否流式
```

支持四种结果：

| 模式 | 上游路径 | 内容块格式 | 用途 |
| --- | --- | --- | --- |
| `native` | `/v1/responses` | `input_text` | 标准原生 Responses |
| `native_text_compat` | `/v1/responses` | `text` | SGLang 原生 Responses 文本兼容 |
| `chat_completions` | `/v1/chat/completions` | `text` | Responses 与 Chat 双向转换 |
| `unknown` | 未确定 | 不固定 | 鉴权、限流、网络等不明确失败 |

`native_text_compat` 直接写入已有 `channel_responses_capabilities` 表的 `varchar(32)` 模式字段，不新增表和字段，不需要手工执行数据库迁移，SQLite、MySQL 和 PostgreSQL 均可继续使用原结构。

## 3. 三阶段探针

自动模式按以下顺序执行最小探针：

```text
/v1/responses + input_text
        ↓ 明确为 input_text/ChatCompletionRequest 校验不兼容
/v1/responses + text
        ↓ 仍不兼容
/v1/chat/completions + text
```

### 3.1 标准原生探针

首先发送标准 Responses 数组输入：

```json
{
  "model": "mapped-upstream-model",
  "input": [
    {
      "role": "user",
      "content": [
        {"type": "input_text", "text": "Reply with OK"}
      ]
    }
  ],
  "max_output_tokens": 16,
  "stream": false
}
```

成功返回有效 Responses JSON 或 SSE 时记录为 `native`。

### 3.2 原生文本兼容探针

只有标准探针返回明确的 `ChatCompletionRequest + input_text + expected text` 校验错误时，才在同一 `/v1/responses` 路径把探针内容块改为 `text`。

成功时记录为 `native_text_compat`。该模式仍使用 SGLang 原生 Responses 响应，因此不会丢失原生 Responses SSE 封装。

如果兼容探针返回 `400`、`422` 或明确的端点不支持错误，再执行 Chat 探针；鉴权、限流、网络、超时和 `5xx` 不会继续扩大探测请求。

### 3.3 Chat Completions 探针

最后使用已有通用 Responses → Chat 转换器访问 `/v1/chat/completions`。只有上游实际返回有效 Chat JSON 或 SSE 时，才记录为 `chat_completions`。

流式和非流式探针独立执行、独立保存，继续使用 singleflight 合并同一能力键的并发探测。

## 4. 请求格式修正

`native_text_compat` 只修改发往上游的请求副本，不修改客户端原始请求：

```text
客户端 /v1/responses + input_text
        ↓
new-api 复制 OpenAIResponsesRequest
        ↓
仅标准化 input 或 input[].content[] 中的 input_text/output_text → text
        ↓
SGLang /v1/responses + text
        ↓
原生 Responses JSON/SSE 返回客户端
```

标准化器不会递归替换任意 JSON 字符串，也不会进入 `function_call_output.output` 等用户或工具自有载荷，避免错误修改工具结果、metadata、JSON Schema 或函数参数里的同名字段。

当能力模式需要改写请求时，协议兼容处理优先于请求体透传；标准 `native` 模式仍保持原有透传行为。

## 5. 运行时安全纠偏

探针结果可能因为 SGLang 镜像升级、回滚或路由调整而过期。自动模式下，真实请求在上游开始输出前遇到明确协议错误时允许纠偏一次：

```text
native
  └─ input_text 校验错误 → native_text_compat

native / native_text_compat
  └─ 404、405、501 或明确端点不支持 → chat_completions
```

纠偏限制：

- 只在渠道设置为“自动检测”时启用；
- 管理员明确选择“原生 Responses”时不擅自改变协议；
- `401`、`403`、`429`、超时、网络错误和 `5xx` 不触发格式切换；
- 已经开始向客户端写入 SSE 后不会重试；
- 每个模式在单个真实请求中最多尝试一次，不形成循环；
- 只有纠偏后的请求和响应完整成功，才更新数据库能力记录与进程缓存；
- Chat 转换器不支持的状态型字段继续返回明确错误，不进行不完整语义模拟。

## 6. 管理界面

渠道编辑页的能力结果增加：

```text
Native Responses (text compatibility)
原生 Responses（文本兼容）
```

自动检测说明同步更新为三阶段探测语义。管理员仍可查看流式、非流式独立结果，并使用“重新检测”在 SGLang 升级后刷新旧能力。

新增文案通过项目 i18n 脚本写入并同步到：

- English；
- 简体中文；
- 繁體中文；
- Français；
- 日本語；
- Русский；
- Tiếng Việt。

## 7. 回归测试

新增覆盖：

- 标准原生 Responses 探针成功；
- 标准 `input_text` 失败后，`text` 原生探针成功；
- 两种原生格式都失败后，Chat 探针成功；
- `input_text` 和 `output_text` 只在 Responses 内容块位置被修改；
- 工具输出中的同名字段保持不变；
- `429` 不被误判为协议兼容错误；
- Responses 路由消失时切换到 Chat 兼容；
- Chat 模式失败后不再循环纠偏；
- `native_text_compat` 能力值可由持久化模型接受。

已执行并通过：

```bash
GOCACHE=/tmp/new-api-go-cache go test ./relay -count=1
GOCACHE=/tmp/new-api-go-cache go build ./...

cd web
./node_modules/.bin/oxlint -c .oxlintrc.json \
  src/features/channels/components/drawers/channel-mutate-drawer.tsx \
  src/features/channels/types.ts
./node_modules/.bin/tsgo -b
node scripts/add-missing-keys.mjs
node scripts/find-missing-keys.mjs
node scripts/sync-i18n.mjs
./node_modules/.bin/rsbuild build
```

当前执行环境没有可用的 `bun` 命令，因此前端验证使用仓库已安装的同版本 Node 脚本及 `node_modules/.bin` 二进制完成。完整 `model` 测试包所需的既有 `miniredis` 依赖下载发生网络超时，未进入该包测试；`relay` 完整测试和后端全量构建均会编译本次修改的生产模型代码，已通过。

## 8. 部署与使用

拉取 `main` 后重新编译前后端并启动 new-api：

```bash
cd web
bun install
bun run build
cd ..

go build -o ./bin/new-api-local .
./bin/new-api-local --port 3000 --log-dir /data/logs
```

渠道的“Responses 上游协议”保持“自动检测”即可使用新逻辑。部署后建议在渠道编辑页点击一次“重新检测”，确认对应模型显示为以下三种状态之一：

```text
原生 Responses
原生 Responses（文本兼容）
Chat Completions 兼容
```

客户端调用地址保持 `/v1/responses`，不需要修改请求 URL。
