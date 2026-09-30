# Enhancement: Glance Image Member Sharing

| Field | Value |
|-------|-------|
| **Status** | implementable |
| **Author(s)** | @zunken1337 |
| **Created** | 2026-09-30 |
| **Last Updated** | 2026-09-30 |
| **Tracking Issue** | TBD |

## Summary

Add support for sharing a Glance `Image` with other OpenStack projects via Glance's member API.
Two new pieces are proposed: a `members` field on the existing `Image` resource for the owner-side
grant step, and a new `ImageMember` resource for the consumer-side accept step. These are separate
resources because Glance's accept operation must be performed with the *member* project's own
credentials, which the owning `Image` object's `cloudCredentialsRef` cannot provide - no existing
K-ORC resource currently needs two different projects' credentials to reach a single desired state,
so this enhancement also surfaces that as a design question for maintainer input, not just a field
addition.

## Motivation

`Image` already supports `visibility: shared`, but K-ORC has no way to declare *which* projects a
shared image is actually shared with - Glance's member list, which is what visibility: shared
actually depends on, isn't represented anywhere in the CRD.

Concretely: in a real GitOps repo we maintain (OpenShift/HyperShift hosted control planes on
OpenStack), a RHCOS worker image is owned by an admin project and must be shared into each
customer's own project so their NodePool workers can boot from it - CAPO's image lookup using the
customer's own credential finds nothing until the image is both granted and accepted. Today this is
a raw Glance API script that authenticates with two separate credentials (the owner's, to grant;
the customer's own, to accept) - exactly the two-sided operation this enhancement would let K-ORC
express declaratively instead.

## Goals

- Declaratively grant image access to one or more OpenStack projects from the owning `Image`
  object, using that object's own credentials.
- Declaratively accept a shared image on the consuming project's side, using that project's own
  credentials - the step the owner's credentials cannot perform.
- Full drift correction: the declared member list matches Glance's real member list, extra members
  are revoked, same "full sync" principle already used elsewhere in this codebase (e.g.
  `SecurityGroup`'s rule reconciliation).
- Removing the `ImageMember` object revokes that project's access.

## Non-Goals

- `community`-visibility images, which don't use the member list / accept flow at all - already
  fully expressible via the existing `visibility` field alone.
- Managing the lifecycle of the projects being shared with - this assumes they already exist,
  whether via K-ORC's own `Project` CRD or provisioned externally.
- Supporting OpenStack deployments whose Glance policy permits the image *owner* to accept on a
  member's behalf (some custom `policy.json` configurations allow this via an elevated role). The
  design below assumes Glance's more common default policy, where accept requires the member
  project's own token - the safer, more portable assumption. Deployments with a relaxed policy
  could still use this design; they'd simply never need it.

## Proposal

### Grant: a `members` field on `Image`

```go
// ImageMemberRef identifies a project an image is shared with.
type ImageMemberRef struct {
    // projectID is the Keystone project id to share this image with.
    // +required
    ProjectID string `json:"projectID,omitempty"`
}
```

Added to `ImageResourceSpec`:

```go
// members is a list of projects this image is shared with, in addition to
// whatever its own visibility already grants. Only meaningful when
// visibility is "shared" - Glance's member list has no effect for any other
// visibility value. This performs only the OWNER side of the share
// (POST .../v2/images/{id}/members) - each member project must separately
// accept the share (see the ImageMember resource) before the image becomes
// usable there.
// +optional
Members []ImageMemberRef `json:"members,omitempty"`
```

Reconciled by a new single-concern reconciler (same shape as `SecurityGroup`'s `updateRules`,
registered in `GetResourceReconcilers` alongside the existing tag/property reconcilers): list the
image's current members from Glance, diff against `spec.resource.members`, `POST` new ones,
`DELETE` ones no longer declared.

### Accept: a new `ImageMember` resource

```go
type ImageMemberResourceSpec struct {
    // imageRef references the (owner's) Image object to accept a share for.
    // +required
    ImageRef KubernetesNameRef `json:"imageRef,omitempty"`
}
```

Example usage:

```yaml
apiVersion: openstack.k-orc.cloud/v1alpha1
kind: ImageMember
metadata:
  name: acme-rhcos-accept
  namespace: openstack-customers
spec:
  cloudCredentialsRef:
    # The CONSUMING project's own credential - not the image owner's. This is what actually
    # performs the accept (PUT .../members/{member_id}), which Glance's default policy restricts
    # to the member project's own token.
    secretName: acme-cloud-config
    cloudName: acme
  resource:
    imageRef: x2s-rhcos-image
```

The controller resolves `imageRef` to the owning `Image` object (same-namespace reference, the
existing `dependency.FetchDependency` pattern used throughout the codebase) to get its
`status.id`, resolves its own credential's project ID (from the auth token's scope, the same
mechanism already used for `ProjectRef` resolution elsewhere), and calls Glance's member-accept
endpoint (`PUT .../members/{member_id}`, `{"status": "accepted"}`) using its **own** credential.

## Risks and Edge Cases

- **Two-credential design is new for K-ORC.** Every existing resource type reconciles against a
  single project via one `cloudCredentialsRef`. This enhancement's two halves (`Image` and
  `ImageMember`) each still only use one credential individually, but achieving the *combined*
  desired state (image usable in the member project) genuinely requires both objects, in two
  different projects, to reconcile correctly. Flagging this explicitly as something that may
  warrant broader discussion beyond this one feature, not deciding it unilaterally here.

- **Revoke is asymmetric with accept, not just grant.** Glance's `DELETE .../members/{member_id}`
  (revoking a member entirely) is an owner-side operation - it requires the **image owner's**
  credential, not the member's, same as grant. This means when an `ImageMember` object is deleted,
  its own `cloudCredentialsRef` (the member project's) *cannot* perform the revoke - the delete
  path would need to resolve `imageRef` back to the `Image` object and use *its* credential
  instead, inverting the credential story between the create and delete paths of the same
  resource. This is a real, non-obvious wrinkle rather than a simple mirror of the accept logic,
  and is left open for reviewer input on the right way to model it (candidates: `ImageMember`'s
  delete path borrows the referenced `Image`'s credentials via `imageRef`, similar to
  `DeletionGuardDependency`'s pattern of holding a finalizer on a referenced object; or deleting
  `ImageMember` deliberately does *not* revoke Glance-side, only removes the K8s record, and
  revocation is left to removing the entry from `Image.spec.resource.members` instead - which
  *does* use the owner's own credential naturally).

- **Ordering.** `ImageMember` must wait for the referenced `Image` to be `Available` *and* for its
  member-grant reconciler to have actually added this project before attempting accept - otherwise
  Glance returns 404/409 for a member that doesn't exist yet. Standard K-ORC dependency-wait
  pattern (`progress.WaitingOnObject`).

- **Idempotency.** Re-accepting an already-accepted member should be a no-op (Glance's `PUT` to
  the same status is idempotent) - matters for a member re-added after being removed and re-granted.

## Alternatives Considered

- **Grant only, no accept resource** - add `members` to `Image` and stop there, leaving accept as
  a manual or externally-scripted step. Smaller in scope and stays within K-ORC's existing
  single-credential-per-object model, but only solves half of the real-world problem motivating
  this enhancement (the image still isn't usable in the member project without a separate,
  non-declarative accept step). Noted as the minimal fallback if the two-credential design below
  isn't accepted.

- **A single `members` field with a per-member accept-credential reference, all reconciled inside
  `Image`'s own controller** - would require a single `Image` object's reconcile loop to hold and
  use credentials for N *other* projects simultaneously. Rejected in favor of the two-resource
  design above: it's a bigger deviation from "one object, one credential, one project" than adding
  a second resource type, and mixes the owner's and every member's failure/permission domains into
  one reconcile loop instead of keeping them independently owned and independently failing.

## Implementation History

- 2026-09-30: Enhancement proposed
