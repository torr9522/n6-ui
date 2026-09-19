# New Codex Recovery Test

This file records the questions a new Codex should be able to answer using only
GitHub clone content.

| ID | Question | Expected Answer Source | Status |
|---|---|---|---|
| A | Current Stable? | `v0.2.0` in `docs/CODEX_HANDOFF.md` | PASS |
| B | Stable commit? | `1265df551af46b7abe5a7e35fc2621d73cbae4be` | PASS |
| C | Clean-install verified Candidate? | `604f8afe084a11d0f706c836af77dbf16eb52a09` | PASS |
| D | Xray fixed version? | Xray `26.5.3` amd64 | PASS |
| E | Why not generic Xray runtime? | Custom runtime preserves validated behavior and Access IP log compatibility | PASS |
| F | Why keep `x-ui` runtime naming? | Runtime compatibility with service, command, DB, paths, and APIs | PASS |
| G | Simple/Advanced ownership? | Simple owns `n5-simple-exec|` and legacy `n5-simple|`; Advanced cannot mutate it | PASS |
| H | How is ALL implemented? | `TrafficPolicy.defaultTarget`, not fake catch-all rule | PASS |
| I | Routing priority? | direct exact > suffix > keyword > regexp > custom group > AI > Game > Streaming > ALL | PASS |
| J | Does creation order affect Simple? | No | PASS |
| K | Is Custom Group snapshotted? | Yes, execution uses snapshot semantics | PASS |
| L | Does rename change `groupId`? | No | PASS |
| M | P1-01? | Referenced egress deletion protection | PASS |
| N | P1-02? | Inbound delete cleanup scoped to deleted inbound Simple state | PASS |
| O | P1-03? | Advanced API protects Simple-managed state | PASS |
| P | P2-01? | Simple create/update/delete are transactional | PASS |
| Q | P2-02? | Disabled builtin groups are rejected by backend | PASS |
| R | RC Mobile? | Mobile table actions and modal footer are reachable | PASS |
| S | RC SQLite? | Busy/locked errors are sanitized; no HTTP 500/raw SQL leakage | PASS |
| T | v0.2.0 gates? | clean install, reboot, browser, mobile, TCP/UDP, DB, automated tests | PASS |
| U | Upgrade policy? | Clean Install supported; in-place upgrade not guaranteed | PASS |
| V | ARM64 runtime policy? | Not claimed for Stable runtime | PASS |
| W | Where to continue development? | Start from `v0.2.0` or current `main` if it is at/after Stable | PASS |
| X | How to test changes? | Go test/build/vet/diff/race plus runtime gates when needed | PASS |
| Y | How to release next Stable? | Follow `docs/history/11_RELEASE_PROCESS.md` | PASS |
| Z | Deferred features? | Upgrade Gate, SS2022, port IP limit redesign, ARM64 Stable Runtime | PASS |
