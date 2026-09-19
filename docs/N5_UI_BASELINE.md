# N5-UI Baseline

## Base Version

- Source project: `n2-ui`
- Source path: `/tmp/cert-audit-1785225391/n2-ui`
- Base commit: `654aa2c0a5cc02823922ad5a0f5d9d61a3c2461d`
- Source working-tree delta included: Trojan Web visibility restore from the previous task

## Initialization

- Initialized at: `2026-07-28T08:57:43Z`
- Target project path: `/root/n5-ui`
- Git branch: `main`

## Source Inventory

- Copied source files before n5-ui documentation: `1146`
- Source copy excluded from n2-ui transfer: source `.git` metadata and top-level built `x-ui` binary
- Initial Git commit excludes compiled binaries and release packages via `.gitignore`

## Current Function Status

- Go backend source is inherited from `n2-ui`.
- Web frontend resources are inherited from `n2-ui`.
- CLI scripts are inherited and continue to expose `x-ui`.
- systemd units are inherited and continue to use `x-ui.service`.
- API prefix remains `/xui/`.
- Go module remains `module x-ui`.
- Database compatibility remains based on `/etc/x-ui/x-ui.db`.

## Known Issues And Notes

- Trojan UI hidden issue: restored in the copied baseline by exposing `Protocols.TROJAN` in the inbound modal and allowing Trojan stream settings.
- Certificate system: still uses the old model and requires a separate certificate-system design/implementation phase.
- Branding: development project name is `n5-ui`, but runtime identity intentionally remains `x-ui` compatible.
- Build outputs and release packages are not part of the clean initial Git commit.
