# n6-ui Development Handoff

Project: `n6-ui`

Initial baseline: validated N5 SS2022 single-user RC `b59074e158877bfc72ea15613fad4695f165fc70`.

Runtime: Custom Xray 26.5.3 amd64, binary SHA256 `128f9c34811ee74b3770eef7010d011e3946e85dfab28f2ed1804e380461b05e`.

Repository: `https://github.com/torr9522/n6-ui`

N6 is an independent development line. Installation, application releases, update checks, raw scripts, and Xray runtime assets must resolve only from `torr9522/n6-ui` after bootstrap. N5 is permitted only as the documented bootstrap parent and provenance source; N6 must not retain a runtime fallback to N5.

The validated SS2022 scope is single-user only:

- `2022-blake3-aes-128-gcm`
- `2022-blake3-aes-256-gcm`
- `2022-blake3-chacha20-poly1305`

Multi-user, EIH, multi-key, per-user accounts, DB schema changes, routing semantic changes, and Xray rebuilds are out of scope.

Compatibility identifiers such as `n5_*`, `n5-simple`, `n5-egress-*`, `/n5/`, `web/service/n5`, `x-ui.service`, and `x-ui.db` must not be renamed without an explicit migration decision. See `N6_COMPATIBILITY_IDENTIFIERS.md`.
