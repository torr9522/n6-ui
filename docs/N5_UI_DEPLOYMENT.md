# N5-UI Deployment

更新时间：2026-09-06
适用版本：`v0.2.0`

## 1. 部署原则

- N5-UI 是基于 `n3-ui` 的个人使用型二开面板，但安装源、发布源、更新源已独立切换到 `torr9522/n5-ui`。
- 运行兼容层继续保持 `x-ui`：
  - 服务名：`x-ui`
  - systemd：`x-ui.service`
  - 安装命令：沿用 `x-ui` 兼容入口，但默认源为 `n5-ui`
  - 默认运行目录：`/usr/local/x-ui`
  - 默认数据库：`/etc/x-ui/x-ui.db`
- 发布整理阶段不改：
  - 安装脚本逻辑
  - systemd 逻辑
  - Xray 核心逻辑
  - x-ui 原兼容层命令名

## 2. 安装方式

### 2.1 一键安装

当前默认安装命令：

```bash
bash <(curl -Ls https://raw.githubusercontent.com/torr9522/n5-ui/v0.2.0/install.sh)
```

英文安装：

```bash
bash <(curl -Ls https://raw.githubusercontent.com/torr9522/n5-ui/v0.2.0/install_en.sh)
```

说明：

- `x-ui` 仍然是运行时命令名，但安装脚本默认拉取 `torr9522/n5-ui`。
- 安装脚本会把程序部署到 `/usr/local/x-ui`。
- 本版本推荐全新安装，暂不承诺旧版本原地升级兼容性。

### 2.2 Go 版本要求

- 源码安装最低要求：`Go 1.16+`
- 当前安装脚本内置自动安装版本：`Go 1.22.7`
- 如果系统已存在更高版本 Go，安装脚本会直接复用

### 2.3 本地源码构建

```bash
cd /root/n5-ui
go test ./...
go build -o x-ui .
```

说明：

- 构建产物名仍然是 `x-ui`，不是 `n5-ui`。

## 3. 运行目录与关键文件

| 路径 | 说明 |
|---|---|
| `/usr/local/x-ui/` | 程序主目录 |
| `/usr/local/x-ui/x-ui` | 主程序二进制 |
| `/usr/local/x-ui/bin/config.json` | 当前最终 Xray 配置 |
| `/usr/local/x-ui/bin/xray-linux-amd64` | Xray 主程序 |
| `/etc/x-ui/x-ui.db` | SQLite 数据库 |
| `/etc/x-ui/certs/` | 托管证书目录 |
| `/usr/bin/x-ui` | 运维管理脚本入口 |
| `/etc/systemd/system/x-ui.service` 或系统加载副本 | systemd 服务 |

## 4. 升级策略

本版本暂不承诺旧版本原地升级兼容性，也不保证直接覆盖安装不会产生数据或配置问题。

如需更换版本，推荐：

1. 完整备份 `/etc/x-ui`、数据库和重要配置。
2. 使用全新 Debian 11 amd64/x86_64 系统安装 `v0.2.0`。
3. 根据实际兼容性和备份内容，由管理员自行评估数据恢复。

正式 Upgrade Gate 将作为独立的发布后兼容性实验执行。

## 5. 备份

### 5.1 升级前备份

推荐目录：

- `/root/n5-ui-backups/`

推荐内容：

- `/usr/local/x-ui/`

推荐命名：

```bash
tar -czf /root/n5-ui-backups/x-ui-$(date +%Y%m%d-%H%M%S).tar.gz /usr/local/x-ui
```

### 5.2 数据与配置保护

即使备份程序目录，也建议单独保护：

- `/etc/x-ui/x-ui.db`
- `/usr/local/x-ui/bin/config.json`

## 6. 恢复

### 6.1 二进制和前端恢复

```bash
systemctl stop x-ui
tar -xzf /root/n5-ui-backups/<backup>.tar.gz -C /
systemctl start x-ui
systemctl is-active x-ui
```

### 6.2 仅回滚程序，不回滚数据

适用于：

- 代码升级异常
- 数据库不希望回退

做法：

- 只恢复 `/usr/local/x-ui/`
- 不覆盖 `/etc/x-ui/x-ui.db`
- 不覆盖当前 `bin/config.json`，除非确认运行配置损坏

## 7. 测试服务器部署流程

当前项目使用过的测试部署基线：

- 目标目录：`/usr/local/x-ui`
- 备份目录：`/root/n5-ui-backups`
- 服务名：`x-ui`

### 7.1 部署前

1. 备份当前 `/usr/local/x-ui`
2. 确认不覆盖数据库与最终配置
3. 同步最新源码

### 7.2 重点同步内容

- `database/model/n5/`
- `database/n5_phase2.go`
- `web/controller/n5/`
- `web/service/n5/`
- `web/html/n5/`
- `web/web.go`
- `web/html/xui/common_sider.html`
- `web/html/xui/setting.html`
- `x-ui.sh`
- 其他本版本实际修改文件

### 7.3 远端编译与重启

```bash
cd /usr/local/x-ui
go test ./...
go build -o x-ui.new .
mv x-ui.new x-ui
systemctl restart x-ui
systemctl is-active x-ui
```

### 7.4 页面验证

至少验证：

- `/xui/`
- `/n5/egress`
- `/n5/pools`
- `/n5/traffic-policy`
- `/n5/simple`
- `/n5/simple/rules`
- `/n5/xray-status`

### 7.5 N5 功能验证

- N5 左侧菜单是否存在
- 出口列表能否加载
- Simple 出口能否加载
- Simple 规则能否加载
- `n5XrayExtensionEnable` 设置项是否存在
- 旧入站功能是否仍正常

## 8. 发布前检查

### 8.1 本地检查

- `git status`
- `go test ./...`
- `go build -o x-ui.new .`
- 文档是否完整
- 未提交文件中是否包含敏感信息

### 8.2 远端检查

- `systemctl is-active x-ui` 必须为 `active`
- 如开启 N5 merge，`/n5/api/xray/status` 应能返回状态
- `bin/config.json` 需可通过 `xray.TestConfig`

## 9. 敏感信息处理建议

发布前不要把以下内容提交到公开仓库：

- 服务器密码
- 私有域名证书内容
- `/etc/x-ui/x-ui.db`
- 真实运行态 `bin/config.json`
- 含公网 IP、面板地址、测试结果的临时回归产物
- 含测试账号密码的临时脚本

## 10. 与当前版本对应的运维入口

### 面板设置

- `web/html/xui/setting.html`
- 开关：`N5 Xray Extension Enable`

### 脚本证书救援

- 命令：`x-ui`
- 菜单：
  - `16` 证书管理
  - 证书管理子菜单中 `6-8` 为 Phase A HTTPS Rescue

### 服务控制

```bash
systemctl status x-ui
systemctl restart x-ui
systemctl stop x-ui
systemctl start x-ui
```

## 11. 当前版本发布建议

当前发布名：

- `v0.2.0`

当前发布说明核心点：

- 完成 N5 品牌迁移
- 完成高级出口 / 线路池 / 分流 / merge
- 完成轻量检测
- 完成 Simple Mode
- 完成 HTTPS Rescue Phase A
