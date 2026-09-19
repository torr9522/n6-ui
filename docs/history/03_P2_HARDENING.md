# P2 Hardening

## P2-01: Simple Transactional Mutations

- Commit: `292b362b19fa8bf316e975df246419e3711ce9c3`
- Public tag: `n5-local-p2-01-simple-transaction`

Problem: Simple create/update/delete touched multiple tables and could leave
partial state if a mid-flow operation failed.

Design decision: each Simple mutation must run inside one DB transaction, using
the transaction handle for all related TrafficPolicy writes.

Covered operations:

- create execution
- update execution
- delete execution

Failure injection validation:

- failure after policy create rolls back
- failure after snapshot/rule create rolls back
- failure after default target update rolls back
- failure during rule update rolls back
- failure during rule delete rolls back

Final semantics: Simple mutations either fully succeed or leave previous state
unchanged.

## P2-02: Disabled Builtin Backend Protection

- Commit: `3b733af4d7a5c1a2cadb7844363283e5ee2d0124`
- Public tag: `n5-local-p2-02-disabled-builtin-protection`

Problem: UI hiding or disabling of builtin groups was not enough; backend API
requests could still create new Simple execution state for disabled groups.

Design decision: backend rejects new Simple execution creation for disabled
builtin groups:

- AI
- Game
- Streaming

Final semantics:

- disabled builtin group creation is rejected from direct traffic type and
  builtin group ID paths
- re-enabled builtin groups can be used again
- existing execution snapshots are not automatically deleted or rewritten
- ALL, direct Custom, and custom groups keep their existing behavior

Validation: UI/API rejection tests, re-enable positive path, and snapshot
preservation checks.
