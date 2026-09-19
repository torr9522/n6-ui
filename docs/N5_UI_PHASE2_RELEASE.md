# N5-UI Phase 2 Release

Date: 2026-08-09

## Architecture Summary

N5-UI Phase 2 keeps the original x-ui runtime compatibility layer unchanged and adds N5 capabilities as isolated modules.

Current layering:

- legacy compatibility layer
  - inbound management
  - original Xray template generation
  - original service and controller chain
- N5 extension layer
  - egress management
  - egress pool management
  - traffic policy management
  - additive outbound/routing/balancer generation
  - merge history and status APIs

Merge model:

1. legacy base config is generated first
2. N5 extension is applied only when `n5XrayExtensionEnable=true`
3. final config is validated by existing `xray.TestConfig`
4. existing restart flow remains in place

## Completed Features

- stable N5 tags
  - `n5-egress-%010d`
  - `n5-pool-%010d`
- N5 data models and migration
- egress CRUD and config validation
- egress pool CRUD and member management
- traffic policy CRUD and inbound binding
- additive Xray merge
  - outbound append
  - routing append
  - balancer generation
- config history
  - base hash
  - extension hash
  - final hash
  - apply status
  - apply error
- N5 runtime status
- N5 config history page
- egress test entry design reservation
- real-server rollback protection for hot restart failure

## API

N5 APIs currently available:

- `/n5/api/egress/list`
- `/n5/api/egress/get/:id`
- `/n5/api/egress/add`
- `/n5/api/egress/update/:id`
- `/n5/api/egress/del/:id`
- `/n5/api/egress/validate`

- `/n5/api/pool/list`
- `/n5/api/pool/get/:id`
- `/n5/api/pool/add`
- `/n5/api/pool/member/list/:id`
- `/n5/api/pool/member/add`
- `/n5/api/pool/member/del`

- `/n5/api/traffic-policy/list`
- `/n5/api/traffic-policy/add`
- `/n5/api/traffic-policy/rule/list/:id`
- `/n5/api/traffic-policy/rule/add`
- `/n5/api/traffic-policy/rule/del/:id`
- `/n5/api/traffic-policy/binding/list`
- `/n5/api/traffic-policy/bind`
- `/n5/api/traffic-policy/fragments`

- `/n5/api/xray/status`
- `/n5/api/xray/history/list`
- `/n5/api/xray/egress-test/entry`

## Data Tables

Phase 2 N5 tables:

- `n5_egresses`
- `n5_egress_pools`
- `n5_egress_pool_members`
- `n5_traffic_policies`
- `n5_traffic_policy_rules`
- `n5_traffic_policy_bindings`
- `n5_xray_config_history`

## UI Pages

Phase 2 N5 pages:

- `/n5/egress`
- `/n5/pools`
- `/n5/traffic-policy`
- `/n5/xray-status`
- `/n5/config-history`
- `/n5/egress-test`

## Known Limits

- `fallbackTag` is not enabled in normal runtime scenarios unless the required Xray observatory capability is introduced.
- routing is additive and appended after legacy rules; already-matched legacy rules still keep priority.
- egress test page is currently a reserved design entry, not an active temporary Xray execution tool.
- history page currently focuses on operational audit fields, not full config diff visualization.
- N5 status and history are panel-level views; no per-inbound runtime breakdown is exposed yet.
