# Codex 开发任务：New API 集群状态模块（第一阶段）

## 0. 任务背景

请在当前 **New API** 项目中新增一个仅管理员可见的「集群状态」模块。

集群监控数据来自：

```text
https://github.com/V1ewLine/sglang_telemetry_agent
```

`sglang_telemetry_agent` 与 SGLang 部署在同一台机器上。New API 所在机器需要按照固定时间间隔，主动请求每个 Agent 的遥测接口，再将结果转换为页面需要的数据。

我会同时提供 UI 参考图片。请以图片作为布局与视觉参考，但最终实现必须与当前 New API 项目的 UI、组件、路由、权限、交互和代码规范保持一致。

---

# 1. 本阶段目标

本阶段采用“一级页面完整、后续页面可演示”的范围控制。

## 1.1 需要重点完成

完整实现：

1. 集群状态一级页面。
2. 一级页面需要的后端接口与数据模型。
3. 添加集群流程。
4. 添加集群时与 New API 已部署模型数据联动。
5. Agent 连接、轮询和数据适配的可扩展接口。
6. 管理员权限控制。
7. Loading、Empty、Error、分页、筛选等基础交互。

## 1.2 只做 Naive 版本

以下页面只实现基本导航和操作逻辑，不要求一次做完全部指标：

1. 某个模型的集群列表页。
2. 某个集群的详情页。
3. 详情页中的：
   - 总览
   - 引擎指标
   - 机器指标

Naive 版本至少应做到：

- 路由可进入。
- 能展示当前模型或集群的基本信息。
- 三个 Tab 可以切换。
- 可以使用少量 Mock 数据展示布局。
- 页面结构和数据接口已经预留，后续可以直接扩展。
- 不要为了本阶段实现完整的历史图表、告警系统或全量指标存储。

---

# 2. 开发前必须先检查当前项目

开始修改代码前，请先检查并说明：

1. 当前前端与后端技术栈。
2. 路由与管理员权限控制方式。
3. 当前 UI 组件库、图标库和图表库。
4. 现有页面容器、侧边栏、顶部导航、表格、分页、Dialog、Form、Select、Tag、Notification 等组件。
5. 已部署模型数据当前来自哪个数据库表、Service、Store 或 API。
6. 后台任务、定时任务或 Worker 的现有实现方式。
7. 敏感信息在当前项目中的存储方式。
8. 项目现有的数据库迁移方案。
9. 项目现有的 API 命名和错误返回格式。
10. i18n、主题和暗色模式的现有支持方式。

然后先输出：

```text
1. 实现计划
2. 预计修改文件
3. 数据库变更
4. API 设计
5. 需要复用的现有组件
6. 当前阶段明确不实现的内容
```

确认计划后再开始编码。

不要脱离当前项目另起一套架构。

---

# 3. 核心约束

## 3.1 与 New API 保持一致

所开发的 UI 和操作逻辑必须与原 New API 对齐：

- 使用现有 Layout。
- 使用现有侧边栏和顶部导航。
- 使用现有管理员权限判断。
- 使用现有 Button、Form、Modal/Dialog、Table、Pagination、Select、Tag 等组件。
- 使用现有主题变量与间距规范。
- 使用现有消息提示和错误处理方式。
- 使用现有 i18n 机制。
- 使用现有请求封装。
- 使用现有数据库与迁移方式。

不要引入新的 UI 框架。

不要为了还原截图使用大量绝对定位或独立 CSS 体系。

## 3.2 方便扩展和修改

本功能不能写成一个巨大的页面组件，也不能让 UI 直接依赖 Agent 的原始 JSON。

必须建立清晰的分层：

```text
Agent 原始响应
    ↓
Agent Client
    ↓
Schema Adapter / Normalizer
    ↓
Cluster Domain Service
    ↓
Repository / Cache / History 接口
    ↓
New API HTTP API
    ↓
Frontend API Client
    ↓
Page ViewModel
    ↓
UI 组件
```

以后新增以下内容时，不应大幅修改现有页面：

- 新 Agent schema 版本。
- 新 GPU 类型。
- 新机器 Collector。
- 多节点集群。
- 多个 SGLang 实例。
- Prometheus 或其他遥测来源。
- 历史时序数据库。
- 新指标卡和图表。
- 新集群状态规则。

---

# 4. SGLang Telemetry Agent 接入要求

## 4.1 当前 Agent 架构

Agent 与 SGLang 位于同一台机器。

New API 主动请求：

```http
GET {agent_base_url}/v1/telemetry
Authorization: Bearer {agent_token}
```

Agent 返回：

- SGLang 引擎状态。
- 请求与 Token 状态。
- 每张 GPU 的功率、利用率、温度、显存和时钟。
- GPU 总板卡功率。
- CPU、内存和 Load Average。
- 最近采样窗口的平均值、最小值和最大值。
- Agent 与引擎采样时间的对齐信息。

注意：

```text
gpu_power_total_watts
```

表示所选 GPU 板卡功率之和，不代表整台机器的总功率。

## 4.2 主动轮询

New API 服务器需要周期性请求已启用的集群 Agent。

轮询间隔必须可配置，不要散落硬编码。

建议提供配置项，例如：

```text
CLUSTER_TELEMETRY_POLL_INTERVAL
CLUSTER_TELEMETRY_REQUEST_TIMEOUT
CLUSTER_TELEMETRY_MAX_CONCURRENCY
CLUSTER_TELEMETRY_FAILURE_THRESHOLD
```

默认轮询间隔可以参考 5 秒，但必须通过统一配置读取。

轮询实现至少考虑：

- 请求超时。
- 最大并发。
- Context 或取消机制。
- 服务关闭时优雅退出。
- 失败计数。
- 连续失败后的退避。
- 少量随机抖动，避免所有集群同一时刻请求。
- Agent 离线。
- 部分数据可用。
- Agent 返回未知 schema。
- 响应体大小限制。
- 禁止记录 Authorization 和连接密钥。
- 手动刷新与后台轮询不能互相破坏。
- 同一集群避免重复并发采集。

不要简单地为每个集群创建一个永久且不可控的无限循环。

优先复用项目已有定时任务、Worker Pool 或后台任务机制。

## 4.3 Agent Client 抽象

建立独立 Agent Client 接口，业务层不要直接拼接 HTTP 请求。

概念接口示例：

```ts
interface TelemetryAgentClient {
  fetchTelemetry(
    connection: ResolvedAgentConnection,
    signal?: AbortSignal,
  ): Promise<RawTelemetryPayload>;

  checkHealth?(
    connection: ResolvedAgentConnection,
    signal?: AbortSignal,
  ): Promise<AgentHealthResult>;
}
```

后端不是 TypeScript 时，请用当前语言实现等价接口。

该接口以后应允许替换为：

- HTTP Agent Client。
- Mock Client。
- Cluster Master Client。
- Prometheus Client。
- 其他遥测来源。

## 4.4 Schema Adapter

Agent 响应包含：

```text
schema_version
```

请建立版本适配层，不要在页面或业务层到处判断字段。

概念结构：

```text
TelemetrySchemaAdapter
├── supports(schemaVersion)
└── normalize(rawPayload) -> NormalizedTelemetrySnapshot
```

第一阶段实现 `1.0` Adapter。

未知版本应：

- 返回明确错误。
- 保留原始响应的诊断信息。
- 不导致整个轮询 Worker 崩溃。
- 在 UI 中显示“数据版本不受支持”或类似状态。

## 4.5 原始数据和规范化数据

不要只保留页面当前用到的五个字段。

建议至少区分：

```text
ClusterConfig
LatestTelemetrySnapshot
NormalizedTelemetrySummary
TelemetryHistoryRepository（接口预留）
```

Agent 当前主要返回“最近状态 + 最近窗口摘要”，并不直接提供完整的一小时历史曲线。

因此：

- 本阶段不要假装 Agent 已经提供完整历史时序。
- 一级页面只需要最新汇总值。
- 二三级页面的趋势图可以暂时使用 Mock 或短期内存数据。
- 需要预留 `TelemetryHistoryRepository` 或类似接口，后续可接数据库或时序数据库。
- 不要求本阶段实现完整历史数据持久化和保留策略。

---

# 5. 集群连接密钥

## 5.1 添加集群只允许三个输入

添加集群表单只包含：

1. 模型
2. 集群名称
3. 集群连接密钥

不要在 UI 中额外增加节点 ID、引擎 ID、URL、Token、区域等必填字段。

表单示例：

```text
模型             [请选择已经部署的模型]
集群名称         [例如：华东 A100 集群]
集群连接密钥     [请输入连接密钥]
```

## 5.2 连接密钥作为不透明凭证

前端和普通业务代码应把“集群连接密钥”视为 opaque secret，不要在页面中解析和展示其内容。

由于当前 Agent 文档使用的是：

```text
Agent URL + Bearer Token
```

请先检查 Agent 项目是否已经定义统一的连接密钥或 Enrollment Key 格式。

### 已存在统一格式时

直接复用该格式，不要重复设计。

### 尚未存在统一格式时

不要在页面组件中临时发明编码协议。

请建立可替换接口：

```ts
interface ClusterLinkResolver {
  resolve(linkSecret: SecretValue): Promise<ResolvedAgentConnection>;
}

interface ResolvedAgentConnection {
  baseUrl: string;
  bearerToken: SecretValue;
}
```

实际语言请适配当前后端。

第一阶段允许：

- 使用 Mock Resolver。
- 或使用一个明确标记为临时实现的 Resolver。
- 将真实解析逻辑留在单独模块。
- 在 README 或代码注释中说明替换点。

不要把解析逻辑写进 Controller、数据库 Model 或页面组件。

## 5.3 密钥安全

必须满足：

- 密钥提交后不再完整返回给前端。
- 列表和详情页不显示密钥。
- 日志中不记录密钥。
- 错误信息不包含密钥。
- 数据库不得明文保存，优先复用项目现有加密或 Secret 管理方式。
- 如果当前项目没有安全存储机制，建立 `SecretProtector` 接口，并在本阶段使用清晰隔离的实现。
- 编辑集群时只允许“保留原密钥”或“替换密钥”。
- API 响应只返回：

```text
has_link_secret: true
```

或脱敏状态，不返回原文。

---

# 6. 与 New API 已部署模型联动

## 6.1 模型必须来自现有模型数据

“添加集群”中的模型下拉框必须读取 New API 当前已经部署或已启用的模型列表。

禁止：

- 在前端硬编码模型数组。
- 新建一份与 New API 模型管理无关的独立模型表。
- 让用户自由输入任意模型名称。
- 从 Agent 返回的 `identity.model` 直接创建 New API 模型。

必须复用现有：

- Model Service。
- Channel Model 数据。
- 已部署模型 API。
- 或项目实际使用的等价数据源。

请先检查项目中“模型广场”“模型管理”“渠道模型”等页面使用的数据来源，再选择正确的接口。

## 6.2 数据关系

集群记录需要关联现有模型的稳定标识。

建议：

```ts
interface ClusterConfig {
  id: string;
  modelId: string;
  modelNameSnapshot: string;
  name: string;
  enabled: boolean;
  hasLinkSecret: boolean;
  createdAt: string;
  updatedAt: string;
}
```

如果当前 New API 没有稳定的 `modelId`，请使用项目现有可稳定关联的键，并明确说明限制。

建议同时保存：

- 稳定模型引用。
- 创建时的模型名称快照。

这样模型重命名后仍可审计。

## 6.3 联动规则

需要处理：

- 新增模型后，添加集群下拉框可读取到。
- 模型被禁用时，不允许新建关联集群。
- 已有关联模型被禁用或删除时，集群记录不要直接丢失。
- 一级页面应显示“模型不可用”或类似状态。
- Agent 返回的 `identity.model` 与所选模型不一致时：
  - 不自动改写关联模型。
  - 记录可诊断的 mismatch。
  - UI 显示警告状态。
- 模型列表刷新后，集群页面能够正确更新。
- 权限和可见范围沿用原有模型接口规则。

---

# 7. 推荐的可扩展后端边界

名称可适配现有项目，但职责应清晰。

```text
ClusterRepository
ClusterService
ClusterModelCatalog
ClusterLinkResolver
SecretProtector
TelemetryAgentClient
TelemetrySchemaAdapter
TelemetryPollScheduler
LatestTelemetryRepository
TelemetryHistoryRepository
ClusterHealthEvaluator
```

职责建议：

## `ClusterRepository`

负责集群配置的增删改查，不负责发送 HTTP 请求。

## `ClusterModelCatalog`

封装“当前已部署模型列表”，内部复用 New API 现有模型服务。

## `ClusterLinkResolver`

将不透明连接密钥解析为 Agent 连接信息。

## `SecretProtector`

负责密钥加密、解密或引用外部 Secret。

## `TelemetryAgentClient`

只负责与 Agent 通信。

## `TelemetrySchemaAdapter`

将 Agent 原始 JSON 转换为稳定的领域模型。

## `TelemetryPollScheduler`

按照配置间隔调度采集，不负责 UI。

## `LatestTelemetryRepository`

保存每个集群最后一次采集结果和状态。

## `TelemetryHistoryRepository`

只预留接口，本阶段可以是短期内存实现或空实现。

## `ClusterHealthEvaluator`

将 Agent 的：

```text
status
engine.up
machine.up
alignment.quality
collector.status
```

转换为 New API 页面使用的：

```text
online
partial
abnormal
offline
unknown
```

状态规则集中放置，不要散落在前后端组件中。

---

# 8. 推荐 API

请优先遵循当前 New API 的路由风格。下面只是职责参考，不要求照搬路径。

## 8.1 一级页面

```http
GET /api/clusters/overview
GET /api/clusters/models
```

支持：

```text
search
model_type
status
page
page_size
```

返回：

```ts
interface ClusterStatusPageResponse {
  overview: {
    totalClusters: number;
    onlineClusters: number;
    abnormalClusters: number;
    totalRequests: number;
    totalTokens: number;
  };
  modelGroups: ModelClusterGroup[];
  alerts: ClusterAlertSummary[];
  pagination: Pagination;
}
```

也可以拆成多个接口，但不要让前端产生大量无必要请求。

## 8.2 添加集群

```http
POST /api/clusters
```

请求：

```ts
interface CreateClusterRequest {
  modelId: string;
  name: string;
  linkSecret: string;
}
```

响应中禁止返回 `linkSecret`。

## 8.3 已部署模型选项

优先复用已有接口。

如果现有接口不适合 Select，再增加轻量适配接口：

```http
GET /api/clusters/model-options
```

它必须调用现有 Model Service，而不是维护第二份模型数据。

返回：

```ts
interface ModelOption {
  id: string;
  name: string;
  logo?: string;
  type?: string;
  enabled: boolean;
}
```

## 8.4 Naive 二三级页面

```http
GET /api/clusters/models/:modelId
GET /api/clusters/:clusterId
GET /api/clusters/:clusterId/telemetry/latest
```

只实现当前演示需要的最小字段。

为后续历史趋势预留：

```http
GET /api/clusters/:clusterId/telemetry/history
```

本阶段可以返回 Mock、空数据或明确的 Not Implemented 业务结果，但路由和 Service 接口要清晰。

---

# 9. 数据库模型

按当前项目数据库规范增加迁移。

建议的最低字段：

```text
id
model_reference
model_name_snapshot
name
link_secret_ciphertext 或 secret_reference
enabled
health_status
last_polled_at
last_success_at
consecutive_failures
last_error_code
created_at
updated_at
```

不要保存完整错误响应中的敏感内容。

最新遥测可以：

- 使用独立表。
- 使用现有缓存。
- 或第一阶段使用内存存储。

请根据当前项目架构选择，不要为了第一阶段过度设计。

但要确保服务重启后：

- 集群配置仍然存在。
- 密钥仍然安全可用。
- 一级页面不会因为最新遥测暂时为空而报错。

---

# 10. 一级页面 UI

## 10.1 页面位置和权限

在管理员侧边栏增加：

```text
集群状态
```

仅管理员可见。

直接访问路由时也必须经过后端或路由权限校验，不能只隐藏菜单。

## 10.2 页面头部

标题：

```text
集群状态
```

操作区从左到右：

1. 搜索框。
2. 全部模型。
3. 全部状态。
4. 添加集群。

必须满足：

- 「添加集群」按钮紧跟在「全部状态」右侧。
- 使用 New API 现有主按钮样式。
- 不单独放到页面最右上角。

## 10.3 总览卡片

只保留五项：

1. 集群总数
2. 在线集群
3. 异常集群
4. 总请求数
5. 总 Token 数

不要增加总实例数等其他指标。

建议类型：

```ts
interface ClusterOverview {
  totalClusters: number;
  onlineClusters: number;
  abnormalClusters: number;
  totalRequests: number;
  totalTokens: number;
}
```

要求：

- 在线使用项目现有绿色语义。
- 异常使用项目现有警告或错误语义。
- 请求和 Token 数做好千分位与单位格式化。
- 小趋势图可先使用 Mock。
- 保持卡片尺寸和间距统一。

## 10.4 模型分组

按照模型分类展示，例如：

```text
大语言模型
Embedding / 向量模型
图像 / 语音模型
```

实际分组应优先使用 New API 已有模型元数据。

没有分类数据时：

- 使用集中式分类映射。
- 或统一归入“其他模型”。
- 不要在每个组件里独立判断模型名称。

## 10.5 模型卡片

每张卡片包含：

- 模型 Logo。
- 模型名称。
- 模型类型 Tag。
- 集群整体状态。
- 可用率或健康度。
- 已部署集群数量。
- 小型趋势图。
- 「查看集群详情」入口。

示例：

```text
[Logo] GPT-4o  大语言模型

● 在线
可用率 99.8%

已部署 4 个集群

查看集群详情 →
```

Logo 要复用 New API 当前模型 Logo 或图标映射。

缺少 Logo 时使用统一占位，不要出现破图。

## 10.6 告警区域

保留参考图右侧的异常告警区域。

本阶段可以根据最近轮询结果生成简单告警：

- Agent 无法连接。
- Agent 返回 `partial`。
- Agent 返回 `unavailable`。
- 引擎离线。
- 机器采集失败。
- 模型 identity mismatch。
- schema 不支持。

不要求实现完整告警历史和通知系统。

## 10.7 分页

底部使用标准分页：

```text
<  1  2  3  4  >
10 条 / 页
```

不要使用：

```text
查看全部集群
查看全部模型
```

搜索、筛选与分页必须可组合。

---

# 11. 添加集群 Dialog

## 11.1 字段

只保留三个字段：

```text
模型
集群名称
集群连接密钥
```

## 11.2 模型 Select

必须从 New API 已部署模型列表加载。

Select 至少支持：

- Loading。
- 搜索。
- Empty。
- 禁用模型不可选。
- 显示模型 Logo 和名称。
- 不允许自由创建不存在的模型。

## 11.3 表单交互

需要：

- 必填校验。
- 集群名称长度校验。
- 连接密钥基本格式校验通过 Resolver 完成。
- 防止重复提交。
- 保存时 Loading。
- 成功后关闭 Dialog。
- 刷新一级页面。
- 使用 New API 原有成功提示。
- 失败时保留用户已填写内容。
- 错误提示与 New API 原有 Form 风格一致。

可以增加：

```text
测试连接
```

作为按钮或提交前流程，但不要增加新的表单字段。

如果测试连接在本阶段无法真实实现，可以通过接口和 Mock 展示操作逻辑。

---

# 12. 二级页面 Naive 版本

点击模型卡片进入该模型的集群列表。

页面至少展示：

- 模型 Logo。
- 模型名称。
- 总实例数。
- 正在运行数。
- 故障数。
- 当前总 Token 数。
- 当前总请求数。
- 集群列表。

集群列表字段：

```text
集群名称
状态
Token 数
请求数
GPU 总功率
操作
```

操作：

```text
查看详情
```

要求：

- 能正确接收模型 ID。
- 能从一级页面跳转。
- 能返回一级页面。
- 能展示真实集群配置与少量 Mock 遥测。
- 不要求本阶段完成复杂图表。

---

# 13. 三级页面 Naive 版本

集群详情页只保留三个 Tab：

```text
总览
引擎指标
机器指标
```

不要新增其他 Tab。

## 13.1 总览

只展示少量基础信息和演示性卡片：

- 集群名称。
- 状态。
- 模型。
- 最近采集时间。
- 当前请求数。
- 当前 Token 数。
- Agent 或引擎是否在线。

## 13.2 引擎指标

Naive 版本展示：

- 引擎状态。
- 运行请求数。
- 等待请求数。
- Token 使用率。
- 生成吞吐。
- 缓存命中率。

可以使用少量 Mock 折线图。

## 13.3 机器指标

Naive 版本展示：

- GPU 总功率。
- 每张 GPU 的功率。
- GPU 利用率。
- GPU 温度。
- CPU 利用率。
- 内存利用率。

必须按 GPU 数组动态渲染，不能写死只有 GPU 0 和 GPU 1。

本阶段不要求完成完整历史查询、导出和全量图表。

---

# 14. 前端组件边界

请根据现有项目结构调整，概念上建议拆分为：

```text
ClusterStatusPage
ClusterOverviewCards
ClusterToolbar
ModelGroupSection
ModelClusterCard
ClusterAlertPanel
ClusterPagination
AddClusterDialog
ModelSelector
ModelClustersPage
ClusterDetailPage
ClusterDetailTabs
EngineMetricsPreview
MachineMetricsPreview
```

页面组件只负责组合，不要把：

- 请求逻辑。
- 字段适配。
- 数值格式化。
- 状态规则。
- Secret 逻辑。
- 图表数据转换。

全部堆在页面中。

---

# 15. 状态和异常处理

必须至少处理：

```text
Loading
Empty
Error
Refreshing
Partial
Offline
Unsupported Schema
Model Mismatch
```

规则：

- 首次加载使用现有 Skeleton。
- 后台刷新不要清空旧数据。
- Agent 失败时保留最后一次成功值，并标记数据已过期。
- 没有成功采集过时显示 Empty 或 Offline。
- 页面不要因为某个集群异常而整体报错。
- 错误信息应可诊断，但不能泄露密钥。
- 用户手动刷新与后台轮询应共用 Service，不重复写请求逻辑。

---

# 16. 安全要求

管理员提供的连接信息会导致 New API 服务器主动发起网络请求，因此至少考虑：

- SSRF 风险。
- URL scheme 限制。
- 重定向限制。
- 请求超时。
- 响应大小限制。
- TLS 校验。
- 私网地址策略。
- 可配置的目标网段 Allowlist。
- 禁止将 Secret 写入日志。
- 禁止在错误响应中回显 Secret。
- 只有管理员可以创建、更新、删除和测试集群。

由于 Agent 通常部署在私网机器上，不要简单粗暴地禁止所有私网地址。

请将目标地址校验封装成独立 Policy 或 Validator，方便部署方配置。

---

# 17. 测试要求

至少补充以下测试：

## 后端

- 创建集群时模型不存在。
- 创建集群时模型被禁用。
- Secret 不出现在 API 响应。
- Agent Client 正确设置 Bearer Header。
- Agent 超时。
- Agent 返回 `partial`。
- Agent 返回未知 schema。
- 模型 identity mismatch。
- Poller 连续失败。
- Health 状态映射。
- GPU 数量为 0、1、2、8。
- 列表分页和筛选。

## 前端

- 一级页面正常渲染。
- 五个总览指标正确显示。
- 添加按钮位置与筛选器同一行。
- 模型 Select 使用已部署模型接口。
- 添加集群成功后刷新。
- Empty、Loading、Error 状态。
- 分页交互。
- 模型卡片跳转。
- Naive 二三级页面可以进入和切换 Tab。

测试工具和写法必须沿用当前项目。

---

# 18. 明确的非目标

本阶段不要过度实现：

- 完整时序数据库。
- 长期历史数据保留策略。
- 完整告警历史与通知渠道。
- 自动扩缩容。
- 多机 Master Agent 聚合。
- Agent 主动上报。
- 完整导出系统。
- 自定义仪表盘布局。
- 复杂 RBAC。
- 全量指标图表。
- 修改 SGLang 推理链路。

但需要留下清晰接口，方便后续增加这些能力。

---

# 19. 验收标准

## 一级页面

- [ ] 管理员可以看到「集群状态」菜单。
- [ ] 非管理员无法访问对应 API 和页面。
- [ ] UI 风格与现有 New API 一致。
- [ ] 操作区顺序为搜索、模型筛选、状态筛选、添加集群。
- [ ] 总览只显示五项指定指标。
- [ ] 模型按照分类展示。
- [ ] 每张模型卡有 Logo、状态、健康度、集群数和详情入口。
- [ ] 右侧保留异常告警区域。
- [ ] 底部使用标准分页。
- [ ] 搜索、筛选和分页可以组合。

## 添加集群

- [ ] 表单只有模型、集群名称、集群连接密钥。
- [ ] 模型来自 New API 已部署模型列表。
- [ ] 不允许选择禁用模型。
- [ ] Secret 不回显、不记录日志、非明文保存。
- [ ] 创建成功后一级页面刷新。
- [ ] 数据关系不会因为模型重命名而完全失效。

## 数据采集架构

- [ ] 有独立 Agent Client。
- [ ] 有 schema adapter。
- [ ] 有轮询调度接口。
- [ ] 轮询间隔可配置。
- [ ] 有超时、并发限制和失败处理。
- [ ] Agent 原始 JSON 不直接进入 UI。
- [ ] 未知 schema 不会使后台任务崩溃。
- [ ] 预留历史仓库接口。
- [ ] GPU 数量动态处理。

## 二三级页面

- [ ] 模型集群列表可以进入。
- [ ] 集群详情可以进入。
- [ ] 详情页只有总览、引擎指标、机器指标三个 Tab。
- [ ] 基础数据和操作逻辑可演示。
- [ ] 未完成内容有明确接口和 TODO，而不是写死在页面里。

## 工程质量

- [ ] 没有引入不必要的新依赖。
- [ ] 数据库迁移可正常执行与回滚。
- [ ] lint、typecheck、test 和 build 通过。
- [ ] 修改集中在相关模块，没有破坏原功能。
- [ ] README 或开发文档说明了 Agent 接入、配置项和后续扩展点。
- [ ] 项目根目录存在 `docs/`，并按三位递增序号记录本次各阶段开发日志。

---

# 20. 最终交付说明

完成后请输出：

1. 实际修改的文件列表。
2. 数据库迁移说明。
3. 新增 API 列表。
4. Agent 轮询流程。
5. 模型数据联动方式。
6. Secret 的存储与脱敏方式。
7. 一级页面已完成功能。
8. 二三级页面当前完成程度。
9. 后续扩展接口的位置。
10. 运行和测试命令。
11. lint、test、typecheck、build 的执行结果。
12. 当前仍存在的限制或需要我确认的事项。
13. `docs/` 中本次开发日志的文件列表与内容摘要。

不要只输出静态页面或孤立组件。最终代码必须能集成到当前 New API 项目中运行，并为后续完整开发保留稳定、清晰的扩展边界。

---

# 21. 开发日志与项目文档

需要在项目根目录新增：

```text
docs/
```

并在该目录中按序号记录本次开发过程。

建议文件命名：

```text
docs/phase1
          ├── 001-cluster-module-plan.md
          ├── 002-cluster-backend-and-storage.md
          ├── 003-cluster-telemetry-poller.md
          ├── 004-cluster-status-page.md
          ├── 005-cluster-naive-detail-pages.md
          └── 006-cluster-testing-and-handoff.md
```

具体文件数量可以根据实际开发步骤调整，但必须满足：

1. 文件名前缀使用三位递增序号，例如 `001`、`002`、`003`。
2. 按实际开发顺序记录，不要事后只补一份笼统总结。
3. 每完成一个重要阶段，就同步新增或更新对应日志。
4. 已提交的历史日志原则上不要重写结论；后续修正应在新日志中说明。
5. 文档使用 Markdown。
6. 不要在日志中记录：
   - 集群连接密钥。
   - Bearer Token。
   - 数据库密码。
   - 完整私网敏感地址。
   - 其他 Secret。
7. 日志中的代码片段、接口路径和文件名必须与实际实现一致。
8. 如果项目根目录已经存在 `docs`，不要覆盖已有文档，应在其中按现有规范新增本次日志。
9. 如果项目已有 ADR、changelog 或开发日志规范，优先兼容已有规范，同时保留递增序号。

每篇日志至少包含：

```markdown
# 标题

## 日期

## 本阶段目标

## 实际修改

## 新增或修改的文件

## 数据库与接口变更

## 关键设计决策

## 遇到的问题

## 解决方式

## 测试与验证

## 当前限制

## 下一步
```

第一篇日志应先记录：

- 项目技术栈检查结果。
- 现有 New API 可复用组件。
- 模型数据来源。
- 权限控制方式。
- 数据库迁移方式。
- Agent 接入方案。
- 本阶段范围与非目标。

最后一篇日志应记录：

- 最终文件清单。
- 数据库迁移结果。
- API 清单。
- 配置项。
- 测试结果。
- lint、typecheck 和 build 结果。
- 未完成事项。
- 后续开发入口。
- 二三级页面继续开发时应复用的接口。

开发日志属于本次交付内容。功能完成但缺少对应日志，不视为完整交付。
