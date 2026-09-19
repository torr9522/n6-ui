# Stable Release Process

## Required Gates

A future Stable release must pass:

- Source of Truth verification
- commit chain audit
- checkpoint/tag audit
- clean install on a clean Debian amd64/x86_64 environment
- desktop browser gate
- mobile browser gate
- inbound CRUD smoke
- egress CRUD and test smoke
- routing priority and creation-order independence
- conflict warning
- subscription outputs
- Access IP validation
- SQLite busy/concurrency audit
- TCP E2E
- UDP E2E
- Xray configuration test
- DB consistency checks
- real OS reboot gate
- post-reboot browser/runtime/data checks
- `go test ./...`
- `go build ./...`
- `go vet ./...`
- `git diff --check`
- N5 race tests

## Release Commit Scope

The final release commit should contain only:

- version metadata
- release notes
- README/install documentation
- release asset references
- other necessary release metadata

It must not secretly include business logic, routing, DB schema, runtime, or
frontend behavior changes.

## Tags

- Stable tags must be annotated.
- Published Stable tags must never be moved.
- Internal or checkpoint tags must be audited before publishing.
- Never use `git push --tags` during a controlled release.

## Assets

Upload only public release assets:

- release package
- runtime asset
- checksums
- release notes

Do not upload:

- Git bundles
- full repo archives containing `.git`
- checkpoint archives
- DB backups
- logs
- credentials
- internal manifests with test host details

## Fresh Clone Verification

After publishing:

```bash
git clone https://github.com/torr9522/n5-ui.git
cd n5-ui
git fetch --tags
git checkout vNEXT
go test ./...
go build ./...
go vet ./...
git diff --check
go test -race ./web/service/n5/... ./web/controller/n5/...
```

Also verify version output and installer/runtime references resolve to the new
Stable tag.
