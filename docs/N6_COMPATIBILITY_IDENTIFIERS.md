# n6-ui Compatibility Identifiers

The product brand and repository are `n6-ui`. The initial business source is the validated N5 SS2022 single-user RC, so several internal identifiers intentionally remain unchanged.

The following names are compatibility identifiers, not product branding:

- `n5_*` database tables and fields
- `n5-simple` and `n5-simple-exec` metadata
- `n5-egress-*` and `n5-pool-*` runtime tags
- `/n5/` API and route paths
- `web/service/n5` and `web/controller/n5` package directories
- `x-ui` binary, service, CLI, `/etc/x-ui`, `/usr/local/x-ui`, and `x-ui.db`

They remain stable to avoid database migration, routing changes, API breaks, and regressions in the already-validated runtime. User-facing labels and N6 repository/release URLs use `n6-ui`.
