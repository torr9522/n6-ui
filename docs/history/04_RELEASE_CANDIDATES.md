# Release Candidates

## RC-01: Mobile Release Readiness

- Commit: `e87c9753491452a6709c3bd94ff883d5152e2d7a`
- Public tag: `n5-local-rc-01-mobile-release-readiness`

Problem: narrow mobile viewports could make N5 action buttons hard or impossible
to reach. The risky actions included edit, delete, manage rules, and test.

Validated viewports:

- 320x720
- 360x800
- 390x844
- 412x915

Design decision: do not rebuild the mobile UI for this release. Apply the
smallest release-readiness operability fix:

- local horizontal scrolling for wide N5 tables
- page-wide overflow protection
- modal footer protection so save/cancel remain reachable

Final semantics: core N5 operations remain reachable on mobile even when table
content is wider than the viewport.

Validation: Playwright mobile gates and real horizontal scroll interaction.

## RC-02: SQLite Busy Handling

- Commit: `604f8afe084a11d0f706c836af77dbf16eb52a09`
- Public tag: `n5-local-rc-02-sqlite-busy-handling`

Problem: concurrent SQLite writes could expose low-level lock errors to users or
return uncontrolled HTTP 500 responses.

Design decision: map SQLite busy/locked errors to a controlled user-facing JSON
message while keeping ordinary business validation errors distinct.

Final semantics:

- no raw `database is locked` text in HTTP responses
- no raw SQL leakage
- no HTTP 500 for handled SQLite busy conditions
- duplicate business errors remain separate from SQLite busy errors
- user-facing message is understandable: database is busy; retry later

Validation: concurrency smoke, duplicate AI concurrency, different rule
concurrency, update/delete concurrency, and HTTP status/error body audit.
