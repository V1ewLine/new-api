# 集群详情趋势图表界面

## 1. 页面范围

本阶段在集群状态三级页面的三个标签页中增加历史曲线：

### 概览

- 遥测可用性：采集成功率、引擎可用率、机器可用率；
- 请求压力：运行中请求、等待中请求；
- 吞吐量；
- 资源利用率：CPU、内存、缓存。

### 引擎指标

- 运行中请求；
- 等待中请求；
- Token 使用量；
- 吞吐量；
- 缓存使用率。

### 机器指标

- GPU 板卡总功耗；
- 单个 GPU 功耗，多 GPU 使用独立序列；
- CPU 利用率；
- 内存利用率。

桌面端每行显示两个图表，窄屏设备自动改为单列。

## 2. 页面级时间窗口

集群详情页只保留一个公共时间窗口控件，三个标签页的全部图表共用该设置。

预设范围：

- 最近 15 分钟；
- 最近 1 小时，默认值；
- 最近 6 小时；
- 最近 24 小时；
- 最近 7 天。

管理员也可以使用精确到秒的开始时间和结束时间。输入时间使用浏览器本地时区，发送接口前转换为 RFC3339 UTC。

相对时间窗口会按照“系统设置 → 控制台内容 → 数据仪表盘”中的集群状态刷新间隔自动向前移动并重新查询。自定义固定时间窗口不自动移动，但页面原有手动刷新按钮会继续使集群详情和趋势查询缓存失效。

超过当前历史保留天数的预设会禁用；自定义范围顺序错误或跨度超限时不能提交，并显示明确错误。

## 3. 图表交互

每张图表卡片右下角提供向右箭头按钮。点击后打开大尺寸浮层：

- 保留原页面、标签页和滚动位置；
- 使用相同指标的放大曲线；
- 浮层拥有独立时间窗口；
- 修改浮层时间窗口不会改变页面公共时间窗口；
- 放大查询使用最多 1440 个点，普通卡片使用最多 720 个点。

浮层使用项目现有 Dialog 组件，包含可访问标题、描述、关闭操作和键盘焦点管理。

## 4. 图表规则

- 使用项目现有 VChart 和主题适配；
- 监控曲线使用直线连接，不启用可能暗示额外插值的平滑曲线；
- 缺失采样保持断点，不用零值连接；
- 百分比指标固定显示 0～100% 纵轴；
- 请求数、Token、吞吐量和功耗从零开始；
- 密集时间范围隐藏数据点，短时间范围显示数据点；
- 多序列图表显示图例；
- 工具提示使用浏览器本地时区显示精确到秒的时间；
- 单个 GPU 使用 UUID 保持跨采样序列稳定。

页面提供加载骨架、查询失败重试、时间范围无采样、单项指标缺失四种反馈状态。

## 5. 请求与刷新策略

普通页面对一个时间窗口只发送一次趋势请求，概览、引擎和机器图表共享同一份 React Query 数据，避免每张图表分别请求。

趋势查询键包含：

```text
clusterId + 时间范围描述 + maxPoints
```

相对范围使用稳定查询键，每次自动刷新时在查询函数内重新计算当前起止时间；自定义范围使用开始和结束毫秒作为键。放大浮层仅在打开时启用自己的查询。

## 6. 国际化

新增图表、时间窗口、加载、空状态和错误文案，使用现有 i18next 体系。通过 `web/scripts/add-missing-keys.mjs` 写入并执行 `scripts/sync-i18n.mjs` 校验，七个前端语言文件均无缺失键。

## 7. 主要改动文件

- `web/src/features/cluster-status/cluster-detail.tsx`
- `web/src/features/cluster-status/api.ts`
- `web/src/features/cluster-status/types.ts`
- `web/src/features/cluster-status/query-keys.ts`
- `web/src/features/cluster-status/lib/trend-range.ts`
- `web/src/features/cluster-status/hooks/use-cluster-telemetry-trends.ts`
- `web/src/features/cluster-status/components/telemetry-time-range-control.tsx`
- `web/src/features/cluster-status/components/telemetry-trend-chart.tsx`
- `web/src/features/cluster-status/components/telemetry-trend-groups.tsx`
- `web/src/features/cluster-status/__tests__/trend-range.test.ts`
- `web/src/i18n/locales/{en,zh,zh-TW,fr,ja,ru,vi}.json`
