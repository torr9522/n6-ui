# N5-UI Database

更新时间：2026-09-06
适用版本：`v0.2.0` Stable

## 1. 数据库说明

- 原数据库：SQLite
- 原核心表：`users`、`inbounds`、`settings`、`access_ip_records`
- N5 扩展策略：不改旧表主结构，新增 `n5_` 独立表；必要状态字段追加到 N5 自己的表中
- v0.2.0 hardening 未引入大规模 DB schema redesign；重点是引用保护、事务边界、ownership 检查和一致性验证。
- Simple-managed ownership markers: `n5-simple-exec|` and legacy `n5-simple|`.

## 2. N5 表清单

### 2.1 `n5_egresses`

用途：

- 记录所有出口线路定义。
- 存储可直接 merge 的 `outbound_json`。
- 记录最近测试状态与出口 IP。

字段：

| 字段 | 类型 | 用途 |
|---|---|---|
| `id` | int | 主键 |
| `name` | string | 出口名称 |
| `remark` | string | 备注 |
| `protocol` | string | 归一化协议名 |
| `tag` | string | 稳定 outbound tag，唯一 |
| `enabled` | bool | 是否启用 |
| `outbound_json` | text | 完整 Xray outbound 片段 |
| `last_status` | string | 最近状态，如 `success` / `failed` |
| `last_exit_ip` | string | 最近测试得到的出口 IP |
| `last_test_time` | int64 | 最近测试时间戳 |
| `last_test_status` | string | 最近测试状态镜像 |
| `last_test_latency_ms` | int | 最近测试延迟 |
| `last_test_error` | text | 最近测试错误信息 |
| `last_test_at` | int64 | 最近测试时间戳镜像 |
| `created_at` | int64 | 创建时间 |
| `updated_at` | int64 | 更新时间 |

关系：

- 1 对多 `n5_egress_test`
- 多对多 `n5_egress_labels` 通过 `n5_egress_label_relations`
- 多对多 `n5_egress_pools` 通过 `n5_egress_pool_members`
- 被 `n5_traffic_policies` 和 `n5_traffic_policy_rules` 以 `target_type=egress` 引用

### 2.2 `n5_egress_test`

用途：

- 记录手动出口测试结果。

字段：

| 字段 | 类型 | 用途 |
|---|---|---|
| `id` | int | 主键 |
| `egress_id` | int | 关联出口 ID |
| `status` | string | 测试状态 |
| `latency` | int | 延迟毫秒 |
| `exit_ip` | string | HTTP 出口 IP |
| `message` | text | 错误或说明 |
| `tested_at` | int64 | 测试时间 |

关系：

- 多对一 `n5_egresses.id`

### 2.3 `n5_egress_labels`

用途：

- 定义出口标签。

字段：

| 字段 | 类型 | 用途 |
|---|---|---|
| `id` | int | 主键 |
| `name` | string | 标签名 |
| `type` | string | 标签类型，如 `region` / `usage` / `quality` |
| `created_at` | int64 | 创建时间 |
| `updated_at` | int64 | 更新时间 |

关系：

- 多对多 `n5_egresses` 通过 `n5_egress_label_relations`

### 2.4 `n5_egress_label_relations`

用途：

- 出口与标签的多对多关系表。

字段：

| 字段 | 类型 | 用途 |
|---|---|---|
| `id` | int | 主键 |
| `egress_id` | int | 出口 ID |
| `label_id` | int | 标签 ID |
| `created_at` | int64 | 创建时间 |

约束：

- `(egress_id, label_id)` 唯一

关系：

- 多对一 `n5_egresses.id`
- 多对一 `n5_egress_labels.id`

### 2.5 `n5_egress_pools`

用途：

- 定义线路池与 balancer 元信息。

字段：

| 字段 | 类型 | 用途 |
|---|---|---|
| `id` | int | 主键 |
| `name` | string | 线路池名称 |
| `remark` | string | 备注 |
| `tag` | string | 稳定 balancer tag |
| `strategy` | string | 线路池策略，如 `random` |
| `fallback_type` | string | 回退目标类型 |
| `fallback_target_id` | int | 回退目标 ID |
| `enabled` | bool | 是否启用 |
| `created_at` | int64 | 创建时间 |
| `updated_at` | int64 | 更新时间 |

关系：

- 1 对多 `n5_egress_pool_members`
- 被 `n5_traffic_policies` / `n5_traffic_policy_rules` 以 `target_type=pool` 引用

### 2.6 `n5_egress_pool_members`

用途：

- 定义线路池中的出口成员。

字段：

| 字段 | 类型 | 用途 |
|---|---|---|
| `id` | int | 主键 |
| `pool_id` | int | 线路池 ID |
| `egress_id` | int | 出口 ID |
| `weight` | int | 权重 |
| `sort_order` | int | 顺序 |
| `enabled` | bool | 是否启用 |
| `created_at` | int64 | 创建时间 |
| `updated_at` | int64 | 更新时间 |

约束：

- `(pool_id, egress_id)` 唯一

关系：

- 多对一 `n5_egress_pools.id`
- 多对一 `n5_egresses.id`

### 2.7 `n5_traffic_policies`

用途：

- 定义一个完整分流策略的根对象。

字段：

| 字段 | 类型 | 用途 |
|---|---|---|
| `id` | int | 主键 |
| `name` | string | 策略名 |
| `remark` | string | 备注；Simple Mode 用 remark 写入元信息前缀 |
| `enabled` | bool | 策略是否启用 |
| `default_target_type` | string | 未匹配流量目标类型：`egress` / `pool` |
| `default_target_id` | int | 未匹配流量目标 ID |
| `created_at` | int64 | 创建时间 |
| `updated_at` | int64 | 更新时间 |

关系：

- 1 对多 `n5_traffic_policy_rules`
- 1 对多 `n5_traffic_policy_bindings`

### 2.8 `n5_traffic_policy_rules`

用途：

- 定义策略中的单条分流规则。

字段：

| 字段 | 类型 | 用途 |
|---|---|---|
| `id` | int | 主键 |
| `policy_id` | int | 所属策略 ID |
| `rule_type` | string | 规则类型，如 `domain` / `ip` |
| `match_mode` | string | 匹配模式，如 `full` / `suffix` / `keyword` |
| `match_value` | text | 匹配内容 |
| `target_type` | string | 目标类型：`egress` / `pool` |
| `target_id` | int | 目标 ID |
| `sort_order` | int | 规则顺序 |
| `enabled` | bool | 规则是否启用 |
| `created_at` | int64 | 创建时间 |
| `updated_at` | int64 | 更新时间 |

关系：

- 多对一 `n5_traffic_policies.id`

### 2.9 `n5_traffic_policy_bindings`

用途：

- 将入站绑定到某个策略。

字段：

| 字段 | 类型 | 用途 |
|---|---|---|
| `id` | int | 主键 |
| `inbound_id` | int | 关联原 `inbounds.id` |
| `policy_id` | int | 关联 `n5_traffic_policies.id` |
| `enabled` | bool | 绑定是否启用 |
| `created_at` | int64 | 创建时间 |
| `updated_at` | int64 | 更新时间 |

约束：

- `inbound_id` 唯一索引
- 语义：一个 inbound 同时只能生效一个 N5 policy

关系：

- 多对一 `inbounds.id`
- 多对一 `n5_traffic_policies.id`

### 2.10 `n5_xray_config_history`

用途：

- 保存每次 N5 merge 产物摘要和应用状态。

字段：

| 字段 | 类型 | 用途 |
|---|---|---|
| `id` | int | 主键 |
| `source` | string | 触发来源 |
| `base_config_hash` | string | 原 x-ui base config hash |
| `extension_config_hash` | string | N5 扩展片段 hash |
| `config_hash` | string | 合并后最终配置 hash |
| `config_json` | text | 合并后完整配置 JSON |
| `apply_status` | string | `pending` / `validated` / `applied` / `failed` |
| `apply_error` | text | 应用失败错误 |
| `created_at` | int64 | 创建时间 |

关系：

- 独立历史表，不做外键绑定

## 3. 迁移逻辑

文件：`database/n5_phase2.go`

### 3.1 AutoMigrate

- 在 `initN5Phase2()` 中统一 `AutoMigrate` 所有 N5 表。

### 3.2 稳定 tag 迁移

- `migrateN5StableTags()`
- 把旧格式：
  - `n5-egress-{id}`
  - `n5-pool-{id}`
- 迁移为固定宽度：
  - `n5-egress-%010d`
  - `n5-pool-%010d`

目的：

- 避免 balancer selector 前缀误匹配
- 保持 tag 可预测与稳定

### 3.3 Binding 唯一索引迁移

- `migrateN5TrafficPolicyBindingUniqueIndex()`
- 先清理重复 `inbound_id`
- 再创建唯一索引：
  - `idx_n5_policy_binding_inbound`

## 4. 数据流关系

```text
n5_egresses
  -> n5_egress_test
  -> n5_egress_label_relations -> n5_egress_labels
  -> n5_egress_pool_members -> n5_egress_pools
  -> n5_traffic_policies.default_target_id (when default_target_type=egress)
  -> n5_traffic_policy_rules.target_id (when target_type=egress)

n5_egress_pools
  -> n5_egress_pool_members
  -> n5_traffic_policies.default_target_id (when default_target_type=pool)
  -> n5_traffic_policy_rules.target_id (when target_type=pool)

n5_traffic_policies
  -> n5_traffic_policy_rules
  -> n5_traffic_policy_bindings

inbounds
  -> n5_traffic_policy_bindings.inbound_id
```

### 4.1 v0.2.0 Consistency Expectations

Stable validation requires these counters to be zero after runtime gates:

- orphan policy
- orphan rule
- orphan binding
- missing egress target
- missing inbound binding
- metadata missing rule IDs
- duplicate binding
- pool member missing egress

## 5. 与运行时的映射

- `n5_egresses.outbound_json`
  - 直接作为 outbound 片段来源
- `n5_egress_pools`
  - 生成 balancer
- `n5_traffic_policies + rules + bindings`
  - 生成 routing.rules
- `n5_xray_config_history`
  - 记录 merge 历史

## 6. 当前数据库边界

- 不修改原 `users` / `inbounds` / `settings` 主结构
- N5 所有增强功能使用独立 `n5_` 表
- N5 与原 x-ui 的交点仅有：
  - `n5_traffic_policy_bindings.inbound_id -> inbounds.id`
  - `settings.n5XrayExtensionEnable`
