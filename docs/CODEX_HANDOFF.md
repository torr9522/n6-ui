# N5-UI Development Handoff

This is the first document a new Codex agent should read after cloning the
public repository.

Start from a fresh clone:

```bash
git clone https://github.com/torr9522/n5-ui.git
cd n5-ui
git fetch --tags
```

Read these files in order:

1. `docs/N5_UI_PROJECT_RULES.md`
2. `docs/N5_UI_DEVELOPMENT_ARCHITECTURE.md`
3. `docs/N5_UI_SOURCE_TREE.md`
4. `docs/N5_UI_DATABASE.md`
5. `docs/N5_UI_RUNTIME.md`
6. `docs/N5_UI_SIMPLE_MODE_DESIGN.md`
7. `docs/DEVELOPMENT_SKILL_TREE.md`
8. `docs/history/README.md`
9. `docs/history/07_ARCHITECTURE_DECISIONS.md`
10. `docs/history/10_CONTINUATION_GUIDE.md`

## Current Stable Baseline

- Stable tag: `v0.2.0`
- Release commit: `1265df551af46b7abe5a7e35fc2621d73cbae4be`
- Verified business candidate: `604f8afe084a11d0f706c836af77dbf16eb52a09`
- Candidate tree: `6cd286b3e2932736233814e7ea915805110ce086`
- Xray baseline: `26.5.3` amd64
- Xray binary SHA256:
  `128f9c34811ee74b3770eef7010d011e3946e85dfab28f2ed1804e380461b05e`

## Release Policy

- Clean Install: supported for the validated Stable path.
- In-place Upgrade: not currently guaranteed.
- Official Stable Runtime: Debian 11 amd64/x86_64 with N5 custom Xray 26.5.3.
- ARM64 Stable Runtime: not claimed for `v0.2.0`.

## Operating Principles

1. Continue development from the current Stable tag or release commit.
2. Treat any deployment host as TEST/UAT only unless a separate source-of-truth
   migration is explicitly performed.
3. Change business source only in the development repository.
4. Never let a test server become the only Source of Truth.
5. For major work: develop, test, create a local commit, create a checkpoint,
   then deploy to UAT.
6. Before the next Stable release, complete clean install, browser, TCP, UDP,
   reboot, runtime, and DB consistency gates.
7. Never move or recreate an existing Stable tag.
8. If a published Stable has a bug, fix it in a new version such as `v0.2.1`.

## Fast Checks

```bash
git checkout v0.2.0
git rev-list -n 1 v0.2.0
git cat-file -t v0.2.0
go test ./...
go build ./...
go vet ./...
git diff --check
go test -race ./web/service/n5/... ./web/controller/n5/...
```

Expected Stable commit:

```text
1265df551af46b7abe5a7e35fc2621d73cbae4be
```

For the full historical map, read `docs/history/09_CHECKPOINT_MAP.md` and
`docs/history/checkpoints.json`.
