# PostgreSQL 与 Redis 快速安装配置

本文适用于 Debian/Ubuntu，以及已经进入运行容器、无法使用 Docker 命令的场景。以下命令使用 `root` 用户执行。

## 1. 下载 PostgreSQL 与 Redis

安装软件：

```bash
apt-get update
apt-get install -y \
  postgresql \
  postgresql-client \
  redis-server \
  redis-tools \
  openssl
```

启动服务：

```bash
service postgresql start
service redis-server start
```

确认服务正常：

```bash
pg_isready
redis-cli PING
```

正常情况下会分别看到 `accepting connections` 和 `PONG`。

> 如果运行环境可能被重建，请确保 `/var/lib/postgresql` 和 `/var/lib/redis` 位于持久化磁盘，否则数据库文件可能随环境重建而丢失。

## 2. 快速配置 PostgreSQL 与 Redis，让本项目使用

### 配置 PostgreSQL

生成密码，并创建 new-api 专用账号和数据库：

```bash
export NEWAPI_PG_PASSWORD="$(openssl rand -hex 24)"
printf '请保存 PostgreSQL 密码：%s\n' "$NEWAPI_PG_PASSWORD"

runuser -u postgres -- \
  psql \
  -v ON_ERROR_STOP=1 \
  -c "CREATE ROLE new_api_app LOGIN PASSWORD '$NEWAPI_PG_PASSWORD';"

runuser -u postgres -- \
  createdb \
  --owner new_api_app \
  new_api_prod
```

验证连接：

```bash
PGPASSWORD="$NEWAPI_PG_PASSWORD" \
  psql \
  -h 127.0.0.1 \
  -p 5432 \
  -U new_api_app \
  -d new_api_prod \
  -c 'SELECT current_database(), current_user;'
```

new-api 第一次启动时会自动创建表，不需要手工导入表结构。

### 配置 Redis

生成密码、设置密码并开启 AOF 持久化：

```bash
export NEWAPI_REDIS_PASSWORD="$(openssl rand -hex 24)"
printf '请保存 Redis 密码：%s\n' "$NEWAPI_REDIS_PASSWORD"

redis-cli CONFIG SET requirepass "$NEWAPI_REDIS_PASSWORD"
REDISCLI_AUTH="$NEWAPI_REDIS_PASSWORD" redis-cli CONFIG SET appendonly yes
REDISCLI_AUTH="$NEWAPI_REDIS_PASSWORD" redis-cli CONFIG SET appendfsync everysec
REDISCLI_AUTH="$NEWAPI_REDIS_PASSWORD" redis-cli CONFIG REWRITE
```

验证连接：

```bash
REDISCLI_AUTH="$NEWAPI_REDIS_PASSWORD" redis-cli PING
```

正常结果为 `PONG`。

### 配置并启动 new-api

保存数据库连接配置：

```bash
install -d -m 0700 /api_data/new-api/secrets
umask 077

{
  printf "export SQL_DSN='postgresql://new_api_app:%s@127.0.0.1:5432/new_api_prod?sslmode=disable'\n" \
    "$NEWAPI_PG_PASSWORD"
  printf "export REDIS_CONN_STRING='redis://:%s@127.0.0.1:6379/0'\n" \
    "$NEWAPI_REDIS_PASSWORD"
} > /api_data/new-api/secrets/new-api.env
```

以后每次启动项目前，先加载配置：

```bash
source /api_data/new-api/secrets/new-api.env
```

然后进入项目目录并启动：

```bash
cd /api_data/new-api/code/new-api
./scripts/update-build-run.sh
```

启动日志中出现下面两项，表示 PostgreSQL 和 Redis 已成功启用：

```text
using PostgreSQL as database
Redis is enabled
```

连接串格式：

```text
SQL_DSN=postgresql://用户名:密码@主机:端口/数据库名?sslmode=disable
REDIS_CONN_STRING=redis://:密码@主机:端口/数据库编号
```

`REDIS_CONN_STRING` 必须保留 `redis://`，不能只填写 `127.0.0.1:6379`。
