# N2-UI Baseline

## 项目来源

本项目基于 n-ui 当前本地源码基线创建，作为独立二开项目 `n2-ui` 使用。

## 当前功能

- Custom Share Address Override
- 在入站分享功能中支持临时输入自定义分享地址。
- 仅影响本次分享链接生成，不写入数据库，不修改 Xray 运行配置。
- VMess 保持传统 `vmess://` + Base64(JSON) 格式。

## 已验证协议

- VMess
- VLESS
- Shadowsocks

## Trojan 状态

当前 UI 新增入站不支持创建 Trojan，本轮 UAT 未覆盖 Trojan。

## 生产 UAT

- 生产 UAT 服务器：`<TEST_SERVER>`
- 域名测试：`xgaws.527270.xyz`

## 验证内容

- `go build` 通过。
- systemd 生产式安装通过。
- reboot 后自启动通过。
- VMess 自定义域名可连接。
- 浏览器真实操作通过。
- 原始分享链接、原始二维码、自定义分享链接、自定义二维码均通过浏览器路径验证。

## 后续二开注意

- 生产模式使用 Go embed，改 HTML/JS 后必须重新编译 binary。
- 不要只替换 `web/html` 或 `web/assets` 文件后认为生产已生效。
- `install.sh` 当前仍可能使用 release asset，发布流程需单独规划。
- 新功能应优先在本地 build、binary 字符串检查、浏览器 UAT、systemd reboot UAT 后再发布。
