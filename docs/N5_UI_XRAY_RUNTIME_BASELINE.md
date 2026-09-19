# N5-UI Xray Runtime Baseline

日期：2026-08-09

## 结论

当前工作区没有正在运行的 Xray 进程，也没有已安装的：

```text
/root/n5-ui/bin/xray-linux-amd64
/usr/local/x-ui/bin/xray-linux-amd64
```

因此本次不能从本机活动进程读取线上实例版本。

根据当前运行代码和安装流程，N5-UI 的正式运行基线为：

```text
/usr/local/x-ui/bin/xray-linux-{arch}
```

本次使用 N5 自有 release 地址下载并验证了 amd64 Xray：

```text
来源：
https://github.com/torr9522/n5-ui/releases/download/v0.1.1-runtime-26.5.3-amd64/Xray-linux-64.zip

版本：
Xray 26.5.3

构建标识：
228f1e1-dirty

平台：
linux/amd64
```

## 1. Xray 路径

面板代码通过：

```go
xray.GetBinaryPath()
```

生成相对路径：

```text
bin/xray-linux-amd64
```

依据：

- `xray/process.go`

systemd 工作目录：

```text
/usr/local/x-ui/
```

依据：

- `x-ui.service`

所以部署后的默认绝对路径为：

```text
/usr/local/x-ui/bin/xray-linux-amd64
```

## 2. Xray 来源

安装脚本 `sync_default_xray_assets()`：

1. 检查本地 `releases/xray-linux-{arch}.zip`
2. 检查 `/usr/local/x-ui/releases/xray-linux-{arch}.zip`
3. 否则从 `${XUI_RELEASES_BASE}` 下载
4. 提取压缩包内的 `xray`
5. 写入 `/usr/local/x-ui/bin/xray-linux-{arch}`

依据：

- `install.sh`

## 4. 本次切换范围

- 仅支持 `amd64/x86_64`
- `arm64` 不纳入本次 runtime 切换

本次能力验证使用的临时文件：

```text
/tmp/n5-xray-capability/xray
/tmp/n5-xray-capability/xray-linux-amd64.zip
```

SHA256：

```text
xray：
128f9c34811ee74b3770eef7010d011e3946e85dfab28f2ed1804e380461b05e

zip：
98e1cfe7b8a85d833edcd5101530f2d67609505d832eac15ac2a236f3374bbbe
```

## 3. 面板构建与 Xray 构建关系

根项目构建的是：

```text
x-ui
```

根 `go.mod` 使用：

```text
github.com/xtls/xray-core v1.4.2
```

该依赖用于面板侧 Go 编译，不会自动生成运行用 Xray 二进制。

仓库内的：

```text
xray-core/
```

是独立 Go module，根项目没有使用：

```go
replace github.com/xtls/xray-core => ./xray-core
```

当前安装流程也不会从该子树构建 Xray，而是下载 release zip。

## 4. 支持能力冻结

使用真实命令：

```bash
N5_XRAY_TEST_BINARY=/tmp/n5-xray-capability/xray \
go test ./web/service/n5 -run TestXrayRuntimeCapabilities -v
```

能力结果：

| 能力 | 结果 | 说明 |
|---|---|---|
| outbound tag | 通过 | 带 tag 的 outbound 可加载 |
| balancer selector | 通过 | selector 可加载并建立 balancer |
| balancer strategy | 通过 | `strategy.type=random` 可加载 |
| fallbackTag | 有条件通过 | 必须同时配置 `observatory` |
| strategy + fallbackTag | 有条件通过 | 必须同时配置 `observatory` |

## 5. fallbackTag 前置条件

只配置：

```json
{
  "fallbackTag": "n5-egress-0000000003"
}
```

真实 Xray `run -test` 会失败：

```text
not all dependencies are resolved
```

加入：

```json
{
  "observatory": {
    "subjectSelector": [
      "n5-egress-0000000001",
      "n5-egress-0000000002"
    ],
    "probeUrl": "https://www.google.com/generate_204"
  }
}
```

后，真实 Xray 配置测试通过。

因此 Phase 2.4 如果启用 `fallbackTag`，必须同时满足：

- 运行 Core 支持 balancer fallback
- 最终配置提供有效的 `observatory`
- `subjectSelector` 覆盖需要健康判断的候选出口
- 目标 `fallbackTag` 指向已存在的 outbound 或可解析目标

## 6. Runtime 基线冻结值

Phase 2.4 默认按以下信息进行配置验证：

```text
运行平台：linux/amd64
运行文件：bin/xray-linux-amd64
来源：n5-ui v0.1.1-runtime-26.5.3-amd64 release zip
已验证版本：Xray 26.5.3
fallbackTag：必须配合 observatory
```

如果部署实例的 Xray 版本与上述基线不同，必须重新执行 capability test，不能直接复用本报告结论。

## 7. 风险

### 高风险

- 当前机器没有活动 Xray 实例，无法证明线上服务器正在运行同一版本。
- release zip 来源没有在仓库内固定 commit 和构建流水线。
- 如果线上二进制不是 `26.5.3` 对应能力，`fallbackTag` 语义可能不同。

### 中风险

- `fallbackTag` 会隐式引入 observatory 依赖。
- Phase 2.4 合并层必须在最终配置验证阶段检查 observatory 依赖，而不是只检查 JSON 字段。
