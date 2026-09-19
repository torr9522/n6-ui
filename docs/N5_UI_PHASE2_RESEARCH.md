# N5-UI Phase 2 Research

Research date: August 8, 2026

## 1. Scope and Constraints

Phase 2 is the first feature phase after Phase 1 branding migration.

This phase must design and later implement two modules as one integrated system:

- 出口线路池系统
- 流量分流系统

Required flow:

`入口节点 -> 分流策略 -> 出口线路 -> Xray outbound/routing`

Current project constraints already frozen by local project rules:

- Do not modify `x-ui.sh`, `x-ui_en.sh`, `install.sh`, `install_en.sh`, `x-ui.service`
- Do not change `/xui/` existing API paths
- Do not rename runtime service/binary/path family
- Do not modify `xray/` core package behavior
- Do not directly expand legacy core tables for new business fields

Local reference:

- `docs/N5_UI_DEVELOPMENT_ARCHITECTURE.md`
- `docs/N5_UI_PROJECT_RULES.md`

## 2. Research Sources

### 2.1 Official Xray documentation

- Xray outbound configuration: <https://xtls.github.io/en/config/outbound.html>
- Xray routing configuration: <https://xtls.github.io/en/config/routing.html>
- Xray API configuration: <https://xtls.github.io/en/config/api.html>
- Xray multi-file configuration: <https://xtls.github.io/en/config/features/multiple.html>

### 2.2 Local Xray-core source in current repository

- `xray-core/infra/conf/router.go`
- `xray-core/infra/conf/router_strategy.go`
- `xray-core/app/proxyman/command/command.proto`
- `xray-core/app/router/command/command.proto`
- `xray/process.go`

### 2.3 Local N5-UI current implementation

- `web/service/xray.go`
- `database/db.go`
- `xray/config.go`
- `xray/process.go`

### 2.4 3x-ui implementation reference

- `/root/3x-ui/internal/web/service/outbound_subscription.go`
- `/root/3x-ui/internal/web/service/xray.go`
- `/root/3x-ui/internal/xray/api.go`
- `/root/3x-ui/internal/web/controller/xray_setting.go`
- `/root/3x-ui/internal/web/service/node.go`
- `/root/3x-ui/internal/database/model/model.go`

### 2.5 Other mature panel reference

- Marzban repository: <https://github.com/Gozargah/Marzban>

Marzban is useful as a contrast reference because it already treats multi-node orchestration and custom Xray configuration as first-class concerns, but its operating model is not `x-ui` compatibility oriented.

## 3. Current N5-UI Baseline

Current `n5-ui` is still the older `n3-ui` style architecture:

- final Xray config is built in `web/service/xray.go`
- the panel reads one stored Xray template JSON
- local inbound records are appended into that template
- final config is validated by `xray.TestConfig`
- runtime apply path is full restart, not gRPC hot-reload

Important local evidence:

- `web/service/xray.go` only builds base config plus dokodemo/tunnel routing helper
- `xray/process.go` supports `TestConfig`, but `Process.Start()` writes the running config to fixed path `bin/config.json`
- `database/db.go` currently auto-migrates only legacy core models (`User`, `Inbound`, `Setting`, `AccessIPRecord`)

Conclusion:

- Phase 2 cannot assume 3x-ui style hot outbound/routing management already exists
- Phase 2 should not be designed around modifying legacy inbound/core models
- Phase 2 needs its own data models and its own additive config generation layer

## 4. Technical Findings

## 4.1 Xray outbound dynamic generation

Official docs and `xray-core` confirm that outbound is a top-level Xray object and can be added as an independent config fragment.

Important evidence:

- `xray-core/app/proxyman/command/command.proto` defines `HandlerService.AddOutbound`, `RemoveOutbound`, `AlterOutbound`, `ListOutbounds`
- `3x-ui/internal/xray/api.go` converts outbound JSON into `conf.OutboundDetourConfig`, builds it through vendored `xray-core`, then calls `AddOutbound`
- `3x-ui/internal/web/service/outbound_subscription.go` validates fetched outbound JSON against `xray-core` before keeping it

Design implication:

- N5-UI Phase 2 should store egress lines as independent outbound definitions
- outbound validity must be checked against `xray-core` before enable/apply
- outbound tags must be stable, because routing and balancer references depend on tags

## 4.2 Xray routing rule generation

`xray-core/infra/conf/router.go` shows the real routing object model:

- top-level routing contains `rules`, `domainStrategy`, `balancers`
- each field rule must target either `outboundTag` or `balancerTag`
- `domainStrategy` supports `AsIs`, `IPIfNonMatch`, `IPOnDemand`

Important evidence:

- `RouterConfig` in `xray-core/infra/conf/router.go`
- `parseFieldRule()` rejects a rule if neither `outboundTag` nor `balancerTag` is present
- `3x-ui/internal/xray/api.go` notes `ApplyRoutingConfig` cannot change `routing.domainStrategy/domainMatcher` at runtime
- `xray-core/app/router/command/command.proto` exposes `TestRoute`, `AddRule`, `RemoveRule`, `GetBalancerInfo`, `OverrideBalancerTarget`

Design implication:

- policy routing should be generated as standard Xray field rules, not custom side logic
- policy default egress should be the last generated rule in that policy block
- rule order matters
- any design that requires frequent `domainStrategy` mutation is high risk

## 4.3 Xray balancer design

`xray-core/infra/conf/router.go` and `router_strategy.go` show balancer semantics clearly:

- balancer has `tag`, `selector`, `strategy`, `fallbackTag`
- supported strategies include `random`, `leastping`, `roundrobin`, `leastload`
- `leastload` has structured health/latency related settings
- observatory-related code exists in `xray-core/app/observatory/...`

Important evidence:

- `xray-core/infra/conf/router.go`
- `xray-core/infra/conf/router_strategy.go`
- official routing docs: `BalancerObject`
- `3x-ui/internal/xray/api.go` and `internal/web/controller/xray_setting.go` expose balancer status, override, and route test

Design implication:

- N5-UI should treat “线路池” as an Xray balancer abstraction, not as ad-hoc application-side random selection
- Phase 2 MVP should use simple balancer strategies that do not require health observatory data
- `leastPing` and `leastLoad` should be deferred until health-detection phases

## 4.4 3x-ui outbound implementation

3x-ui provides several useful implementation patterns:

- independent outbound subscription model
- stable tag prefix generation
- validation against vendored `xray-core`
- additive merge into final `outbounds` array
- route test and balancer control endpoints

Important evidence:

- `internal/web/service/outbound_subscription.go` stores outbound lists as JSON, validates them, and maintains stable tag prefixes
- `internal/web/service/xray.go` merges subscription outbounds into the final config
- `internal/web/controller/xray_setting.go` exposes `testOutbound`, `testOutbounds`, `balancerStatus`, `balancerOverride`, `routeTest`

Design implication:

- stable tags are mandatory
- outbound/pool/policy should be independent entities
- test capability should exist outside the legacy inbound CRUD path

## 4.5 3x-ui node binding to outbound

3x-ui also implements “node -> outbound” binding, but its implementation detail should be separated into two parts:

Part A: persistent config injection

- `internal/database/model/model.go` adds `Node.OutboundTag`
- `internal/web/service/xray.go` injects loopback SOCKS inbounds and prepend routing rules for nodes with `OutboundTag`

Part B: temporary probe bridge

- `internal/web/service/node.go` function `withOutboundBridge()` creates a temporary loopback SOCKS inbound
- it prepends one routing rule to either `outboundTag` or `balancerTag`
- it uses the running Xray API to add/remove the bridge dynamically during probe

Design implication:

- the “bind traffic from a specific inbound to a specific outbound/balancer” idea is valid
- but the 3x-ui storage choice of adding `OutboundTag` directly to the core `Node` model is not suitable for N5-UI Phase 2
- N5-UI should bind existing inbounds to policies through a separate binding table, not by modifying legacy inbound/core tables

## 5. Borrowable Approaches

The following approaches are suitable for N5-UI:

- treat egress lines as independent outbound entities
- use stable generated tags for every egress and pool
- validate outbound fragments with vendored `xray-core`
- generate routing rules with standard `outboundTag` and `balancerTag`
- use balancer as the real implementation of “线路池”
- keep config generation additive instead of rewriting the base template
- isolate route test and outbound test into dedicated services/controllers

## 6. Rejected Approaches and Reasons

## 6.1 Directly modifying legacy core tables

Rejected:

- add many egress/policy fields to legacy `Inbound` table
- add new routing fields to legacy `Setting` records

Reason:

- violates local architecture freeze
- raises migration and rollback risk for existing `x-ui` compatible installations
- makes later Phase 3 to Phase 5 harder to maintain

## 6.2 Reusing 3x-ui `Node.OutboundTag` style binding

Rejected:

- write new outbound/policy fields directly into legacy core models

Reason:

- N5-UI local rules explicitly prefer new tables over core schema expansion
- current repository does not even have the same node architecture as 3x-ui
- policy binding should be a Phase 2 module concern, not a base inbound schema concern

## 6.3 Depending on Xray hot API as the primary Phase 2 path

Rejected for MVP:

- design Phase 2 around `AddOutbound` and `ApplyRoutingConfig` as the primary apply path

Reason:

- current `n5-ui` codebase does not yet have a runtime Xray API wrapper like 3x-ui
- current panel apply path is restart-oriented
- routing `domainStrategy` is not freely hot-reloadable, which weakens a hot-only design

Decision:

- Phase 2 MVP should use full config regeneration plus normal Xray restart for apply
- hot-apply can be evaluated later as an optimization phase, not as MVP foundation

## 6.4 Changing startup scripts to use multi-file Xray configuration

Rejected:

- use Xray `confdir` or startup argument changes to attach new module files

Reason:

- official Xray docs support multi-file config, but using it here would require touching frozen script/service/startup behavior
- local rules explicitly freeze install scripts, service definitions, and startup conventions

## 6.5 Using advanced balancer strategies in MVP

Rejected for Phase 2 MVP:

- `leastPing`
- `leastLoad`

Reason:

- they are closely tied to observatory/health sampling
- health detection is already planned for later phases
- choosing them now would force premature observatory design

Decision:

- MVP default strategy should be `random`
- `roundrobin` can be a secondary supported strategy if implementation cost stays low

## 6.6 Distributed control-plane model like Marzban

Rejected:

- adopt a multi-node orchestration architecture similar to Marzban as the base Phase 2 model

Reason:

- Marzban is useful as a mature reference, but its operating assumptions are not `x-ui` compatibility first
- N5-UI Phase 2 target is local Xray outbound/routing extension, not replacing the panel with a new control plane

## 7. Final N5-UI Design Choice

## 7.1 Core design decision

Phase 2 should be implemented as:

- independent database tables
- independent controller/service/model modules
- independent frontend pages
- additive Xray config fragment generation
- one controlled merge point into the final runtime config

Not as:

- direct edits to `xray/` core package
- direct edits to old `/xui/` APIs
- direct edits to legacy core business tables

## 7.2 Backend module layout

Recommended backend layout:

- `web/model/egress`
- `web/model/trafficpolicy`
- `web/service/egress`
- `web/service/trafficpolicy`
- `web/service/xrayext`
- `web/controller/egress`
- `web/controller/trafficpolicy`

Purpose:

- `egress`: egress lines and line pools
- `trafficpolicy`: split policies, rules, bindings
- `xrayext`: only responsible for generating additive outbound/routing/balancer fragments and merging them into the already-generated base Xray config

## 7.3 Frontend layout

Current repository is server-rendered HTML plus Vue 2, not a separate SPA `pages/` tree.

Recommended frontend layout for this repository:

- `web/html/n5/egress.html`
- `web/html/n5/traffic_policies.html`
- `web/html/n5/components/...`

New navigation should be independent from old inbound management pages.

## 7.4 API namespace

Do not modify old `/xui/` APIs.

Recommended new API namespace:

- `/n5/api/egress/...`
- `/n5/api/traffic-policy/...`

Recommended new page routes:

- `/n5/egress`
- `/n5/traffic-policies`

## 7.5 Database design

Recommended Phase 2 tables:

- `n5_egresses`
- `n5_egress_pools`
- `n5_egress_pool_members`
- `n5_traffic_policies`
- `n5_traffic_policy_rules`
- `n5_traffic_policy_bindings`

Recommended responsibilities:

### `n5_egresses`

One row per outbound line.

Suggested stored content:

- display name
- stable generated tag
- enabled flag
- outbound protocol
- outbound JSON fragment fields needed to build Xray outbound
- last test result fields

### `n5_egress_pools`

One row per logical pool.

Suggested content:

- display name
- stable generated balancer tag
- strategy
- fallback target
- enabled flag

### `n5_egress_pool_members`

Pool membership and order.

Suggested content:

- pool id
- egress id
- weight or order field
- enabled flag

### `n5_traffic_policies`

One row per split policy.

Suggested content:

- name
- enabled flag
- default target type: `egress` or `pool`
- default target id

### `n5_traffic_policy_rules`

Ordered rules inside one policy.

Suggested content:

- policy id
- rule type: `domain` or `ip`
- match mode
- match value
- target type: `egress` or `pool`
- target id
- sort order
- enabled flag

### `n5_traffic_policy_bindings`

Bind legacy inbounds to Phase 2 policies without changing legacy tables.

Suggested content:

- inbound id
- policy id
- enabled flag

## 7.6 Tag strategy

Stable tags are required.

Recommended strategy:

- egress outbound tag: `n5-egress-{id}`
- pool balancer tag: `n5-pool-{id}`

Do not let users edit runtime tags manually in MVP.

Reason:

- tag stability is more important than UI flexibility
- routing and balancer references must survive rename operations
- this matches the stable-tag lesson learned from 3x-ui

## 7.7 Xray config merge strategy

Final config generation should follow this order:

1. legacy `XrayService` generates the base config exactly as today
2. `xrayext` reads active Phase 2 tables
3. `xrayext` generates additional outbounds
4. `xrayext` generates additional balancers
5. `xrayext` generates additional routing rules
6. `xrayext` merges them into the base config
7. final config goes through normal `xray.TestConfig`
8. panel restarts Xray through existing restart path

Important compatibility rule:

- Phase 2 outbounds must be appended after existing template outbounds
- existing template outbound order must not be changed
- especially the first outbound must not be replaced or reordered by Phase 2 logic

Important routing rule rule-order choice:

- generated policy rules should be prepended before existing routing rules
- every generated rule must include bound `inboundTag`
- because of that inbound scoping, old routing behavior for unbound inbounds remains unchanged

## 7.8 Policy rendering strategy

For each active binding:

1. resolve bound inbound's current tag from legacy inbound record
2. emit one routing block for that inbound tag
3. emit domain rules in configured order
4. emit IP rules in configured order
5. emit one default rule for that inbound tag at the end

Targeting rules:

- if target is one egress line, emit `outboundTag`
- if target is one pool, emit `balancerTag`

This directly matches official Xray routing semantics.

## 7.9 Domain and IP rule scope for MVP

Phase 2 MVP should keep the rule surface intentionally narrow.

Recommended MVP domain matching:

- exact domain
- domain suffix
- keyword

Recommended MVP IP matching:

- single IP
- CIDR

Not recommended for MVP UI:

- unrestricted raw routing JSON
- full geosite/geoip free-form editor
- protocol/process/localIP advanced rule matrix

Reason:

- keep validation deterministic
- keep UX aligned with user-stated MVP
- avoid prematurely exposing every Xray routing feature in the first implementation

## 7.10 `domainStrategy` choice

Final decision:

- keep stored template untouched
- allow Phase 2 merge layer to upgrade runtime `routing.domainStrategy` to `IPIfNonMatch` only when there is at least one active Phase 2 IP rule and the base routing config is empty or `AsIs`

Reason:

- mixed domain rule plus IP rule policies become predictable
- stored compatibility-layer template is still preserved as-is
- the runtime-only adjustment is limited to the feature being enabled

Risk:

- this is the one deliberate runtime behavior shift in Phase 2
- it must be clearly documented and covered by tests

## 7.11 Outbound test design

Phase 2 MVP must support:

- add outbound
- edit outbound
- delete outbound
- test outbound

Current codebase finding:

- `xray.TestConfig` only validates config syntax/compatibility
- current `Process.Start()` writes to fixed running path `bin/config.json`
- therefore the running process helper cannot be reused directly for isolated connectivity tests

Final decision:

- implement outbound test as an isolated temporary Xray test run

Recommended test flow:

1. generate a minimal temporary Xray config
2. add one loopback SOCKS inbound on a random local port
3. add the candidate outbound or candidate pool
4. add one routing rule from that loopback inbound to the candidate target
5. validate config with `xray.TestConfig`
6. start a temporary Xray subprocess with the temporary config path, not the panel's main `bin/config.json`
7. perform HTTP probe through the temporary SOCKS proxy
8. stop the temporary subprocess
9. persist last test result in Phase 2 tables

Reason:

- does not disturb the running panel Xray
- does not require 3x-ui style runtime API layer
- works for both direct outbound and balancer-backed pool targets

## 7.12 Node binding strategy in N5-UI

Final decision:

- do not copy 3x-ui temporary bridge logic for persistent binding
- bind legacy inbound ids to policies through `n5_traffic_policy_bindings`
- resolve the actual inbound tag at render time and generate direct Xray routing rules

Reason:

- `n5-ui` already owns the local inbound definitions
- no extra loopback bridge is needed for normal production traffic
- separate binding table avoids changing legacy schema

## 8. MVP Boundary

Phase 2 MVP should include:

- egress line add
- egress line edit
- egress line delete
- egress line test
- pool creation based on balancer abstraction
- policy creation
- domain rules
- IP rules
- default egress
- inbound-to-policy binding
- final Xray outbound/routing generation and restart-based apply

Phase 2 MVP should not include:

- health-based auto failover
- observatory-driven leastPing/leastLoad scheduling
- dynamic balancer override UI
- advanced geosite/geoip editor
- distributed multi-node orchestration

## 9. Recommended Implementation Sequence

Recommended coding order after this document is approved:

1. add Phase 2 tables and isolated auto-migrate entry points
2. add egress model/service/controller
3. add pool model/service/controller
4. add policy model/service/controller
5. add xray extension merge service
6. add isolated outbound/pool test service
7. add frontend pages and navigation
8. add integration tests for generated outbound/routing merge results
9. add restart/apply tests and rollback cases

## 10. Risk Summary

## 10.1 Main risks

- runtime `domainStrategy` adjustment can affect routing resolution behavior
- routing rule order bugs can accidentally shadow legacy rules
- outbound tag instability would break saved policies
- any attempt to prepend new outbounds before old ones could change legacy default-route behavior
- test subprocess implementation must not overwrite panel runtime config

## 10.2 Main mitigations

- keep stored template unchanged
- append additional outbounds only
- prepend generated rules only when every rule is inbound-scoped
- use deterministic generated tags
- build isolated temporary config files for tests
- keep all new schema in dedicated `n5_*` tables

## 11. Final Recommendation

N5-UI Phase 2 should proceed with:

- independent egress, pool, policy, binding modules
- balancer-backed pool design
- additive Xray config fragment generation
- restart-based apply path for MVP
- isolated temporary-process outbound testing

N5-UI Phase 2 should not proceed with:

- direct edits to frozen compatibility-layer scripts/services/APIs
- schema pollution of legacy inbound/setting tables
- hot-reload-first architecture
- observatory-dependent scheduling in MVP

This is the most compatible path with the current `n5-ui` codebase and the strongest match to the existing project rule set.
