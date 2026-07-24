# 集群连接密钥跨重启持久化

## 日期

2026-07-24

## 本阶段目标

修复 New API 停止、重新编译并启动后，已有集群因 Agent 连接密钥无法解密而全部离线的问题，并确认该问题不是数据库记录或迁移不一致导致。

## 问题定位

在部署实例中进行了只读检查：

- 进程没有设置 `SESSION_SECRET`；
- 进程没有设置 `CRYPTO_SECRET`；
- `/data/new-api/one-api.db` 仍然存在，集群轮询器能够继续读取原有集群 ID；
- 重启后的日志明确返回 `CLUSTER_SECRET_INVALID`。

原实现直接使用 `common.CryptoSecret` 加密 Agent Bearer Token。未设置上述两个环境变量时，`common.CryptoSecret` 会在每次进程启动时重新生成。数据库中的集群配置没有丢失，但新进程无法使用新的随机密钥解开旧密文，因此将轮询失败记录为集群离线。

## 实际修改

- 新增集群专用密钥解析与持久化逻辑。
- 保留显式环境变量的优先级和原有兼容行为：
  1. `CRYPTO_SECRET`
  2. `SESSION_SECRET`
  3. 持久化密钥文件
- 未配置环境变量时，首次启动自动生成 32 字节随机密钥，并以 Base64URL 格式写入密钥文件。
- SQLite 部署默认将密钥文件放在数据库文件同一目录，文件名为 `.new-api-cluster-secret.key`。
- MySQL 或 PostgreSQL 部署默认将密钥文件放在进程工作目录。
- 可通过 `CLUSTER_SECRET_KEY_FILE` 显式指定密钥文件路径。
- 密钥文件使用 `0600` 权限；已有文件权限过宽、内容损坏、不是普通文件或超过大小限制时拒绝启动集群模块，不会静默覆盖。
- 启动日志只记录密钥来源和文件路径，不输出密钥内容。
- 将默认密钥文件名加入 `.gitignore`，避免本地运行时误提交。

## 新增或修改的文件

- `service/clusterstatus/secret_key.go`
- `service/clusterstatus/secret_key_test.go`
- `service/clusterstatus/poller.go`
- `.gitignore`
- `docs/develop/phase1/008-cluster-secret-restart-persistence.md`

## 数据库与迁移结论

本次不新增数据表或字段，不需要数据库迁移。

已有 `clusters` 和 `cluster_telemetry_latest` 数据仍在原数据库中。离线问题发生在读取集群配置之后、解密 Agent 连接信息时，因此不是数据库文件切换、表结构不一致或关联丢失。

数据库只保存 AES-GCM 密文，持久化密钥文件不写入数据库。备份和迁移时必须同时保存：

- 主数据库；
- `.new-api-cluster-secret.key`，或部署所使用的 `CRYPTO_SECRET`/`SESSION_SECRET`。

只恢复数据库而没有对应密钥时，Agent Bearer Token 无法恢复。

## 兼容与恢复说明

升级前已经被旧的临时随机密钥加密、且旧进程已经退出的集群，无法从密文反推出 Bearer Token。这是 AES-GCM 加密的预期安全属性，不是数据迁移能够修复的。

部署本次修改后：

1. 启动一次新版本，让服务生成并保存稳定密钥文件；
2. 删除当前显示 `CLUSTER_SECRET_INVALID` 的旧集群；
3. 使用原 Agent 地址和 Bearer Token 重新添加一次；
4. 后续停止、重新编译和启动会继续读取同一个密钥文件，不再因进程重启失效。

如果升级前本来就配置了固定 `CRYPTO_SECRET` 或 `SESSION_SECRET`，继续使用原值即可直接解密，不需要重建集群。

生产环境或多节点部署仍建议显式配置相同的高强度 `CRYPTO_SECRET`。多个 New API 实例连接同一数据库时，所有实例必须使用同一个环境变量密钥，或访问同一份 `CLUSTER_SECRET_KEY_FILE`。

## 测试与验证

回归测试覆盖：

- `CRYPTO_SECRET` 优先于 `SESSION_SECRET` 和密钥文件；
- 自动生成的密钥跨多次加载保持一致；
- 使用第二次加载得到的密钥可以解开第一次加密的连接信息；
- 自动生成文件权限为 `0600`；
- SQLite 默认密钥文件位于数据库同一目录；
- 损坏的已有密钥文件不会被静默覆盖。

验证命令：

```bash
go test ./service/clusterstatus
go build ./...
```

部署后可在启动日志中确认密钥来源：

```text
cluster secret protection initialized: source=generated_file:/data/new-api/.new-api-cluster-secret.key
```

第二次及后续启动应显示：

```text
cluster secret protection initialized: source=file:/data/new-api/.new-api-cluster-secret.key
```
