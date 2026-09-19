# N6 Repository Autonomy Audit

Scope: executable source (`.sh`, `.go`, `.service`, `.timer`, templates, and release configuration).

Results:

- N6 install source: `raw.githubusercontent.com/torr9522/n6-ui/v0.1.0`.
- N6 source clone fallback: `https://github.com/torr9522/n6-ui.git`, branch `main`.
- N6 application release source: `https://github.com/torr9522/n6-ui/releases/download/v0.1.0`.
- N6 Xray runtime source: same N6 release base; binary SHA is checked after extraction.
- N6 update checker: same N6 release base.
- No executable N6 path falls back to `torr9522/n5-ui`.
- No executable N6 path downloads a runtime from Xray upstream or another panel repository.

The N5 URLs retained in historical documentation are provenance/history only and are not executable fetch paths. `acme.sh`, kernel mirrors, Debian apt, Go module proxy, and other system dependencies remain third-party infrastructure rather than N6 application/runtime sources.

Status: PASS for executable N6-owned fetch paths; final release and fresh-install verification are required before publication.
