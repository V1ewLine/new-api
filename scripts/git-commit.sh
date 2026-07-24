#!/usr/bin/env bash

# Copyright (C) 2023-2026 QuantumNous
# SPDX-License-Identifier: AGPL-3.0-or-later

set -euo pipefail

usage() {
  cat <<'EOF'
用法：
  ./scripts/git-commit.sh
  ./scripts/git-commit.sh "feat: 提交说明"

脚本会：
  1. 用 git add -p 逐段选择已跟踪文件的修改
  2. 逐项询问是否添加新文件
  3. 展示最终暂存范围并确认后提交

脚本不会自动 push，也不会自动添加常见的密钥、依赖和构建产物。
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ ! -t 0 || ! -t 1 ]]; then
  echo "请在交互式终端中运行此脚本。" >&2
  exit 1
fi

repo_root="$(git rev-parse --show-toplevel 2>/dev/null)" || {
  echo "当前目录不在 Git 仓库中。" >&2
  exit 1
}
cd "$repo_root"

commit_message="${1:-}"

echo "当前改动："
git status --short

if git diff --quiet &&
  git diff --cached --quiet &&
  [[ -z "$(git status --porcelain=v1 --untracked-files=normal)" ]]; then
  echo "没有需要提交的改动。"
  exit 0
fi

if ! git diff --quiet; then
  echo
  echo "请选择已跟踪文件中要暂存的修改："
  echo "常用操作：y=添加当前块，n=跳过，s=拆分，q=结束选择，?=帮助"
  git add -p
fi

while IFS= read -r -d '' entry; do
  status="${entry:0:2}"
  [[ "$status" == "??" ]] || continue

  path="${entry:3}"
  case "$path" in
    .env | */.env | .env.local | */.env.local | .env.*.local | */.env.*.local | \
      node_modules/ | */node_modules/ | */node_modules/* | \
      dist/ | */dist/ | */dist/* | build/ | */build/ | */build/* | \
      bin/ | bin/* | logs/ | logs/* | \
      *.log | *.db | *.sqlite | *.sqlite3)
      echo "自动跳过可能包含密钥或生成内容的路径：$path"
      ;;
    *)
      printf "是否暂存新文件/目录 %q？[y/N] " "$path" > /dev/tty
      IFS= read -r answer < /dev/tty
      case "$answer" in
        y | Y | yes | YES)
          git add -- "$path"
          ;;
      esac
      ;;
  esac
done < <(git status --porcelain=v1 -z --untracked-files=normal)

if git diff --cached --quiet; then
  echo "没有暂存任何内容，未创建提交。"
  exit 0
fi

git diff --cached --check

echo
echo "即将提交的内容："
git diff --cached --stat
git diff --cached --name-status

if [[ -z "$commit_message" ]]; then
  printf "请输入提交说明： " > /dev/tty
  IFS= read -r commit_message < /dev/tty
fi

if [[ -z "${commit_message//[[:space:]]/}" ]]; then
  echo "提交说明不能为空，已保留暂存内容。" >&2
  exit 1
fi

printf "确认创建提交？[y/N] " > /dev/tty
IFS= read -r confirmation < /dev/tty
case "$confirmation" in
  y | Y | yes | YES)
    git commit -m "$commit_message"
    echo "提交已创建。确认无误后可运行：git push"
    ;;
  *)
    echo "已取消提交，暂存内容保持不变。"
    ;;
esac
