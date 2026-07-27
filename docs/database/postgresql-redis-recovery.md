# 使用 PostgreSQL 与 Redis 数据目录恢复 new-api

> 本文面向 PostgreSQL + Redis 部署。所有示例路径、账号、端口和密码都必须替换为实际值。
>
> 恢复操作应先在副本和隔离环境中演练，确认无误后再切换生产流量。

## 1. 最重要的结论

new-api **不能直接读取 PostgreSQL 或 Redis 的数据文件夹**。

正确的数据链路是：

```text
PostgreSQL 数据目录 -> PostgreSQL 服务 -> SQL_DSN -> new-api
Redis 数据目录      -> Redis 服务      -> REDIS_CONN_STRING -> new-api
```

给定两个数据文件夹后，需要先用兼容版本的 PostgreSQL 和 Redis 服务加载目录，再让新部署的 new-api 连接这两个服务。

不要把 PostgreSQL 的 `base/`、`global/` 等文件复制到 new-api 目录，也不要让 new-api 直接读取 `dump.rdb` 或 AOF 文件。这样无法恢复数据。

## 2. 数据恢复优先级

| 数据 | 恢复要求 | 原因 |
| --- | --- | --- |
| PostgreSQL 主库 | 必须 | 用户、额度、令牌、渠道、设置、订阅、任务和集群配置都在主库 |
| PostgreSQL 日志库 | 配置过 `LOG_SQL_DSN` 时必须单独恢复 | 请求日志可能不在主库 |
| Redis | 通常可不恢复 | 主要保存缓存、限流窗口、会话快照和渠道亲和性 |
| 原部署密钥 | 必须 | 用于保持登录态、缓存键和集群 Agent 凭据可解密 |

推荐顺序：

1. 恢复 PostgreSQL 主库。
2. 如果有独立日志库，恢复 PostgreSQL 日志库。
3. 使用一个新的空 Redis 启动 new-api，确认业务数据完整。
4. 只有确实需要保留限流窗口、渠道亲和性等临时状态时，才恢复旧 Redis。

使用空 Redis 不会删除 PostgreSQL 中的用户、渠道、令牌、余额、订阅或集群配置。缓存会在运行过程中重新建立。

## 3. 选择正确的恢复方案

| 现有条件 | 推荐方案 |
| --- | --- |
| 旧 PostgreSQL、Redis 服务仍可连接 | 新 new-api 直接连接旧服务，完成验证后切换流量 |
| 有 `pg_dump`/`pg_dumpall` 逻辑备份 | 恢复到全新的 PostgreSQL 实例 |
| 只有 PostgreSQL 原始数据目录 | 用相同 PostgreSQL 大版本从目录副本启动临时实例，先导出逻辑备份，再恢复到新实例 |
| 只有 Redis 数据目录 | PostgreSQL 恢复完成后先使用空 Redis；确有需要时再从 RDB/AOF 副本恢复 |
| 只有 new-api 文件，没有 PostgreSQL 备份或数据目录 | 无法从 Redis 重建完整业务数据库 |

逻辑备份比直接搬运原始数据目录更适合跨机器、跨容器或升级 PostgreSQL 大版本。

## 4. 恢复前必须收集的信息

### 4.1 PostgreSQL

- 主库数据目录的绝对路径。
- 数据目录中的 `PG_VERSION` 内容。
- 原 PostgreSQL 的完整小版本和 CPU 架构。
- 原数据库名、账号和认证方式。
- 是否配置了表空间；检查 `pg_tblspc/` 中是否存在符号链接。
- WAL 是否位于外部目录。
- 是否另有 `LOG_SQL_DSN` 日志数据库。
- 原 `postgresql.conf`、`pg_hba.conf` 以及部署配置。

只读检查示例：

```bash
export NEWAPI_OLD_PGDATA='/srv/postgresql/data'

test -f "$NEWAPI_OLD_PGDATA/PG_VERSION"
cat "$NEWAPI_OLD_PGDATA/PG_VERSION"
ls -ld "$NEWAPI_OLD_PGDATA"
ls -la "$NEWAPI_OLD_PGDATA/pg_tblspc"
```

`PG_VERSION` 如果是 `15`，恢复原始目录时必须先使用 PostgreSQL 15。不能直接让 PostgreSQL 16、17 或其他大版本打开该目录。

### 4.2 Redis

- Redis 数据目录的绝对路径。
- 原 Redis 版本。
- 原 `redis.conf`。
- 是 RDB、AOF，还是 RDB + AOF。
- Redis 逻辑 DB 编号，例如连接串末尾的 `/0`。
- 是否配置 ACL 文件或 `requirepass`。

只读检查示例：

```bash
export NEWAPI_OLD_REDIS_DATA='/srv/redis/data'

find "$NEWAPI_OLD_REDIS_DATA" -maxdepth 3 -type f \
  \( -name 'dump.rdb' -o -name '*.aof' -o -name '*.manifest' \) \
  -print
```

Redis 7 的多段 AOF 通常位于 `appendonlydir/`，恢复时需要整个目录和 manifest，不能只复制其中一个增量文件。

### 4.3 new-api 部署密钥

安全保存原部署中的：

- `SESSION_SECRET`
- `CRYPTO_SECRET`
- `CLUSTER_SECRET_KEY_FILE` 指向的文件
- 其他包含 OAuth、支付和上游认证信息的部署密钥

集群 Agent 凭据的解密材料按当前实现优先使用 `CRYPTO_SECRET`，其次是 `SESSION_SECRET`；如果两者均未提供，则使用集群密钥文件。新部署必须延续原部署实际使用的那一份材料。

如果密钥丢失，即使 `clusters` 表成功恢复，已有 `link_secret_ciphertext` 也可能无法解密，需要重新轮换 Agent Bearer Token。

## 5. 生产切换前停止写入

备份时必须得到一致的数据快照。

推荐流程：

1. 从负载均衡器移除旧 new-api，停止新的 API 请求和后台管理写入。
2. 如果启用了 `BATCH_UPDATE_ENABLED=true`，等待至少两个 `BATCH_UPDATE_INTERVAL`。默认间隔为 5 秒，因此至少等待 10 秒，并确认日志中出现批量更新完成的信息。
3. 正常停止旧 new-api。
4. 对 PostgreSQL 执行逻辑备份，或在 PostgreSQL 已完全停止后复制物理目录。
5. 复制 Redis 数据时，先执行持久化并正常停止 Redis，或者使用存储系统的一致性快照。

不要在数据库仍运行且持续写入时，用普通文件复制命令复制 PostgreSQL 数据目录。这样的目录可能无法启动，或在不明显报错的情况下缺少一致数据。

切换期间不要让旧 new-api 和新 new-api 同时作为可写主节点连接同一数据库。

## 6. 方案 A：直接复用仍在运行的数据库服务

这是风险最低的迁移方式，不需要操作原始数据目录。

在新 new-api 的环境配置中设置：

```dotenv
SQL_DSN=postgresql://newapi:REDACTED@postgres.example.internal:5432/new_api?sslmode=require
LOG_SQL_DSN=postgresql://newapi_log:REDACTED@postgres-log.example.internal:5432/new_api_log?sslmode=require
REDIS_CONN_STRING=redis://:REDACTED@redis.example.internal:6379/0

SESSION_SECRET=与原部署完全相同
CRYPTO_SECRET=与原部署完全相同
```

如果没有独立日志库，不要设置 `LOG_SQL_DSN`。

注意：

- new-api 运行在容器中时，`127.0.0.1` 指向 new-api 容器本身，不是 PostgreSQL 或 Redis 容器。应使用服务名、容器网络 DNS 名或实际 IP。
- 用户名、密码中包含 `@`、`:`、`/`、`#`、`?` 等字符时，需要按 URI 规则编码。
- 不要把带密码的 `.env` 或 DSN 提交到 Git。
- 先从新环境执行只读连接检查，再启动 new-api。

## 7. 方案 B：从 PostgreSQL 逻辑备份恢复

### 7.1 在旧数据库导出

以下命令可以在 PostgreSQL 服务仍在线时创建一致的逻辑备份，但应先停止 new-api 写入：

```bash
export NEWAPI_SOURCE_DSN='postgresql://backup_user:REDACTED@127.0.0.1:5432/new_api'
export NEWAPI_BACKUP_FILE='/backup/new-api-main.dump'

pg_dump \
  --format=custom \
  --no-owner \
  --no-acl \
  --dbname="$NEWAPI_SOURCE_DSN" \
  --file="$NEWAPI_BACKUP_FILE"

pg_restore --list "$NEWAPI_BACKUP_FILE"
```

如果使用独立日志库，对 `LOG_SQL_DSN` 指向的数据库再导出一份：

```bash
export NEWAPI_LOG_SOURCE_DSN='postgresql://backup_user:REDACTED@127.0.0.1:5432/new_api_log'
export NEWAPI_LOG_BACKUP_FILE='/backup/new-api-log.dump'

pg_dump \
  --format=custom \
  --no-owner \
  --no-acl \
  --dbname="$NEWAPI_LOG_SOURCE_DSN" \
  --file="$NEWAPI_LOG_BACKUP_FILE"
```

如果还需要迁移 PostgreSQL 角色和授权，可另外执行：

```bash
pg_dumpall \
  --globals-only \
  --database="$NEWAPI_SOURCE_DSN" \
  --file=/backup/postgresql-globals.sql
```

全局对象备份可能包含角色密码哈希，应按敏感文件管理。很多场景下更安全的做法是在目标数据库手工创建最小权限账号，而不是直接恢复全部全局对象。

### 7.2 恢复到新数据库

先创建一个全新的空数据库。不要把备份直接覆盖到正在使用的数据库。

```bash
export NEWAPI_TARGET_ADMIN_DSN='postgresql://postgres:REDACTED@127.0.0.1:5432/postgres'
export NEWAPI_TARGET_DSN='postgresql://newapi:REDACTED@127.0.0.1:5432/new_api_restored'

createdb \
  --maintenance-db="$NEWAPI_TARGET_ADMIN_DSN" \
  --owner=newapi \
  --template=template0 \
  new_api_restored

pg_restore \
  --exit-on-error \
  --single-transaction \
  --no-owner \
  --no-acl \
  --dbname="$NEWAPI_TARGET_DSN" \
  "$NEWAPI_BACKUP_FILE"
```

目标数据库必须为空。`createdb` 在同名数据库已经存在时会失败，这可以避免误覆盖。

大数据库不适合单事务恢复时，可以去掉 `--single-transaction`，但应在隔离环境恢复，并在任意错误后废弃该目标库，重新创建空库再恢复。

### 7.3 使用仓库的一键迁移脚本

仓库提供了方案 B 的辅助脚本：

```text
scripts/database/postgresql-logical-migrate.sh
scripts/database/postgresql-logical-migrate.env.example
```

脚本负责：

1. 检查 `psql`、`pg_dump`、`pg_restore` 和 `createdb`。
2. 检查源库、目标 PostgreSQL 版本、目标账号和目标库名称。
3. 拒绝覆盖任何已经存在的目标数据库。
4. 未手工指定名称时，自动生成带日期的目标数据库名。
5. 先生成并校验 PostgreSQL custom 格式逻辑备份。
6. 创建全新的目标数据库，并在单个事务中恢复。
7. 对比源库和目标库的关键表行数。
8. 可选地同时迁移独立的 `LOG_SQL_DSN` 日志库。

脚本不会读取源或目标 PostgreSQL 的 `PGDATA` 原始目录。方案 B 中需要填写的是源库和目标库的连接参数，以及 `.dump` 逻辑备份的保存目录。如果手里只有 `PGDATA`，应先按“方案 C”使用相同 PostgreSQL 大版本启动目录副本，再回到本脚本执行逻辑迁移。

脚本可以在宿主机、运维机或临时工具容器中运行，不要求该机器运行 new-api，但必须能同时访问源、目标 PostgreSQL，并安装 `psql`、`pg_dump`、`pg_restore`、`createdb`。PostgreSQL 客户端版本不能低于源数据库服务端的大版本。

先复制配置模板到仓库外的私密目录：

```bash
cp scripts/database/postgresql-logical-migrate.env.example \
  /data/secrets/postgresql-logical-migrate.env
chmod 600 /data/secrets/postgresql-logical-migrate.env
```

配置模板已经把连接参数拆开，脚本内部会组合成 libpq 连接串。只迁移主库时必须填写：

| 配置项 | 用途 |
| --- | --- |
| `NEWAPI_SOURCE_MAIN_HOST` | 旧 PostgreSQL 地址 |
| `NEWAPI_SOURCE_MAIN_DATABASE` | 旧 new-api 数据库名 |
| `NEWAPI_SOURCE_MAIN_USER` | 旧数据库账号 |
| `NEWAPI_SOURCE_MAIN_PASSWORD` | 旧数据库密码 |
| `NEWAPI_TARGET_MAIN_HOST` | 目标 PostgreSQL 地址 |
| `NEWAPI_TARGET_MAIN_ADMIN_USER` | 目标管理账号，必须拥有 `CREATEDB` 权限 |
| `NEWAPI_TARGET_MAIN_ADMIN_PASSWORD` | 目标管理账号密码 |
| `NEWAPI_TARGET_MAIN_APP_USER` | 已存在的目标应用账号 |
| `NEWAPI_TARGET_MAIN_APP_PASSWORD` | 目标应用账号密码 |
| `NEWAPI_BACKUP_DIR` | `.dump` 逻辑备份保存目录，不是 `PGDATA` |

端口默认是 `5432`，SSL 模式默认是 `prefer`，目标管理库默认是 `postgres`。这些默认值可以分别通过 `NEWAPI_SOURCE_MAIN_PORT`、`NEWAPI_TARGET_MAIN_PORT`、`NEWAPI_SOURCE_MAIN_SSLMODE`、`NEWAPI_TARGET_MAIN_SSLMODE` 和 `NEWAPI_TARGET_MAIN_ADMIN_DATABASE` 修改。

旧部署使用独立 `LOG_SQL_DSN` 时，再填写模板中的 `NEWAPI_SOURCE_LOG_*` 配置。目标日志库默认复用目标主库的服务器、管理连接和应用账号。

如需直接使用完整 DSN，仍可填写 `NEWAPI_SOURCE_MAIN_DSN`、`NEWAPI_TARGET_MAIN_ADMIN_DSN`、`NEWAPI_TARGET_MAIN_DSN_TEMPLATE` 和 `NEWAPI_TARGET_MAIN_OWNER`。完整 DSN 优先于拆分参数；URI 密码中的特殊字符需要编码。

目标应用账号必须提前创建，管理连接使用的账号必须具有 `CREATEDB` 权限。目标数据库本身不能提前创建，脚本会用 `template0` 创建它。

默认名称使用运行脚本机器的当前日期：

```text
主库：new_api_prod_YYYYMMDD
日志库：new_api_log_prod_YYYYMMDD
```

例如在 2026 年 7 月 27 日执行时，主库名称为 `new_api_prod_20260727`。运行 `--check` 时会显示最终生成的名称。需要固定名称时，可以在配置文件中填写 `NEWAPI_TARGET_MAIN_DATABASE`；DSN 中无需同步修改，脚本会自动替换 `__DATABASE__` 占位符。

先执行只读检查：

```bash
./scripts/database/postgresql-logical-migrate.sh \
  --config /data/secrets/postgresql-logical-migrate.env \
  --check
```

停止旧 new-api 写入后执行正式迁移：

```bash
./scripts/database/postgresql-logical-migrate.sh \
  --config /data/secrets/postgresql-logical-migrate.env
```

正式迁移要求在交互式终端输入目标数据库名确认。失败时脚本不会删除备份或目标数据库，也不会自动重试覆盖；应保留现场排查，并换一个新的空目标库重新执行。

脚本只完成 PostgreSQL 逻辑备份、恢复和基础行数核对。之后仍需让新版 new-api 连接目标库，并且只启动一个主节点来执行当前版本的 GORM 数据库迁移。

## 8. 方案 C：只有 PostgreSQL 原始数据目录

### 8.1 不要直接使用原目录

原始目录必须保持不变，用于回退和二次恢复。先在数据库停止或存储快照一致的前提下创建工作副本。

以下命令在数据库宿主机执行：

```bash
export NEWAPI_OLD_PGDATA='/srv/postgresql/data'
export NEWAPI_PG_RECOVERY_COPY='/srv/recovery/postgresql-data-copy'

test -f "$NEWAPI_OLD_PGDATA/PG_VERSION"
test ! -e "$NEWAPI_PG_RECOVERY_COPY"

mkdir -p "$NEWAPI_PG_RECOVERY_COPY"
rsync -aHAX --numeric-ids \
  "$NEWAPI_OLD_PGDATA/" \
  "$NEWAPI_PG_RECOVERY_COPY/"

cat "$NEWAPI_PG_RECOVERY_COPY/PG_VERSION"
```

`test ! -e` 是防误覆盖检查。如果目标路径已存在，应人工确认并选择新的空目录，不要删除或覆盖已有恢复副本。

还必须复制或重新挂载：

- `pg_tblspc/` 中符号链接指向的全部表空间。
- 配置在其他路径的 WAL。
- PostgreSQL 启动所需的证书、密钥和外部配置。

### 8.2 使用相同大版本启动临时恢复实例

临时 PostgreSQL 必须满足：

- 与 `PG_VERSION` 相同的大版本。
- CPU 架构和操作系统兼容。
- 对工作副本具有正确的文件属主和权限。
- 不暴露到公网。
- 不连接生产 new-api。

如果宿主机可使用容器，可在**宿主机**用固定版本镜像启动工作副本。下面以 PostgreSQL 15 为例，实际版本必须替换为 `PG_VERSION` 的值：

```yaml
services:
  postgres-recovery:
    image: postgres:15
    restart: "no"
    ports:
      - "127.0.0.1:55432:5432"
    volumes:
      - /srv/recovery/postgresql-data-copy:/var/lib/postgresql/data
```

这是宿主机编排配置，不是在 new-api 容器里执行的命令。已经进入 new-api 镜像且没有 Docker/Podman 指令时，需要让宿主机或平台管理员启动恢复实例。

原始数据目录已经初始化时，`POSTGRES_PASSWORD` 等初始化变量不会重置旧账号密码。需要使用原数据库账号认证。若账号和认证信息均丢失，应停止操作并由 PostgreSQL 管理员在隔离环境修复认证。

启动失败时重点检查：

- 镜像大版本是否与 `PG_VERSION` 一致。
- 目录 UID/GID 和权限是否匹配。
- `postmaster.pid` 是否来自未正常关闭的副本；不要在原始目录上处理。
- 表空间或 WAL 的符号链接目标是否缺失。
- `postgresql.conf` 中是否引用了不存在的证书、库或外部路径。
- 数据目录是否来自运行中数据库的非一致文件复制。

### 8.3 从临时实例导出，再恢复到正式新实例

临时实例能够正常启动后，不建议把这个物理副本直接作为长期生产库。应通过 `pg_dump` 导出可移植逻辑备份，再按“方案 B”恢复。

```bash
export NEWAPI_RECOVERY_DSN='postgresql://原账号:REDACTED@127.0.0.1:55432/new_api'
export NEWAPI_RECOVERED_DUMP='/srv/recovery/new-api-recovered.dump'

psql "$NEWAPI_RECOVERY_DSN" \
  -c 'SELECT current_database(), current_user, version();'

pg_dump \
  --format=custom \
  --no-owner \
  --no-acl \
  --dbname="$NEWAPI_RECOVERY_DSN" \
  --file="$NEWAPI_RECOVERED_DUMP"

pg_restore --list "$NEWAPI_RECOVERED_DUMP"
```

跨 PostgreSQL 大版本迁移时，使用逻辑备份恢复，或使用 PostgreSQL 官方 `pg_upgrade` 流程。绝不能让新大版本直接打开旧大版本数据目录。

## 9. Redis 恢复

### 9.1 推荐：启动一个新的空 Redis

先恢复 PostgreSQL，再启动固定版本且启用持久化的新 Redis：

```dotenv
REDIS_CONN_STRING=redis://:REDACTED@redis:6379/0
```

空 Redis 会造成：

- 用户和令牌缓存重新建立。
- 登录会话缓存重新建立。
- 当前限流窗口清零。
- 渠道亲和性和部分统计缓存清零。

不会造成：

- 用户、余额、渠道、令牌或系统设置从 PostgreSQL 消失。
- 订阅和订单数据丢失。
- `clusters`、`cluster_telemetry_latest` 和 `cluster_telemetry_history` 表数据丢失。

### 9.2 确实需要时，从 RDB/AOF 恢复

先复制工作副本，绝不在唯一原件上检查或修复：

```bash
export NEWAPI_OLD_REDIS_DATA='/srv/redis/data'
export NEWAPI_REDIS_RECOVERY_COPY='/srv/recovery/redis-data-copy'

test -d "$NEWAPI_OLD_REDIS_DATA"
test ! -e "$NEWAPI_REDIS_RECOVERY_COPY"

mkdir -p "$NEWAPI_REDIS_RECOVERY_COPY"
rsync -aHAX --numeric-ids \
  "$NEWAPI_OLD_REDIS_DATA/" \
  "$NEWAPI_REDIS_RECOVERY_COPY/"
```

在与原 Redis 相同版本的工具中检查副本：

```bash
redis-check-rdb /srv/recovery/redis-data-copy/dump.rdb
redis-check-aof /srv/recovery/redis-data-copy/appendonlydir/appendonly.aof.manifest
```

只执行实际存在的文件对应的检查命令。不要对唯一原件执行 `redis-check-aof --fix`；修复操作可能改写文件。

启动恢复实例时必须复用与原数据一致的关键配置：

- Redis 大版本。
- `dir`、`dbfilename`。
- `appendonly`、`appenddirname` 和 AOF manifest。
- ACL 文件或密码配置。
- 实际使用的逻辑 DB 编号。

宿主机编排示例，Redis 版本必须替换为原实例的固定版本：

```yaml
services:
  redis-recovery:
    image: redis:7.2
    restart: "no"
    ports:
      - "127.0.0.1:56379:6379"
    volumes:
      - /srv/recovery/redis-data-copy:/data
      - /srv/recovery/redis.conf:/usr/local/etc/redis/redis.conf:ro
    command:
      - redis-server
      - /usr/local/etc/redis/redis.conf
```

恢复后先在隔离环境检查：

```bash
export NEWAPI_RECOVERY_REDIS_URL='redis://:REDACTED@127.0.0.1:56379/0'

redis-cli -u "$NEWAPI_RECOVERY_REDIS_URL" PING
redis-cli -u "$NEWAPI_RECOVERY_REDIS_URL" DBSIZE
redis-cli -u "$NEWAPI_RECOVERY_REDIS_URL" INFO persistence
```

RDB/AOF 中的过期时间会被保留。恢复时已经过期的键不会重新生效，这是正常现象。

### 9.3 当前仓库 Compose 配置的注意事项

当前仓库的 `docker-compose.yml` 为 PostgreSQL 配置了：

```text
pg_data:/var/lib/postgresql/data
```

但 Redis 服务没有挂载 `/data` 持久化卷，并且使用了浮动的 `redis:latest` 镜像。删除并重建 Redis 容器后，Redis 数据可能无法保留。

生产环境建议：

- 固定 Redis 版本，不使用 `latest`。
- 为 `/data` 配置持久化卷。
- 根据业务要求启用 RDB，或启用 `appendonly yes` 与 `appendfsync everysec`。
- PostgreSQL 与 Redis 都应有独立于容器生命周期的备份。

## 10. 新部署的 new-api 配置

完整示例：

```dotenv
SQL_DSN=postgresql://newapi:REDACTED@postgres:5432/new_api_restored?sslmode=require

# 只有旧部署使用了独立日志库时才设置
LOG_SQL_DSN=postgresql://newapi_log:REDACTED@postgres-log:5432/new_api_log_restored?sslmode=require

REDIS_CONN_STRING=redis://:REDACTED@redis:6379/0

SESSION_SECRET=与原部署完全相同
CRYPTO_SECRET=与原部署完全相同

# 仅在原部署通过文件保存集群密钥时配置
CLUSTER_SECRET_KEY_FILE=/data/secrets/.new-api-cluster-secret.key
```

集群密钥文件应从原部署安全复制，挂载到新容器的稳定路径，并限制文件权限：

```bash
chmod 600 /data/secrets/.new-api-cluster-secret.key
```

不要为了让实例“先跑起来”而临时生成新的 `SESSION_SECRET` 或 `CRYPTO_SECRET`。这可能让历史会话失效，并使已有加密字段无法解密。

## 11. 启动顺序

1. 启动恢复后的 PostgreSQL 主库。
2. 如有独立日志库，启动日志数据库。
3. 启动新 Redis，或经过验证的 Redis 恢复实例。
4. 从 new-api 运行环境测试 PostgreSQL 和 Redis 网络连通性。
5. 确认已经保存首次启动前备份。
6. 仅启动一个 new-api 主节点，让当前代码执行 GORM 迁移。
7. 检查启动日志中是否出现 PostgreSQL、数据库迁移和 Redis 已启用相关信息。
8. 完成数据验收后再接入外部流量。
9. 最后再启动其他从节点或扩容实例。

如果节点被配置为 slave，它不会执行主数据库迁移。首次升级恢复时必须有且只有一个主节点完成迁移。

## 12. 恢复后的数据库验收

### 12.1 连接与版本

```bash
export NEWAPI_VERIFY_DSN='postgresql://newapi:REDACTED@127.0.0.1:5432/new_api_restored'

psql "$NEWAPI_VERIFY_DSN" \
  -c 'SELECT current_database(), current_user, version();'

psql "$NEWAPI_VERIFY_DSN" -c '\dt+ public.*'
```

### 12.2 关键表数量

在旧库和恢复库分别执行，并对比结果：

```sql
SELECT 'users' AS table_name, COUNT(*) AS row_count FROM users
UNION ALL
SELECT 'tokens', COUNT(*) FROM tokens
UNION ALL
SELECT 'channels', COUNT(*) FROM channels
UNION ALL
SELECT 'options', COUNT(*) FROM options
UNION ALL
SELECT 'subscription_plans', COUNT(*) FROM subscription_plans
UNION ALL
SELECT 'user_subscriptions', COUNT(*) FROM user_subscriptions
UNION ALL
SELECT 'clusters', COUNT(*) FROM clusters
UNION ALL
SELECT 'cluster_telemetry_latest', COUNT(*) FROM cluster_telemetry_latest
UNION ALL
SELECT 'cluster_telemetry_history', COUNT(*) FROM cluster_telemetry_history
ORDER BY table_name;
```

如果使用独立日志库，再检查：

```sql
SELECT COUNT(*) AS log_count, MIN(created_at), MAX(created_at)
FROM logs;
```

仅比较总行数还不够。还应抽样核对：

- 管理员账号和普通用户能否登录。
- 用户额度、已用额度和订阅状态。
- API 令牌能否正常鉴权。
- 渠道密钥和模型路由能否正常使用。
- 系统设置是否与旧环境一致。
- 集群状态页面是否能读取集群，Agent Bearer Token 是否可以验证。
- 请求产生后，`logs`、`quota_data` 和计费字段是否继续增加。

## 13. 故障时的回退

在新环境验收完成前：

- 保留旧 PostgreSQL、Redis 和 new-api，不要删除。
- 保留原始物理目录的只读副本。
- 保留恢复前的逻辑备份。
- 不让旧、新两套应用同时写同一数据库。
- 使用负载均衡或服务发现切换流量，而不是覆盖旧数据。

如果新版本迁移或业务验收失败：

1. 停止向新环境发送流量。
2. 停止新 new-api 的写入。
3. 保留失败现场和日志用于排查。
4. 将流量切回仍保持一致的旧环境。
5. 不要试图通过删除迁移后的字段来“降级”数据库。

## 14. 后续备份建议

- PostgreSQL：至少每日逻辑备份，生产环境同时使用带时间点恢复能力的物理备份/WAL 归档。
- 独立日志库：按合规与保留周期单独备份。
- Redis：即使作为缓存，也应固定版本、挂载持久化目录，并按是否需要临时状态连续性决定是否备份。
- 部署密钥：使用密钥管理系统保存，必须能够与数据库备份配套恢复。
- 每次升级 new-api 前创建 PostgreSQL 备份。
- 定期在隔离环境做完整恢复演练。只有实际验证过的备份才算可用备份。

## 15. 官方参考

- [PostgreSQL：SQL Dump](https://www.postgresql.org/docs/current/backup-dump.html)
- [PostgreSQL：File System Level Backup](https://www.postgresql.org/docs/current/backup-file.html)
- [PostgreSQL：pg_restore](https://www.postgresql.org/docs/current/app-pgrestore.html)
- [PostgreSQL：pg_upgrade](https://www.postgresql.org/docs/current/pgupgrade.html)
- [Redis：Persistence](https://redis.io/docs/latest/operate/oss_and_stack/management/persistence/)
