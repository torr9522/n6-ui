# N5-UI Changelog v0.1.0-beta-simple

更新时间：2026-08-10  
整理说明：本文档按当前仓库状态与开发阶段整理，为发布归档清单，不等同于逐 commit 历史。

## Phase 1: 项目迁移

### 功能

- 以 `n3-ui` 为基础建立独立品牌项目 `N5-UI`。
- 保持运行兼容层继续使用 `x-ui` 生态：
  - 服务名不变
  - systemd 名称不变
  - 安装命令不变
  - 运行路径不变
  - 数据库结构不改旧表
  - 原 API 不变

### 主要文件

新增文件：

- `N5_UI_PHASE1_BRANDING_REPORT.md`

修改文件：

- `README.md`
- `README_EN.md`
- `web/html/xui/*` 中项目展示文案
- `web/assets/js/model/models.js`

### 测试

- 品牌替换静态检查
- 运行兼容边界审查

## Phase 2: 出口线路池 + 分流

### 功能

- 建立 N5 独立数据模型与业务层。
- 完成出口、线路池、分流策略、规则、绑定、高级页面和 API。
- 建立 N5 Xray merge 扩展链路，但不重写原 x-ui 生成流程。

### 涉及文件族

- `database/model/n5/models.go`
- `database/n5_phase2.go`
- `web/controller/n5/*.go`
- `web/service/n5/*.go`
- `web/html/n5/*.html`
- `web/web.go`
- `web/html/xui/common_sider.html`
- `web/service/xray.go`
- `web/entity/entity.go`
- `web/service/setting.go`
- `web/html/xui/setting.html`

### 测试

- `go test ./...`
- N5 service / controller / merge 单元测试
- 真实环境联调与回滚验证

## Phase 2.1: 数据库

### 新增文件

- `database/model/n5/models.go`
- `database/n5_phase2.go`
- `database/n5_phase2_test.go`

### 功能

- 新增 `n5_egresses`
- 新增 `n5_egress_pools`
- 新增 `n5_egress_pool_members`
- 新增 `n5_traffic_policies`
- 新增 `n5_traffic_policy_rules`
- 新增 `n5_traffic_policy_bindings`
- 新增 `n5_xray_config_history`

### 后续追加到同一模型文件的扩展

- `n5_egress_test`
- `n5_egress_labels`
- `n5_egress_label_relations`
- egress 测试状态字段
- binding 入站唯一索引迁移
- 稳定 tag 迁移

### 测试

- `database/n5_phase2_test.go`
- N5 service 测试启动时自动迁移验证

## Phase 2.2: Service

### 新增文件

- `web/service/n5/common.go`
- `web/service/n5/egress.go`
- `web/service/n5/egress_pool.go`
- `web/service/n5/traffic_policy.go`
- `web/service/n5/xray_ext.go`
- `web/service/n5/xray_history.go`
- `web/service/n5/xray_status.go`
- `web/service/n5/xray_merge.go`
- `web/service/n5/traffic_template.go`
- `web/service/n5/templates/base.go`
- `web/service/n5/templates/ai.go`
- `web/service/n5/templates/game.go`
- `web/service/n5/templates/streaming.go`
- `web/service/n5/egress_probe.go`
- `web/service/n5/egress_detail.go`
- `web/service/n5/egress_label.go`
- `web/service/n5/traffic_policy_detail.go`

### 功能

- 出口 CRUD 与配置校验
- 稳定 tag 生成
- 线路池与成员管理
- 分流策略、规则、绑定管理
- 模板驱动策略创建
- outbound/routing 片段生成
- merge 后 history 记录
- 状态接口数据聚合
- 出口测试与出口详情聚合
- 标签系统

### 测试文件

- `web/service/n5/n5_test.go`
- `web/service/n5/traffic_policy_manage_test.go`
- `web/service/n5/traffic_template_test.go`
- `web/service/n5/xray_capability_test.go`
- `web/service/n5/xray_history_test.go`
- `web/service/n5/xray_merge_test.go`
- `web/service/n5/xray_status_test.go`

## Phase 2.3: API / UI

### 新增文件

- `web/controller/n5/egress.go`
- `web/controller/n5/pool.go`
- `web/controller/n5/traffic.go`
- `web/controller/n5/traffic_template.go`
- `web/controller/n5/xray.go`
- `web/controller/n5/egress_label.go`
- `web/html/n5/egress.html`
- `web/html/n5/pools.html`
- `web/html/n5/traffic_policy.html`
- `web/html/n5/xray_status.html`
- `web/html/n5/config_history.html`
- `web/html/n5/egress_test.html`
- `web/html/n5/egress_detail.html`
- `web/html/n5/traffic_policy_detail.html`

修改文件：

- `web/web.go`
- `web/html/xui/common_sider.html`

### 功能

- N5 页面入口
- N5 API 路由
- 左侧导航菜单
- 状态页、历史页、详情页

### 测试

- `web/controller/n5/n5_test.go`
- 页面渲染路由检查

## Phase 2.4: Xray Merge

### 修改文件

- `web/service/xray.go`
- `web/service/setting.go`
- `web/entity/entity.go`
- `web/html/xui/setting.html`
- `database/n5_phase2.go`

### 功能

- 在原 x-ui base config 生成完成后追加 `N5 merge` 调用点
- 增加 `n5XrayExtensionEnable` 开关
- merge 历史记录与应用状态更新
- N5 merge 统计日志：
  - outbound 数量
  - routing 数量
  - binding 数量

### 测试

- `web/service/xray_n5_test.go`
- `web/service/n5/xray_merge_test.go`
- `web/service/n5/xray_capability_test.go`
- 实机 `xray -test`

## Phase 3: 轻量检测

### 新增/修改文件

- `database/model/n5/models.go`
- `web/service/n5/egress_probe.go`
- `web/controller/n5/egress.go`
- `web/html/n5/egress.html`
- `web/html/n5/egress_test.html`
- `web/service/n5/egress.go`
- `web/service/n5/egress_pool.go`

### 功能

- 手动测试出口
- 测试 TCP 连通性与 HTTP 出口 IP
- 保存 `n5_egress_test`
- 出口主表增加最近测试状态、出口 IP、测试时间
- 线路池 selector 只纳入测试成功出口

### 测试

- N5 egress probe 测试
- 真实 SOCKS5 / 域名出口测试
- 分流后指定流量走出口、普通流量保持本机出口

## Phase 3.5: 管理增强

### Phase 3.5-A: 标签系统 + 出口详情

新增文件：

- `web/service/n5/egress_label.go`
- `web/service/n5/egress_detail.go`
- `web/controller/n5/egress_label.go`
- `web/html/n5/egress_detail.html`

修改文件：

- `database/model/n5/models.go`
- `database/n5_phase2.go`
- `web/controller/n5/egress.go`
- `web/html/n5/egress.html`

功能：

- 多标签系统
- 出口详情聚合页
- 查看所属线路池与关联策略

测试：

- 标签 CRUD / bind / unbind
- 详情页 API 与页面渲染

### Phase 3.5-B: 模板系统

新增文件：

- `web/service/n5/templates/base.go`
- `web/service/n5/templates/ai.go`
- `web/service/n5/templates/game.go`
- `web/service/n5/templates/streaming.go`
- `web/service/n5/traffic_template.go`
- `web/controller/n5/traffic_template.go`

修改文件：

- `web/html/n5/traffic_policy.html`

功能：

- AI / 游戏 / 流媒体模板
- 模板预览
- 模板一键创建策略与规则

测试：

- 模板列表
- 预览
- 创建 policy / rule / binding

### Phase 3.5-C: Traffic Policy 管理增强

新增文件：

- `web/service/n5/traffic_policy_detail.go`
- `web/html/n5/traffic_policy_detail.html`

修改文件：

- `web/service/n5/traffic_policy.go`
- `web/controller/n5/traffic.go`
- `web/html/n5/traffic_policy.html`

功能：

- 策略详情页
- 规则编辑、启停、重排
- 策略启停、删除、解绑、重绑
- 模板策略与手工策略统一编辑

测试：

- Service 管理测试
- Controller API 测试
- XrayExt 对 disable policy / disable rule 的行为测试

## Phase 3.5: Simple Mode

### Simple-1: Simple 出口

新增文件：

- `web/service/n5/simple/egress.go`
- `web/controller/n5/simple/egress.go`
- `web/service/n5/simple/egress_test.go`
- `web/controller/n5/simple/egress_test.go`
- `web/html/n5/simple.html`

修改文件：

- `web/web.go`
- `web/html/xui/common_sider.html`

功能：

- Simple 出口列表
- 创建、测试、删除
- 复用高级 EgressService / EgressTestService

### Simple-2: Simple 规则

新增文件：

- `web/service/n5/simple/rule.go`
- `web/controller/n5/simple/rule.go`
- `web/service/n5/simple/rule_test.go`
- `web/controller/n5/simple/rule_test.go`
- `web/html/n5/simple_rules.html`

功能：

- 用“入口 + 流量类型 + 出口”创建高级策略
- 支持 `all / ai / game / streaming / custom-domain`

### Simple-3-A: 协议收敛

修改文件：

- `web/service/n5/simple/egress.go`
- `web/html/n5/simple.html`

功能：

- Simple 模式只支持 `SOCKS5` / `SS`
- 自动翻译为 Xray socks / shadowsocks outbound

### Simple-3-B: Simple 编辑

新增文件：

- `web/html/n5/simple_egress_edit.html`

修改文件：

- `web/controller/n5/simple/egress.go`
- `web/service/n5/simple/egress.go`
- `web/html/n5/simple.html`

功能：

- Simple 出口编辑
- 保持内部 tag 不变

### 测试

- `go test ./...`
- Simple controller / service 测试
- 页面路由验证

## Certificate Phase A

### 修改文件

- `x-ui.sh`

### 功能

- 新增 `require_sqlite()`
- 新增 `get_panel_setting()`
- 新增 `cert_key_match()`
- 新增 `show_panel_https_status()`
- 新增 `disable_panel_https()`
- 新增 `repair_panel_https_certificate()`
- `save_panel_cert_setting()` 增加 cert/key 匹配校验
- `cert_manage()` 菜单从 `0-5` 扩到 `0-8`

### 测试

- `bash -n x-ui.sh`
- 临时证书 + 临时 sqlite 脚本测试
- 正常证书设置
- 错误 cert/key 组合
- HTTPS 状态查看
- 关闭 HTTPS
- 修复 HTTPS

## 当前版本总结

`v0.1.0-beta-simple` 当前可交付能力：

- N5 品牌迁移完成
- 高级出口管理
- 线路池
- 分流策略、规则、绑定
- 模板系统
- N5 Xray merge
- 运行状态与配置历史
- 轻量出口测试
- 标签系统
- 出口详情页
- Simple 出口与 Simple 规则
- HTTPS Rescue Phase A
