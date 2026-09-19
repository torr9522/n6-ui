# Source Inventory

## Already Public In GitHub

- source code
- `v0.2.0` release commit
- `v0.2.0` annotated Stable tag
- README and release notes
- existing design docs under `docs/`
- public release assets

## Exists Only In Private Development Archives

- raw checkpoint directory
- raw browser evidence
- raw runtime logs
- DB snapshots
- Git bundles
- full repo tar archives containing `.git`
- release freeze manifests with local machine paths

These are intentionally not copied into the repository.

## Safe To Publish After Redaction

- checkpoint names
- public commit hashes
- public tag names
- design decisions
- test summaries
- release gate outcomes
- runtime version and public SHA256
- generic source tree and database relationships

## Not Safe To Publish

- any raw server, credential, token, cookie, database, access log, browser
  profile, or certificate material
- private test host details
- private domains used only for UAT
- raw UAT artifacts that may contain URLs or runtime state

## Published Handoff Strategy

This directory contains public summaries derived from Git commits, current
source, existing docs, and private checkpoints. It is enough for a new Codex to
reconstruct the project context without access to private evidence.
