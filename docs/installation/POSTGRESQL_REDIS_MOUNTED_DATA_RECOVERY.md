# 使用挂载的 PostgreSQL 与 Redis 数据恢复 new-api

本文适用于以下场景：

- 新容器中已经挂载旧部署的 PostgreSQL 数据目录。
- 需要安装新的 PostgreSQL 与 Redis。
- 需要通过仓库迁移脚本把旧数据库恢复到新数据库。
- 最后让新版 new-api 继续使用原有用户、令牌、渠道、设置和集群数据。

本文以 Debian/Ubuntu 容器为例，并假设命令由 `root` 用户执行。

相关文件：

- [PostgreSQL 与 Redis 快速安装配置](./POSTGRESQL_REDIS_SETUP.md)
- [PostgreSQL 与 Redis 完整恢复说明](../database/postgresql-redis-recovery.md)
- [PostgreSQL 逻辑迁移脚本](../../scripts/database/postgresql-logical-migrate.sh)
- [迁移配置模板](../../scripts/database/postgresql-logical-migrate.env.example)

## 1. 恢复流程

PostgreSQL 原始数据目录不能由 new-api 直接读取，也不能直接交给不同大版本的 PostgreSQL 使用。

正确流程如下：

```text
挂载的旧 PostgreSQL 数据目录
            ↓
复制为恢复工作副本
            ↓
使用相同 PostgreSQL 大版本启动临时源库（55432）
            ↓
postgresql-logical-migrate.sh
            ↓
新的目标 PostgreSQL（5432）
            ↓
SQL_DSN
            ↓
新版 new-api
```

Redis 不由 `postgresql-logical-migrate.sh` 迁移。new-api 的永久业务数据主要位于 PostgreSQL，因此推荐先使用新的空 Redis。

## 2. 恢复前检查

假设旧 PostgreSQL 数据挂载在：

```bash
export NEWAPI_OLD_PGDATA='/mnt/old-postgresql'
```

检查它是否为 PostgreSQL 原始数据目录：

```bash
test -f "$NEWAPI_OLD_PGDATA/PG_VERSION"
cat "$NEWAPI_OLD_PGDATA/PG_VERSION"
```

如果返回：

```text
15
```

说明必须先使用 PostgreSQL 15 启动该目录。不能直接使用 PostgreSQL 16、17 或其他大版本。

恢复前还要确认：

1. 旧 PostgreSQL 已经正常停止，或者挂载目录来自一致性存储快照。
2. `pg_tblspc/` 中引用的外部表空间已经同时挂载。
3. 外置 WAL、证书和 PostgreSQL 配置文件没有缺失。
4. 已保存旧部署使用的数据库名、数据库账号和密码。
5. 已保存旧部署的 `SESSION_SECRET`、`CRYPTO_SECRET` 和集群密钥文件。

如果挂载的是 `.dump` 文件，而不是包含 `PG_VERSION`、`base/` 和 `global/` 的目录，则不需要启动临时源库，应直接使用 `pg_restore`。本文介绍的迁移脚本需要一个可以连接的源 PostgreSQL 服务。

## 3. 安装 PostgreSQL 与 Redis

读取旧 PostgreSQL 大版本：

```bash
export NEWAPI_OLD_PG_MAJOR="$(cat "$NEWAPI_OLD_PGDATA/PG_VERSION")"
printf '旧 PostgreSQL 大版本：%s\n' "$NEWAPI_OLD_PG_MAJOR"
```

安装相同大版本的 PostgreSQL，以及迁移所需工具：

```bash
apt-get update
apt-get install -y \
  "postgresql-${NEWAPI_OLD_PG_MAJOR}" \
  "postgresql-client-${NEWAPI_OLD_PG_MAJOR}" \
  redis-server \
  redis-tools \
  rsync \
  openssl
```

启动新安装的 PostgreSQL。它将作为目标数据库：

```bash
service postgresql start
pg_isready -h 127.0.0.1 -p 5432
```

如果目标 PostgreSQL 没有监听 `5432`，使用下面的命令查看实际端口：

```bash
pg_lsclusters
```

后续配置中的 `NEWAPI_TARGET_MAIN_PORT` 和 `SQL_DSN` 必须使用实际端口。

## 4. 创建旧 PostgreSQL 数据的工作副本

不要直接启动唯一的挂载原件。先复制到一个新的恢复目录：

```bash
export NEWAPI_PG_RECOVERY_COPY='/api_data/new-api/recovery/postgresql-old'

test ! -e "$NEWAPI_PG_RECOVERY_COPY"
install -d -o postgres -g postgres "$NEWAPI_PG_RECOVERY_COPY"

rsync -aHAX --numeric-ids \
  "$NEWAPI_OLD_PGDATA/" \
  "$NEWAPI_PG_RECOVERY_COPY/"

chown -R postgres:postgres "$NEWAPI_PG_RECOVERY_COPY"
```

`test ! -e` 用于防止覆盖已经存在的恢复副本。如果检查失败，请换一个新的空目录，不要删除或覆盖现有数据。

再次确认副本版本：

```bash
cat "$NEWAPI_PG_RECOVERY_COPY/PG_VERSION"
```

## 5. 启动临时源 PostgreSQL

临时源库使用 `55432`，避免与目标库的 `5432` 冲突：

```bash
export NEWAPI_OLD_PG_PORT='55432'
export NEWAPI_OLD_PG_SOCKET='/run/postgresql-old'
export NEWAPI_OLD_PG_LOG='/api_data/new-api/recovery/postgresql-old.log'

install -d -o postgres -g postgres "$NEWAPI_OLD_PG_SOCKET"

runuser -u postgres -- \
  "/usr/lib/postgresql/${NEWAPI_OLD_PG_MAJOR}/bin/pg_ctl" \
  -D "$NEWAPI_PG_RECOVERY_COPY" \
  -l "$NEWAPI_OLD_PG_LOG" \
  -o "-h 127.0.0.1 -p ${NEWAPI_OLD_PG_PORT} -k ${NEWAPI_OLD_PG_SOCKET}" \
  start
```

检查临时源库：

```bash
pg_isready -h 127.0.0.1 -p 55432
```

如果启动失败，查看：

```bash
tail -n 200 "$NEWAPI_OLD_PG_LOG"
```

常见原因包括 PostgreSQL 大版本不一致、目录权限错误、外部表空间或 WAL 缺失，以及旧配置引用了不存在的证书或扩展。

列出旧数据库：

```bash
runuser -u postgres -- \
  psql \
  -h "$NEWAPI_OLD_PG_SOCKET" \
  -p 55432 \
  -d postgres \
  -c "
SELECT datname
FROM pg_database
WHERE datistemplate = false
ORDER BY datname;
"
```

记录原 new-api 使用的数据库名，例如 `new_api`。

迁移脚本需要通过 TCP 连接临时源库。优先使用原数据库账号和密码。如果原密码未知，但本地 Socket 允许 `postgres` 用户登录，可以只在恢复副本中设置一个临时密码：

```bash
export NEWAPI_SOURCE_PASSWORD="$(openssl rand -hex 24)"
printf '请保存临时源库密码：%s\n' "$NEWAPI_SOURCE_PASSWORD"

runuser -u postgres -- \
  psql \
  -h "$NEWAPI_OLD_PG_SOCKET" \
  -p 55432 \
  -d postgres \
  -c "ALTER ROLE postgres WITH LOGIN PASSWORD '$NEWAPI_SOURCE_PASSWORD';"
```

该操作只修改恢复工作副本，不会修改挂载的原始目录。

## 6. 配置目标 PostgreSQL

恢复场景下只创建目标应用账号，不要提前创建目标数据库。迁移脚本要求目标数据库不存在，并负责创建数据库。

生成目标管理密码和应用密码：

```bash
export NEWAPI_TARGET_ADMIN_PASSWORD="$(openssl rand -hex 24)"
export NEWAPI_TARGET_APP_PASSWORD="$(openssl rand -hex 24)"

printf '请保存目标 PostgreSQL 管理密码：%s\n' \
  "$NEWAPI_TARGET_ADMIN_PASSWORD"
printf '请保存 new-api 数据库密码：%s\n' \
  "$NEWAPI_TARGET_APP_PASSWORD"
```

配置目标 PostgreSQL：

```bash
runuser -u postgres -- \
  psql \
  -p 5432 \
  -c "ALTER ROLE postgres WITH PASSWORD '$NEWAPI_TARGET_ADMIN_PASSWORD';"

runuser -u postgres -- \
  psql \
  -p 5432 \
  -c "CREATE ROLE new_api_app LOGIN PASSWORD '$NEWAPI_TARGET_APP_PASSWORD';"
```

不要执行 `createdb`。目标数据库由迁移脚本创建，以避免误覆盖已有数据库。

## 7. 配置并运行迁移脚本

复制配置模板到不会提交到 Git 的私密目录：

```bash
install -d -m 0700 /api_data/new-api/secrets

cp scripts/database/postgresql-logical-migrate.env.example \
  /api_data/new-api/secrets/postgresql-logical-migrate.env

chmod 600 /api_data/new-api/secrets/postgresql-logical-migrate.env
```

编辑：

```text
/api_data/new-api/secrets/postgresql-logical-migrate.env
```

只迁移主库时，核心配置如下：

```bash
NEWAPI_SOURCE_MAIN_HOST='127.0.0.1'
NEWAPI_SOURCE_MAIN_PORT='55432'
NEWAPI_SOURCE_MAIN_DATABASE='new_api'
NEWAPI_SOURCE_MAIN_USER='postgres'
NEWAPI_SOURCE_MAIN_PASSWORD='替换为临时源库密码'
NEWAPI_SOURCE_MAIN_SSLMODE='disable'

NEWAPI_TARGET_MAIN_HOST='127.0.0.1'
NEWAPI_TARGET_MAIN_PORT='5432'
NEWAPI_TARGET_MAIN_ADMIN_USER='postgres'
NEWAPI_TARGET_MAIN_ADMIN_PASSWORD='替换为目标管理密码'
NEWAPI_TARGET_MAIN_APP_USER='new_api_app'
NEWAPI_TARGET_MAIN_APP_PASSWORD='替换为目标应用密码'
NEWAPI_TARGET_MAIN_SSLMODE='disable'

NEWAPI_TARGET_MAIN_DATABASE='new_api_prod_restored_20260730'
NEWAPI_BACKUP_DIR='/api_data/new-api/backups/postgresql-migration'
```

需要修改的值：

- `NEWAPI_SOURCE_MAIN_DATABASE`：原 new-api 数据库名。
- `NEWAPI_SOURCE_MAIN_USER` 和 `NEWAPI_SOURCE_MAIN_PASSWORD`：临时源库登录信息。
- `NEWAPI_TARGET_MAIN_ADMIN_PASSWORD` 和 `NEWAPI_TARGET_MAIN_APP_PASSWORD`：目标管理账号和应用账号密码。
- `NEWAPI_TARGET_MAIN_DATABASE`：新的目标数据库名。

建议显式填写 `NEWAPI_TARGET_MAIN_DATABASE`，这样迁移后不会忘记数据库名。名称只能使用小写字母、数字和下划线，不能以数字开头。

先执行检查：

```bash
bash scripts/database/postgresql-logical-migrate.sh \
  --config /api_data/new-api/secrets/postgresql-logical-migrate.env \
  --check
```

检查通过后执行正式迁移：

```bash
bash scripts/database/postgresql-logical-migrate.sh \
  --config /api_data/new-api/secrets/postgresql-logical-migrate.env
```

脚本会要求输入类似下面的确认文字：

```text
MIGRATE new_api_prod_restored_20260730
```

迁移脚本将：

1. 从临时源库生成并校验 `pg_dump` 逻辑备份。
2. 创建全新的目标数据库。
3. 使用单个事务恢复备份。
4. 对比源库和目标库的关键表行数。
5. 输出备份文件和最终目标数据库名。

如果旧部署配置了独立的 `LOG_SQL_DSN`，还需要填写配置模板中的 `NEWAPI_SOURCE_LOG_*` 项，单独迁移日志数据库。

## 8. 配置 Redis

### 8.1 推荐使用新的空 Redis

PostgreSQL 保存用户、令牌、渠道、余额、订单、订阅、系统设置、集群配置和遥测历史。Redis 主要保存缓存、限流窗口和临时状态。

因此恢复 PostgreSQL 后，推荐直接使用新的空 Redis：

```bash
service redis-server start

export NEWAPI_REDIS_PASSWORD="$(openssl rand -hex 24)"
printf '请保存 Redis 密码：%s\n' "$NEWAPI_REDIS_PASSWORD"

redis-cli CONFIG SET requirepass "$NEWAPI_REDIS_PASSWORD"
REDISCLI_AUTH="$NEWAPI_REDIS_PASSWORD" redis-cli CONFIG SET appendonly yes
REDISCLI_AUTH="$NEWAPI_REDIS_PASSWORD" redis-cli CONFIG SET appendfsync everysec
REDISCLI_AUTH="$NEWAPI_REDIS_PASSWORD" redis-cli CONFIG REWRITE
REDISCLI_AUTH="$NEWAPI_REDIS_PASSWORD" redis-cli PING
```

使用空 Redis 不会删除 PostgreSQL 中的业务数据。缓存会在 new-api 运行过程中自动重新建立。

### 8.2 确实需要时恢复旧 Redis

旧 Redis 数据不经过 PostgreSQL 迁移脚本。如果必须恢复，需要：

1. 确认旧 Redis 大版本。
2. 将挂载的 Redis 数据目录复制为工作副本。
3. 保留完整的 `dump.rdb` 或 AOF 目录和 manifest。
4. 使用与旧数据兼容的 Redis 版本启动工作副本。
5. 配置正确的 `dir`、`dbfilename`、`appendonly` 和 `appenddirname`。
6. 复用旧 ACL/密码，或者在隔离环境设置新的认证信息。

不要直接在唯一的 Redis 数据原件上运行修复命令。Redis 7 的多段 AOF 必须保留完整的 `appendonlydir`，不能只复制单个 AOF 文件。

## 9. 让 new-api 使用迁移后的数据

创建启动环境文件。数据库名必须与迁移配置中的 `NEWAPI_TARGET_MAIN_DATABASE` 完全一致：

```bash
umask 077

{
  printf "export SQL_DSN='postgresql://new_api_app:%s@127.0.0.1:5432/new_api_prod_restored_20260730?sslmode=disable'\n" \
    "$NEWAPI_TARGET_APP_PASSWORD"
  printf "export REDIS_CONN_STRING='redis://:%s@127.0.0.1:6379/0'\n" \
    "$NEWAPI_REDIS_PASSWORD"
} > /api_data/new-api/secrets/new-api.env

chmod 600 /api_data/new-api/secrets/new-api.env
```

如果密码不是本文生成的十六进制密码，而是包含 `@`、`:`、`/`、`#` 等字符的旧密码，需要先进行 URI 编码，或者使用 libpq 关键字格式的 PostgreSQL 连接串。

每次启动前加载配置：

```bash
source /api_data/new-api/secrets/new-api.env
```

确认变量已设置，但不要输出完整连接串：

```bash
[[ -n "${SQL_DSN:-}" ]] && echo 'SQL_DSN 已设置'
[[ -n "${REDIS_CONN_STRING:-}" ]] && echo 'REDIS_CONN_STRING 已设置'
```

启动 new-api：

```bash
cd /api_data/new-api/code/new-api
./scripts/update-build-run.sh
```

第一次连接恢复数据库时，只启动一个 new-api 主节点，让它完成当前版本需要的 GORM 表结构迁移。确认迁移成功后再启动其他节点。

## 10. 必须恢复原部署密钥

数据库恢复成功并不代表所有加密信息都一定能解密。新部署还需要复用原部署实际使用的：

```text
SESSION_SECRET
CRYPTO_SECRET
CLUSTER_SECRET_KEY_FILE 指向的文件
```

如果旧部署日志显示：

```text
cluster secret protection initialized: source=file:某个路径
```

需要把对应密钥文件安全复制到新部署的持久化目录，并通过 `CLUSTER_SECRET_KEY_FILE` 指向它。

如果丢失这些密钥：

- 历史登录会话可能失效。
- OAuth、支付或其他加密配置可能无法正常使用。
- 已保存的集群 Agent Bearer Token 可能无法解密，需要重新轮换。

密钥文件权限建议设置为：

```bash
chmod 600 /实际路径/.new-api-cluster-secret.key
```

## 11. 恢复验证

确认 PostgreSQL 连接：

```bash
PGPASSWORD="$NEWAPI_TARGET_APP_PASSWORD" \
  psql \
  -h 127.0.0.1 \
  -p 5432 \
  -U new_api_app \
  -d new_api_prod_restored_20260730 \
  -c 'SELECT current_database(), current_user;'
```

检查核心表数据量：

```bash
PGPASSWORD="$NEWAPI_TARGET_APP_PASSWORD" \
  psql \
  -h 127.0.0.1 \
  -p 5432 \
  -U new_api_app \
  -d new_api_prod_restored_20260730 \
  -c "
SELECT 'users' AS table_name, COUNT(*) FROM users
UNION ALL
SELECT 'tokens', COUNT(*) FROM tokens
UNION ALL
SELECT 'channels', COUNT(*) FROM channels;
"
```

确认 Redis：

```bash
REDISCLI_AUTH="$NEWAPI_REDIS_PASSWORD" redis-cli PING
```

检查 new-api 启动日志中是否出现：

```text
using PostgreSQL as database
Redis is enabled
database migration started
New API ... started
```

最后通过管理界面确认：

- 原用户可以登录。
- Token、渠道和系统设置仍然存在。
- 用户余额和订阅数据正确。
- 集群列表和遥测历史正确。
- 已有集群 Agent 凭据可以正常解密和连接。

## 12. 验证完成后的处理

确认新版 new-api 已稳定运行后，可以停止临时源 PostgreSQL：

```bash
runuser -u postgres -- \
  "/usr/lib/postgresql/${NEWAPI_OLD_PG_MAJOR}/bin/pg_ctl" \
  -D "$NEWAPI_PG_RECOVERY_COPY" \
  -m fast \
  stop
```

在完成业务验收和备份之前，不要删除：

- 挂载的原始 PostgreSQL 数据。
- PostgreSQL 恢复工作副本。
- 迁移脚本生成的 `.dump` 文件。
- 原部署密钥文件。

目标 PostgreSQL、Redis 数据目录和 `/api_data/new-api/secrets` 都应放在持久化存储中，避免容器重建后再次丢失。
