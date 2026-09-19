# N6 Repository Fetch Graph

Bootstrap provenance is documented separately: N5 v0.2.0 supplied the validated Xray asset during migration. After N6 release, every N6-owned fetch resolves only to `torr9522/n6-ui`.

| Component | Bootstrap source | N6 final source | Fetch method | SHA / pin | Status |
| --- | --- | --- | --- | --- | --- |
| Main source | local validated RC `b59074e` | `https://github.com/torr9522/n6-ui.git` | git clone/release archive | commit/tag pin | PASS |
| Install scripts | local validated RC | `raw.githubusercontent.com/torr9522/n6-ui/v0.1.0` | curl/wget | tag pin | PASS |
| Application tarball | local N6 build | N6 `v0.1.0` release | release asset | recorded in SHA256SUMS | READY |
| Xray zip | N5 v0.2.0 bootstrap asset | N6 runtime release `n6-runtime-26.5.3-amd64` | release asset | binary SHA `128f9c34811ee74b3770eef7010d011e3946e85dfab28f2ed1804e380461b05e` | VERIFIED bootstrap, READY |
| Web assets | local source tree | application package / N6 release | bundled | source commit pin | PASS |
| Geo assets | N5 bootstrap zip | N6 runtime release | bundled in Xray zip | archive SHA recorded | PENDING N6 upload |
| Update checker | local source | `github.com/torr9522/n6-ui/releases` | HTTPS API/release URL | N6 tag | PASS |

Third-party system services such as Debian apt, ACME endpoints, Go modules, and DNS remain external dependencies by design; they are not N6 application/runtime sources.
