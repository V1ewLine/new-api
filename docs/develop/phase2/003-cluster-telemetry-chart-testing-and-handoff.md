# 集群趋势图表测试与交接

## 1. 自动化验证

### 后端

执行：

```bash
GOCACHE=/tmp/new-api-go-cache go test ./service/clusterstatus ./router
```

覆盖：

- 依据时间跨度自动选择分桶；
- 同一桶选择最后一条成功采样；
- 失败采样参与采集成功率计算；
- 失败桶的业务指标保持空值；
- GPU UUID、单卡功耗和板卡总功耗映射；
- 集群不存在、时间顺序错误、超过保留周期和点数超限；
- 趋势路由受管理员鉴权保护。

### 前端

执行：

```bash
cd web
node --test src/features/cluster-status/__tests__/*.test.ts
./node_modules/.bin/tsgo -b
./node_modules/.bin/oxlint -c .oxlintrc.json \
  src/features/cluster-status \
  scripts/add-missing-keys.mjs
./node_modules/.bin/rsbuild build
```

趋势范围测试覆盖：

- 相对时间窗口按自动刷新时间向前移动；
- 相对范围保持稳定查询键；
- 自定义范围保留精确到秒的 API 参数；
- 自定义范围查询键区分不同起止时间；
- 反向范围和超过保留周期的范围被拒绝。

## 2. 部署后的人工验收

1. 保证历史保留天数大于需要查看的时间窗口。
2. 保证 Agent 连续运行并至少积累数分钟采样。
3. 进入“集群状态 → 模型 → 集群详情”。
4. 验证概览、引擎指标、机器指标三个标签页均能看到对应曲线。
5. 切换最近 15 分钟、1 小时、6 小时，确认所有图表同时更新。
6. 选择精确到秒的固定时间范围，等待一个自动刷新周期，确认时间范围不会向前移动。
7. 点击任意图表右下角箭头，确认浮层打开且原页面状态保留。
8. 在浮层中切换时间窗口，确认页面公共时间窗口没有改变。
9. 点击页面手动刷新按钮，确认最新状态和相对时间趋势重新查询。
10. 在 Agent 暂时不可用的时间段检查“遥测可用性”，确认成功率下降且其他指标显示断点而不是零值。

## 3. 数据语义提醒

- `running_requests` 和 `waiting_requests` 是采样时刻的请求压力，不是时间段累计请求量。
- `token_usage` 是采样时刻的 Token 占用，不是时间段累计 Token 消耗。
- `throughput` 是 Agent 上报的生成吞吐量。
- `gpu_board_power_watts` 是所有 GPU 板卡功耗之和，不是整机功耗。
- 每个分桶的业务指标取桶内最后一条成功采样；`poll_success_percent` 使用桶内全部成功和失败采样计算。
- 趋势接口不进行线性插值，缺失数据应在图表上表现为断点。

## 4. 运维注意事项

- 图表数据来自主数据库的 `cluster_telemetry_history`，Redis 不承担趋势持久化。
- 缩短 `ClusterTelemetryRetentionDays` 会由清理任务永久删除超期采样，之后无法通过图表恢复。
- 增大图表时间窗口不会突破保留周期。
- 前端普通图表最多请求 720 点，放大图表最多请求 1440 点；接口硬上限为 2000 点。
- 如果页面只有最新数值而没有曲线，先检查历史表是否有该集群记录，以及采样的 `normalized_payload` 是否包含对应字段。

## 5. 本阶段数据库影响

没有新增表、字段或索引，不需要执行额外迁移。

现有 PostgreSQL、MySQL 或 SQLite 部署升级后，服务会直接读取原有 `cluster_telemetry_history` 数据。Redis 数据不参与恢复。
