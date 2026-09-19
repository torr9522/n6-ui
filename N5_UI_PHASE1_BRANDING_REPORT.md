# N5-UI Phase-1 Branding Report

## Source Location

- Source baseline path: `/root/n3-ui`
- New project path: `/root/n5-ui`
- Phase-1 branding time: `2026-08-08`

## Compatibility Kept

- Service name remains `x-ui.service`.
- CLI command remains `x-ui`.
- Runtime install path family remains `/usr/local/x-ui`.
- Config directory remains `/etc/x-ui`.
- Database path remains `/etc/x-ui/x-ui.db`.
- Go module remains `module x-ui`.
- API prefix remains `/xui/`.
- Frontend internal `x-ui` naming was not mass-renamed.
- Install scripts and install commands remain unchanged.
- Xray config generation and node management logic remain unchanged.

## Brand Scope

- Project brand changed from `n3-ui` to `n5-ui` where the string is display-only.
- Runtime identity remains `x-ui` compatible.
- No architecture or feature work was added in this phase.

## Added Or Renamed Project Files

- `docs/N5_UI_PROJECT_RULES.md`
- `docs/N5_UI_BASELINE.md`
- `N5_UI_PHASE1_BRANDING_REPORT.md`

## Execution Notes

- n5-ui is created as an independent source copy of n3-ui.
- This phase changes branding only: README, page title, display text, repo references, and project docs.
- No backend protocol code was changed.
- No service, binary, path, database, API, or script compatibility identifier was renamed.

## Verification Plan

- Check source file inventory.
- Run `go build` if the local Go environment allows it.
- Confirm compatibility-sensitive identifiers remain unchanged.
