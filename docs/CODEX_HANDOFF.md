# n6-ui Development Handoff

## Stable Baseline

- Repository: `torr9522/n6-ui`
- Current Stable: `v0.1.0`
- Stable code commit: `0b4f648511ca13054cef298223a80bd39e7980b1`
- Release status: `v0.1.0` is formally published.
- Application asset: `x-ui-linux-amd64.tar.gz`
- Application asset SHA256: `cec81fea7a8e280f95a3d7ce7702c231e69dd9fb30196b8a4c40b51d6c55b6a3`
- Runtime release: `n6-runtime-26.5.3-amd64`
- Xray: Custom Xray 26.5.3 amd64
- Xray binary SHA256: `128f9c34811ee74b3770eef7010d011e3946e85dfab28f2ed1804e380461b05e`

The `v0.1.0` tag is the stable rollback point for this baseline. It must continue to resolve to the stable code commit above.

## Repository Autonomy

N6 is an independent development line. N6 application, install, update, raw-script, release, and runtime fetch paths resolve only from `torr9522/n6-ui`.

- N6 application/install/update/runtime source: `n6-ui` only
- N5 fetch dependency: `0`
- N5 repository runtime dependency: none

The N5 repository is historical provenance only. It is not an N6 runtime, installation, update, release, or recovery dependency.

## Validated Stable Scope

The released SS2022 scope is single-user only. All three supported methods passed real external TCP and UDP validation:

- `2022-blake3-aes-128-gcm`
- `2022-blake3-aes-256-gcm`
- `2022-blake3-chacha20-poly1305`

Additional stable acceptance results:

- Legacy Shadowsocks: PASS
- VMess: PASS
- VLESS: PASS
- AI real routing through an independent egress: PASS
- Advanced real routing through an independent egress: PASS
- Pool-member real routing through an independent egress: PASS
- Pool fallback: NOT_APPLICABLE because the v0.1.0 UI has no fallback configuration entry
- Clean install: PASS
- Real external TCP/UDP: PASS
- Real OS reboot and post-reboot recovery: PASS
- Second independent clean-server UAT: PASS
- `STABLE_V0_1_0_ACCEPTANCE=PASS`

SS2022 multi-user and EIH are not supported by v0.1.0. Classic Clash output does not emit SS2022 nodes; use the Mihomo subscription format for SS2022.

## Future Development Bootstrap

Future N6 development must use this recovery sequence by default:

1. Clone or fetch `torr9522/n6-ui`.
2. Read `docs/CODEX_HANDOFF.md`.
3. Read the latest Stable tag and its release notes.
4. Use remote `main` as the development source.
5. Use the most recent Stable tag as the stable rollback point.

Do not treat a local `/root` directory, local-only tags, bundles, archives, frozen evidence directories, or a Codex session as a long-term dependency. They may disappear without notice. GitHub repository history, remote `main`, published Stable tags, release notes, and release assets are the durable source of truth.

## Compatibility Identifiers

The following internal compatibility identifiers must not be casually renamed:

- `n5_*` database tables
- `n5-simple`
- `n5-simple-exec`
- `n5-egress-*`
- `n5-pool-*`
- `/n5/` API paths
- `web/service/n5`
- `web/controller/n5`
- x-ui service, installation path, configuration path, and database names, including `x-ui.service`, `/usr/local/x-ui`, `/etc/x-ui`, and `x-ui.db`

These are internal compatibility identifiers, not product branding. User-visible product branding remains N6/n6-ui. Renaming any compatibility identifier requires an explicit migration design, compatibility review, upgrade-path analysis, and regression evidence.

See `docs/N6_COMPATIBILITY_IDENTIFIERS.md` for additional context.
