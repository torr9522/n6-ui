# SS2022 Compatibility Assessment

## Scope

This change adds only single-user Shadowsocks 2022 methods to the existing
Shadowsocks inbound and N5 egress paths. It does not add multi-user, EIH, user
lists, per-user accounts, or a new protocol model.

## Frozen-area impact

- `web/service/inbound.go`: adds validation before the existing opaque settings
  JSON is stored. The model and generated Xray JSON shape do not change.
- `web/controller/inbound.go`: adds one backward-compatible authenticated API
  for cryptographically secure key generation. Existing routes are unchanged.
- Existing `/xui/` paths, service names, binary names, runtime directories and
  database paths remain unchanged.

## Compatibility

- Legacy Shadowsocks remains in the same protocol and settings structure.
- Legacy passwords remain arbitrary non-empty strings and are not subject to
  SS2022 Base64 or decoded-length validation.
- The three SS2022 methods require canonical padded standard Base64 keys of the
  exact length accepted by the formal N5 Xray 26.5.3 runtime.
- Existing `inbounds` and `n5_egresses.outbound_json` storage is sufficient.

## Migration and rollback

- Database migration: none.
- Runtime replacement: none.
- Installer change: none.
- Rollback: check out the preceding local checkpoint; no stored schema needs to
  be reversed.
