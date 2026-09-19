# N5-UI API Reference

更新时间：2026-08-10  
适用版本：`v0.1.0-beta-simple`

说明：

- 所有 `/n5/api/*` 接口都要求已登录。
- 返回结构沿用项目统一 `entity.Msg` 形式：
  - `success`
  - `msg`
  - `obj`

## 1. Egress API

### `POST /n5/api/egress/list`

- 参数：无
- 返回：`[]Egress`
- 用途：获取高级出口列表

### `POST /n5/api/egress/get/:id`

- 参数：
  - 路径：`id`
- 返回：`Egress`
- 用途：获取单个出口原始记录

### `GET /n5/api/egress/detail/:id`

- 参数：
  - 路径：`id`
- 返回：`EgressDetail`
- 用途：获取出口详情页聚合数据

### `POST /n5/api/egress/add`

- 参数：
  - `name`
  - `remark`
  - `protocol`
  - `enabled`
  - `outboundJson`
- 返回：创建后的 `Egress`
- 用途：新增高级出口

### `POST /n5/api/egress/update/:id`

- 参数：
  - 路径：`id`
  - Body 同新增
- 返回：更新后的 `Egress`
- 用途：更新高级出口

### `POST /n5/api/egress/del/:id`

- 参数：
  - 路径：`id`
- 返回：成功/失败消息
- 用途：删除高级出口

### `POST /n5/api/egress/validate`

- 参数：
  - `protocol`
  - `outboundJson`
  - `tag`
- 返回：
  - `protocol`
  - `outboundJson`
- 用途：校验和归一化 outbound JSON

### `POST /n5/api/egress/test`

- 参数：
  - `id`
- 返回：`EgressTest`
- 用途：手动测试出口连通性和出口 IP

## 2. Egress Label API

### `GET /n5/api/egress-label/list`

- 参数：无
- 返回：`[]EgressLabel`
- 用途：获取标签列表

### `POST /n5/api/egress-label/add`

- 参数：
  - `name`
  - `type`
- 返回：创建后的 `EgressLabel`
- 用途：新增标签

### `POST /n5/api/egress-label/update/:id`

- 参数：
  - 路径：`id`
  - `name`
  - `type`
- 返回：更新后的 `EgressLabel`
- 用途：更新标签

### `POST /n5/api/egress-label/del/:id`

- 参数：
  - 路径：`id`
- 返回：成功/失败消息
- 用途：删除标签并清理关系

### `POST /n5/api/egress-label/bind`

- 参数：
  - `egressId`
  - `labelId`
- 返回：`EgressLabelRelation`
- 用途：给出口绑定标签

### `POST /n5/api/egress-label/unbind`

- 参数：
  - `egressId`
  - `labelId`
- 返回：成功/失败消息
- 用途：解除出口与标签关系

## 3. Pool API

### `POST /n5/api/pool/list`

- 参数：无
- 返回：`[]EgressPool`
- 用途：获取线路池列表

### `POST /n5/api/pool/get/:id`

- 参数：
  - 路径：`id`
- 返回：`EgressPool`
- 用途：获取单个线路池

### `POST /n5/api/pool/add`

- 参数：
  - `name`
  - `remark`
  - `strategy`
  - `fallbackType`
  - `fallbackTargetId`
  - `enabled`
- 返回：创建后的 `EgressPool`
- 用途：新增线路池

### `POST /n5/api/pool/member/list/:id`

- 参数：
  - 路径：`id`，即 `poolId`
- 返回：`[]EgressPoolMember`
- 用途：获取线路池成员

### `POST /n5/api/pool/member/add`

- 参数：
  - `poolId`
  - `egressId`
  - `weight`
  - `sortOrder`
- 返回：`EgressPoolMember`
- 用途：向线路池添加成员

### `POST /n5/api/pool/member/del`

- 参数：
  - `poolId`
  - `egressId`
- 返回：成功/失败消息
- 用途：移除线路池成员

## 4. Traffic Policy API

### `POST /n5/api/traffic-policy/list`

- 参数：无
- 返回：`[]TrafficPolicy`
- 用途：获取策略列表

### `POST /n5/api/traffic-policy/add`

- 参数：
  - `name`
  - `remark`
  - `enabled`
  - `defaultTargetType`
  - `defaultTargetId`
- 返回：创建后的 `TrafficPolicy`
- 用途：新增分流策略

### `GET /n5/api/traffic-policy/get/:id`

- 参数：
  - 路径：`id`
- 返回：`TrafficPolicyDetail`
- 用途：获取策略详情页数据

### `POST /n5/api/traffic-policy/update/:id`

- 参数：
  - 路径：`id`
  - `name`
  - `remark`
  - `enabled`
  - `defaultTargetType`
  - `defaultTargetId`
- 返回：更新后的 `TrafficPolicy`
- 用途：更新策略

### `POST /n5/api/traffic-policy/del/:id`

- 参数：
  - 路径：`id`
- 返回：成功/失败消息
- 用途：删除策略及其 N5 自有规则/绑定

### `POST /n5/api/traffic-policy/enable/:id`

- 参数：
  - 路径：`id`
- 返回：更新后的 `TrafficPolicy`
- 用途：启用策略

### `POST /n5/api/traffic-policy/disable/:id`

- 参数：
  - 路径：`id`
- 返回：更新后的 `TrafficPolicy`
- 用途：停用策略

## 5. Traffic Rule API

### `POST /n5/api/traffic-policy/rule/list/:id`

- 参数：
  - 路径：`id`，即 `policyId`
- 返回：`[]TrafficPolicyRule`
- 用途：获取策略规则列表

### `POST /n5/api/traffic-policy/rule/add`

- 参数：
  - `policyId`
  - `ruleType`
  - `matchMode`
  - `matchValue`
  - `targetType`
  - `targetId`
  - `sortOrder`
  - `enabled`
- 返回：创建后的 `TrafficPolicyRule`
- 用途：新增规则

### `POST /n5/api/traffic-policy/rule/update/:id`

- 参数：
  - 路径：`id`
  - 其余同新增
- 返回：更新后的 `TrafficPolicyRule`
- 用途：更新规则

### `POST /n5/api/traffic-policy/rule/del/:id`

- 参数：
  - 路径：`id`
- 返回：成功/失败消息
- 用途：删除规则

### `POST /n5/api/traffic-policy/rule/enable/:id`

- 参数：
  - 路径：`id`
- 返回：更新后的 `TrafficPolicyRule`
- 用途：启用规则

### `POST /n5/api/traffic-policy/rule/disable/:id`

- 参数：
  - 路径：`id`
- 返回：更新后的 `TrafficPolicyRule`
- 用途：停用规则

### `POST /n5/api/traffic-policy/rule/reorder`

- 参数：
  - `policyId`
  - `ruleIds[]`
- 返回：成功/失败消息
- 用途：按给定顺序重排规则

## 6. Traffic Binding API

### `POST /n5/api/traffic-policy/binding/list`

- 参数：无
- 返回：`[]TrafficPolicyBinding`
- 用途：获取所有入站绑定

### `POST /n5/api/traffic-policy/bind`

- 参数：
  - `inboundId`
  - `policyId`
- 返回：`TrafficPolicyBinding`
- 用途：绑定 inbound 到策略

### `POST /n5/api/traffic-policy/unbind`

- 参数：
  - `inboundId`
- 返回：成功/失败消息
- 用途：解绑 inbound

### `POST /n5/api/traffic-policy/rebind`

- 参数：
  - `inboundId`
  - `policyId`
- 返回：`TrafficPolicyBinding`
- 用途：重绑 inbound 到新策略

### `POST /n5/api/traffic-policy/fragments`

- 参数：无
- 返回：routing 片段对象
- 用途：调试查看当前策略生成的 routing 片段

## 7. Traffic Template API

### `GET /n5/api/traffic-template/list`

- 参数：无
- 返回：`[]TrafficTemplateSummary`
- 用途：获取模板列表

### `GET /n5/api/traffic-template/preview/:name`

- 参数：
  - 路径：`name`
- 返回：`TrafficTemplatePreview`
- 用途：查看模板规则

### `POST /n5/api/traffic-template/create`

- 参数：
  - `name`
  - `inboundId`
  - `targetType`
  - `targetId`
  - `policyName`
- 返回：`TrafficTemplateCreateResult`
- 用途：一键按模板创建策略、规则、绑定

## 8. Xray Status / History API

### `GET /n5/api/xray/status`

### `POST /n5/api/xray/status`

- 参数：无
- 返回：`XrayStatus`
- 用途：获取 N5 开关状态、最后应用时间、hash、计数

### `GET /n5/api/xray/history/list`

### `POST /n5/api/xray/history/list`

- 参数：
  - `limit`
- 返回：`[]XrayConfigHistory`
- 用途：获取最近 N5 merge 配置历史

### `GET /n5/api/xray/egress-test/entry`

### `POST /n5/api/xray/egress-test/entry`

- 参数：无
- 返回：
  - `supported`
  - `planned`
  - `mode`
  - `description`
  - `inputs`
  - `outputs`
- 用途：返回当前版本手动出口测试入口说明

## 9. Simple Egress API

### `GET /n5/api/simple/egress/list`

- 参数：无
- 返回：`[]SimpleEgress`
- 用途：Simple 出口列表

### `GET /n5/api/simple/egress/get/:id`

- 参数：
  - 路径：`id`
- 返回：`SimpleEgress`
- 用途：Simple 出口编辑页读取

### `POST /n5/api/simple/egress/add`

- 参数：
  - `name`
  - `protocol`：`socks5` / `ss`
  - `address`
  - `port`
  - `username`
  - `method`
  - `password`
  - `enabled`
- 返回：`SimpleEgress`
- 用途：创建 Simple 出口

### `POST /n5/api/simple/egress/update/:id`

- 参数：
  - 路径：`id`
  - 其余同新增
- 返回：`SimpleEgress`
- 用途：更新 Simple 出口

### `POST /n5/api/simple/egress/test`

- 参数：
  - `id`
- 返回：`SimpleEgressTestResult`
- 用途：测试 Simple 出口

### `POST /n5/api/simple/egress/delete`

- 参数：
  - `id`
- 返回：成功/失败消息
- 用途：删除 Simple 出口

## 10. Simple Rule API

### `GET /n5/api/simple/rule/list`

- 参数：无
- 返回：`SimpleRuleListResult`
- 用途：获取 Simple 规则列表，同时返回可选入口、出口、流量类型

### `POST /n5/api/simple/rule/add`

- 参数：
  - `inboundId`
  - `trafficType`：`all` / `ai` / `game` / `streaming` / `custom-domain`
  - `egressId`
  - `customDomain`
- 返回：`SimpleRule`
- 用途：创建 Simple 规则，本质上生成高级策略与绑定

### `POST /n5/api/simple/rule/delete`

- 参数：
  - `id`，这里的 `id` 为 policyId
- 返回：成功/失败消息
- 用途：删除 Simple 规则对应的高级策略

## 11. 页面路径清单

这些不是 API，但属于 N5 前端入口：

- `GET /n5/egress`
- `GET /n5/egress-detail`
- `GET /n5/pools`
- `GET /n5/traffic-policy`
- `GET /n5/traffic-policy-detail`
- `GET /n5/xray-status`
- `GET /n5/config-history`
- `GET /n5/egress-test`
- `GET /n5/simple`
- `GET /n5/simple/edit`
- `GET /n5/simple/rules`
