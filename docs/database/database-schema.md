# new-api 数据库表结构与数据归属

> 适用代码基线：`7d695754`
>
> 适用数据库：PostgreSQL、MySQL、SQLite；本文重点说明 PostgreSQL 部署。
>
> 目的：帮助开发、运维和交接人员理解数据存放位置、表之间的关系以及迁移边界。

## 1. 先明确数据存在哪里

new-api 的持久化数据分为三部分：

| 数据层 | 配置 | 主要内容 | 是否为业务数据源 |
| --- | --- | --- | --- |
| 主数据库 | `SQL_DSN` | 用户、令牌、渠道、模型、系统配置、充值订阅、任务、集群配置等 | 是 |
| 日志数据库 | `LOG_SQL_DSN` | 请求日志 `logs` | 是；未配置时与主数据库共用 |
| Redis | `REDIS_CONN_STRING` | 用户/令牌/会话缓存、限流计数、渠道亲和性等 | 否，主要是缓存和临时状态 |

PostgreSQL 是业务数据的权威来源。Redis 丢失通常不会删除用户、渠道、令牌、余额或系统设置，但会清空缓存、限流窗口和渠道亲和性等临时状态。

如果未配置 `LOG_SQL_DSN`，`logs` 与其他表位于 `SQL_DSN` 指向的数据库中。如果配置了独立日志库，必须把主库和日志库分别备份。

## 2. 表结构的来源与使用方式

项目启动时，主节点会在 `model/main.go` 中通过 GORM `AutoMigrate` 自动补充表和字段。以下结构来自当前代码中的 GORM 模型，是跨数据库的“逻辑结构”，不等同于某套线上 PostgreSQL 的逐字 DDL。

以下情况可能让线上物理结构与本文略有差异：

- 线上实例从旧版本连续升级，保留了历史字段、索引或默认值。
- PostgreSQL、MySQL、SQLite 的实际字段类型和默认值由 GORM 按方言生成。
- 项目升级后新增了迁移，而本文尚未同步。
- 运维人员在数据库中增加了自定义索引。

因此：

1. 本文用于理解和交接。
2. 迁移前应以目标版本代码中的模型和实际数据库导出的 schema 为准。
3. 不要仅依据本文手工新建业务表；应恢复数据后让 new-api 主节点执行迁移。

## 3. 主要逻辑关系

项目的大多数关系由应用代码维护，并不保证数据库中存在外键或级联删除约束。

```mermaid
erDiagram
    users ||--o{ tokens : owns
    users ||--o{ user_sessions : logs_in
    users ||--o{ logs : generates
    users ||--o{ quota_data : aggregates
    users ||--o{ top_ups : pays
    users ||--o{ user_subscriptions : subscribes
    channels ||--o{ abilities : routes
    channels ||--o{ logs : serves
    channels ||--o{ tasks : executes
    vendors ||--o{ models : provides
    models ||--o{ clusters : deploys
    clusters ||--|| cluster_telemetry_latest : reports
    clusters ||--o{ cluster_telemetry_history : samples
    subscription_plans ||--o{ subscription_orders : purchased_as
    subscription_plans ||--o{ user_subscriptions : grants
    user_subscriptions ||--o{ subscription_pre_consume_records : reserves
    custom_oauth_providers ||--o{ user_oauth_bindings : binds
```

关键逻辑引用：

- `users.id` 被令牌、会话、日志、配额、充值、订阅、OAuth、Passkey、两步验证等表引用。
- `channels.id` 被路由能力、日志、仪表板汇总、异步任务等表引用。
- `vendors.id` 被 `models.vendor_id` 引用。
- `models.id` 被 `clusters.model_id` 引用。
- `clusters.id` 被 `cluster_telemetry_latest.cluster_id` 和 `cluster_telemetry_history.cluster_id` 引用。
- `subscription_plans.id` 被订阅订单和用户订阅引用。
- `user_subscriptions.id` 被预消费记录引用。

删除或修复数据前，必须检查这些逻辑引用，不要假设 PostgreSQL 会通过外键自动处理。

## 4. 表目录

当前主数据库迁移共涉及 36 张表。

| 分类 | 表 |
| --- | --- |
| 核心访问与配置 | `channels`、`tokens`、`users`、`options`、`abilities`、`models`、`vendors`、`prefill_groups`、`setups`、`casbin_rule`、`authz_roles` |
| 认证与安全 | `user_sessions`、`auth_flows`、`external_identity_claims`、`passkey_credentials`、`two_fas`、`two_fa_backup_codes`、`custom_oauth_providers`、`user_oauth_bindings` |
| 计费与用量 | `redemptions`、`top_ups`、`checkins`、`logs`、`quota_data`、`perf_metrics` |
| 异步任务 | `midjourneys`、`tasks` |
| 订阅 | `subscription_plans`、`subscription_orders`、`user_subscriptions`、`subscription_pre_consume_records` |
| 系统任务与节点 | `system_instances`、`system_tasks`、`system_task_locks` |
| 集群状态 | `clusters`、`cluster_telemetry_latest`、`cluster_telemetry_history` |

## 5. 核心访问与配置表

### 5.1 `channels`

上游渠道及其路由配置。

- 主键：`id`
- 字段：`type`、`key`、`open_ai_organization`、`test_model`、`status`、`name`、`weight`、`created_time`、`test_time`、`response_time`、`base_url`、`other`、`balance`、`balance_updated_time`、`models`、`group`、`used_quota`、`model_mapping`、`status_code_mapping`、`priority`、`auto_ban`、`other_info`、`tag`、`setting`、`param_override`、`header_override`、`remark`、`channel_info`、`settings`
- 重要索引：`name`、`tag`
- 敏感字段：`key` 及覆盖配置中可能存在的认证信息

### 5.2 `tokens`

用户创建的 API 令牌。

- 主键：`id`
- 字段：`user_id`、`key`、`status`、`name`、`created_time`、`accessed_time`、`expired_time`、`remain_quota`、`unlimited_quota`、`model_limits_enabled`、`model_limits`、`allow_ips`、`used_quota`、`group`、`cross_group_retry`、`deleted_at`
- 重要约束/索引：`key` 唯一；`user_id`、`name`、`deleted_at` 有索引
- 说明：`deleted_at` 用于软删除；恢复时不能只恢复 `users` 而遗漏本表

### 5.3 `users`

用户账号、额度、角色和第三方身份摘要。

- 主键：`id`
- 字段：`username`、`password`、`display_name`、`role`、`status`、`email`、`github_id`、`discord_id`、`oidc_id`、`wechat_id`、`telegram_id`、`access_token`、`quota`、`used_quota`、`request_count`、`group`、`aff_code`、`aff_count`、`aff_quota`、`aff_history`、`inviter_id`、`deleted_at`、`linux_do_id`、`setting`、`remark`、`stripe_customer`、`created_at`、`last_login_at`、`auth_version`
- 重要约束/索引：`username`、`access_token`、`aff_code` 唯一；多个第三方身份、邮箱、邀请人和软删除字段有索引
- 敏感字段：`password`、`access_token`、第三方身份标识

### 5.4 `options`

系统设置键值表。

- 主键：`key`
- 字段：`value`
- 说明：后台“系统设置”中的大量配置存放在此表；迁移时不可遗漏

### 5.5 `abilities`

用户组、模型与渠道之间的可用路由关系。

- 复合主键：`group`、`model`、`channel_id`
- 字段：`enabled`、`priority`、`weight`、`tag`
- 逻辑引用：`channel_id -> channels.id`

### 5.6 `models`

模型广场和模型元数据。

- 主键：`id`
- 字段：`model_name`、`description`、`icon`、`tags`、`vendor_id`、`endpoints`、`status`、`sync_official`、`created_time`、`updated_time`、`deleted_at`、`name_rule`
- 重要约束：`model_name + deleted_at` 组合唯一
- 逻辑引用：`vendor_id -> vendors.id`

### 5.7 `vendors`

模型厂商元数据。

- 主键：`id`
- 字段：`name`、`description`、`icon`、`status`、`created_time`、`updated_time`、`deleted_at`
- 重要约束：`name + deleted_at` 组合唯一

### 5.8 `prefill_groups`

预填模型组或预设集合。

- 主键：`id`
- 字段：`name`、`type`、`items`、`description`、`created_time`、`updated_time`、`deleted_at`
- 重要约束：`name` 唯一

### 5.9 `setups`

实例首次初始化记录。

- 主键：`id`
- 字段：`version`、`initialized_at`
- 说明：用于判断实例是否已经完成初始化

### 5.10 `casbin_rule`

Casbin 权限策略。

- 主键：`id`
- 字段：`ptype`、`v0`、`v1`、`v2`、`v3`、`v4`、`v5`
- 重要约束：上述策略字段组合唯一，并建立查询复合索引

### 5.11 `authz_roles`

可授权角色定义。

- 主键：`id`
- 字段：`key`、`name`、`description`、`built_in`、`enabled`、`sort`、`created_at`、`updated_at`
- 重要约束：`key` 唯一

## 6. 认证与安全表

### 6.1 `user_sessions`

登录会话和刷新令牌轮换状态。

- 主键：`sid`
- 字段：`user_id`、`version`、`user_auth_version`、`status`、`refresh_hash`、`previous_refresh_hash`、`previous_valid_until`、`login_method`、`ip`、`user_agent`、`created_at`、`last_active_at`、`expires_at`、`revoked_at`、`revoked_reason`
- 逻辑引用：`user_id -> users.id`
- 敏感字段：刷新令牌哈希、IP、User-Agent
- 说明：更换 `SESSION_SECRET` 可能导致既有登录态失效

### 6.2 `auth_flows`

OAuth、注册、绑定等短期认证流程。

- 主键：`id`
- 字段：`token_hash`、`purpose`、`provider`、`intent`、`user_id`、`session_id`、`payload`、`created_at`、`expires_at`、`consumed_at`
- 重要约束：`token_hash` 唯一

### 6.3 `external_identity_claims`

外部身份与本地用户之间的唯一归属关系。

- 主键：`id`
- 字段：`provider`、`subject`、`user_id`、`created_at`
- 重要约束：`provider + subject` 唯一；`provider + user_id` 唯一

### 6.4 `passkey_credentials`

WebAuthn/Passkey 凭据。

- 主键：`id`
- 字段：`user_id`、`credential_id`、`public_key`、`attestation_type`、`aa_guid`、`sign_count`、`clone_warning`、`user_present`、`user_verified`、`backup_eligible`、`backup_state`、`transports`、`attachment`、`last_used_at`、`created_at`、`updated_at`、`deleted_at`
- 重要约束：`user_id`、`credential_id` 唯一
- 敏感字段：凭据 ID、公钥及认证器信息

### 6.5 `two_fas`

用户两步验证配置。

- 主键：`id`
- 字段：`user_id`、`secret`、`is_enabled`、`failed_attempts`、`locked_until`、`last_used_at`、`created_at`、`updated_at`、`deleted_at`
- 重要约束：`user_id` 唯一
- 敏感字段：`secret`

### 6.6 `two_fa_backup_codes`

两步验证备用码。

- 主键：`id`
- 字段：`user_id`、`code_hash`、`is_used`、`used_at`、`created_at`、`deleted_at`
- 敏感字段：`code_hash`

### 6.7 `custom_oauth_providers`

管理员配置的自定义 OAuth/OIDC 提供商。

- 主键：`id`
- 字段：`name`、`slug`、`icon`、`enabled`、`client_id`、`client_secret`、`authorization_endpoint`、`token_endpoint`、`user_info_endpoint`、`scopes`、`user_id_field`、`username_field`、`display_name_field`、`email_field`、`well_known`、`auth_style`、`access_policy`、`access_denied_message`、`created_at`、`updated_at`
- 重要约束：`slug` 唯一
- 敏感字段：`client_secret`

### 6.8 `user_oauth_bindings`

用户与自定义 OAuth 提供商账号的绑定关系。

- 主键：`id`
- 字段：`user_id`、`provider_id`、`provider_user_id`、`created_at`
- 重要约束：`user_id + provider_id` 唯一；`provider_id + provider_user_id` 唯一
- 逻辑引用：`user_id -> users.id`，`provider_id -> custom_oauth_providers.id`

## 7. 计费与用量表

### 7.1 `redemptions`

兑换码及使用状态。

- 主键：`id`
- 字段：`user_id`、`key`、`status`、`name`、`quota`、`created_time`、`redeemed_time`、`used_user_id`、`deleted_at`、`expired_time`
- 重要约束：`key` 唯一

### 7.2 `top_ups`

充值订单。

- 主键：`id`
- 字段：`user_id`、`amount`、`money`、`trade_no`、`payment_method`、`payment_provider`、`create_time`、`complete_time`、`status`
- 重要约束：`trade_no` 唯一

### 7.3 `checkins`

签到奖励记录。

- 主键：`id`
- 字段：`user_id`、`checkin_date`、`quota_awarded`、`created_at`
- 重要约束：`user_id + checkin_date` 唯一

### 7.4 `logs`

请求、消费和系统事件日志。

- 主键：`id`
- 字段：`user_id`、`created_at`、`type`、`content`、`username`、`token_name`、`model_name`、`quota`、`prompt_tokens`、`completion_tokens`、`use_time`、`is_stream`、`channel_id`、`channel_name`、`token_id`、`group`、`ip`、`request_id`、`upstream_request_id`、`other`
- 重要索引：时间、用户、令牌、渠道、模型、日志类型、请求 ID 等检索字段
- 特别说明：配置 `LOG_SQL_DSN` 后，运行时读写的是独立日志数据库中的 `logs`

### 7.5 `quota_data`

数据仪表板使用的小时级用量聚合。

- 主键：`id`
- 字段：`user_id`、`username`、`model_name`、`created_at`、`use_group`、`token_id`、`channel_id`、`node_name`、`token_used`、`count`、`quota`
- 说明：`created_at` 按小时归档；部分数据会先在进程内聚合再落库

### 7.6 `perf_metrics`

模型广场使用的性能指标聚合。

- 主键：`id`
- 字段：`model_name`、`group`、`bucket_ts`、`request_count`、`success_count`、`total_latency_ms`、`ttft_sum_ms`、`ttft_count`、`output_tokens`、`generation_ms`
- 重要约束：`model_name + group + bucket_ts` 唯一

## 8. 异步任务表

### 8.1 `midjourneys`

Midjourney 类型任务及历史兼容数据。

- 主键：`id`
- 字段：`code`、`user_id`、`action`、`mj_id`、`prompt`、`prompt_en`、`description`、`state`、`submit_time`、`start_time`、`finish_time`、`image_url`、`video_url`、`video_urls`、`status`、`progress`、`fail_reason`、`channel_id`、`quota`、`buttons`、`properties`

### 8.2 `tasks`

统一异步任务记录。

- 主键：`id`
- 字段：`created_at`、`updated_at`、`task_id`、`platform`、`user_id`、`group`、`channel_id`、`quota`、`action`、`status`、`fail_reason`、`submit_time`、`start_time`、`finish_time`、`progress`、`properties`、`private_data`、`data`
- 逻辑引用：`user_id -> users.id`，`channel_id -> channels.id`

## 9. 订阅表

### 9.1 `subscription_plans`

订阅计划定义。

- 主键：`id`
- 字段：`title`、`subtitle`、`price_amount`、`currency`、`duration_unit`、`duration_value`、`custom_seconds`、`enabled`、`sort_order`、`allow_balance_pay`、`allow_wallet_overflow`、`stripe_price_id`、`creem_product_id`、`waffo_pancake_product_id`、`max_purchase_per_user`、`upgrade_group`、`downgrade_group`、`total_amount`、`quota_reset_period`、`quota_reset_custom_seconds`、`created_at`、`updated_at`
- 金额字段：`price_amount` 使用定点小数，避免浮点金额误差

### 9.2 `subscription_orders`

订阅购买订单。

- 主键：`id`
- 字段：`user_id`、`plan_id`、`money`、`trade_no`、`payment_method`、`payment_provider`、`status`、`create_time`、`complete_time`、`provider_payload`
- 重要约束：`trade_no` 唯一

### 9.3 `user_subscriptions`

用户已经生效或历史订阅。

- 主键：`id`
- 字段：`user_id`、`plan_id`、`amount_total`、`amount_used`、`start_time`、`end_time`、`status`、`source`、`last_reset_time`、`next_reset_time`、`upgrade_group`、`prev_user_group`、`downgrade_group`、`allow_wallet_overflow`、`created_at`、`updated_at`

### 9.4 `subscription_pre_consume_records`

订阅额度预消费记录，用于请求结算和幂等控制。

- 主键：`id`
- 字段：`request_id`、`user_id`、`user_subscription_id`、`pre_consumed`、`status`、`created_at`、`updated_at`
- 重要约束：`request_id` 唯一

## 10. 系统任务与节点表

### 10.1 `system_instances`

运行节点心跳和实例信息。

- 主键：`node_name`
- 字段：`info`、`started_at`、`last_seen_at`、`created_at`、`updated_at`

### 10.2 `system_tasks`

后台系统任务状态。

- 主键：`id`
- 字段：`task_id`、`type`、`status`、`active_key`、`payload`、`state`、`result`、`error`、`locked_by`、`created_at`、`updated_at`
- 重要约束：`task_id` 唯一；`active_key` 唯一

### 10.3 `system_task_locks`

系统任务的分布式数据库锁。

- 主键：`type`
- 字段：`task_id`、`locked_by`、`locked_until`、`updated_at`

## 11. 集群状态表

### 11.1 `clusters`

监控 Agent 集群配置、凭据状态和最近一次轮询状态。

- 主键：`id`
- 字段：`model_id`、`model_name_snapshot`、`name`、`link_secret_ciphertext`、`credential_status`、`credential_version`、`credential_issued_at`、`credential_verified_at`、`enabled`、`health_status`、`last_polled_at`、`last_success_at`、`consecutive_failures`、`last_error_code`、`last_failure_payload`、`next_poll_at`、`poll_locked_by`、`poll_locked_until`、`created_at`、`updated_at`
- 逻辑引用：`model_id -> models.id`
- 敏感字段：`link_secret_ciphertext`
- 恢复要求：必须同时保留原 `CRYPTO_SECRET`、原 `SESSION_SECRET`，或原集群密钥文件，否则已有 Agent 密钥可能无法解密

### 11.2 `cluster_telemetry_latest`

每个集群最新一份遥测数据。

- 主键：`cluster_id`
- 字段：`schema_version`、`collection_id`、`raw_payload`、`normalized_payload`、`collected_at`、`updated_at`
- 逻辑引用：`cluster_id -> clusters.id`

### 11.3 `cluster_telemetry_history`

集群遥测历史采样。成功和失败轮询都会写入，历史保留天数由 `options.ClusterTelemetryRetentionDays` 控制。

- 主键：`id`
- 字段：`cluster_id`、`collection_id`、`status`、`health_status`、`schema_version`、`normalized_payload`、`error_code`、`collected_at`、`created_at`
- 逻辑引用：`cluster_id -> clusters.id`
- 成功采样：`status=success`，保存 `collection_id`、`schema_version` 和 `normalized_payload`
- 失败采样：`status=error`，只保存脱敏的 `error_code`，不保存原始响应和诊断载荷
- 时间索引：`(cluster_id, collected_at, id)` 和 `(collected_at, id)`
- 去重索引：`(cluster_id, collection_id)`；失败采样的 `collection_id` 为 `NULL`
- 清理规则：主节点按保留天数每小时分批删除过期记录

## 12. 时间、软删除和敏感数据约定

- 多数 `*_time`、`*_at` 字段存 Unix 时间戳，具体单位以对应模型和调用代码为准；当前主要为秒。
- `deleted_at` 表示 GORM 软删除。直接查询数据库时，需要明确是否包含已经软删除的数据。
- `group`、`key` 等名称在某些数据库中是保留字；项目通过 GORM 和共享列名适配处理，不应在业务代码中拼接不带转义的原始 SQL。
- 数据库备份包含用户密码哈希、API 令牌、渠道密钥、OAuth 密钥、两步验证密钥等敏感数据，应加密存储并限制读取权限。

## 13. 从实际 PostgreSQL 导出结构

以下命令只读取 schema，不导出业务行。建议通过 `.pgpass`、临时环境或密钥管理系统提供凭据，不要把完整 DSN 提交到仓库。

```bash
export NEWAPI_SCHEMA_DSN='postgresql://newapi:REDACTED@127.0.0.1:5432/new_api?sslmode=require'

pg_dump \
  --schema-only \
  --no-owner \
  --no-acl \
  --dbname="$NEWAPI_SCHEMA_DSN" \
  --file=new-api-schema.sql

psql "$NEWAPI_SCHEMA_DSN" -c '\dt+ public.*'
psql "$NEWAPI_SCHEMA_DSN" -c '\d+ public.users'
psql "$NEWAPI_SCHEMA_DSN" -c '\d+ public.channels'
psql "$NEWAPI_SCHEMA_DSN" -c '\d+ public.clusters'
```

如果配置了独立日志数据库，再对 `LOG_SQL_DSN` 指向的数据库执行一次 schema 导出。

## 14. 迁移和交接检查清单

- 记录 new-api 版本或 Git commit、PostgreSQL 大版本、Redis 版本。
- 同时保存主库和独立日志库的逻辑备份。
- 保存部署环境变量的安全副本，尤其是 `SESSION_SECRET`、`CRYPTO_SECRET` 和集群密钥文件。
- 记录 `SQL_DSN`、`LOG_SQL_DSN`、`REDIS_CONN_STRING` 的数据库名、主机、端口和 Redis DB 编号，但不要把密码写入本文或提交到 Git。
- 恢复后核对 `users`、`tokens`、`channels`、`options`、计费订阅和集群相关表的数量。
- 首次启动新版本前再做一次备份；只允许主节点执行迁移。
- 不要在没有备份的情况下手工删除列、修改主键或清空缓存与数据库。
