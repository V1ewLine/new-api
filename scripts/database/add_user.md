# PostgreSQL 创建 `new_api_app` 用户

本文档用于在目标 PostgreSQL 实例中创建供 new-api 使用的登录用户 `new_api_app`，并验证用户是否创建成功。

## 需要确认的参数

| 参数 | 示例 | 说明 |
| --- | --- | --- |
| PostgreSQL 地址 | `127.0.0.1` | PostgreSQL 服务监听地址 |
| PostgreSQL 端口 | `5433` | 目标 PostgreSQL 实例的端口 |
| 管理员用户 | `postgres` | 需要具备创建角色权限 |
| 应用用户 | `new_api_app` | new-api 连接数据库时使用的用户名 |

下面的命令以 PostgreSQL 地址 `127.0.0.1`、管理员用户 `postgres` 为例。请将命令中的 `<目标端口>` 替换为实际端口。

## 创建用户

首先连接目标 PostgreSQL 实例：

```bash
psql \
  -h 127.0.0.1 \
  -p <目标端口> \
  -U postgres \
  -W \
  -d postgres
```

进入 `psql` 后，依次执行：

```sql
CREATE ROLE new_api_app WITH LOGIN;
\password new_api_app
```

执行 `\password new_api_app` 后，根据提示输入并确认 `new_api_app` 的密码。密码不会显示在终端中。

设置完成后退出：

```text
\q
```

## 验证用户

在终端中执行以下命令：

```bash
psql \
  -h 127.0.0.1 \
  -p <目标端口> \
  -U postgres \
  -W \
  -d postgres \
  -c "SELECT rolname, rolcanlogin FROM pg_roles WHERE rolname = 'new_api_app';"
```

正常情况下应返回类似结果：

```text
   rolname   | rolcanlogin
-------------+-------------
 new_api_app | t
```

其中 `rolcanlogin` 为 `t`，表示该用户可以登录 PostgreSQL。

## 用户已存在时

如果执行 `CREATE ROLE` 时提示 `role "new_api_app" already exists`，不需要重复创建。可以直接重新设置密码：

```sql
\password new_api_app
```

## 注意事项

- 请连接迁移脚本实际使用的目标 PostgreSQL 实例，确保地址和端口与迁移配置一致。
- 不要把数据库密码直接写进本文档、Git 仓库或终端历史。
- 创建用户后，还需要为它创建或授权目标数据库；仅创建角色并不会自动获得现有数据库的访问权限。
