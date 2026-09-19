# Rejected and Deferred Work

## In-place Upgrade

Status: deferred.

`v0.2.0` does not claim old-version in-place upgrade compatibility. Upgrade
risk audit and Upgrade Gate are separate post-release work.

## SS2022

Status: deferred.

The release does not add or newly certify SS2022 functionality.

## Port IP Limit Redesign

Status: deferred.

Expected future direction:

- count only meaningful proxy activity
- use activity TTL
- avoid counting scanner noise as real usage
- combine valid Xray activity signals with nftables enforcement

This is not implemented in `v0.2.0`.

## Broad Mobile Redesign

Status: rejected for v0.2.0.

The release needed a minimal operability fix, not a redesign of the mobile UI.
RC Mobile therefore focused on action reachability, horizontal table scroll, and
modal footer accessibility.

## DB Schema Redesign

Status: rejected for v0.2.0.

The hardening chain avoided large schema changes because Stable readiness was
achievable through ownership checks, transaction boundaries, and consistency
audits.

## ARM64 Stable Runtime

Status: deferred.

The project may later validate and publish ARM64 runtime assets, but `v0.2.0`
does not claim ARM64 Stable Runtime support.
