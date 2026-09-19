# Checkpoint Map

| Checkpoint | Problem | Commit | Public Tag | Stable Inclusion | Validation |
|---|---|---|---|---|---|
| Fix 01 | Rule group first-level page mixed concepts | `a898bd14f702b75b8e9178fa6ee05f6fa10ce47d` | `n5-local-fix-01-rule-group-list-page` | `v0.2.0` | Browser CRUD, service tests |
| Fix 02 | Rule group needed focused detail management | `a898bd14f702b75b8e9178fa6ee05f6fa10ce47d` | `n5-local-fix-02-rule-group-detail-page` | `v0.2.0` | Browser CRUD, service tests |
| Fix 03 | Matcher labels exposed raw values | `a898bd14f702b75b8e9178fa6ee05f6fa10ce47d` | `n5-local-fix-03-domain-match-chinese-labels` | `v0.2.0` | Rule CRUD UAT |
| Fix 04 | Rule group edit flow incomplete | `a898bd14f702b75b8e9178fa6ee05f6fa10ce47d` | `n5-local-fix-04-rule-group-editing` | `v0.2.0` | Add/edit/delete/rename |
| Fix 05 | Direct custom required recreate-style changes | `a898bd14f702b75b8e9178fa6ee05f6fa10ce47d` | `n5-local-fix-05-direct-custom-edit` | `v0.2.0` | Direct custom edit |
| Fix 06 | Direct custom and custom group flows were unclear | `a898bd14f702b75b8e9178fa6ee05f6fa10ce47d` | `n5-local-fix-06-custom-flow-separation` | `v0.2.0` | UI/API flow checks |
| Fix 07 | Creation order affected routing winner | `a898bd14f702b75b8e9178fa6ee05f6fa10ce47d` | `n5-local-fix-07-deterministic-routing-priority` | `v0.2.0` | Priority and order tests |
| Fix 08 | Overlapping rules lacked warning | `a898bd14f702b75b8e9178fa6ee05f6fa10ce47d` | `n5-local-fix-08-routing-conflict-warning` | `v0.2.0` | Conflict warning UAT |
| Fix 09 | Custom group display name was generic | `6145d9acf24225ce607dd657fce837703533bb23` | `n5-local-fix-09-rule-group-display-name` | `v0.2.0` | Rename/dropdown checks |
| Fix 10 | N5 menu collapsed on N5 routes | `6777b8545163ab60d4bc83e4b87d2e192e065ad5` | `n5-local-fix-10-n5-menu-expanded` | `v0.2.0` | Navigation checks |
| P1-01 | Referenced egress could be deleted | `1d89561c21e6238e8a8a224e62f7e954ddd8c8fb` | `n5-local-p1-01-egress-reference-protection` | `v0.2.0` | Reject/delete-after-unreference |
| P1-02 | Inbound deletion could orphan or over-delete state | `93e2616fcfd6ea9a486788554aa0d36c550fc322` | `n5-local-p1-02-inbound-delete-cleanup` | `v0.2.0` | Orphan counters zero |
| P1-03 | Advanced APIs could mutate Simple-owned state | `61c622d169d12ab667981f18333d2fb46f27cfb2` | `n5-local-p1-03-advanced-simple-protection` | `v0.2.0` | Rejection matrix |
| P2-01 | Simple mutations could partially apply | `292b362b19fa8bf316e975df246419e3711ce9c3` | `n5-local-p2-01-simple-transaction` | `v0.2.0` | Fault-injection rollback |
| P2-02 | Disabled builtin groups lacked backend enforcement | `3b733af4d7a5c1a2cadb7844363283e5ee2d0124` | `n5-local-p2-02-disabled-builtin-protection` | `v0.2.0` | Disable/re-enable matrix |
| RC Mobile | Mobile action columns could be unreachable | `e87c9753491452a6709c3bd94ff883d5152e2d7a` | `n5-local-rc-01-mobile-release-readiness` | `v0.2.0` | 320/390 mobile gates |
| RC SQLite Busy | SQLite lock errors could leak or become HTTP 500 | `604f8afe084a11d0f706c836af77dbf16eb52a09` | `n5-local-rc-02-sqlite-busy-handling` | `v0.2.0` | Concurrency and HTTP audit |

## Legacy Duplicate Local Tags

These tags are retained locally where present but are not needed for public
reconstruction:

- `n5-local-fix-01-simple-rule-group-and-priority`
- `n5-local-fix-02-rule-group-display-name`
- `n5-local-fix-03-n5-menu-expanded`

The canonical public tags above cover the same meaningful checkpoints.
