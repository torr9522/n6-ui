# N5-UI Phase 3.5 Design

Date: 2026-08-10

## 1. 目标

Phase 3.5 的目标不是增加复杂调度，而是在现有 Phase 3 轻量出口检测基础上补齐“管理增强”。

本阶段只设计以下能力：

- 出口线路标签系统
- 出口详情页
- AI / 游戏 / 流媒体分流模板
- 一键创建策略
- 后续用户策略绑定预留

必须保持：

- 低资源消耗
- 不增加常驻后台任务
- 不修改 `x-ui` 核心逻辑
- 不改安装脚本、systemd、旧 `/xui/` API

实现边界：

- 所有新增能力继续放在 `N5` 扩展层
- 所有新增 API 继续放在 `/n5/api/*`
- 所有模板能力优先采用静态内置定义，不引入模板执行引擎

## 2. 设计原则

### 2.1 轻量优先

- 不引入 observatory
- 不引入周期扫描 worker
- 不引入高频统计任务
- 不引入独立缓存服务

### 2.2 展示与运行分离

- 标签、模板、详情页属于管理增强层
- 出口测试结果、线路池过滤、策略绑定属于运行扩展层
- 管理增强层不得反向侵入 `xray.go` 核心生成流程

### 2.3 兼容冻结

以下区域仍视为冻结区：

- `x-ui.sh`
- `install.sh`
- `x-ui.service`
- `xray/` 核心生成逻辑
- 旧 `controller`
- 旧 `/xui/` API

## 3. 出口线路标签系统

### 3.1 目标

为出口线路增加低成本分类能力，用于：

- 按地区筛选
- 按用途筛选
- 按稳定性标记
- 为模板和一键创建策略提供候选出口选择

### 3.2 标签范围

建议标签只做“弱约束分类”，不参与底层运行逻辑。

推荐预设分类：

- 地区类：`hk` `sg` `jp` `us`
- 用途类：`ai` `game` `stream`
- 属性类：`stable` `cheap` `ipv4` `ipv6`

标签系统不直接影响 Xray 配置生成。

运行期仍由：

- 出口启用状态
- 最近测试状态
- 线路池成员关系
- 流量策略绑定

共同决定最终生效结果。

### 3.3 设计选择

采用多对多关系，不在 `n5_egresses` 中堆叠逗号字符串字段。

原因：

- 查询稳定
- 后续可扩展筛选
- 不污染现有出口模型语义

## 4. 出口详情页

### 4.1 目标

提供单出口的聚合视图，减少运维操作的跳转成本。

详情页重点展示：

- 基本信息
- 最近测试状态
- 最近出口 IP
- 最近测试错误
- 所属标签
- 所在线路池
- 被哪些流量策略引用
- 原始 outbound JSON

### 4.2 数据来源

详情页优先复用现有数据，不新增重型统计表。

直接查询：

- `n5_egresses`
- `n5_egress_test`
- `n5_egress_pool_members`
- `n5_traffic_policy_rules`
- `n5_traffic_policies`

### 4.3 展示原则

- 只显示最近 N 条测试记录
- 不做图表引擎
- 不做实时推送
- 不做自动刷新

推荐默认：

- 最近 10 条测试记录
- 手动刷新按钮
- 手动测试按钮

## 5. AI / 游戏 / 流媒体分流模板

### 5.1 目标

降低策略创建成本，让用户无需手写规则即可快速建立常见分流方案。

### 5.2 模板形态

模板不建议落库为动态 DSL。

Phase 3.5 推荐采用：

- 后端内置静态模板定义
- 控制器提供模板列表与预览
- 用户点击后一键生成标准 `policy + rules`

原因：

- 资源消耗最低
- 不需要模板解释器
- 便于版本控制
- 便于测试

### 5.3 模板内容

#### AI 模板

目标域名建议包含：

- `openai.com`
- `chatgpt.com`
- `claude.ai`
- `anthropic.com`
- `oaistatic.com`

默认策略：

- 模板命中流量走指定出口或线路池
- 未命中流量保持原默认出口

#### 游戏模板

模板设计原则：

- 先只做“基础框架”
- 不预置过大域名库
- 采用用户可见、可编辑的小型规则集合

建议首版只提供：

- `steamcontent.com`
- `steampowered.com`
- `steamstatic.com`
- `riotgames.com`

#### 流媒体模板

建议首版包含：

- `netflix.com`
- `nflxvideo.net`
- `youtube.com`
- `googlevideo.com`
- `disneyplus.com`

### 5.4 不采用方案

不采用以下方案：

- 在线更新模板库
- 第三方模板订阅
- 自动同步规则集
- geosite 级大型模板管理界面

原因：

- 增加维护成本
- 增加后台任务需求
- 破坏 Phase 3 的轻量目标

## 6. 一键创建策略

### 6.1 目标

让用户从“选择模板 + 选择出口/线路池 + 选择绑定入站”直接生成可用策略。

### 6.2 输入

一键创建最小输入建议为：

- 模板类型
- 目标类型
- 目标对象
- 绑定入站
- 策略名称

其中目标类型为：

- `egress`
- `pool`

### 6.3 输出

一次操作生成：

- `n5_traffic_policies` 一条
- `n5_traffic_policy_rules` 多条
- `n5_traffic_policy_bindings` 一条

### 6.4 交互原则

- 生成前展示规则预览
- 生成后允许继续手工编辑
- 不锁定为只读模板策略

模板只负责初始生成，不负责持续托管。

## 7. 后续用户策略绑定预留

### 7.1 背景

当前 Phase 2/3 的策略绑定核心是：

- `Inbound -> Policy`

后续如果要支持更细粒度绑定，可能出现：

- `User -> Policy`
- `Client -> Policy`
- `Email/UUID -> Policy`

### 7.2 预留方式

不建议直接修改现有 `n5_traffic_policy_bindings` 语义。

建议新增独立保留结构：

- `n5_traffic_policy_subjects`

字段建议：

- `id`
- `subject_type`
- `subject_id`
- `subject_key`
- `policy_id`
- `enabled`
- `created_at`
- `updated_at`

说明：

- `subject_type` 预留 `inbound` `client` `user`
- `subject_id` 用于内部主键映射
- `subject_key` 用于未来 email、uuid、client id 等非数字绑定对象

### 7.3 当前阶段原则

Phase 3.5 只做预留设计，不接入运行逻辑。

原因：

- 当前 Xray merge 仍以 `inboundTag` 绑定最稳定
- 用户级绑定需要重新审计旧入站与客户端模型
- 不应在本阶段扩大运行复杂度

## 8. 数据库变化

Phase 3.5 建议数据库变化如下。

### 8.1 新增表

- `n5_egress_labels`
- `n5_egress_label_relations`

建议结构：

`n5_egress_labels`

- `id`
- `name`
- `code`
- `color`
- `remark`
- `sort_order`
- `enabled`
- `created_at`
- `updated_at`

约束建议：

- `code` 唯一
- `name` 普通索引

`n5_egress_label_relations`

- `id`
- `egress_id`
- `label_id`
- `created_at`

约束建议：

- `egress_id + label_id` 唯一

### 8.2 预留表

后续用户绑定建议预留但不急于实现：

- `n5_traffic_policy_subjects`

### 8.3 不建议新增的表

Phase 3.5 不建议新增：

- 模板表
- 模板版本表
- 模板发布表
- 周期任务表

原因：

- 首版模板完全可以内置
- 没必要把静态预设变成动态系统

### 8.4 不建议修改的表

不建议改动：

- 旧 `x-ui` 表
- 现有 `inbounds`
- 现有 `clients`
- 现有 `settings`

可以接受的小范围 N5 自表扩展：

- 如后续需要，可在 `n5_traffic_policies` 增加 `template_key`
- 但首版可不加，避免无效字段

## 9. 页面设计

### 9.1 出口线路列表页增强

页面：

- `/n5/egress`

新增区域：

- 标签筛选栏
- 快速过滤按钮
- 一键测试入口保留原位

新增列：

- 标签
- 最近状态
- 最近出口 IP
- 最后测试时间

新增操作：

- 测试
- 查看详情
- 打标签

### 9.2 出口详情页

页面建议：

- `/n5/egress-detail?id=:id`

模块布局建议：

- 基本信息卡片
- 最近测试状态卡片
- 标签卡片
- 所在线路池卡片
- 策略引用卡片
- 最近测试记录卡片
- 原始 outbound JSON 卡片

### 9.3 流量分流页面增强

页面继续使用：

- `/n5/traffic-policy`

新增区域：

- 模板卡片区
- 一键创建策略弹窗
- 模板规则预览弹窗

原因：

- 不新增一级导航
- 避免模板页与策略页割裂
- 保持用户操作路径最短

### 9.4 用户绑定预留页面

Phase 3.5 不建议立即开放独立页面。

建议先只在文档和 API 层预留，等后续运行链路明确后再新增：

- `/n5/policy-subjects`

## 10. API 设计

所有 API 继续放在 `/n5/api/*` 下。

### 10.1 标签系统 API

- `POST /n5/api/egress-label/list`
- `POST /n5/api/egress-label/add`
- `POST /n5/api/egress-label/update/:id`
- `POST /n5/api/egress-label/del/:id`
- `POST /n5/api/egress-label/bind`
- `POST /n5/api/egress-label/unbind`

说明：

- `bind` 输入：`egressId` `labelId`
- `unbind` 输入：`egressId` `labelId`

### 10.2 出口详情 API

- `POST /n5/api/egress/detail/:id`

返回建议包含：

- `egress`
- `labels`
- `latestTest`
- `recentTests`
- `pools`
- `policies`

### 10.3 模板 API

- `POST /n5/api/traffic-template/list`
- `POST /n5/api/traffic-template/preview`
- `POST /n5/api/traffic-template/create`

输入建议：

`preview`

- `templateKey`
- `targetType`
- `targetId`

`create`

- `templateKey`
- `policyName`
- `targetType`
- `targetId`
- `bindInboundId`

返回建议：

- `policy`
- `rules`
- `binding`

### 10.4 预留 API

只做后续规划，不建议在 Phase 3.5 实现：

- `POST /n5/api/traffic-policy-subject/list`
- `POST /n5/api/traffic-policy-subject/bind`
- `POST /n5/api/traffic-policy-subject/unbind`

## 11. 开发顺序

推荐开发顺序如下。

### Phase 3.5.1 标签系统

- 新增标签表与关联表
- 标签 CRUD
- 出口列表筛选
- 出口线路绑定标签

原因：

- 风险最低
- 与运行链路耦合最弱
- 可先提升管理体验

### Phase 3.5.2 出口详情页

- 聚合出口基础信息
- 聚合测试记录
- 聚合线路池引用
- 聚合策略引用

原因：

- 主要是读接口和页面组织
- 不需要改 Xray 逻辑

### Phase 3.5.3 模板列表与预览

- 实现内置模板定义
- 实现模板 list / preview API
- 在流量分流页增加模板卡片

原因：

- 先让用户看到模板内容
- 便于后续确认规则集

### Phase 3.5.4 一键创建策略

- 基于模板生成 `policy + rules + binding`
- 保留生成后可编辑能力

原因：

- 写操作风险高于展示和预览
- 应放在模板内容稳定后进行

### Phase 3.5.5 用户绑定预留

- 只补文档、模型预留、接口草案
- 不接入主运行链路

原因：

- 当前阶段最容易扩大边界
- 应单独进入下一阶段审计

## 12. 风险与控制

主要风险：

- 标签系统如果直接影响运行逻辑，会造成语义混乱
- 模板规则如果预置过大，会迅速膨胀为规则库系统
- 用户绑定如果提前接入，会冲击当前 `Inbound -> Policy` 稳定模型

控制原则：

- 标签只做管理分类，不做调度依据
- 模板只做静态预设，不做在线更新
- 用户绑定只预留，不在本阶段接入
- 所有新增能力只走 `N5` 扩展层

## 13. 结论

Phase 3.5 应定位为“轻量管理增强”，而不是“新调度阶段”。

最小可行落地顺序应是：

1. 标签系统
2. 出口详情页
3. 模板预览
4. 一键创建策略
5. 用户绑定预留

这样可以在不增加后台负担、不修改 `x-ui` 核心的前提下，明显提升 N5-UI 的可管理性和后续扩展空间。
