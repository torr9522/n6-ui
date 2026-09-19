# N5-UI Project Rules

## Project Identity

- Project name: `n5-ui`
- Source baseline: `n3-ui`
- Development model: source-level fork based on `n3-ui`

## Runtime Compatibility

`n5-ui` must continue to run as an `x-ui` compatible panel. Existing user environments, service names, commands, paths, database files, and API paths must remain compatible.

## Must Keep Unchanged

- Service name: `x-ui.service`
- CLI command name: `x-ui`
- Install/runtime directory family: `/usr/local/x-ui`
- Configuration directory: `/etc/x-ui`
- Database path: `/etc/x-ui/x-ui.db`
- Database schema compatibility with existing `x-ui` installations
- API path prefix: `/xui/`
- Go module name: `module x-ui`
- Existing frontend internal `x-ui` naming and variables unless a narrow feature requires otherwise

## Allowed Development Areas

- Brand presentation updates
- Project documentation updates
- Repository metadata updates
- Future feature work only after compatibility review

## Compatibility Rules

- Do not rename `x-ui.service`.
- Do not rename the installed binary or CLI command away from `x-ui`.
- Do not move the database away from `/etc/x-ui/x-ui.db`.
- Do not introduce database schema changes without explicit migration and rollback planning.
- Do not change existing `/xui/` API paths unless backward compatible aliases remain.
- Do not mass-replace `x-ui` strings only for branding.
- Document every compatibility-sensitive change before implementation.
