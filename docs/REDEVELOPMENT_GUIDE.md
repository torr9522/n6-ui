# Redevelopment Guide

## Clone

```bash
git clone git@github.com:torr9522/n5-ui.git
cd n5-ui
```

如使用 HTTPS：

```bash
git clone https://github.com/torr9522/n5-ui.git
cd n5-ui
```

## Build

```bash
go build -o x-ui .
```

## Local Run

建议使用临时数据库和临时端口做本地验证，避免覆盖生产 `/etc/x-ui/x-ui.db`。

```bash
./x-ui
```

如需生产式验证，应使用独立测试服务器，按 `/usr/local/x-ui`、`/etc/x-ui`、`/usr/bin/x-ui`、systemd 的安装结构部署。

## Verify Go Embed

修改 `web/html` 或 `web/assets` 后必须重新编译：

```bash
go build -o /tmp/x-ui-check .
grep -aE "复制自定义分享链接|自定义二维码|留空则使用原始地址|normalizeShareAddress" /tmp/x-ui-check
```

如果 binary 中没有新字符串，说明生产运行不会加载新 UI。

## UAT

建议最小 UAT 流程：

1. `go build`。
2. 检查 binary 中是否包含新 UI 字符串。
3. 生产式 systemd 安装到测试服务器。
4. 浏览器登录面板。
5. 通过 UI 创建或复用入站。
6. 验证原始分享链接。
7. 验证自定义分享链接。
8. 验证原始二维码和自定义二维码。
9. 执行 reboot。
10. 验证 x-ui、Xray、入站端口和面板登录均恢复正常。

## Avoid Pushing To Other Repositories

本项目 remote 必须指向 `torr9522/n5-ui`。

```bash
git remote -v
git branch --show-current
```

推送前确认不要出现 `torr9522/n3-ui` 或 `torr9522/n2-ui`。

## Feature Branch

```bash
git checkout main
git pull origin main
git checkout -b feature/your-change
```

功能完成后先本地 build 和 UAT，再合并或提交 PR。

## Rollback

优先使用 `git revert` 回滚已提交变更：

```bash
git revert <commit>
```

不要在共享分支上使用 `git reset --hard` 或 force push，除非明确确认可以重写历史。
