# 集群状态一级页面

## 日期

2026-07-23

## 本阶段目标

按照参考图完成管理员集群状态一级页面，并沿用 New API 的布局、组件、主题、权限和国际化机制。

## 实际修改

- 完成搜索、模型筛选、状态筛选和添加集群操作区。
- 完成五个指定总览指标、模型分类卡片、实际在线率、异常告警和标准分页。
- 完成添加集群 Dialog，表单包含模型、集群名称、Agent IP 与端口、Agent Bearer Token。
- 模型选择器支持接口加载、搜索、Empty、Loading、禁用状态和模型图标。
- 接入 React Query 查询、后台刷新、创建后缓存失效、Skeleton、Empty 和 Error 状态。

## 新增或修改的文件

- `web/src/features/cluster-status/index.tsx`
- `web/src/features/cluster-status/api.ts`
- `web/src/features/cluster-status/query-keys.ts`
- `web/src/features/cluster-status/types.ts`
- `web/src/features/cluster-status/components/add-cluster-dialog.tsx`
- `web/src/features/cluster-status/components/model-selector.tsx`
- `web/src/features/cluster-status/components/model-avatar.tsx`
- `web/src/features/cluster-status/components/overview-toolbar.tsx`
- `web/src/features/cluster-status/components/overview-content.tsx`
- `web/src/features/cluster-status/components/status-badge.tsx`
- `web/src/features/cluster-status/lib/format.ts`
- `web/src/i18n/locales/*.json`

## 数据库与接口变更

页面使用：

- `GET /api/clusters/overview`
- `GET /api/clusters/model-options`
- `POST /api/clusters/`

搜索、模型、状态、页码和每页数量由一级聚合接口组合处理。

## 关键设计决策

- 使用现有 `SectionPageLayout`、Base UI/shadcn 组件和 Tailwind 语义主题，不创建独立 CSS 体系。
- 模型卡片的在线率只根据当前真实集群状态计算；Agent 未提供历史数据时不绘制虚构趋势。
- 请求和 Token 缺失时显示 `—`，不把缺失值伪装为实际零值。
- 所有用户可见新增文案通过 `useTranslation()`，并同步七个前端 locale。
- 前端不再要求管理员生成 `sgta1.` 连接密钥，也不持久化或回显 Agent Bearer Token。

## 遇到的问题

- 参考图包含小时趋势线，但 Agent 第一阶段只返回当前状态和最近窗口摘要。
- 原生 Select 无法同时满足模型搜索和图标展示。

## 解决方式

- 使用真实在线率进度条表达当前健康程度，并明确不伪造历史。
- 复用现有 Popover、Command、Button 和模型图标映射组合可搜索模型选择器。

## 测试与验证

- TypeScript 类型检查通过。
- 涉及文件 oxlint 检查通过。
- 前端生产构建通过。
- 单元测试覆盖缺失指标、GPU 动态平均值、严重度排序和分页窗口。

## 当前限制

- 暂无真实历史趋势图。
- 告警区域展示当前异常摘要，不包含告警历史和通知。

## 下一步

完成模型集群列表页和集群详情三个 Tab 的 Naive 版本。
