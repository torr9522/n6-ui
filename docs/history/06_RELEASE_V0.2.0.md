# Release v0.2.0

## Identity

- Release: N5-UI v0.2.0 Stable
- Tag: `v0.2.0`
- Tag type: annotated
- Release commit: `1265df551af46b7abe5a7e35fc2621d73cbae4be`
- Release commit parent:
  `604f8afe084a11d0f706c836af77dbf16eb52a09`
- Verified business candidate:
  `604f8afe084a11d0f706c836af77dbf16eb52a09`
- Candidate tree: `6cd286b3e2932736233814e7ea915805110ce086`
- Stable tree: `5f45f7491a64b8970ae5dd819b93e8c59b3f1db5`

## Runtime

- Xray version: 26.5.3
- Architecture: amd64
- Binary SHA256:
  `128f9c34811ee74b3770eef7010d011e3946e85dfab28f2ed1804e380461b05e`

## Release Commit Scope

The release commit contains version, README, release note, install source, and
runtime asset URL metadata. It intentionally does not change N5 routing,
database schema, controller behavior, service business logic, Xray core, or
frontend functional behavior.

## Gates

- Pre-release freeze: PASS
- Freeze bundle verify: PASS
- Freeze bundle restore: PASS
- Candidate automated tests: PASS
- Clean install: PASS
- Reboot: PASS
- Full E2E: PASS
- Release commit tests: PASS
- GitHub Release: published
- GitHub asset download SHA verification: PASS
- Fresh clone checkout `v0.2.0`: PASS
- Fresh clone tests/build/vet/race/diff-check: PASS
- Fresh clone version output: `v0.2.0`

## Public Assets

- `x-ui-linux-amd64.tar.gz`
- `Xray-linux-64.zip`
- `SHA256SUMS`
- `RELEASE-v0.2.0.md`

Internal bundle archives and checkpoint archives are not release assets.

## Upgrade

Clean Install is supported. In-place upgrade is not claimed or guaranteed for
this release. Upgrade risk audit and Upgrade Gate are separate post-release
work.
