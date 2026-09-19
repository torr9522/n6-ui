# N5-UI Phase 2 Xray Integration Audit

日期：2026-08-08

范围：
- N5 扩展数据模型与 Service 生成逻辑
- 当前 `x-ui` 入站模型与 Xray 配置生成链路
- `web/service/n5/xray_ext.go`
- `web/service/xray.go`
- 项目内 `xray-core/` 子树
- 当前 `go.mod` 依赖的 `github.com/xtls/xray-core v1.4.2`

说明：
- 本报告只做技术审计，不修改代码。
- 本报告的“必须修改项”是对 Phase 2.4 接入前的阻断项，不代表本阶段已实施。

## 当前状态

### 1. TrafficPolicyBinding 到 inboundTag 的映射方式

当前绑定表只保存：
- `n5_traffic_policy_bindings.inbound_id`
- `n5_traffic_policy_bindings.policy_id`

当前路由片段生成时，并不是直接使用 `inboundId`，而是运行时按 `inboundId` 查询旧 `Inbound` 记录，再读取 `Inbound.Tag` 写入 `routing.rules[].inboundTag`。

依据：
- `database/model/n5/models.go`
- `web/service/n5/xray_ext.go`
- `database/model/model.go`

结论：
- `inboundId -> 当前 inboundTag` 的运行时映射是成立的。
- 但 `inboundId -> 固定不变 inboundTag` 并不成立。

原因：
- 旧 `xui` 新增入站时把 tag 设为 `inbound-{port}`。
- 旧 `xui` 更新入站时也会重写 tag 为 `inbound-{port}`。

依据：
- `web/controller/inbound.go`
- `web/service/inbound.go`

这意味着：
- 只要端口变了，旧 inbound 的 tag 就会变。
- Binding 通过 `inboundId` 重新查表后，能跟上“当前 tag”。
- 但任何缓存、历史快照、外部引用都不能假设 inboundTag 是永久稳定值。

### 2. XrayExtService 当前生成的 outbound 片段

`GenerateOutboundFragments()` 当前行为：
- 读取所有启用的 `n5_egresses`
- 反序列化 `outbound_json`
- 原样输出为 `[]map[string]interface{}`

依据：
- `web/service/n5/xray_ext.go`

当前出口创建/更新阶段会保证：
- `tag` 被写回 JSON
- `protocol` 与 JSON 一致
- JSON 至少是对象，且包含 `settings`

依据：
- `web/service/n5/egress.go`
- `web/service/n5/common.go`

结论：
- 对 N5 自己的最小结构来说，outbound 片段能稳定生成。
- 但这不等于“已经严格符合最终接入的 Xray Core 配置结构”。

原因有两个：
- 当前项目同时存在两套 Xray Core 基线。
- 无本地 `xray` 二进制时，校验会退化为最小形状校验，不是完整协议级校验。

### 3. 当前项目存在两套 Xray Core 基线

基线 A：
- 面板 Go 模块依赖：`github.com/xtls/xray-core v1.4.2`
- 见根目录 `go.mod`

基线 B：
- 仓库内自带 `xray-core/` 子树
- 其 `go.mod` 为更高版本体系，且 `routing.balancers` 已支持 `strategy` 与 `fallbackTag`

依据：
- `/root/n5-ui/go.mod`
- `/root/n5-ui/xray-core/go.mod`
- `xray-core/infra/conf/router.go`

结论：
- “当前 vendored xray-core 配置结构”与“当前面板编译时依赖的 xray-core v1.4.2”不是同一套结构。
- Phase 2.4 如果不先选定唯一基线，接入会产生语义漂移。

## 已验证部分

### 1. Binding 唯一约束

当前迁移已经确保：
- `n5_traffic_policy_bindings.inbound_id` 唯一
- 重复记录保留最新一条

依据：
- `database/n5_phase2.go`

这保证了：
- 一个 inbound 同时只会有一个生效策略
- Phase 2.4 不需要再处理“一入站多策略并发命中”的歧义

### 2. routing rules 基本结构

当前生成的每条规则结构为：
- `type: "field"`
- `inboundTag: [currentInboundTag]`
- 规则命中字段之一：
  - `domain`
  - `ip`
- 目标字段之一：
  - `outboundTag`
  - `balancerTag`

依据：
- `web/service/n5/xray_ext.go`

这与 Xray 路由规则的基本字段模型一致。

依据：
- `xray-core/infra/conf/router.go`
- `xray-core` 对应 `parseFieldRule(...)`

### 3. 规则优先级的当前实现

当前顺序是：
1. 按 `binding.id asc` 遍历绑定
2. 每个策略内按 `sort_order asc, id asc` 生成规则
3. 每个策略的默认出口规则放在该策略规则的最后

依据：
- `web/service/n5/xray_ext.go`

结论：
- 同一 inbound 内，显式规则优先于默认出口规则。
- 不同 inbound 之间因为 `inboundTag` 不同，互不竞争。

### 4. 线路池 balancer 基本结构

当前生成：
- `tag = pool.Tag`
- `selector = []egress.Tag`
- 可选 `strategy`
- 可选 `fallbackTag`

依据：
- `web/service/n5/xray_ext.go`

### 5. 默认出口逻辑

当前 N5 默认出口不是“全局默认 outbound”。

当前只实现了：
- 每个 `TrafficPolicy` 可生成一条“该 inbound 的默认命中规则”

如果请求没有命中任何 N5 规则：
- 是否走 N5 默认，不由 N5 决定
- 仍取决于最终主 Xray 配置中的默认路由行为

当前模板默认 outbounds：
1. 第一个是未命名 `freedom`
2. 第二个是 `blocked`

依据：
- `web/service/config.json`

这意味着：
- 在不改动默认模板顺序的前提下，未命中任何规则的流量会回落到原模板的默认出口语义，而不是 N5 出口。

## 风险点

### 1. inboundTag 不是冻结标识

风险等级：高

问题：
- Binding 存的是 `inboundId`
- Xray 认的是 `inboundTag`
- 而旧 `xui` 会在入站改端口时重写 tag

影响：
- Phase 2.4 必须在每次生成主配置时实时解析 `inboundId -> 当前 tag`
- 不能把历史 tag 直接缓存成长期真值

### 2. balancer selector 是前缀匹配，不是精确匹配

风险等级：高

Xray outbound selector 的匹配逻辑是 `strings.HasPrefix(tag, selector)`。

依据：
- `xray-core/app/proxyman/outbound/outbound.go`

当前 N5 线路池成员使用：
- selector = `n5-egress-{id}`

这会导致前缀碰撞：
- selector `n5-egress-1` 会同时命中
  - `n5-egress-1`
  - `n5-egress-10`
  - `n5-egress-11`
  - ...

影响：
- 当前线路池 balancer 在出口数量增多后可能把错误出口纳入池中。
- 这是 Phase 2.4 前的阻断问题。

### 3. Xray Core 基线不一致

风险等级：高

当前存在分叉：
- 根模块依赖 `github.com/xtls/xray-core v1.4.2`
- 仓库内 `xray-core/` 子树是更新实现

差异点之一：
- `go.mod` 对应的 `v1.4.2` 路由 balancer 结构只认 `tag` 和 `selector`
- 仓库内 `xray-core/infra/conf/router.go` 已支持：
  - `strategy`
  - `fallbackTag`

影响：
- 如果最终运行二进制基于旧 Core，N5 生成的 `strategy/fallbackTag` 不会按预期生效。
- 如果最终运行二进制基于仓库内新 Core，则当前设计方向是对的，但需要明确构建/发布链。

### 4. outbound 校验在无本地 xray 二进制时偏弱

风险等级：中

当前降级校验只保证：
- `protocol` 在允许列表
- `settings` 是对象

不保证：
- 协议字段完整性
- 传输层配置合法性
- 多协议细节符合目标 Core 版本

依据：
- `web/service/n5/common.go`
- `web/service/n5/egress.go`

### 5. 主配置合并顺序尚未定义

风险等级：高

当前 N5 只生成片段，不接入 `web/service/xray.go`。

真正接入时必须明确：
- N5 outbounds 是追加到模板 outbounds 后面，还是前面
- N5 routing rules 是插入到模板 rules 前面，还是后面，还是按锚点插入

影响：
- outbounds 顺序会影响“无规则命中时的默认 handler”
- rules 顺序会影响旧模板规则与 N5 规则谁先命中

### 6. 默认出口目前是“按 inbound 的默认规则”，不是“系统默认出口”

风险等级：中

这本身没有问题，但必须在 Phase 2.4 明确：
- N5 默认出口只对已绑定策略的 inbound 生效
- 未绑定策略的 inbound 完全沿用原模板默认行为

否则前后端容易误把“默认出口”理解成“全局默认出口”。

## 必须修改项

### 1. 先选定唯一 Xray Core 基线

这是最优先项。

必须先明确以下二选一：
- 以根模块当前依赖 `github.com/xtls/xray-core v1.4.2` 为准
- 以仓库内 `xray-core/` 子树及其配套二进制为准

在未统一前，不能宣称 `strategy` / `fallbackTag` 已可接入。

### 2. 修复 balancer selector 前缀碰撞

这是接入前阻断项。

当前方案不能直接用于生产。

必须至少满足其一：
- 调整 egress tag 方案，使“单个 selector 不会成为其他 tag 的前缀”
- 或引入专门的 balancer 选择标签，不再直接拿实际 outbound tag 当 selector

在未修复前：
- 线路池选路结果不可信

### 3. 明确主配置合并锚点

Phase 2.4 必须明确：
- N5 outbounds 只能追加到现有模板 outbounds 后面，不能前置

原因：
- Xray outbound manager 的默认 handler 是第一个加入的 outbound
- 当前模板第一个 outbound 是原始 `freedom`
- 如果把 N5 tagged outbound 放到前面，会改变未命中流量的默认出口行为

依据：
- `xray-core/app/proxyman/outbound/outbound.go`
- `web/service/config.json`

### 4. 明确 routing rules 的插入顺序

建议约束：
- 保留旧模板的 API 规则最高优先级
- 保留旧模板安全阻断规则优先级
- N5 规则插入到这些保留规则之后
- N5 每个 inbound 的默认出口规则必须放在该 inbound 的具体规则之后

否则会出现：
- API 流量被 N5 抢走
- 旧安全规则失效
- 默认出口规则吞掉更细的策略规则

### 5. 增加“生成时解析 inboundTag”的冻结约束

Phase 2.4 必须把以下逻辑写成硬约束：
- 永远以 `binding.inbound_id` 为主键
- 每次生成主配置时实时查 `Inbound.Tag`
- 不能在 Binding 表中持久化冗余 `inboundTag` 作为主来源

原因：
- 旧 `xui` 仍会改写入站 tag

### 6. 明确默认出口语义

必须在实现和文档中固定：
- `TrafficPolicy.DefaultTarget*` 是“绑定 inbound 的默认目标”
- 不是“系统全局默认出口”

否则后续页面和 API 命名会误导。

## Phase 2.4 接入方案

### 方案目标

在不改动旧 `xray.go` 语义边界的前提下，引入一个独立的 N5 Xray 扩展合并层：

1. 读取旧模板配置
2. 读取旧 inbound 配置
3. 生成 N5 outbounds
4. 生成 N5 routing/balancers
5. 把 N5 片段合并进最终 `xray.Config`
6. 交给原有 Xray 启停流程测试和启动

### 推荐接入步骤

#### Step 1. 固定 Core 基线

先决定：
- 运行二进制、配置语义、测试标准，以哪套 Core 为准

建议：
- 如果要使用 `strategy` 和 `fallbackTag`，应以仓库内 `xray-core/` 子树对应的二进制语义为准。

#### Step 2. 新增 N5 合并服务，不直接散改旧逻辑

建议新增独立扩展 service，例如：
- `web/service/n5/xray_merge.go`

职责：
- 接收旧 `xray.Config`
- 调用 `XrayExtService`
- 合并 outbounds / routing / balancers
- 返回新 `xray.Config`

不要在多个旧 service 中分散拼装。

#### Step 3. outbounds 合并规则

建议：
- 保留模板原有 outbounds 顺序不变
- N5 outbounds 统一追加到模板 outbounds 后面

原因：
- 避免改变原模板默认 handler
- 保持 x-ui 原生态兼容

#### Step 4. routing 合并规则

建议顺序：
1. 保留原模板 API 规则
2. 保留原模板安全规则
3. 插入 N5 routing rules
4. 保留原模板剩余规则

最低要求：
- N5 默认规则必须在本 inbound 的显式规则之后
- 不得覆盖 API inbound 的既有路由

#### Step 5. inbound 绑定解析规则

生成阶段：
- 从 `TrafficPolicyBinding` 取 `inboundId`
- 实时查询 `Inbound`
- 读取当前 `Inbound.Tag`
- 写入 `routing.rules[].inboundTag`

如果 inbound 已删除或 tag 为空：
- 跳过该 binding
- 记录审计日志或配置历史

#### Step 6. balancer 设计修正

在修复 selector 前缀问题之前，不要把当前 pool selector 直接接入主配置。

接入前需先确定一种无碰撞方案，例如：
- 固定宽度 tag
- 或专用 selector namespace

#### Step 7. 应用前验证

Phase 2.4 接入后必须增加两层验证：
- 结构验证：最终合并后的完整 `xray.Config`
- 行为验证：至少覆盖
  - egress 直连
  - pool balancer
  - domain rule
  - ip rule
  - default target
  - inbound 绑定变更后重生成

#### Step 8. 历史记录

当前已有：
- `n5_xray_config_history`

建议在 Phase 2.4 真正落地时用于记录：
- 合并后的最终 N5 片段
- 配置 hash
- 应用结果
- 应用错误

这样可以审计：
- 某次策略修改是否真正进入最终 Xray 配置
- 入站 tag 变化是否导致 routing 变化

## 审计结论

可以进入 Phase 2.4，但前提不是“直接接入”，而是“先消除接入歧义”。

当前最关键的三个阻断点：
1. Xray Core 基线不一致
2. balancer selector 前缀碰撞
3. 主配置合并顺序尚未冻结

如果这三点不先收敛，Phase 2.4 即使能拼出 JSON，也不能保证运行语义正确。
