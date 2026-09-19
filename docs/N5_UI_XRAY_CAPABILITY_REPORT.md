# N5-UI Xray Capability Report

日期：2026-08-09

## 测试对象

真实 Xray 二进制：

```text
/tmp/n5-xray-capability/xray
```

版本：

```text
Xray 26.5.3
228f1e1-dirty
go1.26.0 linux/amd64
```

来源：

```text
https://github.com/torr9522/n3-ui/releases/download/n3-ui-assets/xray-linux-amd64.zip
```

测试命令：

```bash
N5_XRAY_TEST_BINARY=/tmp/n5-xray-capability/xray \
go test ./web/service/n5 -run TestXrayRuntimeCapabilities -v
```

测试实现：

- `web/service/n5/xray_capability_test.go`

每个用例都会执行：

```bash
xray run -test -c <temporary-config>
```

## 测试结果

| 测试项 | 结果 |
|---|---|
| outbound tag | PASS |
| balancer selector | PASS |
| strategy.random | PASS |
| fallbackTag + 无 observatory | FAIL，符合预期 |
| fallbackTag + observatory | PASS |
| strategy + fallbackTag + observatory | PASS |

## 1. Outbound Tag

测试配置包含：

```json
{
  "protocol": "freedom",
  "settings": {},
  "tag": "n5-egress-0000000001"
}
```

结果：

```text
Configuration OK.
```

结论：

- N5 固定长度 outbound tag 可以被真实 Xray 加载。

## 2. Balancer Selector

测试配置包含：

```json
{
  "tag": "n5-pool-0000000001",
  "selector": [
    "n5-egress-0000000001",
    "n5-egress-0000000002"
  ]
}
```

结果：

```text
Configuration OK.
```

结论：

- balancer 的 `tag` 和 `selector` 字段结构可被真实 Xray 接受。
- selector 仍然是前缀匹配，因此固定长度 ID 方案仍然必须保留。

## 3. Balancer Strategy

测试配置包含：

```json
{
  "strategy": {
    "type": "random"
  }
}
```

结果：

```text
Configuration OK.
```

结论：

- 当前 runtime Xray 接受 `strategy.type=random`。
- N5 当前允许的 `random` 策略可以进入 Phase 2.4。

## 4. fallbackTag

### 4.1 无 observatory

仅增加：

```json
{
  "fallbackTag": "n5-egress-0000000003"
}
```

真实 Xray 失败：

```text
Failed to start:
main: failed to create server >
core: not all dependencies are resolved.
```

结论：

- `fallbackTag` 不是独立字段。
- 运行时会要求 observatory 依赖。

### 4.2 配置 observatory

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

结果：

```text
Configuration OK.
```

结论：

- `fallbackTag` 在 observatory 配置存在时可以通过真实 Xray 配置测试。
- Phase 2.4 merge 层必须检查：
  - `fallbackTag` 是否启用
  - observatory 是否存在
  - subjectSelector 是否覆盖候选 outbound

## 5. N5 Tag 方案验证

当前 N5 tag：

```text
n5-egress-0000000001
n5-egress-0000000010
n5-pool-0000000001
n5-pool-0000000010
```

测试验证多个 ID 两两之间不存在 selector 前缀误匹配。

对应测试：

- `TestN5StableTagsDoNotPrefixCollide`

## 6. Capability Test 运行方式

没有真实 Xray 时：

```bash
go test ./...
```

capability test 会跳过：

```text
N5_XRAY_TEST_BINARY is not set
```

有真实 Xray 时：

```bash
N5_XRAY_TEST_BINARY=/absolute/path/to/xray \
go test ./web/service/n5 -run TestXrayRuntimeCapabilities -v
```

## 7. Phase 2.4 结论

当前 runtime Xray 已验证：

- outbound tag：可用
- balancer selector：可用
- random strategy：可用
- fallbackTag：可用，但必须配置 observatory

因此 Phase 2.4 可以继续设计 merge 接入，但不能把 fallbackTag 当作无依赖字段直接生成。

