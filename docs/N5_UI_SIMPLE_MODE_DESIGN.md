# N5-UI Simple Mode Design

Date: 2026-08-10

## 1. 目标

`N5 Simple Mode` 的目标不是替代现有高级架构，而是在保留现有 `Egress / Pool / Traffic Policy / Merge` 体系的前提下，增加一个普通用户可直接使用的轻量入口。

设计原则：

- 保留现有高级模式，不下线、不重构
- Simple Mode 只做“更简单的管理入口”，不改变底层运行模型
- 普通用户不需要理解：
  - `outboundTag`
  - `policy`
  - `rule`
  - `binding`
- 底层仍映射到现有：
  - `n5_egresses`
  - `n5_traffic_policies`
  - `n5_traffic_policy_rules`
  - `n5_traffic_policy_bindings`

目标用户体验：

`添加出口`  
`->` `测试出口`  
`->` `选择入口`  
`->` `选择流量`  
`->` `选择出口`  
`->` `完成`

## 2. Simple Mode 页面结构

Simple Mode 建议采用“单入口、多步骤”的页面结构。

推荐页面：

- `/n5/simple`
- `/n5/simple/egress`
- `/n5/simple/rules`

其中：

### 2.1 Simple 首页

展示三个卡片区域：

- 我的出口
- 我的简单规则
- 快速向导

首页目标：

- 让用户先看到已存在出口
- 让用户先看到是否已有简单规则
- 给出明显的“开始配置”按钮

### 2.2 出口管理页

面向普通用户，只保留最少输入：

- 名称
- 协议
- 地址
- 端口
- 用户名
- 密码
- 标签
- 测试按钮

不直接展示：

- 原始 `outbound JSON`
- 底层 `tag`
- 复杂配置片段

### 2.3 简单规则页

采用接近 `x-ui` 入站列表的表单风格：

- 选择入口
- 选择流量类型
- 选择出口
- 保存

用户只需要看到“业务语义”，而不是 `policy` / `rule` / `binding`。

## 3. 导航调整

保持当前高级导航不变，但增加一层入口分组。

推荐导航：

- `N5增强`
- `Simple Mode`
- `高级模式`

Simple Mode 下：

- `快速分流`
- `简单出口`

高级模式下保留：

- `出口线路池`
- `流量分流`
- `N5运行状态`
- `配置历史`

导航原则：

- 新用户先看到 `Simple Mode`
- 老用户仍可进入高级模式
- 不隐藏高级能力
- 不把高级概念直接暴露给普通用户

## 4. 出口列表设计

Simple Mode 出口列表应比当前 `N5 Egress` 页面更轻。

推荐字段：

- 名称
- 协议
- 地址
- 端口
- 标签
- 状态
- 出口 IP
- 最后测试时间

推荐操作：

- 添加
- 测试
- 编辑
- 删除
- 用于分流

不展示字段：

- 原始 `outboundJson`
- 内部 `tag`
- 历史 hash
- 高级错误细节全文

详情查看可以保留“展开”或“更多”，但默认折叠。

## 5. 简单出口添加流程

Simple Mode 中新增出口应采用结构化表单，不让用户手写 JSON。

### 5.1 用户输入

按协议拆分：

#### SOCKS5

- 名称
- 地址
- 端口
- 用户名
- 密码

#### Shadowsocks

- 名称
- 地址
- 端口
- 加密方式
- 密码

后续协议可以继续扩展，但首版建议只覆盖：

- `socks`
- `shadowsocks`

### 5.2 表单行为

用户点击保存时：

- 前端提交结构化字段
- Simple Service 负责转换为标准 `outbound JSON`
- 内部调用现有 `EgressService.Create`

### 5.3 保存后动作

保存成功后：

- 自动生成底层 egress
- 自动显示测试入口
- 建议引导用户立即测试

## 6. 出口规则设计

Simple Mode 不让用户直接编辑通用 rule 列表，而是采用“业务模板 + 目标出口”的方式。

推荐规则模型：

- 规则名称
- 绑定入口
- 流量类型
- 目标出口
- 启用状态

### 6.1 流量类型

Simple Mode 只暴露可理解的选项：

- AI
- 游戏
- 流媒体
- 自定义域名

其中：

#### AI

内部映射到现有 AI 模板规则集。

#### 游戏

内部映射到现有游戏模板规则集。

#### 流媒体

内部映射到现有流媒体模板规则集。

#### 自定义域名

允许输入少量域名：

- `api.ipify.org`
- `openai.com`
- `example.com`

Simple Mode 首版建议只支持：

- 精确域名
- 后缀域名

不开放：

- IP CIDR
- 多级复杂规则组合
- 手工排序

### 6.2 默认流量语义

Simple Mode 里统一使用：

- `未匹配流量保持原出口`

即：

- 不强制用户指定默认出口
- 不显式要求用户理解 `freedom`
- 默认不污染原入口未命中的流量

## 7. Simple API 设计

Simple Mode 应新增独立 API，不直接复用高级 API 的输入结构。

推荐前缀：

- `/n5/api/simple/*`

建议 API：

### 7.1 出口

- `GET /n5/api/simple/egress/list`
- `POST /n5/api/simple/egress/add`
- `POST /n5/api/simple/egress/update/:id`
- `POST /n5/api/simple/egress/del/:id`
- `POST /n5/api/simple/egress/test`

### 7.2 简单规则

- `GET /n5/api/simple/rule/list`
- `GET /n5/api/simple/rule/get/:id`
- `POST /n5/api/simple/rule/add`
- `POST /n5/api/simple/rule/update/:id`
- `POST /n5/api/simple/rule/del/:id`
- `POST /n5/api/simple/rule/enable/:id`
- `POST /n5/api/simple/rule/disable/:id`

### 7.3 模板辅助

- `GET /n5/api/simple/template/list`
- `GET /n5/api/simple/template/preview/:name`

API 输出原则：

- 输出用户语义字段
- 隐藏底层 `policyId` / `bindingId` / `ruleId` 组合细节
- 调试模式下可增加 `debug` 字段，但默认不返回

## 8. Simple Service 设计

Simple Mode 不应侵入旧 service，而应新增独立 service 层。

推荐目录：

- `web/service/n5/simple/`

建议服务：

### 8.1 SimpleEgressService

职责：

- 结构化字段转标准 outbound JSON
- 调用现有 `EgressService`
- 调用现有 `EgressTestService`
- 返回简化后的视图模型

### 8.2 SimpleRuleService

职责：

- 把“入口 + 流量类型 + 出口”的简单配置
- 映射到现有 `TrafficPolicy + Rule + Binding`

### 8.3 SimpleTemplateService

职责：

- 复用现有模板定义
- 提供 Simple Mode 需要的简化模板描述

### 8.4 SimpleViewService

职责：

- 聚合出口列表
- 聚合规则列表
- 聚合入口候选
- 输出 Simple Mode 页面所需视图模型

## 9. 与现有 Egress / TrafficPolicy 映射关系

Simple Mode 不新增新的运行核心，只做“视图层 + 映射层”。

### 9.1 出口映射

Simple 出口
`->`
`EgressService.Create`
`->`
`n5_egresses`

### 9.2 简单规则映射

Simple 规则
`->`
创建或更新一条 `TrafficPolicy`
`->`
创建对应 `TrafficPolicyRule`
`->`
创建对应 `TrafficPolicyBinding`

### 9.3 模板规则映射

AI / 游戏 / 流媒体
`->`
复用现有模板定义
`->`
生成标准 `TrafficPolicyRule`

### 9.4 绑定策略

Simple Mode 要保证：

- 一个入口在 Simple Mode 下只对应一个“简单策略”
- 如果入口已存在高级策略，需要给出冲突提示

推荐策略：

- 默认阻止覆盖高级策略
- 用户明确确认后才允许替换

## 10. 页面交互建议

推荐使用“分步向导”而不是“复杂表格先行”。

步骤建议：

1. 选择或添加出口
2. 测试出口
3. 选择入口
4. 选择流量类型
5. 选择目标出口
6. 保存并启用

每一步都应尽量只出现一个核心动作。

不建议：

- 首屏展示过多底层字段
- 首屏同时出现 JSON 编辑器
- 首屏要求用户理解 `defaultTargetType`

## 11. 冲突处理设计

Simple Mode 和高级模式会共享同一套底层数据，因此必须有冲突识别。

建议识别以下情况：

- 入口已绑定其他高级策略
- Simple 规则对应的底层 policy 被高级模式手工修改过
- 出口被线路池使用且正在被多个策略引用

处理建议：

- 在 Simple 页面显示“已被高级模式接管”
- 对被高级修改的对象转为只读或半只读
- 不自动覆盖未知状态

## 12. 数据模型策略

Simple Mode 首版建议尽量不新增运行核心表。

可选方案：

### 12.1 最保守方案

不新增表。

通过约定命名或 remark 标记识别 Simple 创建对象。

优点：

- 实现最轻
- 无迁移成本

风险：

- 元数据表达弱
- 高级模式修改后难以识别来源

### 12.2 推荐方案

新增轻量映射表，例如：

- `n5_simple_rules`
- `n5_simple_rule_targets`

只记录 Simple Mode 的展示语义和与底层对象的映射关系。

运行时仍由既有 `egress / policy / rule / binding` 表生效。

优点：

- 可以清楚区分 Simple 与高级对象
- 后续页面管理更稳定

风险：

- 需要新增轻量迁移

首版推荐：

- 如果要快速上线，先做最保守方案
- 如果准备长期维护，优先做轻量映射表方案

## 13. 不修改部分

Simple Mode 设计明确不修改以下内容：

- `xray.go`
- `xray merge`
- 数据库核心旧表结构
- `x-ui` 兼容层
- 安装脚本
- systemd service
- `/xui/` 原有 API

Simple Mode 只能：

- 新增页面
- 新增 controller
- 新增 service
- 新增轻量映射层
- 复用现有高级能力

## 14. 分阶段开发建议

### Phase S1

- 新增 Simple 导航
- 新增 Simple 出口页面
- 支持结构化添加 SOCKS / SS 出口
- 支持出口测试

### Phase S2

- 新增 Simple 规则页面
- 支持入口选择
- 支持 AI / 游戏 / 流媒体 / 自定义域名
- 支持绑定出口

### Phase S3

- 增加冲突检测
- 增加 Simple / 高级模式切换提示
- 增加对象来源标识

### Phase S4

- 如有必要，再补轻量映射表
- 优化 Simple 详情和批量操作

## 15. 结论

`N5 Simple Mode` 的本质不是新增第二套运行逻辑，而是对现有高级能力做“简单视图封装”。

最终应做到：

- 新手能快速完成“出口 -> 测试 -> 入口 -> 流量 -> 目标出口”
- 老手仍可进入高级模式精细控制
- 运行底层继续复用现有 `Egress / TrafficPolicy / Binding / Merge`
- 不破坏当前 `x-ui` 兼容层与 N5 高级架构
