# Continuation Guide

## Where To Start

Start new development from `v0.2.0` or from the current `main` if it is still at
or ahead of the Stable release commit.

```bash
git clone https://github.com/torr9522/n5-ui.git
cd n5-ui
git fetch --tags
git checkout main
```

Before editing:

```bash
git status --short
git rev-parse HEAD
git tag --points-at HEAD
```

## Development Rules

- Keep business changes in the development repository.
- Use deployment hosts for TEST/UAT only.
- Do not commit directly on a test server unless doing an explicit emergency
  recovery workflow.
- Keep Simple and Advanced ownership boundaries intact.
- Do not move old Stable tags.
- Do not rewrite published history.
- Do not add secrets, DB backups, raw logs, cookies, or private keys to Git.

## Test Matrix For Meaningful Changes

Run at least:

```bash
go test ./...
go build ./...
go vet ./...
git diff --check
go test -race ./web/service/n5/... ./web/controller/n5/...
```

For runtime-facing changes, also perform:

- browser UAT
- mobile viewport UAT
- Xray config test
- TCP E2E
- UDP E2E
- DB consistency checks
- real reboot gate before Stable

## Ownership Rules

Simple-managed state is marked by `n5-simple-exec|` or legacy `n5-simple|`.
Advanced APIs must not mutate it. Simple service internals manage its lifecycle.

## Routing Rules

Simple routing priority is deterministic:

```text
direct exact > direct suffix > direct keyword > direct regexp > custom group > AI > Game > Streaming > ALL
```

Creation order must not affect Simple routing.

## Release Rule

If a Stable release is already public and a bug is found, publish a new version.
Do not move the old tag.
