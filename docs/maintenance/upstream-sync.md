# Fork 长期同步官方上游指南

本文适用于持续在 fork 中开发，同时定期合并
`QuantumNous/new-api` 官方 `main` 的场景。目标是保留 fork 的集群状态、协议兼容、
数据导出和部署工具等额外开发，并让每次上游同步都可审查、可验证、可回滚。

## 1. 远端约定

固定使用两个远端：

```text
origin    自己的 fork，可拉取、可推送
upstream  QuantumNous/new-api，只用于获取官方更新
```

首次配置：

```bash
git remote add upstream https://github.com/QuantumNous/new-api.git
git fetch upstream --prune
```

检查：

```bash
git remote -v
```

不要把 `main` 的跟踪分支改为 `upstream/main`。本地 `main` 应继续跟踪
`origin/main`，官方更新通过独立同步分支合入。

## 2. 为什么使用 merge 而不是 rebase

fork 已经包含长期部署和数据库相关提交，公开提交也已经推送给其他环境使用。
对这些提交执行 rebase 会重写提交 ID，增加误用强制推送和丢失改动的风险。

本项目采用以下历史结构：

```text
upstream/main ────────┐
                     ├─ merge commit ── fork main
fork 自定义提交 ──────┘
```

merge commit 会明确记录每次同步的官方基线，后续可以用第一父提交回退整个同步，
也能继续利用 Git 的三方合并结果。不要在 fork 的 `main` 上运行：

```bash
git pull upstream main
git rebase upstream/main
git push --force
```

## 3. 每次同步前检查

运行仓库提供的检查命令：

```bash
bash scripts/git/upstream-sync.sh check
```

该命令会：

1. 校验或创建 `upstream` 远端；
2. 获取最新 `upstream/main`；
3. 显示双方独有提交数量和共同基线；
4. 进行虚拟合并并列出文本冲突；
5. 保持当前分支和当前工作区不变。

返回码含义：

```text
0  获取成功，虚拟合并没有文本冲突
2  获取成功，虚拟合并发现冲突，需要人工处理
其他  配置、网络或 Git 状态异常
```

“没有文本冲突”不代表功能一定正确。上游可能修改接口、DTO、数据库迁移、构建
版本或前端组件，需要继续执行完整回归测试。

## 4. 创建隔离同步环境

```bash
bash scripts/git/upstream-sync.sh prepare
```

脚本会从 fork 的本地 `main` 创建：

```text
分支：sync/upstream-YYYYMMDD-HHMMSS
目录：系统临时目录中的 new-api-upstream-YYYYMMDD-HHMMSS
```

然后只在该 worktree 中执行：

```bash
git merge --no-ff --no-commit upstream/main
```

原项目目录里的未提交文件不会被 stash、移动或覆盖。脚本也不会自动解决冲突、
创建提交、更新 `main` 或推送远端。

## 5. 冲突判断原则

发生冲突后，把脚本输出的同步目录和冲突列表交给 Codex。处理时按语义判断，
不能简单地整文件选择 `ours` 或 `theirs`。

优先级如下：

1. 保留官方的安全修复、数据库兼容、计费边界和协议修复；
2. 保留 fork 新增的功能入口、数据模型、配置项、路由和开发文档；
3. fork 修改的旧代码已经被官方重构时，把 fork 行为迁移到官方新结构；
4. 同名配置字段同时变化时，保留双方字段并补充序列化和回归测试；
5. i18n 文件保留双方 key，通过项目同步脚本校验，不手工删除未知 key；
6. 不通过删除测试、关闭校验或回退官方安全限制来消除失败。

重点复核 fork 当前的长期功能：

- 集群状态、Agent 凭据、轮询、历史趋势与导出；
- 集群遥测保存天数、刷新间隔等系统设置；
- 模型调用分析导出；
- Responses → Chat 兼容、能力探测和 SGLang 文本格式兼容；
- Claude 工具调用流式 content block 索引修复；
- PostgreSQL、Redis、数据库迁移和容器编译脚本。

## 6. 验证清单

在隔离 worktree 中确认没有冲突标记：

```bash
git diff --name-only --diff-filter=U
git grep -nE '^(<<<<<<<|=======|>>>>>>>)'
git diff --check
```

后端与独立 RelayKit：

```bash
go test ./...
go build ./...

cd relaykit
go test ./...
cd ..
```

前端：

```bash
cd web
bun install --frozen-lockfile
bun run format:check
bun run lint
bun run typecheck
bun run i18n:sync
bun run build
cd ..
```

如果 `i18n:sync` 产生文件变化，需要审查并纳入同步提交，然后重新执行检查。

最后执行 fork 功能的定向测试，例如：

```bash
go test ./service/clusterstatus ./service/dashboardexport ./relay ./router
```

数据库相关变更必须确认 SQLite、MySQL 和 PostgreSQL 均有兼容路径。涉及已有生产
数据时，先备份 PostgreSQL 数据库和集群密钥文件，再在测试实例启动迁移后的程序。

## 7. 提交和更新 main

确认冲突解决与验证结果后，在同步 worktree 中创建 merge commit：

```bash
git status
git add <本次冲突解决和必要联动文件>
git commit
```

Git 会使用合并信息生成提交。提交信息建议：

```text
merge: sync upstream/main YYYY-MM-DD
```

回到原项目目录前，先处理或提交原工作区的本地修改。为避免覆盖，只有工作区完全
干净时才更新 `main`：

```bash
git switch main
git status --short
git merge --ff-only sync/upstream-YYYYMMDD-HHMMSS
git push origin main
```

禁止使用 `git reset --hard`、`git checkout -- .` 或强制推送来“清理”工作区。

## 8. 回滚与清理

合并提交推送后如果出现问题，优先创建反向提交，不改写已发布历史：

```bash
git revert -m 1 <merge-commit-id>
git push origin main
```

确认同步分支不再需要后，可以清理临时 worktree：

```bash
git worktree list
git worktree remove <同步 worktree 路径>
git branch -d sync/upstream-YYYYMMDD-HHMMSS
```

只有在确认目录中没有未提交内容时才执行清理。Git 拒绝删除含未提交内容的
worktree 时，应先检查，不要添加 `--force`。

## 9. 日常建议

- 每周或每两周同步一次，避免一次积累数百个文件；
- 每个 fork 功能保持独立提交，并为核心行为保留回归测试；
- 不要把二进制、压测输出、数据库文件或密钥提交到 Git；
- 同步前记录当前生产版本和数据库备份位置；
- 上游出现大规模重构时，先完成一次专门的兼容同步，不与新业务开发混在同一提交。
