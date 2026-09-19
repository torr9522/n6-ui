# Project Timeline

## Baseline

`6eeb747c3c78a5e51531bb25ee1d6c7a73063989`

This was the public baseline before the v0.2.0 Stable hardening chain. It
already contained the N5 independent install/runtime direction and custom Xray
26.5.3 amd64 release asset support.

## Checkpoint Chain

| Stage | Commit | Purpose |
|---|---|---|
| Fix 01-08 | `a898bd14f702b75b8e9178fa6ee05f6fa10ce47d` | Rule group UX, direct custom editing, flow separation, deterministic routing, conflict warning |
| Fix 09 | `6145d9acf24225ce607dd657fce837703533bb23` | Real custom rule group names in Simple rules |
| Fix 10 | `6777b8545163ab60d4bc83e4b87d2e192e065ad5` | Keep N5 navigation expanded |
| P1-01 | `1d89561c21e6238e8a8a224e62f7e954ddd8c8fb` | Protect referenced egress deletion |
| P1-02 | `93e2616fcfd6ea9a486788554aa0d36c550fc322` | Clean only deleted inbound Simple state |
| P1-03 | `61c622d169d12ab667981f18333d2fb46f27cfb2` | Block Advanced mutation of Simple-owned state |
| P2-01 | `292b362b19fa8bf316e975df246419e3711ce9c3` | Transactional Simple mutations |
| P2-02 | `3b733af4d7a5c1a2cadb7844363283e5ee2d0124` | Backend protection for disabled builtin groups |
| RC Mobile | `e87c9753491452a6709c3bd94ff883d5152e2d7a` | Mobile operability release fix |
| RC SQLite Busy | `604f8afe084a11d0f706c836af77dbf16eb52a09` | Sanitized SQLite busy handling |
| Stable release | `1265df551af46b7abe5a7e35fc2621d73cbae4be` | Version, docs, release metadata only |

## Stable Chain

```text
v0.2.0
-> 1265df551af46b7abe5a7e35fc2621d73cbae4be
-> 604f8afe084a11d0f706c836af77dbf16eb52a09
-> clean install / reboot / full E2E verified business candidate
```

The `v0.2.0` commit is intentionally a small release-prep commit on top of the
verified business candidate.
