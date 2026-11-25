# Enhancement: API Compatibility for ORC v3

| Field | Value |
|-------|-------|
| **Status** | Draft |
| **Author(s)** | mandre |
| **Created** | 2026-11-05 |
| **Last Updated** | 2026-11-24 |
| **Tracking Issue** | TBD |

## Summary

Restore the v2.6.0 `openstack.k-orc.cloud/v1alpha1` desired-state contract
before stable ORC v3.0.0. Preserve live objects and their OpenStack resource
associations during supported upgrades. Whether v3 needs `v1alpha2` depends
on whether we want to keep the new Server field layout and which prerelease
clients we intend to support.

## Motivation

Both v2.6.0 and v3.0.0-pre1 use `v1alpha1`, but pre1 moved Server
`spec.resource.serverGroupRef` under `schedulerHints`. The API version no
longer tells clients which shape to expect. The published pre1 Port CRD also
reduced the per-item length limit on `spec.resource.securityGroupRefs` from
255 to 253 characters. These changes affect existing manifests and live
objects regardless of what the ORC release or Go module version permits.

## Goals

- Keep valid v2.6.0 manifests usable without edits. Restore the top-level
  Server group reference in `v1alpha1`, keep its effective-value immutability
  behavior, and restore the Port reference item limit to 255 characters.
  Restoring that limit does not make a longer string a valid Kubernetes
  object name.
- Keep compatible additions, including the other Server scheduler hints and
  Network `spec.resource.segments`. Restore the compatibility contract, not
  the exact historical CRDs or Go types.
- Upgrade objects in place. Keep desired state, UIDs, finalizers, management
  policies, and OpenStack resource associations. Do not delete and recreate
  CRs or CRDs as part of migration.
- Make any pre1 upgrade and rollback commitments explicit. If an in-place
  upgrade from pre1 is supported, preserve its desired state and associations
  through a tested procedure.

## Non-Goals

- Graduate the API to beta or stable or revert unrelated Go-only changes.
- Make old controllers understand features introduced after their release.
- Guarantee that an old typed client can replace an object and retain fields
  its type does not know about. This is a separate support commitment from
  accepting old manifests and is only considered in alternative C.
- Guarantee lossless observed status round trips or preserve omissions on
  legacy `/status` writes.

## Proposal

### Compatibility policy

ORC releases and exported Go APIs continue to follow semantic versioning.
After this repair, use a new Kubernetes API version and an upgrade plan for
incompatible changes to the supported desired-state API. Decide separately
whether prerelease clients need ongoing support and when to retire a served
API version.

### Status compatibility versus upgrade safety

Observed status is not a stable API contract for this transition. Network
`status.resource.provider` may be removed, including from `v1alpha1`. Document
that readers should use `status.resource.segments`. Keep the complete segment
list where it is exposed. Do not archive `provider` in an annotation or
reconstruct it from a segment: the two observations are not equivalent.

Preserve operational state through supported upgrades. The generic reconciler
uses `status.id` to find the existing OpenStack resource before considering
import, adoption, or creation. Other controllers rely on compound
associations, such as the RoleAssignment role, actor, and scope tuple, or on
checkpoints, such as the Image download-attempt count and the User applied
password reference. Retain these and any dependency inputs that would make
reconciliation unsafe if lost. This is an upgrade invariant, not a promise
that old clients can safely replace `/status`. None of the alternatives
requires a `/status` preservation webhook or routing controller status writes
through a new API version for the sake of that webhook.

## Alternatives Considered

### A. Restore `v1alpha1` only

Keep `spec.resource.serverGroupRef` as the canonical field. Retain other
scheduler hints. If supporting pre1 upgrades, provide a one-time transition
schema that accepts both paths. With writers that still use the pre1 path
paused, move each affected live reference in place, verify its effective
value, and update source manifests before installing the final schema. Reject
conflicting aliases instead of picking one. The migration must keep unrelated
hints and must not loosen immutability for actual placement changes.

This avoids a second served version, conversion, and write-preservation
admission. It is the lowest-cost option if the nested spelling is not worth
keeping as a long-term API. Pre1 clients do not retain ongoing support.

### B. Restore `v1alpha1` and introduce `v1alpha2` with a bounded pre1 transition

Keep the top-level reference in the final `v1alpha1` schema and the nested
reference in `v1alpha2`. Use a CRD conversion webhook to map between them and a
single reconciliation implementation. Conversion must preserve desired state
and operational state without calling OpenStack or resolving Secret
references. Decide whether to serve `v1alpha2` for Server alone or for all
kinds as a consistent set. Keep `v1alpha1` as storage for the first release
serving both versions; defer storage migration and version retirement.

If supporting pre1 upgrades, accept both historical shapes during the
transition. Normalize live references to a representation the final storage
schema can retain before tightening it. Pre1 users then update **custom
resource manifests** and clients to `v1alpha2`; ORC supplies the CRDs. There
is no continuing promise that the pre1 `v1alpha1` spelling works, and no
write-preservation admission for old clients.

Choose this option if the nested reference is worth retaining. It adds a
conversion webhook and its deployment and recovery requirements without
adding a second compatibility policy for updates.

### C. Support both historical `v1alpha1` client shapes during the transition

Serve `v1alpha2` as in alternative B, but keep both Server aliases in
`v1alpha1` for a defined support window. Treat equal aliases as one logical
value and reject unequal aliases. If existing objects must expose both aliases
to `v1alpha1` readers, explicitly normalize them: a direct read at the storage
version does not do that for objects already stored in that version.

If the support commitment includes legacy full-object writes, protect a
reviewed set of desired-state fields on `v1alpha1` main-resource updates. An
omission can mean either that the client does not know the field or that it
wants to remove it. Preserving the value by default changes historical
omission-based removal behavior. Require `v1alpha2` for intentional removal
of protected values. Do not make `/status` fields sticky.

Choose this option only if both legacy client shapes need ongoing support
and the changed removal semantics are acceptable. It has the largest
validation, admission, and rollout surface.

## Risks and Edge Cases

- **Pruning during a partial migration:** A rewrite can drop fields that the
  CRD schema no longer recognizes. Back up live objects and verify the migrated
  references before dropping either field. Account for GitOps or other writers
  that could restore the old representation.
- **Serving versus storage:** Serving a new version or editing a manifest's
  `apiVersion` does not rewrite live objects. Applying the edited manifest
  writes through the selected version, but the CRD still controls the storage
  version. Changing that storage version does not rewrite existing records.
- **Conflicting aliases or unintended placement changes:** Reject unequal
  aliases. Test that a representation-only move is allowed but does not
  authorize a change to the effective group.
- **Unsafe downgrade:** Document supported recovery steps for each release
  stage. Do not recommend reinstalling historical CRDs as a generic rollback.
- **Webhook failure in dual-version deployments:** Deploy the webhook before
  configuring CRDs to call it. Verify API-server trust and availability, not
  only that the webhook process reports ready.

## Release Gates

**All alternatives:** Use historical CRDs and representative object fixtures
to test an in-place upgrade from v2.6.0. Check manifest validation, desired
state, UIDs, finalizers, management policies, and OpenStack associations. Verify
that omitted observations do not affect reconciliation safety. If supporting
pre1, test the published migration order, including retry after interruption
and preservation of the live reference before tightening its schema. If no
tested procedure ships, explicitly state in the release notes that in-place
upgrade from v3.0.0-pre1 is unsupported.

- **A:** Test the one-time field move against the final `v1alpha1` schema if
  pre1 migration is supported. Verify that other scheduler hints and effective
  placement survive. No conversion webhook or storage rewrite is required.
- **B:** Test cross-version reads and writes, preservation of logical spec
  and operational state, and SSA behavior across the rename. Validate
  conversion webhook deployment, API-server trust, certificate handling,
  readiness, outage behavior, and recovery. A storage rewrite is not a gate
  for the first dual-version release.
- **C:** In addition to the dual-version gates, test both historical client
  shapes. If spec-write preservation is promised, test replacements, patches,
  same-owner SSA, conflicts, admission failure and recovery, and intentional
  removal through `v1alpha2` without resurrection or bypassing validation.

## Implementation History

- 2026-11-05: Initial draft.
- 2026-11-24: Separated the release alternatives and their upgrade requirements.

## References

- [Kubernetes CRD versioning and storage migration](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/)
- [Kubernetes API change guidance](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md)
