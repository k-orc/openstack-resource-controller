# Enhancement: Versioned API Compatibility for ORC v3

| Field | Value |
|-------|-------|
| **Status** | Draft |
| **Author(s)** | mandre |
| **Created** | 2026-11-05 |
| **Last Updated** | 2026-11-05 |
| **Tracking Issue** | TBD |

## Summary

Introduce `openstack.k-orc.cloud/v1alpha2` before stable ORC v3.0.0. Restore
compatibility with the v2.6.0 `v1alpha1` API without losing fields written by
v3 prereleases.

Both v2.6.0 and v3.0.0-pre1 use `v1alpha1`, but define different fields. The
upgrade needs to support objects created by either release. Future breaking
Kubernetes API changes will use a new API version instead of redefining
`v1alpha1`.

## Motivation

ORC's compatibility policy protects Kubernetes and exported Go APIs within a
stable major release, but allows breaking changes across major releases and
during prereleases. In v3.0.0-pre1, Server `serverGroupRef` moved under
`schedulerHints`, but the API remained `v1alpha1`.

The version no longer tells clients which fields to expect. v2.6.0 manifests
need to keep working, and pre1 objects must not lose their newer fields.
Introducing `v1alpha2` before stable v3.0.0 gives clients a way to distinguish
these APIs.

## Goals

- Introduce `v1alpha2` for incompatible Kubernetes API changes before stable
  v3.0.0.
- Restore the removed v2.6.0 fields in `v1alpha1` with compatible field
  behavior, validation, and immutability rules.
- Preserve data written using the v3.0.0-pre1 `v1alpha1` schema.
- Convert between the served API versions without losing data.
- Migrate existing objects in place, keeping their identity and leaving
  OpenStack resources unchanged.
- Test upgrades from both v2.6.0 and v3.0.0-pre1 and document when users can
  still roll back.

## Non-Goals

- Graduate the API to beta or stable.
- Revert every Go type change, additive feature, or controller bug fix in v3.
- Make old controllers understand features introduced after their release.

## Proposal

### API compatibility policy

ORC releases and exported Go APIs still follow semantic versioning, including
prerelease rules. Breaking changes to a released Kubernetes API need a new
API version and a conversion and migration plan, regardless of the ORC release.

Decide separately how long to support `v1alpha1` and when to retire it.

### Supporting both v1alpha1 schemas

Restore these fields for v2.6.0 clients and use the new fields in `v1alpha2`:

| Resource | v2.6.0 field to restore in `v1alpha1` | `v1alpha2` field |
| --- | --- | --- |
| Server | `spec.resource.serverGroupRef` | `spec.resource.schedulerHints.serverGroupRef` |
| Network | `status.resource.provider` | `status.resource.segments` |

Keep pre1 fields such as Server `schedulerHints` and Network `segments` in
`v1alpha1` too. Clients from both releases must be able to read and update
objects without losing information; `apiVersion` cannot identify which
release created them.

Keep additive features that preserve existing behavior. Compatibility
includes validation and field behavior, but does not require reverting
Go-only changes.

Before implementation, decide whether to introduce `v1alpha2` for all ORC
kinds or only those with incompatible changes.

### Conversion and reconciliation

Use a CRD conversion webhook between the two versions and a common form for
reconciliation. Both versions must use the same reconciliation implementation
and produce the same OpenStack behavior.

Conversion must preserve spec, status, object identity, finalizers, and
unrelated metadata. It must not perform OpenStack operations or replace
Secret references with the Secrets' current contents.

Two cases need further design:

- **Server group references:** map `serverGroupRef` to
  `schedulerHints.serverGroupRef`. Define how edits work when both fields are
  present, including old-client updates. Do not silently pick one if they
  differ. New immutability rules must not block a migration that only moves
  data between fields.
- **Network segments:** the old `provider` field cannot hold multiple
  segments. Keep the full list when converting to `v1alpha1` and back, using
  additional fields or conversion annotations. Do not keep only the first
  segment.

The compatibility fields, conversion annotations, limits, and ownership
rules remain undecided. Annotations must have size limits and validation,
and must not contain private credentials.

A future KeyPair API might accept a Secret reference as an alternative to an
inline public key. Conversion would need to keep the reference, not resolve
it or copy private key material into annotations or status.

### Rollout and storage migration

1. Capture CRD and object fixtures from v2.6.0 and v3.0.0-pre1 before changing
   schemas. Include spec, status, finalizers, and management policies.
2. Update `v1alpha1` to accept and preserve fields from both releases. Test
   upgrades from both before removing fields or tightening their validation.
3. Deploy the conversion webhook before configuring CRDs to use it. Serve
   `v1alpha2` alongside `v1alpha1`, keeping the current storage version. Verify
   that clients can use both versions. Update clients, examples, and GitOps
   source manifests.
4. After a release with both versions has been tested, switch storage to
   `v1alpha2` and rewrite existing objects using a supported storage-migration
   procedure. Keep UIDs, finalizers, status IDs, and OpenStack associations.
   Verify that all objects have been rewritten before marking migration
   complete. Setting `storage: true` or reading objects is not enough.
5. Keep conversion for the old version as long as stored objects or supported
   clients need it. Remove its entry from CRD `status.storedVersions` only
   after verifying migration. Retiring the API version and removing temporary
   fields can happen later; neither is required to introduce `v1alpha2`.

Do not delete and recreate CRs or CRDs to migrate them. Test and publish
release-specific upgrade instructions covering installation order, webhook
readiness, backups, and recovery.

### Tests and release requirements

Add upgrade tests using the historical CRD and object fixtures. Keep these
separate from validation and reconciliation tests for the current API.

Cover:

- Upgrades from both v2.6.0 and pre1.
- Converting spec and status to the other version and back, including absent
  fields, explicit false values, multiple segments, and conflicting inputs.
- Creates, edits, and status-subresource writes through both versions. Include
  updates by old clients that do not know about newer fields, not only
  conversions of unchanged objects.
- CEL validation and immutability checks when moving data between fields.
- Preservation of metadata, UIDs, finalizers, status IDs, and import/detach
  policies, with no unintended OpenStack changes during migration.
- Storage rewrites, completion checks, and interrupted or retried migrations.
- Webhook readiness, certificate handling, failures, and recovery.
- Rollback at each supported stage. Reject unsupported downgrades or document
  them before users reach a point where they cannot roll back.

Do not ship this transition in stable v3.0.0 until these tests pass and the
tested upgrade procedure is published.

## Risks and Edge Cases

- **Data loss:** old schemas can drop pre1 fields. Keep support for those
  fields until migration is verified, and back up objects before changing CRDs.
- **Conflicting inputs:** an object may contain both `serverGroupRef` and
  `schedulerHints.serverGroupRef`. Define how edits to either field affect the
  other, including edits by old clients.
- **Webhook outages:** API conversion depends on webhook availability. Set up
  certificate management, readiness checks, upgrade ordering, and recovery
  before enabling it.
- **Rollback and GitOps:** document which releases users can roll back to at
  each stage. Old binaries may not understand new fields, and GitOps sources
  must be updated so old manifests do not undo the migration.
- **Scope:** versioning affects code generation, clients, CRDs, reconciliation
  adapters, packaging, and documentation. Inventory the work in each before
  estimating effort or committing to a release schedule.

## Alternatives Considered

- **Keep changing `v1alpha1` at ORC major releases:** follows the existing
  release policy, but clients still cannot tell which fields `v1alpha1`
  supports, and existing objects still need migration.
- **Restore the exact v2.6.0 schema:** supports old clients but loses pre1
  fields and leaves pre1 users without an upgrade path.
- **Add `v1alpha2` without restoring compatibility:** gives the new fields
  their own version but leaves v2.6.0 compatibility and preservation of
  existing objects unresolved.
- **Keep both sets of fields in one version permanently:** avoids a conversion
  webhook, but leaves duplicate inputs and conflicts in the API. We need both
  sets temporarily for pre1 users, not as the long-term API.

## Implementation History

- 2026-11-05: Initial draft.

## References

- [Kubernetes CRD versioning and storage migration](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/)
- [Kubernetes API change guidance](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md)
