# N5-UI Xray Merge Layer Design

日期：2026-08-08

状态：Phase 2.4-Pre 设计，不包含主配置接入实现。

## 1. 设计目标

建立独立的 N5 Xray 配置合并层，后续把：

```text
旧 x-ui 模板配置
        +
N5 outbound
        +
N5 routing rules
        +
N5 balancers
```

合并为最终 `xray.Config`。

约束：

- 不修改旧 `xray.go` 的核心生成逻辑。
- 不修改旧 service。
- 不修改旧 controller。
- 不改变旧安装命令、服务名称、配置路径和默认端口。
- N5 只追加增强配置，不接管原 x-ui 兼容层。

## 2. Merge 入口

建议新增独立 service：

```text
web/service/n5/xray_merge.go
```

建议接口：

```go
type XrayMergeService struct {
}

func (s *XrayMergeService) Merge(base *xray.Config) (*xray.Config, error)
```

职责：

1. 接收旧配置生成链路得到的 `xray.Config`
2. 读取启用的 N5 egress、pool、policy、binding
3. 生成 N5 outbound fragments
4. 生成 N5 routing fragments
5. 校验 tag 引用关系
6. 合并配置并返回新对象
7. 生成配置 hash
8. 记录 `n5_xray_config_history`

Phase 2.4-Pre 不调用该入口，也不把它接入旧 `xray.go`。

## 3. Outbound 追加规则

### 3.1 保留旧 outbounds 顺序

最终顺序：

```text
base.OutboundConfigs
        +
N5 outbounds
```

N5 outbounds 必须追加，不能前置。

原因：

- Xray outbound manager 的默认 handler 是第一个加入的 outbound。
- 当前模板第一个 outbound 是原始 `freedom`。
- 将 N5 outbound 前置会改变未命中规则时的默认行为。

依据：

- `web/service/config.json`
- `xray-core/app/proxyman/outbound/outbound.go`

### 3.2 N5 outbound 要求

每个启用出口必须满足：

- `protocol` 存在
- `settings` 结构合法
- `tag` 存在
- `tag` 与数据库 `Egress.Tag` 一致
- tag 在最终 outbounds 中唯一

禁止：

- 覆盖旧 outbound
- 修改旧 outbound 的 tag
- 将未启用出口追加到最终配置

### 3.3 Tag 规范

当前 N5 tag 采用固定长度 ID：

```text
n5-egress-0000000001
n5-pool-0000000001
```

固定长度的目标是避免 Xray selector 前缀匹配造成 ID 碰撞。

## 4. Routing 插入规则

### 4.1 基本结构

N5 routing fragment：

```json
{
  "rules": [],
  "balancers": []
}
```

每条 N5 规则必须有：

- `type: "field"`
- `inboundTag`
- `outboundTag` 或 `balancerTag`

### 4.2 Legacy 规则优先级

为了保持 x-ui 兼容，旧模板规则必须保留原有相对顺序。

推荐合并原则：

1. 保留 API inbound 规则
2. 保留原有安全阻断规则
3. 插入 N5 规则
4. 保留其他旧规则

如果无法可靠识别旧规则类型，默认策略应为：

- 保留整个旧 rules 数组原顺序
- 将 N5 rules 追加到旧 rules 之后

这样优先保护旧 x-ui 行为，但意味着：

- 已被旧通用规则命中的流量不会再进入 N5 规则。

该行为必须在 Phase 2.4 文档和测试中明确。

### 4.3 N5 规则内部顺序

同一 inbound：

1. 显式域名规则
2. 显式 IP/CIDR 规则
3. 默认出口规则

数据库排序：

```text
sort_order asc, id asc
```

默认出口必须最后追加，避免吞掉更具体的规则。

### 4.4 inboundTag 实时解析

Binding 的主来源是：

```text
TrafficPolicyBinding.InboundId
```

每次 Merge 时必须：

1. 按 `InboundId` 查询旧 `model.Inbound`
2. 读取当前 `Inbound.Tag`
3. 将当前 tag 写入 `routing.rules[].inboundTag`

禁止：

- 把 `inboundTag` 作为 Binding 的永久主来源
- 使用历史配置中的 tag 覆盖当前数据库 tag
- 依赖端口直接拼接 tag

原因：

- 旧 x-ui 更新入站时会根据端口重写 tag。

如果 inbound 不存在或 tag 为空：

- 跳过该 binding
- 记录错误
- 不生成无法命中的 routing rule

## 5. Balancer 合并规则

### 5.1 Balancer 结构

目标结构：

```json
{
  "tag": "n5-pool-0000000001",
  "selector": [
    "n5-egress-0000000001",
    "n5-egress-0000000002"
  ],
  "strategy": {
    "type": "random"
  },
  "fallbackTag": "n5-egress-0000000003"
}
```

### 5.2 引用校验

Merge 前必须校验：

- pool tag 唯一
- selector 对应的 egress tag 存在
- selector 对应出口已启用
- fallback tag 存在
- fallback tag 指向有效 egress 或有效 pool 目标
- routing rule 引用的 balancer tag 存在
- routing rule 引用的 outbound tag 存在

任何悬空引用都应阻止配置应用。

### 5.3 Selector 前缀语义

Xray selector 使用前缀匹配：

```go
strings.HasPrefix(tag, selector)
```

因此 selector 不能使用可互相成为前缀的可变长度 ID tag。

Phase 2.4-Pre 的修正方案：

```text
n5-egress-{10位固定数字}
n5-pool-{10位固定数字}
```

要求：

- 所有新生成 tag 使用固定长度数字
- 测试覆盖多个 ID 的两两前缀关系
- 生产已有旧 tag 在正式启用线路池前必须迁移或隔离

## 6. Default Target 规则

`TrafficPolicy.DefaultTargetType` 与 `DefaultTargetId` 的语义定义为：

```text
当前绑定 inbound 的默认目标
```

不是：

```text
系统全局默认 outbound
```

生成规则：

```json
{
  "type": "field",
  "inboundTag": ["current-inbound-tag"],
  "outboundTag": "n5-egress-0000000001"
}
```

或：

```json
{
  "type": "field",
  "inboundTag": ["current-inbound-tag"],
  "balancerTag": "n5-pool-0000000001"
}
```

默认规则必须位于该 inbound 的显式规则之后。

未绑定 N5 策略的 inbound：

- 不生成 N5 默认规则
- 继续沿用旧模板默认路由行为

## 7. Config History 记录

Merge 成功或失败都应记录 `n5_xray_config_history`。

建议字段：

- `source`
  - `n5-merge`
- `configHash`
  - 最终完整配置的 SHA-256
- `configJson`
  - 合并后的完整配置或受控脱敏版本
- `applyStatus`
  - `generated`
  - `validated`
  - `applied`
  - `failed`
- `applyError`
  - 失败原因
- `createdAt`
  - 创建时间

推荐状态流：

```text
generated
    -> validated
    -> applied
```

失败状态：

```text
generated
    -> failed
```

注意：

- 配置历史必须保存最终合并结果，而不是只保存 N5 局部片段。
- 这样才能审计 N5 片段是否真正进入最终 Xray 配置。

## 8. Phase 2.4 接入边界

Phase 2.4 允许新增：

- 独立 N5 merge service
- 独立配置校验器
- 独立配置 hash/history service
- 独立测试 fixtures

Phase 2.4 暂不允许：

- 修改旧 `xray.go`
- 修改旧 service 的原始生成逻辑
- 修改旧 controller
- 修改安装脚本
- 修改 systemd service

如果旧 `xray.go` 没有可用扩展入口，必须先设计一个兼容调用边界，再由后续变更单独批准；不能直接散改旧逻辑。

## 9. Phase 2.4 验收标准

### 结构验收

- 最终 outbounds 保留旧顺序
- N5 outbounds 位于旧 outbounds 之后
- 所有 routing target 引用存在
- 所有 balancer selector 无前缀碰撞
- 所有绑定 inboundTag 来自实时数据库查询

### 行为验收

- egress 直连规则
- pool balancer 规则
- 域名规则
- IP/CIDR 规则
- 默认目标规则
- inbound 端口变更后重新生成能使用新 tag
- 未绑定 inbound 保持旧默认行为

### 回滚验收

- N5 merge 失败时不覆盖旧有效配置
- `n5_xray_config_history` 能定位失败原因
- 原 x-ui 配置可以独立恢复

## 10. 当前设计结论

Phase 2.4 的正确边界是：

```text
旧 x-ui 配置生成
        |
        v
独立 N5 Merge Layer
        |
        +--> 追加 N5 outbounds
        +--> 合并 N5 routing rules
        +--> 合并 N5 balancers
        +--> 记录 config history
        |
        v
最终 xray.Config
```

本设计不改变 x-ui 的服务、安装、数据库旧表、API 和原始 Xray 兼容层。
