# Development Skill Tree

This skill tree is current for `v0.2.0` Stable maintenance and later N5
development.

## Backend

- Go
- Gin
- Go embed
- SQLite 数据库
- GORM models and migrations
- SQLite transaction boundaries
- SQLite busy/locked error handling
- systemd 部署与日志排查

## Frontend

- Vue 2
- HTML template
- Ant Design Vue
- 前端状态与弹窗交互
- 前端 UAT / Headless Chromium
- Playwright browser UAT
- mobile viewport UAT

## Xray

- Xray inbound 配置
- Xray outbound 配置
- Xray routing priority
- Xray config generation and `run -test`
- Xray access log interpretation
- VMess 分享链接格式
- VLESS 分享链接格式
- Trojan 分享链接格式
- Shadowsocks 分享链接格式
- TLS / SNI / Host Header 差异

## Release And Operations

- GitHub 发布流程
- Git 分支与回滚
- annotated Git tags
- GitHub Release assets
- release package SHA256 verification
- 安装脚本风险点
- Go build 产物验证
- systemd 服务安装与 reboot 验证
- Debian clean install validation
- TCP/UDP end-to-end validation
- DB consistency checks
- secret scanning and redaction

## Required Verification Skills

- 检查 Go embed 后 binary 是否包含最新 UI 字符串。
- 使用浏览器真实路径验证复制链接和二维码。
- 区分数据库配置、Xray 运行配置、分享链接临时导出字段。
- 验证 install.sh、release asset、源码 build 三种安装路径的差异。
- 维护 Simple/Advanced ownership 边界，禁止 Advanced 破坏 Simple-managed state。
- 验证 deterministic Simple routing priority，不允许 creation order 决定结果。
- 发布前确认 clean install、browser、TCP、UDP、reboot、DB consistency 全部通过。
