# N5_UI_FINAL_ACCEPTANCE_REPORT

N5-UI `v0.1.0-beta-simple` 最终发布验收报告。

验收日期：2026-08-11  
仓库：`https://github.com/torr9522/n5-ui`  
Runtime：`Xray 26.3.27`

## 1. 项目独立化历程

N5-UI 起点是基于 `n3-ui` 的二次开发项目。早期阶段继承了原有的面板基础能力，包括原生 x-ui 入站管理、证书管理、系统服务结构、Web 页面骨架，以及与 Xray 运行相关的基础控制流程。这一继承关系本身没有问题，但如果继续长期依赖上游仓库、安装脚本和 Runtime 发布链，会带来几个直接风险：

1. 发布链不受控。安装脚本和下载源如果仍指向 `n3-ui`，则 N5-UI 的公网安装结果不具备可验证的独立性。
2. Runtime 基线不稳定。运行时版本如果从上游动态获取，N5-UI 的已验证配置和实际安装得到的 Xray 版本可能不一致。
3. 文档与用户认知错位。用户看到的是 N5-UI，但安装、更新、下载链仍可能落到 `n3-ui`，这会造成维护边界不清晰。
4. 后续功能演进受限。N5 扩展包括出口线路、流量分流、Simple Mode、Merge Layer，都要求有自己的版本、文档和运行基线。

因此，本次 `v0.1.0-beta-simple` 发布的核心目标，不只是“功能可用”，而是完成 N5-UI 的独立化。

独立化路径如下：

`n3-ui 基础`
→ `N5-UI 二开`
→ `建立独立 GitHub 仓库`
→ `切换独立安装链`
→ `切换独立 Release 资产`
→ `固定独立 Xray Runtime`
→ `完成公网 Fresh Install 验收`

本次验收确认了以下边界：

- 已继承的部分：
  - 原 x-ui 的基础面板结构
  - 原生入站模型与控制逻辑
  - 原有证书管理基础能力
  - 原有 systemd 服务与基本部署目录结构
- 已独立的部分：
  - GitHub 仓库
  - 安装入口
  - Git clone 来源
  - Release 发布链
  - Xray Runtime 下载来源
  - N5 扩展模块
  - N5 文档体系
  - N5 版本标识

结论：N5-UI 在 `v0.1.0-beta-simple` 阶段，已经从“依附式二开”转入“独立发行项目”状态。

## 2. 源码与发布链验证

本次验收重点验证了源码链、安装链、运行链三者是否已经全部切换到 `torr9522/n5-ui`。

验收结果如下：

- GitHub 仓库：`https://github.com/torr9522/n5-ui`
- 安装入口：`https://raw.githubusercontent.com/torr9522/n5-ui/main/install.sh`
- Git clone：`https://github.com/torr9522/n5-ui.git`
- Release：`v0.1.0-beta-simple`
- Runtime：`Xray 26.3.27`

Fresh install 完成后，远端服务器 `/usr/local/x-ui` 中实际验证到：

- `git remote -v`
  - `origin https://github.com/torr9522/n5-ui.git (fetch)`
  - `origin https://github.com/torr9522/n5-ui.git (push)`
- `git branch -a`
  - `main`
  - `remotes/origin/main`
- `git log -1 --oneline`
  - `e230ecf N5-UI standalone installation chain`

安装日志同时确认：

- 安装脚本不是从本地复制
- 仓库不是从 `n3-ui` clone
- Xray Runtime 不是从 `n3-ui` Release 下载

本次还额外检查了仓库中 `n3-ui` 字符串残留。结果是：

- 运行链、安装链、下载链中未发现 `n3-ui` 依赖
- 仅在 `README.md` 和 `README_EN.md` 中保留“项目来源于 n3-ui”这类历史说明

这类说明属于项目背景，不属于运行依赖，不影响独立发行判断。

因此，本版本可以明确声明：

- 不再依赖 `n3-ui` 源码仓库作为安装目标
- 不再依赖 `n3-ui` 安装链
- 不再依赖 `n3-ui` Runtime 发布链

## 3. Fresh Install 测试

测试服务器：

- IP：`<REDACTED_SOURCE_IP>`
- 系统：`Debian GNU/Linux 11 (bullseye)`
- 内核：`5.10.0-32-amd64`

测试开始前确认：

- 服务器已经重装为全新 Debian 11
- `/usr/local/x-ui` 不存在
- 本次安装不使用本地源码
- 本次安装不使用本地 release 文件
- 安装必须完全走公网资源

实际测试流程如下：

1. 确认系统版本与时间。
2. 检查 `/usr/local/x-ui` 不存在。
3. 使用公网安装入口：
   - `bash <(wget -qO- https://raw.githubusercontent.com/torr9522/n5-ui/main/install.sh)`
4. 安装脚本 clone `torr9522/n5-ui`
5. 安装脚本下载 `v0.1.0-beta-simple` 对应 Xray Runtime
6. 构建并启动 `x-ui.service`
7. 检查服务状态、Git 来源、Xray 版本

安装结果：

- `x-ui.service active`
- x-ui 主进程正常
- Xray 子进程正常
- 面板可访问
- 版本显示正常

测试过程中有一个环境偏差：

- 该 Debian 11 极简镜像初始未预装 `wget`

为继续执行“公网安装脚本”测试，先补装了 `wget`。这属于系统镜像差异，不属于 N5-UI 功能故障，也不改变安装链验证结果。

结论：N5-UI 可以从全新 Debian 11 环境完成独立公网安装，安装链闭环成立。

## 4. Xray Runtime 验证

本次验收要求确认 Runtime 已切换为 N5-UI 自有发布资产。

验证结果：

- Runtime 版本：`Xray 26.3.27`
- 二进制路径：`/usr/local/x-ui/bin/xray-linux-amd64`
- 实际版本输出：
  - `Xray 26.3.27 (Xray, Penetrates Everything.)`
- 下载来源：
  - `https://github.com/torr9522/n5-ui/releases/download/v0.1.0-beta-simple/Xray-linux-64.zip`

这说明：

- 运行时来源已经绑定到 `torr9522/n5-ui`
- 不再依赖 `n3-ui` Runtime 发布页
- Fresh install 获得的运行时，与项目验证过的运行时基线一致

本项意义在于，后续关于原生协议、N5 merge、证书功能、Simple Mode 的所有行为，都是在同一个固定 Runtime 版本上验证完成的，而不是依赖“latest”下载结果。

## 5. 原生协议兼容测试

本轮验收不仅检查 N5 功能，还专门验证了“不启用 N5 分流规则时，原生 x-ui 协议能力没有被破坏”。

测试方式：

- 在 Fresh install 环境上通过面板 API 创建临时真实入站
- 使用本机 Xray 客户端生成真实连接
- 访问 `api.ipify.org`
- 校验：
  - 客户端连通性
  - access.log 命中
  - error.log 无协议错误
  - 出口 IP 为服务器公网 IP

### VMess

测试类型：

- `TCP`
- `security = none`

结果：

- 创建成功
- 客户端连接成功
- 出口 IP：`<REDACTED_SOURCE_IP>`
- 观测延迟：约 `264ms`
- access.log 命中：
  - `accepted tcp:api.ipify.org:443 email: accept-vmess@test`
- error.log 未发现协议异常

结论：VMess TCP none 正常。

### VLESS

测试类型：

- `Reality`

结果：

- 创建成功
- Reality 握手成功
- 客户端连接成功
- 出口 IP：`<REDACTED_SOURCE_IP>`
- 观测延迟：约 `5319ms`
- access.log 命中：
  - `accepted tcp:api.ipify.org:443 email: accept-vless@test`
- error.log 未发现：
  - `handshake failed`
  - `invalid key`
  - `invalid certificate`

结论：VLESS Reality 正常。

说明：本次 Reality 首连明显慢于其他协议，但功能完整可用，见“已知问题”章节。

### Trojan

测试类型：

- `TLS`

测试证书：

- `4.527270.xyz`

结果：

- 创建成功
- 客户端连接成功
- 出口 IP：`<REDACTED_SOURCE_IP>`
- 观测延迟：约 `306ms`
- access.log 命中：
  - `accepted tcp:api.ipify.org:443 email: accept-trojan@test`
- error.log 无异常

结论：Trojan TLS 正常。

### Shadowsocks

测试类型：

- method：`aes-256-gcm`

结果：

- 创建成功
- 客户端连接成功
- 出口 IP：`<REDACTED_SOURCE_IP>`
- 观测延迟：约 `298ms`
- access.log 成功命中目标请求
- error.log 无异常

结论：Shadowsocks 原生入站正常。

### 原生协议结论

本轮验证覆盖了：

- VMess
- VLESS Reality
- Trojan TLS
- Shadowsocks

所有测试协议均可正常创建、启动、连接和出站。说明 N5-UI 在独立化、Simple Mode、Xray Merge、证书补丁迁移之后，没有破坏原生 x-ui 入站协议能力。

## 6. N5 功能测试

本次验收同时覆盖 N5 的核心功能链路，目标是确认 N5-UI 不是“只能安装”，而是“安装后核心扩展能力可实际运行”。

### 出口管理

已验证的能力包括：

- SOCKS5 出口创建
- 出口测试
- Simple 模式出口读取
- 临时出口删除

本次真实测试创建的出口：

- 名称：`ACC-SOCKS5-EGRESS`
- 地址：`n5-uics.527270.xyz:38963`
- 自动生成 tag：`n5-egress-0000000001`

出口测试结果：

- `status = success`
- `exitIp = <REDACTED_SOURCE_IP>`

说明：

- 说明 SOCKS5 出口定义、tag 生成、临时 Xray 测试链路正常
- 说明域名形式地址能够被正确用于出口配置

SS、编辑、删除能力在此前阶段测试和回归中已经覆盖；本次最终公网验收重点落在“真实安装后是否能成功创建并测试出口”。

### Simple Mode

本次最终验收覆盖了 Simple Mode 的两条主路径：

1. Simple 出口
2. Simple 规则

Simple 出口验证点：

- 列表可读
- 新建可用
- 测试可用
- 删除可用

Simple 规则验证点：

- 按入口节点创建规则
- 选择流量类型
- 绑定到指定出口
- 底层自动生成 `Traffic Policy + Rule + Binding`

本次最终公网测试使用的是：

- 流量类型：`custom-domain`
- 匹配内容：`full:api64.ipify.org`
- 目标出口：`n5-egress-0000000001`

### Xray Merge

在启用 `n5XrayExtensionEnable=true` 后，验证了以下内容：

- N5 outbound 能被追加到最终 `config.json`
- N5 routing rule 能被追加到最终 `config.json`
- `/n5/api/xray/status` 返回正常
- config history 有实际应用记录

状态接口返回结果要点：

- `enabled = true`
- `outboundCount = 1`
- `routingCount = 1`
- `lastApply.status = applied`
- `hash`、`baseConfigHash`、`extensionConfigHash` 均存在

说明 N5 merge 不是停留在数据库层，而是已经被真实合并到最终运行配置中。

## 7. 真实分流测试

本节记录本次验收最关键的一项：真实流量分流。

测试目标：

- 验证 N5 规则只影响指定流量
- 验证默认流量不被污染

测试链路：

`VMess inbound`
→ `Simple Rule`
→ `N5 egress`
→ `Xray outbound/routing`

具体对象如下：

- 入口：`VMess inbound-21005`
- 规则：`full:api64.ipify.org`
- 出口：`n5-egress-0000000001`
- 出口服务器：`n5-uics.527270.xyz:38963`
- 出口测试 IP：`<REDACTED_SOURCE_IP>`

生成后的最终配置片段确认如下：

outbound：

```json
{
  "protocol": "socks",
  "settings": {
    "servers": [
      {
        "address": "n5-uics.527270.xyz",
        "port": 38963
      }
    ]
  },
  "tag": "n5-egress-0000000001"
}
```

routing：

```json
{
  "domain": ["full:api64.ipify.org"],
  "inboundTag": ["inbound-21005"],
  "outboundTag": "n5-egress-0000000001",
  "type": "field"
}
```

真实流量结果：

- 访问 `api64.ipify.org`
  - 返回：`<REDACTED_SOURCE_IP>`
- 访问 `api.ipify.org`
  - 返回：`<REDACTED_SOURCE_IP>`

access.log 同时给出直接证据：

```text
accepted tcp:api64.ipify.org:443 [inbound-21005 -> n5-egress-0000000001] email: accept-n5@test
accepted tcp:api.ipify.org:443 email: accept-n5@test
```

这说明：

- 指定域名流量成功走 N5 出口
- 普通流量继续走服务器默认 freedom 出口
- N5 分流逻辑没有污染未命中流量
- inboundTag、outboundTag、routing rule 三者映射正确

这是 `v0.1.0-beta-simple` 达到可发布标准的关键依据之一。

## 8. Certificate Phase A

本版本还包含证书侧的 Phase A 能力，即 HTTPS Rescue。

测试域名：

- `3.527270.xyz`
- `4.527270.xyz`

本轮完整测试了以下内容：

1. 证书申请
2. 证书列表查看
3. 设置面板 HTTPS
4. 查看当前 HTTPS 状态
5. 关闭 HTTPS
6. 修复 / 重新设置 HTTPS
7. cert/key 匹配校验

### 证书申请

- 两个域名均申请成功
- 证书目录生成成功：
  - `/etc/x-ui/certs/3.527270.xyz`
  - `/etc/x-ui/certs/4.527270.xyz`
- 列表显示正常

### 设置 HTTPS

使用 `3.527270.xyz` 成功设置面板 HTTPS。

数据库中对应值：

- `webCertFile=/etc/x-ui/certs/3.527270.xyz/fullchain.pem`
- `webKeyFile=/etc/x-ui/certs/3.527270.xyz/privkey.pem`

验证结果：

- `curl -k https://127.0.0.1:36133/` 返回 `200`
- 服务日志显示面板运行于 HTTPS

### 查看 HTTPS 状态

状态页可正常显示：

- HTTPS 是否启用
- 证书路径
- 私钥路径
- 文件存在状态
- openssl 解析信息
- 证书有效期
- cert/key 匹配状态

### 关闭 HTTPS

执行关闭后：

- `webCertFile` 清空
- `webKeyFile` 清空
- 原证书文件保留
- HTTP 访问恢复正常
- HTTPS 不再可用

说明关闭逻辑只处理设置项，不会删除证书文件，符合 Rescue 能力设计。

### 修复 HTTPS

使用 `4.527270.xyz` 执行修复后：

- `webCertFile=/etc/x-ui/certs/4.527270.xyz/fullchain.pem`
- `webKeyFile=/etc/x-ui/certs/4.527270.xyz/privkey.pem`
- HTTPS 访问恢复成功

### cert/key 错误校验

测试方式：

- 人为将 `3.527270.xyz` 的证书与 `4.527270.xyz` 的私钥组合保存

结果：

- 被正确拒绝
- 返回“证书和私钥不匹配或无法解析”
- 数据库设置未被污染

### Certificate Phase A 结论

本阶段迁移的 HTTPS Rescue 能力已经达到可用状态：

- 证书申请可用
- 面板 HTTPS 设置可用
- HTTPS 状态查看可用
- HTTPS 关闭可用
- HTTPS 修复可用
- cert/key 组合校验可用

## 9. 已知问题

本次验收过程中发现的问题如下。

### 1. HTTPS 修复首次可能出现 `database is locked`

现象：

- 在执行“修复 / 重新设置面板 HTTPS”时，首次操作出现 `database is locked`

结果：

- 立即重试后成功
- 不影响最终修复结果

判断：

- 这是一个轻微的数据库锁竞争问题
- 影响的是修复动作稳定性，不影响主功能正确性

### 2. Reality 首连延迟较高

现象：

- VLESS Reality 测试首连延迟约 `5319ms`

结果：

- 连接成功
- 握手成功
- 无证书或密钥错误

判断：

- 功能正常
- 更像网络环境、目标站点、TLS/Reality 首连路径导致的延迟差异
- 当前不构成发布阻断

### 3. Debian 极简环境依赖差异

现象：

- Fresh install 前系统未自带 `wget`

说明：

- 这是测试镜像差异
- 不属于 N5-UI 业务功能故障
- 但说明安装文档中仍应明确依赖前提，或在脚本中更友好地处理缺失依赖

## 10. 当前版本能力总结

`N5-UI v0.1.0-beta-simple` 当前已确认具备如下能力。

### N5 扩展能力

- 出口线路管理
- SOCKS5 出口
- Shadowsocks 出口
- 出口测试
- Simple Mode
- Simple 出口规则
- Traffic Policy
- Xray Merge
- 真实分流

### 原生协议能力

- VMess
- VLESS Reality
- Trojan
- Shadowsocks

### 证书能力

- Certificate Phase A
- HTTPS Rescue
- HTTPS 状态查看
- HTTPS 关闭
- HTTPS 修复
- cert/key 匹配校验

### 发行能力

- 独立源码仓库
- 独立安装链
- 独立 Runtime 发布链
- 独立版本标识
- 完整文档体系

## 11. 发布结论

最终结论：

`N5-UI v0.1.0-beta-simple` 已达到独立发行标准。

判定依据如下：

1. 已具备独立源码仓库：`torr9522/n5-ui`
2. 已具备独立安装入口：`raw.githubusercontent.com/torr9522/n5-ui/main/install.sh`
3. 已具备独立 clone 来源：`https://github.com/torr9522/n5-ui.git`
4. 已具备独立 Runtime 发布链：`torr9522/n5-ui Releases`
5. 已固定运行时版本：`Xray 26.3.27`
6. 已完成 Fresh install 测试
7. 已完成原生协议兼容测试
8. 已完成 N5 出口与真实分流测试
9. 已完成 Certificate Phase A 功能测试
10. 已完成日志审计与临时测试数据清理

本次验收结束时的环境状态：

- `x-ui.service active`
- 面板 HTTPS 正常运行
- `n5XrayExtensionEnable=false`
- 临时测试入站已删除
- 临时 N5 出口、策略、规则、绑定已删除
- 证书测试文件保留，便于后续运维复查

因此，可以将 `v0.1.0-beta-simple` 视为：

- N5-UI 第一个可公网安装的独立版本
- 第一个完成独立 Runtime 切换的正式基线版本
- 第一个同时覆盖原生协议、N5 扩展、证书 Rescue 的完整验收版本

后续版本工作可在此基线上继续推进，但不影响本版本作为“独立发行起点”的成立。
