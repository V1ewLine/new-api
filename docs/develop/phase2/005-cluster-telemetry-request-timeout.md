# 集群遥测请求默认超时调整

## 1. 调整背景

new-api 定时请求 Agent 的 `/v1/telemetry` 接口时，原默认超时时间为 3 秒。该接口需要同时汇总 SGLang 引擎负载和机器采样；在远程网络存在短暂抖动，或 Agent 正在执行 GPU、CPU 和内存采样时，3 秒可能导致正常但稍慢的响应被记录为 `AGENT_TIMEOUT`。

## 2. 调整内容

`CLUSTER_TELEMETRY_REQUEST_TIMEOUT_SECONDS` 的默认值由 3 秒调整为 10 秒。

未显式设置该环境变量时：

- Agent 请求超时为 10 秒；
- 轮询租约默认值仍为“请求超时加 5 秒”，因此同步调整为 15 秒。

已经显式设置 `CLUSTER_TELEMETRY_REQUEST_TIMEOUT_SECONDS` 的部署继续使用配置值，不受默认值变化影响。

## 3. 生效方式

该配置在集群状态服务初始化时读取。升级代码后需要重启 new-api，无需重新编译 Agent，也不涉及数据库迁移。

如需覆盖默认值，可以在启动 new-api 前设置：

```bash
export CLUSTER_TELEMETRY_REQUEST_TIMEOUT_SECONDS=15
```

## 4. 回归测试

新增测试覆盖：

- 未配置环境变量时，请求超时为 10 秒、轮询租约为 15 秒；
- 显式配置请求超时时，配置值继续生效，轮询租约随之调整。

执行：

```bash
go test ./service/clusterstatus
```

## 5. 影响范围

本次调整只改变 Agent 遥测请求的默认等待时间，不修改：

- 集群状态刷新间隔；
- 连续失败阈值；
- 指数退避策略；
- Agent Bearer Token；
- 历史采样保留周期；
- 数据库表结构与 Redis 数据。
