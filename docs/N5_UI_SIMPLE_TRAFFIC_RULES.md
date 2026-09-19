# N5-UI Simple 分流规则设计

## 1. 模块关系

`N5出口`
=
定义出口线路。

`分流规则`
=
定义哪些流量属于某个分类。

`出口规则`
=
入口 + 分流规则组 + 出口。

关系如下：

```text
分流规则组
   ↓ 创建出口规则时复制
执行 Snapshot
   ↓
TrafficPolicy / Binding
   ↓
Xray Routing
```

## 2. Snapshot 模式

Simple 分流规则采用 Snapshot 模式。

创建出口规则时，系统会把当前分流规则组内容复制成独立执行策略。

后续修改源规则组：

- 不会自动更新已有 execution snapshot
- 不会自动修改已有 TrafficPolicyRule
- 不会自动修改运行中的 Xray routing
- 删除源规则组不会导致已有 snapshot 失效

核心一句：

> Simple 分流规则采用 Snapshot 模式：创建出口规则时复制规则组内容，后续对规则组的修改不会自动同步到已有执行规则。

## 3. 示例

例如规则组：

```text
AI分流

openai.com
claude.ai
```

创建出口规则后，系统会生成一份 execution snapshot。

如果之后再给 AI 分流增加：

```text
accounts.google.com
```

结果是：

- 源规则组会增加 `domain:accounts.google.com`
- 已有 snapshot 不会增加该规则

只有以后重新创建新的出口规则时，才会使用最新规则组内容。

## 4. 为什么这样设计

保持简单：

- 避免修改一个规则组影响多个正在运行的入口
- 不做复杂批量同步
- 不增加数据库引用体系
- 保持 N5-UI 轻量、自用定位

## 5. 已实测确认

已在测试服务器上实测确认：

- 新增前 snapshot 没有 `accounts.google.com`
- 源 group 新增后出现 `domain:accounts.google.com`
- 已有 execution snapshot 没有自动新增
- 运行中的 `config.json` routing 没有自动变化
