#!/usr/bin/env bash

# Copyright (C) 2023-2026 QuantumNous
# SPDX-License-Identifier: AGPL-3.0-or-later

set -Eeuo pipefail

usage() {
  cat <<'EOF'
用法：
  ./scripts/update-build-run.sh [额外的 new-api 启动参数]

执行顺序：
  1. 从远端 fast-forward 拉取当前分支
  2. 安装/同步前端依赖并构建 web/dist
  3. 编译新的 bin/new-api-local
  4. 在前台启动新编译的 new-api

可选环境变量：
  NEW_API_GIT_REMOTE       Git 远端，默认 origin
  NEW_API_GIT_BRANCH       Git 分支，默认当前分支
  NEW_API_PORT             服务端口，默认 3000
  NEW_API_LOG_DIR          日志目录，默认 /api_data/new-api/logs
  NEW_API_BINARY_PATH      二进制路径，默认 bin/new-api-local
  NEW_API_GOEXPERIMENT     Go 实验特性，默认 greenteagc；设为空可禁用

示例：
  ./scripts/update-build-run.sh

  NEW_API_PORT=3001 \
  NEW_API_LOG_DIR=/data/logs \
  ./scripts/update-build-run.sh

说明：
  - SQL_DSN、REDIS_CONN_STRING 等现有环境变量会原样传给 new-api。
  - 脚本不会自动终止已经运行的 new-api，避免误杀容器 PID 1。
  - 当前分支存在已跟踪但未提交的修改时，脚本会停止，避免覆盖本地改动。
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(git -C "$script_dir" rev-parse --show-toplevel 2>/dev/null)" || {
  echo "[ERROR] 无法定位 Git 仓库根目录。" >&2
  exit 1
}
cd "$repo_root"

git_remote="${NEW_API_GIT_REMOTE:-origin}"
current_branch="$(git branch --show-current)"
git_branch="${NEW_API_GIT_BRANCH:-$current_branch}"
port="${NEW_API_PORT:-3000}"
log_dir="${NEW_API_LOG_DIR:-/api_data/new-api/logs}"
binary_path="${NEW_API_BINARY_PATH:-bin/new-api-local}"
go_experiment="${NEW_API_GOEXPERIMENT-greenteagc}"

if [[ -z "$current_branch" ]]; then
  echo "[ERROR] 当前仓库处于 detached HEAD，无法确定要拉取的分支。" >&2
  exit 1
fi

if [[ "$git_branch" != "$current_branch" ]]; then
  echo "[ERROR] 当前分支是 $current_branch，但配置的拉取分支是 $git_branch。" >&2
  echo "请先切换到目标分支，脚本不会自动切换或合并分支。" >&2
  exit 1
fi

if [[ ! "$port" =~ ^[0-9]+$ ]] || ((port < 1 || port > 65535)); then
  echo "[ERROR] NEW_API_PORT 必须是 1 到 65535 之间的整数。" >&2
  exit 1
fi

if ! git diff --quiet || ! git diff --cached --quiet; then
  echo "[ERROR] 当前分支存在已跟踪但未提交的修改，已停止拉取。" >&2
  git status --short
  exit 1
fi

if command -v ss >/dev/null 2>&1 &&
  ss -lntH | awk -v suffix=":$port" '$4 ~ suffix "$" { found = 1 } END { exit !found }'; then
  echo "[ERROR] 端口 $port 已被占用，请先停止现有服务再运行本脚本。" >&2
  echo "脚本不会自动终止进程，以免误杀容器 PID 1。" >&2
  exit 1
fi

if [[ -d /opt/go1.26.5/bin ]]; then
  PATH="/opt/go1.26.5/bin:$PATH"
fi
if [[ -d /root/.bun/bin ]]; then
  export BUN_INSTALL="${BUN_INSTALL:-/root/.bun}"
  PATH="$BUN_INSTALL/bin:$PATH"
fi
export PATH

for command_name in git bun go; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "[ERROR] 未找到 $command_name，请先按照 docs/installation/BUILD_IN_CONTAINER.md 安装构建依赖。" >&2
    exit 1
  fi
done

if [[ "$binary_path" != /* ]]; then
  binary_path="$repo_root/$binary_path"
fi

version="$(tr -d '[:space:]' < "$repo_root/VERSION")"
if [[ -z "$version" ]]; then
  echo "[ERROR] VERSION 文件为空，无法生成版本信息。" >&2
  exit 1
fi

echo "[1/4] 拉取 $git_remote/$git_branch"
git pull --ff-only "$git_remote" "$git_branch"

echo "[2/4] 安装前端依赖并构建"
cd "$repo_root/web"
bun install --frozen-lockfile
DISABLE_ESLINT_PLUGIN=true \
  VITE_REACT_APP_VERSION="$version" \
  bun run build

echo "[3/4] 编译 Go 后端"
cd "$repo_root"
mkdir -p "$(dirname "$binary_path")" "$log_dir"

temporary_binary="${binary_path}.new.$$"
cleanup() {
  if [[ -f "$temporary_binary" ]]; then
    rm -f -- "$temporary_binary"
  fi
}
trap cleanup EXIT

build_environment=(env "CGO_ENABLED=0")
if [[ -n "$go_experiment" ]]; then
  build_environment+=("GOEXPERIMENT=$go_experiment")
fi

"${build_environment[@]}" go build \
  -ldflags "-s -w -X github.com/QuantumNous/new-api/common.Version=$version" \
  -o "$temporary_binary" \
  .

mv -f -- "$temporary_binary" "$binary_path"
trap - EXIT

echo "[4/4] 启动 new-api"
echo "二进制：$binary_path"
echo "端口：$port"
echo "日志目录：$log_dir"

exec "$binary_path" \
  --port "$port" \
  --log-dir "$log_dir" \
  "$@"
