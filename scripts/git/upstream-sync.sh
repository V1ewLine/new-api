#!/usr/bin/env bash

# Copyright (C) 2023-2026 QuantumNous
# SPDX-License-Identifier: AGPL-3.0-or-later

set -Eeuo pipefail

usage() {
  cat <<'EOF'
用法：
  ./scripts/git/upstream-sync.sh check
  ./scripts/git/upstream-sync.sh prepare

命令：
  check    获取官方上游并执行只读分叉、虚拟合并检查，不修改当前工作区。
  prepare  创建独立同步分支和临时 worktree，并在其中合并官方上游。

可选环境变量：
  NEW_API_UPSTREAM_REMOTE     官方远端名，默认 upstream
  NEW_API_UPSTREAM_URL        官方仓库 URL，默认 https://github.com/QuantumNous/new-api.git
  NEW_API_UPSTREAM_BRANCH     官方分支，默认 main
  NEW_API_TARGET_BRANCH       fork 的目标分支，默认 main
  NEW_API_SYNC_BRANCH         prepare 创建的分支名，默认带当前时间
  NEW_API_SYNC_WORKTREE       prepare 创建的 worktree 路径，默认位于系统临时目录

安全约束：
  - 不会 stash、reset、rebase、删除或覆盖当前未提交文件。
  - 不会自动解决冲突、提交、更新 main 或推送远端。
  - prepare 后请在输出的 worktree 中审查冲突并完成验证。
EOF
}

info() {
  echo "[INFO] $*"
}

error() {
  echo "[ERROR] $*" >&2
  exit 1
}

command_name="${1:-check}"
case "$command_name" in
  check | prepare) ;;
  -h | --help)
    usage
    exit 0
    ;;
  *)
    usage >&2
    error "未知命令：$command_name"
    ;;
esac

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(git -C "$script_dir" rev-parse --show-toplevel 2>/dev/null)" ||
  error "无法定位 Git 仓库根目录。"

upstream_remote="${NEW_API_UPSTREAM_REMOTE:-upstream}"
upstream_url="${NEW_API_UPSTREAM_URL:-https://github.com/QuantumNous/new-api.git}"
upstream_branch="${NEW_API_UPSTREAM_BRANCH:-main}"
target_branch="${NEW_API_TARGET_BRANCH:-main}"
timestamp="$(date '+%Y%m%d-%H%M%S')"
sync_branch="${NEW_API_SYNC_BRANCH:-sync/upstream-$timestamp}"
sync_worktree="${NEW_API_SYNC_WORKTREE:-${TMPDIR:-/tmp}/new-api-upstream-$timestamp}"

for required_command in git awk sed; do
  command -v "$required_command" >/dev/null 2>&1 ||
    error "未找到命令：$required_command"
done

cd "$repo_root"

if ! git show-ref --verify --quiet "refs/heads/$target_branch"; then
  error "本地目标分支不存在：$target_branch"
fi

if git remote get-url "$upstream_remote" >/dev/null 2>&1; then
  configured_url="$(git remote get-url "$upstream_remote")"
  if [[ "$configured_url" != "$upstream_url" ]]; then
    error "远端 $upstream_remote 当前指向 $configured_url，而不是 $upstream_url；请人工确认，脚本不会覆盖。"
  fi
else
  info "添加官方远端：$upstream_remote -> $upstream_url"
  git remote add "$upstream_remote" "$upstream_url"
fi

info "获取 $upstream_remote/$upstream_branch"
git fetch "$upstream_remote" "$upstream_branch" --prune

upstream_ref="refs/remotes/$upstream_remote/$upstream_branch"
if ! git show-ref --verify --quiet "$upstream_ref"; then
  error "获取后仍找不到上游引用：$upstream_remote/$upstream_branch"
fi

read -r local_only upstream_only < <(
  git rev-list --left-right --count "$target_branch...$upstream_remote/$upstream_branch"
)
merge_base="$(git merge-base "$target_branch" "$upstream_remote/$upstream_branch")"

info "$target_branch 独有提交：$local_only"
info "$upstream_remote/$upstream_branch 独有提交：$upstream_only"
info "共同基线：$merge_base"

if [[ "$command_name" == "check" ]]; then
  merge_output="$(mktemp "${TMPDIR:-/tmp}/new-api-merge-tree.XXXXXX")"
  cleanup() {
    rm -f -- "$merge_output"
  }
  trap cleanup EXIT

  if git merge-tree --write-tree --messages \
    "$target_branch" "$upstream_remote/$upstream_branch" >"$merge_output"; then
    info "虚拟合并未发现文本冲突。仍需在隔离分支执行完整测试。"
    exit 0
  fi

  info "虚拟合并发现以下冲突："
  sed -n 's/^CONFLICT ([^)]*): Merge conflict in /  - /p' "$merge_output"
  exit 2
fi

if git show-ref --verify --quiet "refs/heads/$sync_branch"; then
  error "同步分支已存在：$sync_branch"
fi
if [[ -e "$sync_worktree" ]]; then
  error "同步 worktree 路径已存在：$sync_worktree"
fi

info "创建隔离同步分支：$sync_branch"
git worktree add -b "$sync_branch" "$sync_worktree" "$target_branch"

info "在隔离 worktree 中合并 $upstream_remote/$upstream_branch"
set +e
git -C "$sync_worktree" merge --no-ff --no-commit \
  "$upstream_remote/$upstream_branch"
merge_status=$?
set -e

echo
info "同步分支：$sync_branch"
info "隔离目录：$sync_worktree"

if ((merge_status == 0)); then
  info "上游已合入隔离目录且没有文本冲突；尚未提交。请先审查和测试。"
else
  info "上游合并停在冲突状态。冲突文件："
  git -C "$sync_worktree" diff --name-only --diff-filter=U |
    sed 's/^/  - /'
  echo
  info "请把上述目录和冲突列表交给 Codex 判断、解决并验证。"
fi

cat <<EOF

后续流程：
  1. 在 $sync_worktree 解决冲突并审查双方改动。
  2. 运行后端、relaykit、前端与 fork 自定义功能测试。
  3. 在同步分支创建 merge commit。
  4. 确认原工作区干净后，使用 --ff-only 更新 $target_branch，再推送 origin。

详细说明：docs/maintenance/upstream-sync.md
EOF

exit "$merge_status"
