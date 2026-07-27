#!/usr/bin/env bash

# Copyright (C) 2023-2026 QuantumNous
# SPDX-License-Identifier: AGPL-3.0-or-later

set -euo pipefail
umask 077

usage() {
  cat <<'EOF'
用法：
  ./scripts/database/postgresql-logical-migrate.sh --config /绝对路径/postgresql-logical-migrate.env
  ./scripts/database/postgresql-logical-migrate.sh --config /绝对路径/postgresql-logical-migrate.env --check

说明：
  按“方案 B”将仍可连接的 PostgreSQL 源数据库逻辑备份并恢复到一个全新的目标数据库。
  脚本不会读取或复制 PostgreSQL 的 PGDATA 原始数据目录，也不会迁移 Redis。
  未指定目标数据库名时，主库默认生成为 new_api_prod_YYYYMMDD。

选项：
  --config FILE  迁移配置文件，必填
  --check        只检查工具、连接、版本、账号和目标库状态，不执行迁移
  -h, --help     显示帮助

安全约束：
  - 目标数据库必须不存在，脚本不会覆盖或删除已有数据库。
  - 迁移前必须停止旧 new-api 写入，并在终端输入确认文字。
  - 失败时保留备份和已创建的目标库，不自动删除，便于排查。
EOF
}

info() {
  printf '[INFO] %s\n' "$*"
}

warn() {
  printf '[WARN] %s\n' "$*" >&2
}

die() {
  printf '[ERROR] %s\n' "$*" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || die "缺少命令：$1"
}

require_variable() {
  local variable_name="$1"
  [[ -n "${!variable_name:-}" ]] || die "配置项不能为空：$variable_name"
}

validate_identifier() {
  local label="$1"
  local value="$2"
  [[ "$value" =~ ^[a-z_][a-z0-9_]{0,62}$ ]] ||
    die "$label 只能包含小写字母、数字和下划线，且不能以数字开头：$value"
}

validate_prefix() {
  local label="$1"
  local value="$2"
  [[ "$value" =~ ^[a-z_][a-z0-9_]*$ ]] ||
    die "$label 只能包含小写字母、数字和下划线，且不能以数字开头：$value"
  ((${#value} <= 54)) ||
    die "$label 最多允许 54 个字符，以便追加日期后不超过 PostgreSQL 名称长度限制"
}

build_target_dsn() {
  local label="$1"
  local database_name="$2"
  local dsn_template="$3"
  [[ "$dsn_template" == *'__DATABASE__'* ]] ||
    die "${label}目标 DSN 模板必须包含 __DATABASE__ 占位符"
  printf '%s' "${dsn_template//__DATABASE__/$database_name}"
}

quote_conninfo_value() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\'/\\\'}"
  printf "'%s'" "$value"
}

build_conninfo() {
  local host="$1"
  local port="$2"
  local database="$3"
  local user="$4"
  local password="$5"
  local sslmode="$6"
  printf 'host=%s port=%s dbname=%s user=%s password=%s sslmode=%s' \
    "$(quote_conninfo_value "$host")" \
    "$(quote_conninfo_value "$port")" \
    "$(quote_conninfo_value "$database")" \
    "$(quote_conninfo_value "$user")" \
    "$(quote_conninfo_value "$password")" \
    "$(quote_conninfo_value "$sslmode")"
}

validate_port() {
  local label="$1"
  local value="$2"
  [[ "$value" =~ ^[0-9]+$ ]] &&
    ((value >= 1 && value <= 65535)) ||
    die "$label 必须是 1 到 65535 之间的端口号：$value"
}

validate_sslmode() {
  local label="$1"
  local value="$2"
  case "$value" in
    disable | allow | prefer | require | verify-ca | verify-full) ;;
    *) die "$label 不是有效的 PostgreSQL sslmode：$value" ;;
  esac
}

psql_scalar() {
  local dsn="$1"
  local sql="$2"
  PGCONNECT_TIMEOUT="$NEWAPI_PGCONNECT_TIMEOUT" \
    psql "$dsn" \
      --no-psqlrc \
      --no-align \
      --tuples-only \
      --quiet \
      --set ON_ERROR_STOP=on \
      --command "$sql" |
    tr -d '\r'
}

database_exists() {
  local admin_dsn="$1"
  local database_name="$2"
  local result
  result="$(psql_scalar "$admin_dsn" \
    "SELECT COUNT(*) FROM pg_database WHERE datname = '$database_name';")"
  [[ "$result" == "1" ]]
}

role_exists() {
  local admin_dsn="$1"
  local role_name="$2"
  local result
  result="$(psql_scalar "$admin_dsn" \
    "SELECT COUNT(*) FROM pg_roles WHERE rolname = '$role_name';")"
  [[ "$result" == "1" ]]
}

check_database_pair() {
  local label="$1"
  local source_dsn="$2"
  local target_admin_dsn="$3"
  local target_database="$4"
  local target_owner="$5"
  local source_database
  local source_version
  local source_version_number
  local target_version
  local target_version_number
  local source_major
  local target_major

  info "检查${label}源数据库连接"
  source_database="$(psql_scalar "$source_dsn" 'SELECT current_database();')"
  source_version="$(psql_scalar "$source_dsn" 'SHOW server_version;')"
  source_version_number="$(psql_scalar "$source_dsn" 'SHOW server_version_num;')"

  info "检查${label}目标 PostgreSQL 管理连接"
  target_version="$(psql_scalar "$target_admin_dsn" 'SHOW server_version;')"
  target_version_number="$(psql_scalar "$target_admin_dsn" 'SHOW server_version_num;')"

  source_major=$((source_version_number / 10000))
  target_major=$((target_version_number / 10000))
  if ((target_major < source_major)); then
    die "${label}目标 PostgreSQL 大版本不能低于源版本：源 ${source_version}，目标 ${target_version}"
  fi

  database_exists "$target_admin_dsn" "$target_database" &&
    die "${label}目标数据库已存在，脚本拒绝覆盖：$target_database"
  role_exists "$target_admin_dsn" "$target_owner" ||
    die "${label}目标账号不存在：$target_owner"

  info "${label}源数据库：${source_database}（PostgreSQL ${source_version}）"
  info "${label}目标数据库：${target_database}（PostgreSQL ${target_version}，当前不存在）"
  info "${label}目标对象所有者：${target_owner}"
}

create_dump() {
  local label="$1"
  local source_dsn="$2"
  local dump_file="$3"
  local partial_file="${dump_file}.partial"

  [[ ! -e "$dump_file" && ! -e "$partial_file" ]] ||
    die "备份文件已经存在，请更换备份目录或时间后重试：$dump_file"

  info "导出${label}逻辑备份：$dump_file"
  if ! PGCONNECT_TIMEOUT="$NEWAPI_PGCONNECT_TIMEOUT" \
    pg_dump \
      --format=custom \
      --no-owner \
      --no-acl \
      --dbname="$source_dsn" \
      --file="$partial_file"; then
    die "${label}逻辑备份导出失败，未创建目标数据库"
  fi

  if ! pg_restore --list "$partial_file" >/dev/null; then
    die "${label}逻辑备份目录校验失败：$partial_file"
  fi

  mv "$partial_file" "$dump_file"
  chmod 600 "$dump_file"
  info "${label}逻辑备份校验通过"
}

restore_dump() {
  local label="$1"
  local target_admin_dsn="$2"
  local target_dsn="$3"
  local target_database="$4"
  local target_owner="$5"
  local dump_file="$6"
  local connected_database
  local connected_user

  info "创建${label}目标数据库：$target_database"
  if ! PGCONNECT_TIMEOUT="$NEWAPI_PGCONNECT_TIMEOUT" \
    createdb \
      --maintenance-db="$target_admin_dsn" \
      --owner="$target_owner" \
      --template=template0 \
      "$target_database"; then
    die "${label}目标数据库创建失败；备份仍保留在 $dump_file"
  fi

  connected_database="$(psql_scalar "$target_dsn" 'SELECT current_database();')"
  [[ "$connected_database" == "$target_database" ]] ||
    die "${label}目标 DSN 实际连接到 ${connected_database}，不是配置的 ${target_database}"

  connected_user="$(psql_scalar "$target_dsn" 'SELECT current_user;')"
  [[ "$connected_user" == "$target_owner" ]] ||
    die "${label}目标 DSN 使用账号 ${connected_user}，应使用对象所有者 ${target_owner}"

  info "恢复${label}逻辑备份"
  if ! PGCONNECT_TIMEOUT="$NEWAPI_PGCONNECT_TIMEOUT" \
    pg_restore \
      --exit-on-error \
      --single-transaction \
      --no-owner \
      --no-acl \
      --dbname="$target_dsn" \
      "$dump_file"; then
    die "${label}恢复失败；目标库 ${target_database} 和备份文件均已保留，请排查后创建另一个空库重试"
  fi
}

verify_table_counts() {
  local label="$1"
  local source_dsn="$2"
  local target_dsn="$3"
  shift 3
  local table_name
  local source_exists
  local target_exists
  local source_count
  local target_count

  info "核对${label}关键表行数"
  for table_name in "$@"; do
    source_exists="$(psql_scalar "$source_dsn" \
      "SELECT CASE WHEN to_regclass('public.${table_name}') IS NULL THEN 0 ELSE 1 END;")"
    [[ "$source_exists" == "1" ]] || continue

    target_exists="$(psql_scalar "$target_dsn" \
      "SELECT CASE WHEN to_regclass('public.${table_name}') IS NULL THEN 0 ELSE 1 END;")"
    [[ "$target_exists" == "1" ]] ||
      die "${label}目标库缺少源库已有的表：$table_name"

    source_count="$(psql_scalar "$source_dsn" "SELECT COUNT(*) FROM \"${table_name}\";")"
    target_count="$(psql_scalar "$target_dsn" "SELECT COUNT(*) FROM \"${table_name}\";")"
    [[ "$source_count" == "$target_count" ]] ||
      die "${label}表 ${table_name} 行数不一致：源 ${source_count}，目标 ${target_count}"
    printf '  %-36s %s\n' "$table_name" "$target_count"
  done
}

config_file=''
check_only=false

while (($# > 0)); do
  case "$1" in
    --config)
      (($# >= 2)) || die '--config 后必须提供配置文件路径'
      config_file="$2"
      shift 2
      ;;
    --check)
      check_only=true
      shift
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      die "未知参数：$1"
      ;;
  esac
done

[[ -n "$config_file" ]] || die '必须使用 --config 指定迁移配置文件'
[[ -r "$config_file" ]] || die "无法读取迁移配置文件：$config_file"

# 配置文件由操作者创建并保存数据库凭据，因此按受信任的 shell 配置加载。
# shellcheck disable=SC1090
source "$config_file"

NEWAPI_PGCONNECT_TIMEOUT="${NEWAPI_PGCONNECT_TIMEOUT:-10}"
NEWAPI_VERIFY_ROW_COUNTS="${NEWAPI_VERIFY_ROW_COUNTS:-true}"
NEWAPI_TARGET_MAIN_DATABASE_PREFIX="${NEWAPI_TARGET_MAIN_DATABASE_PREFIX:-new_api_prod}"

if [[ -z "${NEWAPI_SOURCE_MAIN_DSN:-}" ]]; then
  NEWAPI_SOURCE_MAIN_PORT="${NEWAPI_SOURCE_MAIN_PORT:-5432}"
  NEWAPI_SOURCE_MAIN_SSLMODE="${NEWAPI_SOURCE_MAIN_SSLMODE:-prefer}"
  require_variable NEWAPI_SOURCE_MAIN_HOST
  require_variable NEWAPI_SOURCE_MAIN_DATABASE
  require_variable NEWAPI_SOURCE_MAIN_USER
  require_variable NEWAPI_SOURCE_MAIN_PASSWORD
  validate_port NEWAPI_SOURCE_MAIN_PORT "$NEWAPI_SOURCE_MAIN_PORT"
  validate_sslmode NEWAPI_SOURCE_MAIN_SSLMODE "$NEWAPI_SOURCE_MAIN_SSLMODE"
  NEWAPI_SOURCE_MAIN_DSN="$(build_conninfo \
    "$NEWAPI_SOURCE_MAIN_HOST" \
    "$NEWAPI_SOURCE_MAIN_PORT" \
    "$NEWAPI_SOURCE_MAIN_DATABASE" \
    "$NEWAPI_SOURCE_MAIN_USER" \
    "$NEWAPI_SOURCE_MAIN_PASSWORD" \
    "$NEWAPI_SOURCE_MAIN_SSLMODE")"
fi

if [[ -z "${NEWAPI_TARGET_MAIN_ADMIN_DSN:-}" ]]; then
  NEWAPI_TARGET_MAIN_PORT="${NEWAPI_TARGET_MAIN_PORT:-5432}"
  NEWAPI_TARGET_MAIN_ADMIN_DATABASE="${NEWAPI_TARGET_MAIN_ADMIN_DATABASE:-postgres}"
  NEWAPI_TARGET_MAIN_SSLMODE="${NEWAPI_TARGET_MAIN_SSLMODE:-prefer}"
  require_variable NEWAPI_TARGET_MAIN_HOST
  require_variable NEWAPI_TARGET_MAIN_ADMIN_USER
  require_variable NEWAPI_TARGET_MAIN_ADMIN_PASSWORD
  validate_port NEWAPI_TARGET_MAIN_PORT "$NEWAPI_TARGET_MAIN_PORT"
  validate_sslmode NEWAPI_TARGET_MAIN_SSLMODE "$NEWAPI_TARGET_MAIN_SSLMODE"
  NEWAPI_TARGET_MAIN_ADMIN_DSN="$(build_conninfo \
    "$NEWAPI_TARGET_MAIN_HOST" \
    "$NEWAPI_TARGET_MAIN_PORT" \
    "$NEWAPI_TARGET_MAIN_ADMIN_DATABASE" \
    "$NEWAPI_TARGET_MAIN_ADMIN_USER" \
    "$NEWAPI_TARGET_MAIN_ADMIN_PASSWORD" \
    "$NEWAPI_TARGET_MAIN_SSLMODE")"
fi

if [[ -z "${NEWAPI_TARGET_MAIN_DSN_TEMPLATE:-}" &&
  -z "${NEWAPI_TARGET_MAIN_DSN:-}" ]]; then
  require_variable NEWAPI_TARGET_MAIN_HOST
  require_variable NEWAPI_TARGET_MAIN_APP_USER
  require_variable NEWAPI_TARGET_MAIN_APP_PASSWORD
  NEWAPI_TARGET_MAIN_PORT="${NEWAPI_TARGET_MAIN_PORT:-5432}"
  NEWAPI_TARGET_MAIN_SSLMODE="${NEWAPI_TARGET_MAIN_SSLMODE:-prefer}"
  validate_port NEWAPI_TARGET_MAIN_PORT "$NEWAPI_TARGET_MAIN_PORT"
  validate_sslmode NEWAPI_TARGET_MAIN_SSLMODE "$NEWAPI_TARGET_MAIN_SSLMODE"
  NEWAPI_TARGET_MAIN_DSN_TEMPLATE="$(build_conninfo \
    "$NEWAPI_TARGET_MAIN_HOST" \
    "$NEWAPI_TARGET_MAIN_PORT" \
    '__DATABASE__' \
    "$NEWAPI_TARGET_MAIN_APP_USER" \
    "$NEWAPI_TARGET_MAIN_APP_PASSWORD" \
    "$NEWAPI_TARGET_MAIN_SSLMODE")"
fi

NEWAPI_TARGET_MAIN_OWNER="${NEWAPI_TARGET_MAIN_OWNER:-${NEWAPI_TARGET_MAIN_APP_USER:-}}"
require_variable NEWAPI_SOURCE_MAIN_DSN
require_variable NEWAPI_TARGET_MAIN_ADMIN_DSN
require_variable NEWAPI_TARGET_MAIN_OWNER
require_variable NEWAPI_BACKUP_DIR

target_name_date="$(date '+%Y%m%d')"
main_database_name_generated=false
if [[ -z "${NEWAPI_TARGET_MAIN_DATABASE:-}" ]]; then
  validate_prefix NEWAPI_TARGET_MAIN_DATABASE_PREFIX "$NEWAPI_TARGET_MAIN_DATABASE_PREFIX"
  NEWAPI_TARGET_MAIN_DATABASE="${NEWAPI_TARGET_MAIN_DATABASE_PREFIX}_${target_name_date}"
  main_database_name_generated=true
fi
validate_identifier NEWAPI_TARGET_MAIN_DATABASE "$NEWAPI_TARGET_MAIN_DATABASE"
validate_identifier NEWAPI_TARGET_MAIN_OWNER "$NEWAPI_TARGET_MAIN_OWNER"

if [[ -n "${NEWAPI_TARGET_MAIN_DSN_TEMPLATE:-}" ]]; then
  [[ -z "${NEWAPI_TARGET_MAIN_DSN:-}" ]] ||
    die 'NEWAPI_TARGET_MAIN_DSN 与 NEWAPI_TARGET_MAIN_DSN_TEMPLATE 只能填写一个'
  NEWAPI_TARGET_MAIN_DSN="$(build_target_dsn \
    '主库' \
    "$NEWAPI_TARGET_MAIN_DATABASE" \
    "$NEWAPI_TARGET_MAIN_DSN_TEMPLATE")"
else
  require_variable NEWAPI_TARGET_MAIN_DSN
  [[ "$main_database_name_generated" == false ]] ||
    die '自动生成主库名称时必须配置 NEWAPI_TARGET_MAIN_DSN_TEMPLATE'
fi

[[ "$NEWAPI_PGCONNECT_TIMEOUT" =~ ^[1-9][0-9]*$ ]] ||
  die 'NEWAPI_PGCONNECT_TIMEOUT 必须是正整数'
case "$NEWAPI_VERIFY_ROW_COUNTS" in
  true | false) ;;
  *) die 'NEWAPI_VERIFY_ROW_COUNTS 只能是 true 或 false' ;;
esac

[[ "$NEWAPI_BACKUP_DIR" == /* ]] ||
  die 'NEWAPI_BACKUP_DIR 必须是绝对路径'
[[ "$NEWAPI_BACKUP_DIR" != '/' ]] ||
  die 'NEWAPI_BACKUP_DIR 不能是根目录 /'

log_database_enabled=false
if [[ -n "${NEWAPI_SOURCE_LOG_DSN:-}" ||
  -n "${NEWAPI_SOURCE_LOG_HOST:-}" ||
  -n "${NEWAPI_TARGET_LOG_DSN:-}" ||
  -n "${NEWAPI_TARGET_LOG_DSN_TEMPLATE:-}" ||
  -n "${NEWAPI_TARGET_LOG_DATABASE:-}" ]]; then
  log_database_enabled=true

  if [[ -z "${NEWAPI_SOURCE_LOG_DSN:-}" ]]; then
    NEWAPI_SOURCE_LOG_PORT="${NEWAPI_SOURCE_LOG_PORT:-${NEWAPI_SOURCE_MAIN_PORT:-5432}}"
    NEWAPI_SOURCE_LOG_SSLMODE="${NEWAPI_SOURCE_LOG_SSLMODE:-${NEWAPI_SOURCE_MAIN_SSLMODE:-prefer}}"
    require_variable NEWAPI_SOURCE_LOG_HOST
    require_variable NEWAPI_SOURCE_LOG_DATABASE
    require_variable NEWAPI_SOURCE_LOG_USER
    require_variable NEWAPI_SOURCE_LOG_PASSWORD
    validate_port NEWAPI_SOURCE_LOG_PORT "$NEWAPI_SOURCE_LOG_PORT"
    validate_sslmode NEWAPI_SOURCE_LOG_SSLMODE "$NEWAPI_SOURCE_LOG_SSLMODE"
    NEWAPI_SOURCE_LOG_DSN="$(build_conninfo \
      "$NEWAPI_SOURCE_LOG_HOST" \
      "$NEWAPI_SOURCE_LOG_PORT" \
      "$NEWAPI_SOURCE_LOG_DATABASE" \
      "$NEWAPI_SOURCE_LOG_USER" \
      "$NEWAPI_SOURCE_LOG_PASSWORD" \
      "$NEWAPI_SOURCE_LOG_SSLMODE")"
  fi

  require_variable NEWAPI_SOURCE_LOG_DSN
  NEWAPI_TARGET_LOG_ADMIN_DSN="${NEWAPI_TARGET_LOG_ADMIN_DSN:-$NEWAPI_TARGET_MAIN_ADMIN_DSN}"
  NEWAPI_TARGET_LOG_HOST="${NEWAPI_TARGET_LOG_HOST:-${NEWAPI_TARGET_MAIN_HOST:-}}"
  NEWAPI_TARGET_LOG_PORT="${NEWAPI_TARGET_LOG_PORT:-${NEWAPI_TARGET_MAIN_PORT:-5432}}"
  NEWAPI_TARGET_LOG_SSLMODE="${NEWAPI_TARGET_LOG_SSLMODE:-${NEWAPI_TARGET_MAIN_SSLMODE:-prefer}}"
  NEWAPI_TARGET_LOG_APP_USER="${NEWAPI_TARGET_LOG_APP_USER:-${NEWAPI_TARGET_MAIN_APP_USER:-}}"
  NEWAPI_TARGET_LOG_APP_PASSWORD="${NEWAPI_TARGET_LOG_APP_PASSWORD:-${NEWAPI_TARGET_MAIN_APP_PASSWORD:-}}"
  NEWAPI_TARGET_LOG_OWNER="${NEWAPI_TARGET_LOG_OWNER:-${NEWAPI_TARGET_LOG_APP_USER:-$NEWAPI_TARGET_MAIN_OWNER}}"
  NEWAPI_TARGET_LOG_DATABASE_PREFIX="${NEWAPI_TARGET_LOG_DATABASE_PREFIX:-new_api_log_prod}"
  log_database_name_generated=false
  if [[ -z "${NEWAPI_TARGET_LOG_DATABASE:-}" ]]; then
    validate_prefix NEWAPI_TARGET_LOG_DATABASE_PREFIX "$NEWAPI_TARGET_LOG_DATABASE_PREFIX"
    NEWAPI_TARGET_LOG_DATABASE="${NEWAPI_TARGET_LOG_DATABASE_PREFIX}_${target_name_date}"
    log_database_name_generated=true
  fi
  validate_identifier NEWAPI_TARGET_LOG_DATABASE "$NEWAPI_TARGET_LOG_DATABASE"
  validate_identifier NEWAPI_TARGET_LOG_OWNER "$NEWAPI_TARGET_LOG_OWNER"

  if [[ -z "${NEWAPI_TARGET_LOG_DSN_TEMPLATE:-}" &&
    -z "${NEWAPI_TARGET_LOG_DSN:-}" ]]; then
    require_variable NEWAPI_TARGET_LOG_HOST
    require_variable NEWAPI_TARGET_LOG_APP_USER
    require_variable NEWAPI_TARGET_LOG_APP_PASSWORD
    validate_port NEWAPI_TARGET_LOG_PORT "$NEWAPI_TARGET_LOG_PORT"
    validate_sslmode NEWAPI_TARGET_LOG_SSLMODE "$NEWAPI_TARGET_LOG_SSLMODE"
    NEWAPI_TARGET_LOG_DSN_TEMPLATE="$(build_conninfo \
      "$NEWAPI_TARGET_LOG_HOST" \
      "$NEWAPI_TARGET_LOG_PORT" \
      '__DATABASE__' \
      "$NEWAPI_TARGET_LOG_APP_USER" \
      "$NEWAPI_TARGET_LOG_APP_PASSWORD" \
      "$NEWAPI_TARGET_LOG_SSLMODE")"
  fi

  if [[ -n "${NEWAPI_TARGET_LOG_DSN_TEMPLATE:-}" ]]; then
    [[ -z "${NEWAPI_TARGET_LOG_DSN:-}" ]] ||
      die 'NEWAPI_TARGET_LOG_DSN 与 NEWAPI_TARGET_LOG_DSN_TEMPLATE 只能填写一个'
    NEWAPI_TARGET_LOG_DSN="$(build_target_dsn \
      '日志库' \
      "$NEWAPI_TARGET_LOG_DATABASE" \
      "$NEWAPI_TARGET_LOG_DSN_TEMPLATE")"
  else
    require_variable NEWAPI_TARGET_LOG_DSN
    [[ "$log_database_name_generated" == false ]] ||
      die '自动生成日志库名称时必须配置 NEWAPI_TARGET_LOG_DSN_TEMPLATE'
  fi

  [[ "$NEWAPI_TARGET_LOG_DATABASE" != "$NEWAPI_TARGET_MAIN_DATABASE" ]] ||
    die '主库和独立日志库不能使用相同的目标数据库名'
  [[ "$NEWAPI_SOURCE_LOG_DSN" != "$NEWAPI_SOURCE_MAIN_DSN" ]] ||
    die '独立日志库源 DSN 不能与主库源 DSN 完全相同'
fi

for command_name in psql pg_dump pg_restore createdb tr mv chmod date mkdir; do
  require_command "$command_name"
done

if [[ "$main_database_name_generated" == true ]]; then
  info "自动生成主库名称：$NEWAPI_TARGET_MAIN_DATABASE"
fi
if [[ "$log_database_enabled" == true &&
  "$log_database_name_generated" == true ]]; then
  info "自动生成日志库名称：$NEWAPI_TARGET_LOG_DATABASE"
fi

check_database_pair \
  '主库' \
  "$NEWAPI_SOURCE_MAIN_DSN" \
  "$NEWAPI_TARGET_MAIN_ADMIN_DSN" \
  "$NEWAPI_TARGET_MAIN_DATABASE" \
  "$NEWAPI_TARGET_MAIN_OWNER"

if [[ "$log_database_enabled" == true ]]; then
  check_database_pair \
    '日志库' \
    "$NEWAPI_SOURCE_LOG_DSN" \
    "$NEWAPI_TARGET_LOG_ADMIN_DSN" \
    "$NEWAPI_TARGET_LOG_DATABASE" \
    "$NEWAPI_TARGET_LOG_OWNER"
fi

if [[ "$check_only" == true ]]; then
  info '迁移前检查通过；未导出备份、未创建目标数据库'
  exit 0
fi

if [[ ! -t 0 || ! -t 1 ]]; then
  die '正式迁移必须在交互式终端运行'
fi

confirmation_text="MIGRATE ${NEWAPI_TARGET_MAIN_DATABASE}"
printf '\n请确认：\n'
printf '  1. 已停止旧 new-api 的请求和后台写入。\n'
printf '  2. 没有新旧两个 new-api 同时写入源数据库。\n'
printf '  3. 目标数据库名称和备份目录已经核对无误。\n\n'
printf '请输入 %s 继续：' "$confirmation_text"
IFS= read -r confirmation
[[ "$confirmation" == "$confirmation_text" ]] || die '确认文字不匹配，迁移已取消'

mkdir -p "$NEWAPI_BACKUP_DIR"
[[ -d "$NEWAPI_BACKUP_DIR" && -w "$NEWAPI_BACKUP_DIR" ]] ||
  die "备份目录不可写：$NEWAPI_BACKUP_DIR"

timestamp="$(date -u '+%Y%m%dT%H%M%SZ')"
main_dump_file="${NEWAPI_BACKUP_DIR}/new-api-main-${timestamp}.dump"
log_dump_file="${NEWAPI_BACKUP_DIR}/new-api-log-${timestamp}.dump"

create_dump '主库' "$NEWAPI_SOURCE_MAIN_DSN" "$main_dump_file"
if [[ "$log_database_enabled" == true ]]; then
  create_dump '日志库' "$NEWAPI_SOURCE_LOG_DSN" "$log_dump_file"
fi

restore_dump \
  '主库' \
  "$NEWAPI_TARGET_MAIN_ADMIN_DSN" \
  "$NEWAPI_TARGET_MAIN_DSN" \
  "$NEWAPI_TARGET_MAIN_DATABASE" \
  "$NEWAPI_TARGET_MAIN_OWNER" \
  "$main_dump_file"

if [[ "$log_database_enabled" == true ]]; then
  restore_dump \
    '日志库' \
    "$NEWAPI_TARGET_LOG_ADMIN_DSN" \
    "$NEWAPI_TARGET_LOG_DSN" \
    "$NEWAPI_TARGET_LOG_DATABASE" \
    "$NEWAPI_TARGET_LOG_OWNER" \
    "$log_dump_file"
fi

if [[ "$NEWAPI_VERIFY_ROW_COUNTS" == true ]]; then
  verify_table_counts \
    '主库' \
    "$NEWAPI_SOURCE_MAIN_DSN" \
    "$NEWAPI_TARGET_MAIN_DSN" \
    users \
    tokens \
    channels \
    options \
    redemptions \
    top_ups \
    quota_data \
    subscription_plans \
    user_subscriptions \
    clusters \
    cluster_telemetry_latest \
    cluster_telemetry_history \
    logs

  if [[ "$log_database_enabled" == true ]]; then
    verify_table_counts \
      '日志库' \
      "$NEWAPI_SOURCE_LOG_DSN" \
      "$NEWAPI_TARGET_LOG_DSN" \
      logs
  fi
else
  warn '已按配置跳过关键表精确行数核对'
fi

printf '\n迁移完成。\n'
printf '主库备份：%s\n' "$main_dump_file"
printf '目标主库：%s\n' "$NEWAPI_TARGET_MAIN_DATABASE"
if [[ "$log_database_enabled" == true ]]; then
  printf '日志库备份：%s\n' "$log_dump_file"
  printf '目标日志库：%s\n' "$NEWAPI_TARGET_LOG_DATABASE"
fi
printf '\n下一步：将新版 new-api 的 SQL_DSN'
if [[ "$log_database_enabled" == true ]]; then
  printf ' 和 LOG_SQL_DSN'
fi
printf ' 指向目标库，复用原部署密钥，并且只启动一个主节点完成应用迁移。\n'
