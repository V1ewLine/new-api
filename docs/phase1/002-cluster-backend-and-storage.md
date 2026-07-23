# 集群后端与存储

## 日期

2026-07-23

## 本阶段目标

建立集群配置、最新遥测、Secret 安全存储和管理员 API 的后端基础。

## 实际修改

- 新增集群配置和最新遥测 GORM 模型。
- 新增模型联动、一级页面聚合、模型详情和集群详情 Service。
- 新增临时连接密钥 Resolver 和 AES-GCM SecretProtector。
- 新增管理员集群 API，并接入统一 API 返回格式。

## 新增或修改的文件

- `model/cluster.go`
- `model/main.go`
- `service/clusterstatus/types.go`
- `service/clusterstatus/security.go`
- `service/clusterstatus/service.go`
- `controller/cluster.go`
- `router/api-router.go`

## 数据库与接口变更

新增：

- `clusters`
- `cluster_telemetry_latest`

两张表均加入普通和快速 `AutoMigrate` 列表。接口清单见 `001-cluster-module-plan.md`。

## 关键设计决策

- 集群记录关联 `models.id`，但不使用级联删除，保证模型删除后集群仍可审计。
- `model_name_snapshot` 保留创建时名称。
- `link_secret_ciphertext` 不参与 JSON 序列化。
- 最新遥测同时保存 Agent 原始响应和规范化响应，前端仅接收规范化响应。
- schema 解析失败时在集群记录的隐藏诊断字段保存受响应体上限约束的原始载荷，不覆盖最后一次成功遥测，也不通过 API 返回。
- 一级页面使用一个聚合接口返回统计、模型分组、告警和分页，避免请求瀑布。

## 遇到的问题

模型可能被软删除，普通查询无法为已有集群恢复模型信息。

## 解决方式

详情和聚合查询使用 GORM `Unscoped` 读取历史模型引用，并通过 `DeletedAt` 与 `status` 计算 `model_available`。

## 测试与验证

已补充 SQLite 模型联动、Secret 不泄露、模型禁用、筛选和分页测试；MySQL、PostgreSQL 由统一 GORM 模型与现有迁移流程兼容。

## 当前限制

- 暂未提供集群编辑和删除接口。
- 历史遥测仓库为空实现。

## 下一步

实现 Agent Client、Schema Adapter、健康评估和后台轮询器。
