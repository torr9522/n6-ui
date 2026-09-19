# n6-ui v0.1.0 Stable

## 主要更新

### 1. N5 路由系统

- N5 出口
- 出口规则
- 分流规则
- 自定义域名
- 自定义规则组
- ALL、AI、Game、Streaming

### 2. Deterministic Routing Priority

Simple routing 的优先级固定为：

```text
direct exact
> direct suffix
> direct keyword
> direct regexp
> custom group
> AI
> Game
> Streaming
> ALL
```

创建顺序不再决定 Simple routing 的最终优先级。

### 3. Rule UX

- 规则组一级和二级页面
- matcher 中文化
- 真实规则组名称
- 规则冲突提示

### 4. Data Safety

- 引用中的 egress 删除保护
- inbound 删除清理
- Advanced/Simple ownership protection
- Simple mutation transaction
- disabled builtin protection
- SQLite busy handling

### 5. Mobile

- 320/390 viewport 已验证
- 操作列可触达
- modal footer 可触达

### 6. Runtime

- Xray 26.5.3 amd64
- SHA256: `128f9c34811ee74b3770eef7010d011e3946e85dfab28f2ed1804e380461b05e`

本 Stable Runtime 正式支持 Debian 11 amd64/x86_64。ARM64 Stable Runtime 不在本次承诺范围内。

### 7. Validation

- 全新 Debian 11 安装通过
- 真实 reboot 通过
- Desktop Browser 通过
- Mobile 320/390 通过
- VMess TCP 通过
- VLESS TCP 通过
- Reality configuration 通过
- TCP E2E 通过
- UDP E2E 通过
- Subscription 通过
- Access IP 通过
- DB consistency 通过，检查项全部为 0

### 8. Upgrade

本版本推荐全新安装。

本版本暂不承诺旧版本原地升级兼容性。已有旧版本的用户，请先备份 `/etc/x-ui`、数据库和重要配置；Upgrade Gate 将在发布后独立验证。

### 9. Known Non-blocker

`/favicon.ico` 可能返回 404。这是静态资源视觉问题，不影响核心功能和运行流程。
