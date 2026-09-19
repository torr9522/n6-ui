## 1. 问题背景

N5-UI 在 Phase2、Phase3、Phase3.5 期间已经完成了出口、策略、模板、Simple Mode 等功能，数据库侧的 `n5_egresses`、`n5_traffic_policies`、`n5_traffic_policy_rules`、`n5_traffic_policy_bindings` 等数据可以正常写入。

问题出现在运行态：

- 面板操作后，N5 数据已经写入数据库
- `XrayExtService` 手动生成 fragment 正常
- 但 `/usr/local/x-ui/bin/config.json` 没有出现新的 N5 outbound 和 routing
- `n5_xray_config_history` 没有新增记录
- `/n5/api/xray/status` 中 `lastApply` 为空或未更新

表现结果是：

- 面板里看到的 N5 配置已经存在
- 运行中的 Xray 仍然使用旧配置
- 指定流量没有进入 N5 出口
- 数据库状态与 Xray 运行状态不一致

这个问题在真实测试服务器 `<TEST_SERVER>` 上被复现，并最终确认不是出口测试、不是规则生成、也不是 N5 Merge 逻辑本身故障，而是配置变更后没有自动进入 Xray apply 链。

## 2. 根因

根因是多个 N5 Controller 在完成数据库写入后，没有调用：

```go
SetToNeedRestart()
```

因此缺失了下面这段运行触发链：

```text
N5 配置修改
-> SetToNeedRestart()
-> 定时任务检测到 need restart
-> RestartXray()
-> N5 Merge
-> 写入 config.json
-> history applied
```

在修复前：

- N5 的 add/update/delete/bind/reorder 等操作只改数据库
- 定时任务没有收到重启标记
- `RestartXray()` 不会被触发
- `getXrayConfigWithMeta()` 不会走到 N5 Merge
- `n5_xray_config_history` 不会新增记录

因此问题本质不是“规则不会生成”，而是“规则生成后没有自动进入运行配置”。

## 3. 修复内容

本次修复的核心原则是：

- 不修改 N5 Merge 核心
- 不修改 Xray 配置生成主逻辑
- 不修改数据库结构
- 只在会影响运行配置的 N5 Controller 操作后补齐 `SetToNeedRestart()`

修复范围如下。

### 3.1 Simple Rule

文件：

- `web/controller/n5/simple/rule.go`

补齐操作：

- `add`
- `delete`

新增能力：

- `GET /n5/api/simple/rule/n5-status`
- `POST /n5/api/simple/rule/n5-status`

作用：

- Simple 规则新增后自动进入 apply 队列
- Simple 规则删除后自动重新应用
- Simple 页面可以直接读取和修改 `n5XrayExtensionEnable`
- 修改开关后同样触发自动 apply

### 3.2 Simple Egress

文件：

- `web/controller/n5/simple/egress.go`

补齐操作：

- `add`
- `update`
- `delete`

作用：

- Simple 出口新增、修改、删除后自动触发 Xray 重载链

### 3.3 Advanced Egress

文件：

- `web/controller/n5/egress.go`

补齐操作：

- `add`
- `update`
- `delete`

作用：

- 高级出口管理和 Simple 出口管理都能进入统一 apply 链

### 3.4 Traffic Policy / Rule / Binding

文件：

- `web/controller/n5/traffic.go`

补齐操作：

- `add`
- `update`
- `delete`
- `enable`
- `disable`
- `addRule`
- `updateRule`
- `deleteRule`
- `enableRule`
- `disableRule`
- `reorderRules`
- `bind`
- `unbind`
- `rebind`

作用：

- 所有会影响 routing、outbound target、inbound binding 的策略类操作都能自动触发应用

### 3.5 Pool Member

文件：

- `web/controller/n5/pool.go`

补齐操作：

- `addMember`
- `delMember`

作用：

- 线路池成员变化后自动触发 balancer 相关配置重建

### 3.6 Template Create

文件：

- `web/controller/n5/traffic_template.go`

补齐操作：

- `create`

作用：

- 通过 AI / game / streaming 模板创建策略后立即进入运行配置，不再需要手工重启或额外保存设置

### 3.7 页面同步

文件：

- `web/html/n5/simple_rules.html`

新增：

- N5 分流状态卡
- enabled / disabled 状态显示
- 开启按钮
- 关闭按钮

作用：

- Simple 页面直接映射高级设置中的 `n5XrayExtensionEnable`
- 用户可以在 Simple 规则页面观察当前分流总开关状态
- 开关变更后明确提示“等待自动应用”

### 3.8 测试补充

文件：

- `web/controller/n5/n5_test.go`
- `web/controller/n5/simple/egress_test.go`
- `web/controller/n5/simple/rule_test.go`

覆盖内容：

- N5 配置变更是否设置 restart flag
- Simple egress add/update/delete 是否触发 restart
- Simple rule add/delete 是否触发 restart
- Simple N5 开关接口是否可用，且是否触发 restart

## 4. Apply 链路

修复后，完整 apply 链路如下：

```text
Controller
-> SetToNeedRestart()
-> 定时任务
-> RestartXray()
-> getXrayConfigWithMeta()
-> N5 Merge
-> TestConfig
-> 写入 /usr/local/x-ui/bin/config.json
-> 写入 n5_xray_config_history
-> applied
```

结合当前代码实现，链路拆分如下。

### 4.1 Controller 层

N5 Controller 在成功完成数据库变更后调用：

```go
xrayService.SetToNeedRestart()
```

这一步只负责打标记，不直接重启。

### 4.2 定时任务

后台定时任务周期检测 restart 标记。

一旦发现存在待重启标记，就调用 `RestartXray()`。

### 4.3 RestartXray

`RestartXray()` 进入统一的配置生成链，而不是只重启现有进程。

关键职责：

- 取基础 x-ui 配置
- 读取 N5 开关
- 决定是否执行 N5 Merge

### 4.4 N5 Merge

当 `n5XrayExtensionEnable=true` 时：

- 合成 N5 outbound
- 合成 routing rules
- 合成 balancer / selector 所需扩展片段
- 计算 hash
- 生成 history 记录

### 4.5 TestConfig

合成后的完整 Xray 配置先经过 `TestConfig` 校验。

只有校验通过才继续写入运行配置。

### 4.6 config.json

通过校验后：

- 更新 `/usr/local/x-ui/bin/config.json`
- 重启/拉起 Xray 进程

### 4.7 History

成功应用后：

- `n5_xray_config_history` 新增记录
- 最终状态写成 `applied`

真实轮询中已经观察到状态变化：

- `generated`
- `validated`
- `applied`

## 5. 真实测试

真实验证服务器：

- `<TEST_SERVER>`

测试入口：

- inbound tag: `inbound-39126`
- protocol: `vmess`

测试出口：

- egress tag: `n5-egress-0000000001`
- protocol: `socks`
- address: `<TEST_DOMAIN>:<TEST_PORT>`
- SOCKS 出口公网 IP: `<REDACTED_SOURCE_IP>`

测试规则：

- `full:api64.ipify.org` -> `n5-egress-0000000001`

验证步骤：

1. 通过 Simple Rule 创建 `custom-domain` 规则
2. 等待自动 apply
3. 检查 `config.json` 中是否出现：
   - `n5-egress-0000000001`
   - `inbound-39126`
   - `full:api64.ipify.org`
4. 通过真实 VMess 入站访问目标站点
5. 检查 Xray access log 命中情况

验证结果：

- `api64.ipify.org` 返回：`<REDACTED_SOURCE_IP>`
- `api.ipify.org` 返回：`<TEST_SERVER>`

access 日志命中证据：

```text
accepted tcp:api64.ipify.org:443 [inbound-39126 -> n5-egress-0000000001]
accepted tcp:api.ipify.org:443
```

结论：

- 指定域名流量成功进入 SOCKS 出口
- 未匹配流量仍走服务器默认出口
- 修复后的自动 apply 链真实生效

## 6. N5 开关同步

本次还补齐了 Simple 规则页面与高级设置页面之间的 N5 开关同步。

同步对象：

- Simple 页面：`/n5/simple/rules`
- 高级设置字段：`n5XrayExtensionEnable`

验证结果：

- Simple 页面关闭 N5 分流后：
  - `/n5/api/simple/rule/n5-status` 返回 `enabled=false`
  - 高级设置 `/xui/setting/all` 返回 `n5XrayExtensionEnable=false`
- Simple 页面再次开启 N5 分流后：
  - `/n5/api/simple/rule/n5-status` 返回 `enabled=true`
  - 高级设置 `/xui/setting/all` 返回 `n5XrayExtensionEnable=true`

说明：

- 两个页面使用的是同一个 setting
- 不存在 Simple 页面状态与高级设置不一致的问题
- 开关变化后同样会进入自动 apply 队列

## 7. 已知问题

当前已知问题：

### 7.1 `database is locked` 偶发

在高频轮询 `n5_xray_config_history`、同时触发 apply 的测试过程中，SQLite 偶尔会出现：

```text
database is locked
```

影响范围：

- 主要出现在并发读写的测试观测阶段
- 不影响最终的 config 应用结果
- 不影响 `config.json` 更新
- 不影响最终 `applied` 状态落库

当前结论：

- 这是 SQLite 并发访问下的偶发现象
- 不属于本次 runtime apply trigger 缺失的主故障
- 后续如需进一步优化，可单独评估 history 观测或数据库 busy timeout 策略

## 结论

本次修复已经解决 N5-UI 在 `v0.1.0-beta-simple` 阶段存在的运行态一致性问题：

- N5 配置变更不再停留在数据库层
- 所有关键 N5 配置修改都会自动进入 Xray apply 链
- `config.json` 与数据库状态保持一致
- `n5_xray_config_history` 能记录运行应用结果
- 真实分流已在服务器 `<TEST_SERVER>` 上验证通过

这次修复适合作为 `v0.1.1` 发布内容。
