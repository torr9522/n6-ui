# P1 Hardening

## P1-01: Egress Reference Protection

- Commit: `1d89561c21e6238e8a8a224e62f7e954ddd8c8fb`
- Public tag: `n5-local-p1-01-egress-reference-protection`

Problem: an egress could be deleted while still referenced by routing state.

Previous risk:

- `TrafficPolicy.default_target_id`
- `TrafficPolicyRule.target_id`
- egress pool members
- egress pool fallback target

could point to an egress that no longer existed.

Design decision: deletion is rejected while references exist. The user must
remove or change references first.

Final semantics: no missing egress target should be introduced by the delete
API. Once references are removed, deletion is allowed.

Validation: API/browser delete rejection, reference removal, second delete pass,
and DB consistency checks.

## P1-02: Inbound Delete Cleanup

- Commit: `93e2616fcfd6ea9a486788554aa0d36c550fc322`
- Public tag: `n5-local-p1-02-inbound-delete-cleanup`

Problem: deleting an inbound could leave orphan Simple execution state or remove
too much unrelated N5 data.

Design decision: clean only the deleted inbound binding and its Simple-managed
execution policy/rules when the policy is not shared by another inbound.

Must preserve:

- source custom groups
- other inbound execution state
- ordinary Advanced policies
- reusable group rules

Final semantics: inbound A deletion cleans A only; inbound B and source custom
groups remain intact.

Validation: orphan policy/rule/binding/metadata counters all zero for the deleted
inbound while unrelated records remain.

## P1-03: Advanced Protects Simple-Managed State

- Commit: `61c622d169d12ab667981f18333d2fb46f27cfb2`
- Public tag: `n5-local-p1-03-advanced-simple-protection`

Problem: Advanced APIs could mutate or destroy policies that Simple mode owns.

Ownership markers:

- current marker: `n5-simple-exec|`
- legacy marker: `n5-simple|`

Design decision: Advanced APIs reject destructive or mutating operations against
Simple-managed policies, rules, bindings, default targets, and reorders.

Final semantics:

- Simple service internals retain lifecycle control over Simple-owned state.
- Ordinary Advanced policies remain fully editable through Advanced APIs.

Validation: Advanced API rejection matrix for Simple-managed state and positive
CRUD coverage for ordinary Advanced policy state.
