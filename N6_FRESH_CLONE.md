# N6 Fresh Clone Verification

The repository was cloned from `https://github.com/torr9522/n6-ui.git` into
an independent temporary directory at commit `bd3d92d`.

- Remote: only `origin`, pointing to `torr9522/n6-ui`
- `go test ./... -count=1 -timeout=30m`: PASS
- `go build ./...`: PASS
- `go vet ./...`: PASS
- `go test -race -timeout=30m ./web/service/n5/... ./web/controller/n5/...`: PASS
- `git diff --check`: PASS

The checkout required no N5 working tree, N5 remote, local path dependency, or
untracked embedded template. `web/service/config.json` is tracked explicitly
because the application embeds it at compile time.
