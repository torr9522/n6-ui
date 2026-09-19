# N5-UI Source Tree

更新时间：2026-09-06
适用版本：`v0.2.0` Stable
说明：本文档以公开仓库 `torr9522/n5-ui` 为准，重点描述 N5-UI 自有开发面。`xray-core/` 为 vendored 上游子树，不建议作为日常二开入口。

## v0.2.0 Handoff Map

- DB models: `database/model/n5/models.go`
- DB migration/bootstrap: `database/n5_phase2.go`
- N5 services: `web/service/n5/`
- N5 Simple services: `web/service/n5/simple/`
- N5 controllers: `web/controller/n5/`
- N5 Simple controllers: `web/controller/n5/simple/`
- N5 frontend pages: `web/html/n5/`
- Sidebar navigation: `web/html/xui/common_sider.html`
- Xray config merge: `web/service/n5/xray_merge.go`,
  `web/service/n5/xray_ext.go`, `web/service/xray.go`
- Subscription: `web/service/subscription.go`, `web/service/share_link.go`,
  `web/controller/subscription.go`
- Access IP: `web/controller/access_ip.go`, `web/service/access_ip.go`,
  `web/job/access_ip_aggregation.go`
- Installer/runtime scripts: `install.sh`, `install_en.sh`, `x-ui.sh`,
  `x-ui_en.sh`
- Release/version metadata: `config/version`, `RELEASE-v0.2.0.md`
- Tests: `*_test.go`, especially `web/service/n5/`,
  `web/service/n5/simple/`, `web/controller/n5/`, and `database/`

## 1. 项目整体结构

```text
n5-ui/
├── bin/
│   ├── config.json
│   ├── geoip.dat
│   └── geosite.dat
├── config/
│   ├── config.go
│   ├── name
│   └── version
├── database/
│   ├── db.go
│   ├── n5_phase2.go
│   ├── n5_phase2_test.go
│   ├── model/
│   │   ├── model.go
│   │   └── n5/
│   │       └── models.go
├── docs/
│   ├── BACKUP_LOG.md
│   ├── DEVELOPMENT_SKILL_TREE.md
│   ├── MAINTENANCE_NOTES.md
│   ├── N2_UI_BASELINE.md
│   ├── N5_UI_BASELINE.md
│   ├── N5_UI_DEVELOPMENT_ARCHITECTURE.md
│   ├── N5_UI_PHASE2_RESEARCH.md
│   ├── N5_UI_PHASE2_RELEASE.md
│   ├── N5_UI_PHASE2_XRAY_INTEGRATION_AUDIT.md
│   ├── N5_UI_PHASE24B_TEST_REPORT.md
│   ├── N5_UI_PHASE35D_REGRESSION_REPORT.md
│   ├── N5_UI_PHASE35_DESIGN.md
│   ├── N5_UI_SIMPLE_MODE_DESIGN.md
│   ├── N5_UI_XRAY_CAPABILITY_REPORT.md
│   ├── N5_UI_XRAY_CORE_BASELINE.md
│   ├── N5_UI_XRAY_MERGE_DESIGN.md
│   ├── N5_UI_XRAY_RUNTIME_BASELINE.md
│   └── artifacts/
├── logger/
│   └── logger.go
├── media/
├── scripts/
│   ├── acme_install.sh
│   └── bbr.sh
├── shims/
│   ├── iptables
│   └── ip6tables
├── util/
│   ├── common/
│   │   ├── err.go
│   │   └── multi_error.go
│   ├── json_util/
│   │   └── json.go
│   ├── random/
│   │   └── random.go
│   ├── reflect_util/
│   │   └── reflect.go
│   ├── sys/
│   │   ├── a.s
│   │   ├── psutil.go
│   │   ├── sys_darwin.go
│   │   └── sys_linux.go
│   └── context.go
├── v2ui/
│   ├── db.go
│   ├── models.go
│   └── v2ui.go
├── web/
│   ├── assets/
│   ├── controller/
│   │   ├── access_ip.go
│   │   ├── base.go
│   │   ├── certificate.go
│   │   ├── inbound.go
│   │   ├── index.go
│   │   ├── reality.go
│   │   ├── server.go
│   │   ├── setting.go
│   │   ├── share_address.go
│   │   ├── util.go
│   │   ├── xui.go
│   │   └── n5/
│   │       ├── egress.go
│   │       ├── egress_label.go
│   │       ├── n5_test.go
│   │       ├── pool.go
│   │       ├── traffic.go
│   │       ├── traffic_template.go
│   │       ├── xray.go
│   │       └── simple/
│   │           ├── egress.go
│   │           ├── egress_test.go
│   │           ├── rule.go
│   │           └── rule_test.go
│   ├── entity/
│   │   ├── certificate.go
│   │   └── entity.go
│   ├── global/
│   │   └── global.go
│   ├── html/
│   │   ├── common/
│   │   │   ├── head.html
│   │   │   ├── js.html
│   │   │   ├── prompt_modal.html
│   │   │   ├── qrcode_modal.html
│   │   │   └── text_modal.html
│   │   ├── login.html
│   │   ├── n5/
│   │   │   ├── config_history.html
│   │   │   ├── egress.html
│   │   │   ├── egress_detail.html
│   │   │   ├── egress_test.html
│   │   │   ├── pools.html
│   │   │   ├── simple.html
│   │   │   ├── simple_egress_edit.html
│   │   │   ├── simple_rules.html
│   │   │   ├── traffic_policy.html
│   │   │   ├── traffic_policy_detail.html
│   │   │   └── xray_status.html
│   │   └── xui/
│   │       ├── access_ips.html
│   │       ├── common_sider.html
│   │       ├── inbound_info_modal.html
│   │       ├── inbound_modal.html
│   │       ├── inbounds.html
│   │       ├── index.html
│   │       ├── setting.html
│   │       ├── component/
│   │       │   ├── inbound_info.html
│   │       │   └── setting.html
│   │       └── form/
│   │           ├── inbound.html
│   │           ├── sniffing.html
│   │           ├── tls_settings.html
│   │           ├── protocol/
│   │           │   ├── dokodemo.html
│   │           │   ├── http.html
│   │           │   ├── mixed.html
│   │           │   ├── shadowsocks.html
│   │           │   ├── socks.html
│   │           │   ├── trojan.html
│   │           │   ├── tunnel.html
│   │           │   ├── vless.html
│   │           │   └── vmess.html
│   │           └── stream/
│   │               ├── stream_grpc.html
│   │               ├── stream_http.html
│   │               ├── stream_kcp.html
│   │               ├── stream_quic.html
│   │               ├── stream_settings.html
│   │               ├── stream_tcp.html
│   │               └── stream_ws.html
│   ├── job/
│   │   ├── access_ip_job.go
│   │   ├── check_inbound_job.go
│   │   ├── check_xray_running_job.go
│   │   └── xray_traffic_job.go
│   ├── network/
│   │   ├── auto_https_listener.go
│   │   └── autp_https_conn.go
│   ├── service/
│   │   ├── access_ip.go
│   │   ├── certificate.go
│   │   ├── config.json
│   │   ├── inbound.go
│   │   ├── panel.go
│   │   ├── portlimit.go
│   │   ├── reality.go
│   │   ├── server.go
│   │   ├── setting.go
│   │   ├── share_address.go
│   │   ├── user.go
│   │   ├── xray.go
│   │   ├── xray_n5_test.go
│   │   └── n5/
│   │       ├── common.go
│   │       ├── egress.go
│   │       ├── egress_detail.go
│   │       ├── egress_label.go
│   │       ├── egress_pool.go
│   │       ├── egress_probe.go
│   │       ├── n5_test.go
│   │       ├── traffic_policy.go
│   │       ├── traffic_policy_detail.go
│   │       ├── traffic_policy_manage_test.go
│   │       ├── traffic_template.go
│   │       ├── traffic_template_test.go
│   │       ├── xray_capability_test.go
│   │       ├── xray_ext.go
│   │       ├── xray_history.go
│   │       ├── xray_history_test.go
│   │       ├── xray_merge.go
│   │       ├── xray_merge_test.go
│   │       ├── xray_status.go
│   │       ├── xray_status_test.go
│   │       ├── templates/
│   │       │   ├── ai.go
│   │       │   ├── base.go
│   │       │   ├── game.go
│   │       │   └── streaming.go
│   │       └── simple/
│   │           ├── egress.go
│   │           ├── egress_test.go
│   │           ├── rule.go
│   │           └── rule_test.go
│   ├── session/
│   │   └── session.go
│   ├── translation/
│   │   ├── translate.en_US.toml
│   │   ├── translate.zh_Hans.toml
│   │   └── translate.zh_Hant.toml
│   └── web.go
├── xray/
│   ├── config.go
│   ├── inbound.go
│   ├── process.go
│   ├── process_test.go
│   ├── reality.go
│   └── traffic.go
├── xray-core/
├── Dockerfile
├── LICENSE
├── README.md
├── README_EN.md
├── main.go
├── install.sh
├── install_en.sh
├── x-ui.sh
├── x-ui_en.sh
├── x-ui.service
├── xui-portlimit-sync.service
├── xui-portlimit-sync.sh
├── xui-portlimit-sync.timer
├── N5_UI_PHASE1_BRANDING_REPORT.md
├── go.mod
└── go.sum
```

### 目录职责

- `bin/`: 运行期 Xray 资源与默认最终配置落地目录。
- `config/`: 程序名、版本、调试开关等基础配置。
- `database/`: SQLite/GORM 初始化、旧表模型、N5 新表模型与迁移逻辑。
- `docs/`: 研发、审计、测试、发布归档文档。
- `logger/`: 程序日志封装。
- `scripts/`: 安装脚本依赖脚本，如 `acme` 安装和 `bbr`。
- `util/`: 公共工具函数。
- `v2ui/`: 历史数据迁移兼容层。
- `web/`: 面板 Web 层，含 controller、service、html、assets、jobs。
- `xray/`: 项目内对 Xray 配置与进程的封装。
- `xray-core/`: vendored 上游 Xray-core 子树，仅做编译依赖和运行兼容层。

## 2. 后端技能树

### 2.1 `database/`

#### 原 x-ui 数据层

| 文件 | 职责 | 调用关系 |
|---|---|---|
| `database/db.go` | 初始化 SQLite、执行原 x-ui 与 N5 自动迁移、填充默认用户、清洗旧协议字段 | `main.go` 启动时初始化；`web/service/*` 通过 `database.GetDB()` 访问 |
| `database/model/model.go` | 定义原始 `User`、`Inbound`、`Setting`、`AccessIPRecord` 模型；`Inbound.GenXrayInboundConfig()` 将数据库入站转为 Xray 配置 | `InboundService`、`SettingService`、`AccessIPService`、`XrayService` 直接依赖 |

#### N5 新增数据层

| 文件 | 职责 | 调用关系 |
|---|---|---|
| `database/model/n5/models.go` | 定义所有 `n5_` 表结构 | `web/service/n5/*` 全量依赖 |
| `database/n5_phase2.go` | 注册 N5 AutoMigrate；执行稳定 tag 迁移；为 `TrafficPolicyBinding` 建立 `inbound_id` 唯一索引 | `database/db.go` 初始化时调用 |
| `database/n5_phase2_test.go` | 验证 N5 AutoMigrate、tag 迁移和唯一索引 | `go test ./...` 覆盖数据库迁移层 |

#### N5 表关系

- `n5_egresses`：出口主表。
- `n5_egress_test`：出口测试记录，`egress_id -> n5_egresses.id`。
- `n5_egress_labels`：标签主表。
- `n5_egress_label_relations`：出口与标签多对多关系，`egress_id -> n5_egresses.id`，`label_id -> n5_egress_labels.id`。
- `n5_egress_pools`：线路池主表。
- `n5_egress_pool_members`：线路池成员表，`pool_id -> n5_egress_pools.id`，`egress_id -> n5_egresses.id`。
- `n5_traffic_policies`：分流策略主表。
- `n5_traffic_policy_rules`：策略规则表，`policy_id -> n5_traffic_policies.id`。
- `n5_traffic_policy_bindings`：入站与策略绑定表，`inbound_id -> inbounds.id`，`policy_id -> n5_traffic_policies.id`。
- `n5_xray_config_history`：N5 merge 配置快照与应用状态历史。

### 2.2 `web/controller/`

#### 原 controller

| 文件 | 职责 | 调用关系 |
|---|---|---|
| `web/controller/base.go` | 基础控制器公共逻辑 | 原 x-ui controller 继承使用 |
| `web/controller/index.go` | 登录、首页跳转 | `web/web.go` 注册 |
| `web/controller/server.go` | 系统状态、重启、日志、版本页相关控制 | 对应 `ServerService` |
| `web/controller/xui.go` | 原 x-ui 路由聚合器，挂载 `/xui/*` 页面与子控制器 | `web/web.go` 注册 |
| `web/controller/inbound.go` | 入站 CRUD、分享、重置流量、客户端增删改等 | 调用 `InboundService`、`XrayService` |
| `web/controller/setting.go` | 设置页读取与更新 | 调用 `SettingService` |
| `web/controller/access_ip.go` | 访问 IP 页面和接口 | 调用 `AccessIPService` |
| `web/controller/share_address.go` | 分享地址相关接口 | 调用 `ShareAddressService` |
| `web/controller/certificate.go` | 证书列表、发现、导入、校验 | 调用 `CertificateService` |
| `web/controller/reality.go` | Reality 默认配置与 X25519 密钥生成 | 调用 `RealityService` |
| `web/controller/util.go` | Web 层辅助函数 | 被多个 controller 使用 |

#### N5 controller

| 文件 | 页面/路由 | 职责 | 主要依赖 |
|---|---|---|---|
| `web/controller/n5/egress.go` | `/n5/egress`、`/n5/egress-detail`、`/n5/api/egress/*` | 出口线路 CRUD、详情、校验、测试 | `EgressService`、`EgressDetailService`、`EgressTestService` |
| `web/controller/n5/egress_label.go` | `/n5/api/egress-label/*` | 标签 CRUD、绑定、解绑 | `EgressLabelService` |
| `web/controller/n5/pool.go` | `/n5/pools`、`/n5/api/pool/*` | 线路池 CRUD、成员管理 | `EgressPoolService` |
| `web/controller/n5/traffic.go` | `/n5/traffic-policy`、`/n5/traffic-policy-detail`、`/n5/api/traffic-policy/*` | 策略 CRUD、规则 CRUD、启停、排序、绑定、片段预览 | `TrafficPolicyService`、`TrafficPolicyDetailService`、`XrayExtService` |
| `web/controller/n5/traffic_template.go` | `/n5/api/traffic-template/*` | 内置模板列表、预览、按模板创建策略 | `TrafficTemplateService` |
| `web/controller/n5/xray.go` | `/n5/xray-status`、`/n5/config-history`、`/n5/egress-test`、`/n5/api/xray/*` | N5 运行状态、配置历史、手动出口测试入口说明 | `XrayStatusService`、`XrayHistoryService` |
| `web/controller/n5/n5_test.go` | N5 控制器整合测试 | 控制器链路验证 | `go test ./...` |

#### Simple controller

| 文件 | 页面/路由 | 职责 | 主要依赖 |
|---|---|---|---|
| `web/controller/n5/simple/egress.go` | `/n5/simple`、`/n5/simple/edit`、`/n5/api/simple/egress/*` | Simple 出口列表、创建、编辑、测试、删除 | `web/service/n5/simple.EgressService` |
| `web/controller/n5/simple/rule.go` | `/n5/simple/rules`、`/n5/api/simple/rule/*` | Simple 规则列表、创建、删除 | `web/service/n5/simple.RuleService` |
| `web/controller/n5/simple/egress_test.go` | Simple 出口控制器测试 | API 与路由测试 | `go test ./...` |
| `web/controller/n5/simple/rule_test.go` | Simple 规则控制器测试 | API 与页面路由测试 | `go test ./...` |

### 2.3 `web/service/`

#### 原 service

| 文件 | 职责 | 调用关系 |
|---|---|---|
| `web/service/user.go` | 用户认证与密码处理 | `IndexController`、`SettingController` |
| `web/service/server.go` | 程序运行状态、服务控制 | `ServerController` |
| `web/service/panel.go` | 面板运行信息辅助 | `ServerController` |
| `web/service/portlimit.go` | 端口限制同步 | job / 外部同步脚本 |
| `web/service/reality.go` | Reality 默认模板、X25519 密钥生成 | `RealityController` |
| `web/service/access_ip.go` | 接入 IP 记录、清理、统计 | `AccessIPController`、job |
| `web/service/config.json` | 原 x-ui 默认 Xray 模板 | `SettingService` 默认值 |
| `web/service/setting.go` | 设置读取、默认值、持久化；包含 `n5XrayExtensionEnable` | `SettingController`、`web/web.go`、`XrayService` |
| `web/service/share_address.go` | 分享地址与导出辅助 | `ShareAddressController` |
| `web/service/certificate.go` | 证书扫描、导入、校验、托管目录元数据 | `CertificateController` |
| `web/service/inbound.go` | 入站 CRUD、客户端处理、导入导出、校验 | `InboundController`、`XrayService` |
| `web/service/xray.go` | 原 x-ui Xray 配置生成、测试、重启；N5 merge 调用点就在此处 | `Server`、job、`InboundController` |
| `web/service/xray_n5_test.go` | Xray + N5 merge 相关联动测试 | `go test ./...` |

#### N5 service

| 文件 | 职责 | 主要调用者 | 调用下游 |
|---|---|---|---|
| `web/service/n5/common.go` | N5 通用常量、目标类型、JSON 解析、协议归一化等辅助 | 所有 N5 service | 公共工具 |
| `web/service/n5/egress.go` | 出口 CRUD、稳定 tag、配置校验 | `EgressController`、Simple Egress | `database`、`xray.TestConfig` |
| `web/service/n5/egress_detail.go` | 聚合出口详情：基础信息、标签、线路池、关联策略 | `EgressController.detail` | `EgressLabelService`、数据库查询 |
| `web/service/n5/egress_label.go` | 标签 CRUD、多对多绑定 | `EgressLabelController`、`EgressDetailService` | `database` |
| `web/service/n5/egress_pool.go` | 线路池 CRUD、成员管理、tag 生成 | `PoolController`、`XrayExtService` | `database` |
| `web/service/n5/egress_probe.go` | 手动出口测试，临时拉起独立 xray 子进程，记录出口 IP 与延迟 | `EgressController.test`、Simple Egress | `xray`、`database` |
| `web/service/n5/traffic_policy.go` | 策略 CRUD、规则 CRUD、启停、排序、入站绑定 | `TrafficPolicyController`、Simple Rule、Template Service | `database` |
| `web/service/n5/traffic_policy_detail.go` | 策略详情聚合，用于详情页 | `TrafficPolicyController.get` | `TrafficPolicyService`、数据库 |
| `web/service/n5/traffic_template.go` | 内置模板列表、预览、按模板创建策略/规则/绑定 | `TrafficTemplateController`、Simple Rule | `TrafficPolicyService`、`templates/*` |
| `web/service/n5/xray_ext.go` | 根据数据库生成 outbound/routing 片段，不接主配置写入 | `TrafficPolicyController.fragments`、`XrayMergeService` | `EgressService`、`TrafficPolicyService` |
| `web/service/n5/xray_merge.go` | 把 N5 outbound/routing merge 到原 x-ui base config，并写入 config history | `web/service/xray.go` | `XrayExtService`、`XrayHistoryService` |
| `web/service/n5/xray_status.go` | 读取 N5 enable 状态、last apply、hash、计数 | `XrayController.status` | `SettingService`、`XrayHistoryService` |
| `web/service/n5/xray_history.go` | 读取 `n5_xray_config_history` 历史 | `XrayController.historyList` | `database` |
| `web/service/n5/templates/base.go` | 模板定义结构、模板索引 | `TrafficTemplateService`、Simple Rule | 同目录模板 |
| `web/service/n5/templates/ai.go` | AI 模板规则集 | `TrafficTemplateService` | - |
| `web/service/n5/templates/game.go` | 游戏模板规则集 | `TrafficTemplateService` | - |
| `web/service/n5/templates/streaming.go` | 流媒体模板规则集 | `TrafficTemplateService` | - |
| `web/service/n5/n5_test.go` | N5 service 通用测试桩与辅助 | N5 测试文件 | - |
| `web/service/n5/traffic_policy_manage_test.go` | 策略管理测试 | `go test ./...` | - |
| `web/service/n5/traffic_template_test.go` | 模板创建测试 | `go test ./...` | - |
| `web/service/n5/xray_capability_test.go` | Xray 能力校验测试 | `go test ./...` | - |
| `web/service/n5/xray_history_test.go` | 配置历史测试 | `go test ./...` | - |
| `web/service/n5/xray_merge_test.go` | merge 逻辑测试 | `go test ./...` | - |
| `web/service/n5/xray_status_test.go` | 状态接口测试 | `go test ./...` | - |

#### Simple service

| 文件 | 职责 | 主要调用者 | 调用下游 |
|---|---|---|---|
| `web/service/n5/simple/egress.go` | Simple 出口视图模型、协议约束、SOCKS5/SS outbound 构建、创建/编辑/测试/删除封装 | `Simple Egress Controller` | `n5.EgressService`、`n5.EgressTestService` |
| `web/service/n5/simple/rule.go` | Simple 规则抽象，把“入口 + 流量类型 + 出口”转换为高级 `TrafficPolicy` / `Template` / `Binding` | `Simple Rule Controller` | `InboundService`、`n5.EgressService`、`n5.TrafficPolicyService`、`n5.TrafficTemplateService` |
| `web/service/n5/simple/egress_test.go` | Simple 出口 service 测试 | `go test ./...` | - |
| `web/service/n5/simple/rule_test.go` | Simple 规则 service 测试 | `go test ./...` | - |

## 3. 前端技能树

### 3.1 原 x-ui 页面

| 页面文件 | 路径 | 功能 | API 依赖 |
|---|---|---|---|
| `web/html/login.html` | `/` 登录前入口 | 登录页 | 原登录 API |
| `web/html/xui/index.html` | `/xui/` | 系统状态首页 | `server` 系列 API |
| `web/html/xui/inbounds.html` | `/xui/inbounds` | 入站列表与弹窗入口 | `InboundController` API |
| `web/html/xui/access_ips.html` | `/xui/access-ips` | 接入 IP 列表 | `AccessIPController` API |
| `web/html/xui/setting.html` | `/xui/setting` | 面板设置、Xray 模板、N5 merge 开关 | `SettingController`、`CertificateController` |
| `web/html/xui/common_sider.html` | 所有登录后页 | 左侧导航，已加入 N5 菜单 | 页面跳转 |
| `web/html/xui/inbound_modal.html` | 入站弹窗 | 入站编辑表单容器 | `InboundController` |
| `web/html/xui/inbound_info_modal.html` | 入站详情弹窗 | 连接信息展示 | `ShareAddressController` |
| `web/html/xui/component/inbound_info.html` | 入站信息组件 | 客户端/端口展示 | `InboundController` |
| `web/html/xui/component/setting.html` | 设置项组件 | 表单复用 | `SettingController` |
| `web/html/xui/form/inbound.html` | 入站表单主模板 | 组合协议、TLS、stream 表单 | `InboundController` |
| `web/html/xui/form/sniffing.html` | sniffing 设置 | 入站 sniffing | `InboundController` |
| `web/html/xui/form/tls_settings.html` | TLS / Reality 配置 | 入站 TLS/Reality 设置 | `CertificateController`、`RealityController` |
| `web/html/xui/form/protocol/*.html` | 协议子表单 | 协议特定字段 | `InboundController` |
| `web/html/xui/form/stream/*.html` | 传输子表单 | transport 细项 | `InboundController` |

### 3.2 N5 页面

| 页面文件 | 路径 | 功能 | API 依赖 |
|---|---|---|---|
| `web/html/n5/egress.html` | `/n5/egress` | 高级出口管理列表与表单 | `/n5/api/egress/*` |
| `web/html/n5/egress_detail.html` | `/n5/egress-detail?id=` | 出口详情，显示标签、线路池、策略引用、测试状态 | `/n5/api/egress/detail/:id`、`/n5/api/egress-label/list` |
| `web/html/n5/pools.html` | `/n5/pools` | 线路池与成员管理 | `/n5/api/pool/*`、`/n5/api/egress/list` |
| `web/html/n5/traffic_policy.html` | `/n5/traffic-policy` | 分流策略列表、模板创建、绑定列表 | `/n5/api/traffic-policy/*`、`/n5/api/traffic-template/*`、`/n5/api/egress/list`、`/n5/api/pool/list` |
| `web/html/n5/traffic_policy_detail.html` | `/n5/traffic-policy-detail?id=` | 策略详情、规则编辑、启停、排序 | `/n5/api/traffic-policy/get/:id`、`/rule/*`、`/binding/*` |
| `web/html/n5/xray_status.html` | `/n5/xray-status` | N5 merge 状态面板 | `/n5/api/xray/status` |
| `web/html/n5/config_history.html` | `/n5/config-history` | N5 配置历史列表 | `/n5/api/xray/history/list` |
| `web/html/n5/egress_test.html` | `/n5/egress-test` | 手动出口测试入口说明页 | `/n5/api/xray/egress-test/entry` |

### 3.3 Simple 页面

| 页面文件 | 路径 | 功能 | API 依赖 |
|---|---|---|---|
| `web/html/n5/simple.html` | `/n5/simple` | Simple 出口列表、创建、测试、删除 | `/n5/api/simple/egress/*` |
| `web/html/n5/simple_egress_edit.html` | `/n5/simple/edit?id=` | Simple 出口编辑 | `/n5/api/simple/egress/get/:id`、`/update/:id` |
| `web/html/n5/simple_rules.html` | `/n5/simple/rules` | Simple 规则列表与创建 | `/n5/api/simple/rule/*`、`/n5/api/simple/egress/list` |

## 4. Xray 扩展技能树

### 4.1 基础生成流程

原 x-ui 基线：

1. `web/service/setting.go:GetXrayConfigTemplate()`
2. `web/service/xray.go:getXrayConfigWithMeta()`
3. 反序列化模板为 `xray.Config`
4. `InboundService.GetAllInbounds()`
5. `Inbound.GenXrayInboundConfig()`
6. 如有 `dokodemo/tunnel`，执行 `ensureDokodemoTunnelRouting()`
7. 若 `n5XrayExtensionEnable=false`，直接返回 base config

N5 merge 叠加：

1. `SettingService.GetN5XrayExtensionEnable()`
2. `web/service/n5/xray_merge.go:MergeWithMeta(base)`
3. `web/service/n5/xray_ext.go:GenerateOutboundFragments()`
4. `web/service/n5/xray_ext.go:GenerateRoutingFragments()`
5. 将 outbound / balancer / routing 合并到 base config
6. 写入 `n5_xray_config_history`
7. 返回 `XrayMergeResult`
8. `web/service/xray.go:RestartXray()`
9. `xray.TestConfig()`
10. 通过后启动/重启 Xray；失败则记录 history 为 failed

### 4.2 关键文件与职责

| 文件 | 关键函数 | 职责 |
|---|---|---|
| `web/service/xray.go` | `getXrayConfigWithMeta()`、`RestartXray()` | 原 x-ui 生成 base config，并作为 N5 merge 唯一接入点 |
| `web/service/n5/xray_ext.go` | `GenerateOutboundFragments()`、`GenerateRoutingFragments()` | 根据数据库只生成扩展片段，不改主链路 |
| `web/service/n5/xray_merge.go` | `Merge()`、`MergeWithMeta()`、`UpdateHistoryStatus()` | 把片段 merge 到 base config，并记录 history |
| `web/service/n5/xray_history.go` | `List()` | 提供配置历史页数据 |
| `web/service/n5/xray_status.go` | `GetStatus()` | 提供 N5 状态页数据 |
| `database/n5_phase2.go` | `migrateN5StableTags()` | 保证 egress/pool tag 稳定，避免 balancer selector 前缀碰撞 |
| `web/controller/n5/xray.go` | `status()`、`historyList()` | 对外暴露状态与历史 API |

### 4.3 Merge 数据来源

- 出口：`n5_egresses`
- 线路池：`n5_egress_pools` + `n5_egress_pool_members`
- 策略：`n5_traffic_policies`
- 规则：`n5_traffic_policy_rules`
- 绑定：`n5_traffic_policy_bindings`
- 历史：`n5_xray_config_history`
- 开关：`settings.n5XrayExtensionEnable`

### 4.4 调用链

```text
SettingService.GetXrayConfigTemplate
  -> XrayService.getXrayConfigWithMeta
     -> InboundService.GetAllInbounds
     -> append base inbound configs
     -> SettingService.GetN5XrayExtensionEnable
        -> false: return base config
        -> true:
           -> XrayMergeService.MergeWithMeta
              -> XrayExtService.GenerateOutboundFragments
              -> XrayExtService.GenerateRoutingFragments
              -> merge outbounds / balancers / routing
              -> write n5_xray_config_history
     -> return merged config + history meta
  -> XrayService.RestartXray
     -> xray.TestConfig
     -> start process or rollback
     -> update history status
```

## 5. N5 模块完整技能树

### 5.1 Egress

#### 模型

- `database/model/n5/models.go:Egress`
- `database/model/n5/models.go:EgressTest`
- `database/model/n5/models.go:EgressLabel`
- `database/model/n5/models.go:EgressLabelRelation`

#### Service

- `web/service/n5/egress.go`
  - 稳定 tag：`GenerateStableTag`
  - 校验：`ValidateConfig`
  - CRUD：`Create` / `Update` / `Delete` / `Get` / `List`
- `web/service/n5/egress_probe.go`
  - 手动测试：`Test`
- `web/service/n5/egress_detail.go`
  - 详情聚合：`Get`
- `web/service/n5/egress_label.go`
  - 标签 CRUD / bind / unbind / listByEgress

#### Controller

- `web/controller/n5/egress.go`
- `web/controller/n5/egress_label.go`

#### HTML

- `web/html/n5/egress.html`
- `web/html/n5/egress_detail.html`
- `web/html/n5/egress_test.html`

#### 职责

- 高级出口管理。
- 出口协议原始 JSON 校验。
- 出口测试与状态记录。
- 标签系统与详情页聚合。

### 5.2 Traffic Policy

#### 模型

- `database/model/n5/models.go:TrafficPolicy`
- `database/model/n5/models.go:TrafficPolicyRule`
- `database/model/n5/models.go:TrafficPolicyBinding`

#### Service

- `web/service/n5/traffic_policy.go`
- `web/service/n5/traffic_policy_detail.go`
- `web/service/n5/traffic_template.go`
- `web/service/n5/xray_ext.go`

#### Controller

- `web/controller/n5/traffic.go`
- `web/controller/n5/traffic_template.go`

#### HTML

- `web/html/n5/traffic_policy.html`
- `web/html/n5/traffic_policy_detail.html`

#### 职责

- 定义“入站 -> 规则 -> 出口/线路池”的高级分流模型。
- 管理默认目标、规则顺序、启停与入站唯一绑定。
- 支持内置模板快速生成策略。

### 5.3 Simple Mode

#### Service

- `web/service/n5/simple/egress.go`
- `web/service/n5/simple/rule.go`

#### Controller

- `web/controller/n5/simple/egress.go`
- `web/controller/n5/simple/rule.go`

#### HTML

- `web/html/n5/simple.html`
- `web/html/n5/simple_egress_edit.html`
- `web/html/n5/simple_rules.html`

#### 职责

- 复用高级 `Egress` / `TrafficPolicy` 能力，提供低理解成本入口。
- Simple 出口仅支持 `SOCKS5` 与 `Shadowsocks`。
- Simple 规则把“入口 + 流量类型 + 出口”翻译为高级策略与绑定，不重写底层逻辑。

### 5.4 Template

#### 文件

- `web/service/n5/templates/base.go`
- `web/service/n5/templates/ai.go`
- `web/service/n5/templates/game.go`
- `web/service/n5/templates/streaming.go`
- `web/service/n5/traffic_template.go`
- `web/controller/n5/traffic_template.go`

#### 职责

- 提供内置代码模板，不走数据库。
- 把模板规则集转换为 `TrafficPolicy` + `TrafficPolicyRule` + `Binding`。

### 5.5 Certificate

#### 当前范围

- 仅完成 Phase A：HTTPS Rescue。

#### 关键文件

- `x-ui.sh`

#### 已迁移内容

- `require_sqlite()`
- `get_panel_setting()`
- `cert_key_match()`
- `show_panel_https_status()`
- `disable_panel_https()`
- `repair_panel_https_certificate()`
- `save_panel_cert_setting()` 增加 cert/key 匹配校验
- `cert_manage()` 扩展到 `0-8`

#### 职责

- 只处理面板 HTTPS 救援。
- 不修改 Go 证书系统。
- 不改变安装脚本或运行兼容层。

## 6. 建议的开发入口

### 建议优先进入的文件

- N5 数据：`database/model/n5/models.go`
- N5 路由：`web/controller/n5/*.go`
- N5 业务：`web/service/n5/*.go`
- Simple 入口：`web/service/n5/simple/*.go`
- Xray merge 接入点：`web/service/xray.go`
- N5 merge 核心：`web/service/n5/xray_merge.go`
- N5 片段生成：`web/service/n5/xray_ext.go`
- 前端页面：`web/html/n5/*.html`

### 不建议轻易进入的文件

- `install.sh`
- `install_en.sh`
- `x-ui.service`
- `xray-core/`
- 原 `database/model/model.go` 的旧结构字段
- 原 `web/service/inbound.go`、`web/service/xray.go` 中非 N5 merge 接入点之外的主链路
