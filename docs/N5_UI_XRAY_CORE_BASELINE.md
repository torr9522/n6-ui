# N5-UI Xray Core Baseline

日期：2026-08-08

## 结论摘要

当前 N5-UI 面板和运行时 Xray 不是同一个构建产物：

- 面板 `x-ui` 由根目录 `go.mod` 编译。
- Xray 运行二进制由安装脚本从 release zip 下载并提取。
- 根项目当前 Go 依赖解析到 `github.com/xtls/xray-core v1.4.2`。
- 仓库内 `xray-core/` 是独立 Go module，不通过根项目 `replace` 参与面板编译。
- 当前工作区没有 `bin/xray-linux-*` 文件，因此不能从本地运行文件确认某个实际 release 二进制的版本。

Phase 2.4 之前必须把“运行二进制对应的 Core 配置语义”作为发布契约固定下来。

## 1. 当前运行 Xray 来源

### 1.1 面板调用路径

`xray.GetBinaryPath()` 返回：

```text
bin/xray-{GOOS}-{GOARCH}
```

在 Linux amd64 环境中，实际路径为：

```text
bin/xray-linux-amd64
```

依据：

- `xray/process.go`
  - `GetBinaryName()`
  - `GetBinaryPath()`
  - `Process.Start()`
  - `TestConfig()`

systemd 工作目录为：

```text
/usr/local/x-ui/
```

依据：

- `x-ui.service`

因此正式安装后的默认运行路径为：

```text
/usr/local/x-ui/bin/xray-linux-amd64
```

### 1.2 安装来源

`install.sh` 的 `sync_default_xray_assets()`：

1. 优先读取本地 `releases/xray-linux-{arch}.zip`
2. 其次读取 `/usr/local/x-ui/releases/xray-linux-{arch}.zip`
3. 否则从 `${XUI_RELEASES_BASE}/xray-linux-{arch}.zip` 下载
4. 从压缩包提取名为 `xray` 的文件
5. 写入 `/usr/local/x-ui/bin/xray-linux-{arch}`

默认 release 地址来自：

```text
https://github.com/torr9522/n3-ui/releases/download/n3-ui-assets
```

依据：

- `install.sh`
  - `XUI_RELEASES_BASE`
  - `sync_default_xray_assets()`

### 1.3 面板运行时是否自动构建 Xray

否。

安装脚本的源码安装流程只构建：

```text
x-ui
```

不会执行：

```text
go build ./xray-core
```

也没有发现根项目通过 `go:generate`、Makefile 或其他脚本自动从 `xray-core/` 子树构建运行二进制。

## 2. 当前构建 Xray 来源

### 2.1 根项目编译来源

根项目模块：

```text
module x-ui
```

根 `go.mod` 依赖：

```text
github.com/xtls/xray-core v1.4.2
```

当前环境解析结果：

```text
github.com/xtls/xray-core v1.4.2
/root/go/pkg/mod/github.com/xtls/xray-core@v1.4.2
```

这套依赖主要被面板代码用于：
- Xray API 类型
- stats gRPC 类型
- 面板侧配置/进程控制相关编译

它不会生成安装目录中的 Xray 可执行文件。

### 2.2 仓库内 `xray-core/` 子树

仓库内存在独立模块：

```text
xray-core/go.mod
module github.com/xtls/xray-core
```

该目录有自己的依赖、Go 版本和源码树。

根项目没有发现：

```go
replace github.com/xtls/xray-core => ./xray-core
```

因此：

- 根项目 `go test ./...` 不会把 `xray-core/` 作为根 module 的依赖替换源。
- 根项目构建不会自动使用 `xray-core/` 子树编译面板依赖。
- 安装脚本也不会从该子树构建 Xray 运行程序。

## 3. 配置结构基线差异

根项目依赖的 `v1.4.2` 配置解析结构与仓库内 `xray-core/` 子树存在差异。

重点差异：

### 根模块缓存的 v1.4.2

`routing.balancers[]` 的基础结构主要包括：

```json
{
  "tag": "balancer-tag",
  "selector": ["outbound-prefix"]
}
```

### 仓库内 `xray-core/` 子树

`xray-core/infra/conf/router.go` 的 `BalancingRule` 支持：

```json
{
  "tag": "balancer-tag",
  "selector": ["outbound-prefix"],
  "strategy": {
    "type": "random"
  },
  "fallbackTag": "fallback-outbound"
}
```

因此 N5 当前生成的：

```json
"strategy": {
  "type": "..."
},
"fallbackTag": "..."
```

只有在运行二进制确实基于支持这些字段的 Core 构建时，才能保证语义生效。

## 4. 当前工作区实况

已检查：

- `/root/n5-ui/bin/` 当前只有：
  - `config.json`
  - `geoip.dat`
  - `geosite.dat`
- 当前工作区没有：
  - `bin/xray-linux-amd64`
  - `bin/xray-linux-arm64`
- 当前环境没有发现 PATH 中的 `xray` 命令。

所以本次审计可以确认：

- 面板的二进制路径和安装来源。
- 根项目编译时使用的 Core module 版本。
- 仓库内 Core 子树与根 module 的关系。

但不能从当前工作区确认：

- 线上服务器实际下载的 Xray release 具体版本。
- 线上 release zip 是否来自仓库内 `xray-core/` 子树的构建。

## 5. Phase 2.4 基线决策

在接入 N5 主配置前，发布流程必须选择唯一方案。

### 方案 A：继续使用 release zip 运行 Xray

要求：
- 明确 release zip 的构建 commit 或版本号。
- 将其配置 schema 作为 N5 生成器的唯一校验标准。
- 不以根项目 `go.mod` 的 v1.4.2 推断线上二进制能力。

### 方案 B：从仓库内 `xray-core/` 子树构建 Xray

要求：
- 增加明确、可重复的 Xray 构建流程。
- 记录构建 commit、版本、目标架构和校验值。
- 发布产物必须由安装脚本或 release 流程明确引用。

### 当前建议

在没有完成构建链统一前：

- N5 不应依赖 `strategy` 和 `fallbackTag` 的高级语义。
- Phase 2.4 应先做完整配置 schema 验证。
- 运行前必须保存 Core 版本和配置 hash。

## 6. 风险

### 高风险

- 线上 Xray 二进制版本未被项目源码锁定。
- 根模块 v1.4.2 与仓库内 `xray-core/` 子树配置能力不一致。
- N5 生成字段可能被旧 Core 忽略。

### 中风险

- 当前工作区无法执行真实 `xray run -test`，只能做结构和代码级审计。
- release zip 下载源可变，缺少构建 commit/hash 记录。

## 7. Phase 2.4 前置验收条件

必须满足：

1. 明确唯一 Xray Core 运行基线。
2. 记录 release 或自构建二进制版本、commit、SHA256。
3. 用该二进制执行完整合并配置的 `run -test`。
4. 确认 `strategy`、`fallbackTag`、`balancerTag` 实际生效。
5. 将 Core 基线写入 `n5_xray_config_history` 或等价审计记录。

