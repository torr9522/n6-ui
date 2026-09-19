# N5-UI Runtime

更新时间：2026-09-06
适用版本：`v0.2.0`

## 1. 目标

N5-UI 从本阶段开始固定自己的 Xray Runtime，不再依赖 `torr9522/n3-ui` 的 Runtime 发布资产。

运行兼容层仍然保持：

- 服务名：`x-ui`
- systemd：`x-ui.service`
- 二进制名：`x-ui`
- Xray 二进制路径：`/usr/local/x-ui/bin/xray-linux-*`

## 2. 固定 Runtime 基线

- N5 版本：`v0.2.0`
- 固定 Xray 版本：`26.5.3`
- Xray binary SHA256：`128f9c34811ee74b3770eef7010d011e3946e85dfab28f2ed1804e380461b05e`
- 参考运行验证日期：2026-08-18
- 已验证架构：
  - `amd64`

## 3. Release 资产格式

N5-UI Runtime 资产采用以下固定文件名：

- `Xray-linux-64.zip`

仓库内本地打包目录：

- `releases/Xray-linux-64.zip`

GitHub Release 目标路径格式：

```text
https://github.com/torr9522/n5-ui/releases/download/v0.2.0/<asset>
```

示例：

```text
https://github.com/torr9522/n5-ui/releases/download/v0.2.0/Xray-linux-64.zip
```

## 4. 当前下载链路

### 4.1 安装脚本

`install.sh` 的 Xray Runtime 处理顺序：

1. 优先使用安装源码目录内的 `releases/` 固定资产
2. 其次使用 `/usr/local/x-ui/releases/` 固定资产
3. 如果本地没有资产，则下载 `torr9522/n5-ui` 对应 release 资产
4. 不再回退到 `torr9522/n3-ui`
5. 不再回退到 `XTLS/Xray-core` 官方 release
6. arm64 不纳入本次 runtime 切换，安装脚本直接退出

### 4.2 Web 面板更新 Xray

`web/service/server.go` 的更新流程同样改为：

1. 优先读取本地 `releases/` 固定资产
2. 如果本地没有，再读取 `XUI_RELEASES_BASE`
3. 下载失败直接报错

## 5. 版本绑定

`config/version` 采用键值格式：

```text
version=v0.2.0
xray=26.5.3
```

当前读取规则：

- `config.GetVersion()` 返回 `version`
- `config.GetXrayRuntimeVersion()` 返回 `xray`

## 6. 发布要求

要让公网一键安装完全可用，`torr9522/n5-ui` 的 `v0.2.0` release 至少需要包含：

- `Xray-linux-64.zip`

如果未来新增 `INSTALL_MODE=package` 的发布方式，还应补充：

- `x-ui-linux-amd64.tar.gz`

## 7. 风险说明

- 当前固定 Runtime 策略已经去除对 `n3-ui` 和官方回退的运行依赖
- 如果 GitHub release 未上传上述资产，公网安装会失败
- 本地源码安装可通过仓库内 `releases/` 资产完成独立安装验证
- 本次 runtime 切换仅支持 amd64/x86_64

## 8. Why Custom Runtime

N5-UI fixes the runtime to the validated N5 custom Xray 26.5.3 amd64 asset so
that routing behavior and Access IP log marker compatibility remain stable.
Use redacted examples such as `[inbound-12345]` in public documentation; do not
publish real access logs or source IPs.
