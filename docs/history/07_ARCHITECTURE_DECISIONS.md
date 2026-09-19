# Architecture Decisions

## Keep `x-ui` Runtime Compatibility

N5-UI is the project brand, but the runtime layer keeps the `x-ui` command,
service name, database path, API shape, and runtime directory. This preserves
compatibility with the existing operational ecosystem and reduces deployment
risk.

## Add N5 as an Extension Layer

N5 features are implemented through N5-specific tables, services, controllers,
HTML pages, and Xray merge logic. The original x-ui base config remains the
base layer; N5 generates additional outbound, balancer, and routing fragments.

## Simple and Advanced Ownership

Simple mode owns policies marked with `n5-simple-exec|` and legacy `n5-simple|`.
Advanced APIs must not mutate those policies. Advanced users still own ordinary
Advanced policies and their `sort_order` semantics.

This boundary prevents two control planes from destructively editing the same
runtime state.

## Custom Group Snapshot

Custom group execution uses a snapshot of the group rules at execution time.
This makes runtime behavior stable even if the reusable source group changes
later. The user can update execution state intentionally when needed.

## Group Rename Keeps `groupId`

Renaming a custom group changes display text only. The identity and execution
binding remain `groupId`, so rename does not break routing references.

## ALL Uses `defaultTarget`

ALL is represented by `TrafficPolicy.defaultTarget`, not by a fake catch-all
rule. This keeps fallback semantics separate from specific rules and makes
priority easier to reason about.

## Deterministic Simple Routing Priority

Simple routing cannot depend on creation order. The final order is:

```text
direct exact
> direct suffix
> direct keyword
> direct regexp
> custom group
> AI
> Game
> Streaming
> ALL
```

Advanced policies retain their manual ordering model.

## Avoid DB Schema Churn

The v0.2.0 hardening work intentionally avoided large DB schema changes. Safety
was achieved through service/controller ownership checks, transaction handling,
and consistency validation.

## Custom Xray 26.5.3

The Stable runtime is N5 custom Xray 26.5.3 amd64. It is fixed to preserve
runtime behavior, including Access IP log marker compatibility. The installer
does not fall back to unrelated upstream runtime assets.

## ARM64 Stable Runtime Not Claimed

Source code may contain upstream ARM64 references, but `v0.2.0` only claims the
validated amd64/x86_64 Stable runtime path.

## Clean Install Only

The release was validated on a fresh Debian 11 amd64/x86_64 system. In-place
upgrade from older versions is intentionally not claimed until a separate
Upgrade Gate validates it.
