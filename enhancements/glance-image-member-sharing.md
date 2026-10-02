# Enhancement: Glance Image Member Sharing

| Field | Value |
|-------|-------|
| **Status** | implementable |
| **Author(s)** | @zunken1337 |
| **Created** | 2026-09-30 |
| **Last Updated** | 2026-10-02 |
| **Tracking Issue** | TBD |

## Summary

Add support for sharing a Glance `Image` with other OpenStack projects via Glance's member API.
Two new pieces are proposed: a `members` field on the existing `Image` resource for the owner-side
grant step, and a new `ImageMember` resource for the consumer-side accept step. These are separate
resources because Glance's accept operation must be performed with the *member* project's own
credentials, which the owning `Image` object's `cloudCredentialsRef` cannot provide - no existing
K-ORC resource currently needs two different projects' credentials to reach a single desired state.
Raised as an open design question in the first draft of this proposal; confirmed on review to be
an acceptable pattern (see "Two-credential design" under Risks and Edge Cases).

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
- Let the consuming project declare its desired membership status (accepted/rejected), and let
  removing the `ImageMember` object revert that status to `pending` - revocation itself stays an
  owner-side operation, done by removing the project from `Image.spec.resource.members` instead
  (see "Revoke is owner-side only" under Risks and Edge Cases).

## Non-Goals

- `community`-visibility images, which don't use the member list / accept flow at all - already
  fully expressible via the existing `visibility` field alone.
- Managing the lifecycle of the projects being shared with. Not quite "assumes they already
  exist", though - members are expressed as a `ProjectRef` (see Proposal below), so ORC's normal
  dependency handling means `Image` simply waits for a referenced `Project` to exist before
  granting it access, the same as any other ORC object-to-object reference; the project can be
  created later, by this repo's own `Project` CRD or externally, in either order.
- Supporting OpenStack deployments whose Glance policy permits the image *owner* to accept on a
  member's behalf (some custom `policy.json` configurations allow this via an elevated role). The
  design below assumes Glance's more common default policy, where accept requires the member
  project's own token - the safer, more portable assumption. Deployments with a relaxed policy
  could still use this design; they'd simply never need it.

## Proposal

### Grant: a `members` field on `Image`

```go
// ImageMemberGrant identifies a project an image is shared with. Named
// "...Grant", not "ImageMember" (as a prior review suggested), to avoid
// colliding with the ImageMember resource's own generated Go type in the
// same api/v1alpha1 package - the substance (a ProjectRef, not a raw
// project ID) is otherwise exactly that suggestion.
type ImageMemberGrant struct {
    // projectRef is a reference to a Project ORC object representing
    // the project to share this image with. ORC objects only reference
    // other ORC objects (see the design-principles doc) - the referenced
    // Project does not need to exist yet; the grant simply waits for it.
    // +required
    ProjectRef KubernetesNameRef `json:"projectRef"`
}
```

Added to `ImageResourceSpec`:

```go
// members specifies the list of projects this image is shared with, in
// addition to whatever its own visibility already grants. The image's
// visibility must be set to "shared" for members to be effective - Glance's
// member list has no effect for any other visibility value. This performs
// only the OWNER side of the share (POST .../v2/images/{id}/members) - each
// member project must separately accept the share (see the ImageMember
// resource) before the image becomes usable there.
// +kubebuilder:validation:MaxItems:=256
// +listType=map
// +listMapKey=projectRef
// +optional
Members []ImageMemberGrant `json:"members,omitempty"`
```

Reconciled by a new single-concern reconciler (same shape as `SecurityGroup`'s `updateRules`,
registered in `GetResourceReconcilers` alongside the existing tag/property reconcilers): list the
image's current members from Glance, diff against `spec.resource.members`, `POST` new ones,
`DELETE` ones no longer declared.

### Status: member acceptance surfaced on `Image`

Added to `ImageResourceStatus`, so the owner side can see each member's real acceptance state
without needing cluster access to the member's own project/namespace:

```go
// ImageMemberStatus reports one member project's real Glance acceptance state.
type ImageMemberStatus struct {
    // projectID is the Keystone project id this entry refers to.
    ProjectID string `json:"projectID,omitempty"`
    // status is this member's current acceptance status in Glance:
    // pending, accepted, or rejected.
    Status string `json:"status,omitempty"`
}
```

```go
// members reports the real Glance membership status of every project this
// image has been granted to, independent of whether each has accepted yet.
// +optional
Members []ImageMemberStatus `json:"members,omitempty"`
```

Populated by the same grant reconciler above, from the same Glance member-list response that
already drives the diff - no extra API call.

### Accept: a new `ImageMember` resource

```go
// +kubebuilder:validation:Enum:=accepted;rejected;pending
type ImageMembershipStatus string

const (
    ImageMembershipStatusAccepted ImageMembershipStatus = "accepted"
    ImageMembershipStatusRejected ImageMembershipStatus = "rejected"
    ImageMembershipStatusPending  ImageMembershipStatus = "pending"
)

type ImageMemberResourceSpec struct {
    // imageRef references the (owner's) Image object to accept a share for.
    // +required
    ImageRef KubernetesNameRef `json:"imageRef,omitempty"`

    // status is this project's desired membership status for the image.
    // Defaults to "accepted" - the common case, where a member project
    // just wants the shared image usable. Set to "rejected" to explicitly
    // decline a share while still recording that decision as an ORC
    // object (vs. simply never creating an ImageMember at all, which
    // leaves the membership "pending" and undecided).
    // +kubebuilder:default:=accepted
    // +optional
    Status *ImageMembershipStatus `json:"status,omitempty"`
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
    # status omitted - defaults to "accepted", the common case.
```

The controller resolves `imageRef` to the owning `Image` object (same-namespace reference, the
existing `dependency.FetchDependency` pattern used throughout the codebase) to get its
`status.id`, resolves its own credential's project ID (from the auth token's scope, the same
mechanism already used for `ProjectRef` resolution elsewhere), and calls Glance's member-status
endpoint (`PUT .../members/{member_id}`, `{"status": "<spec.status>"}`) using its **own**
credential - the same single API call sets either `accepted` or `rejected`, so `Status` drives it
directly with no branching logic needed in the reconciler itself.

On delete, the controller makes the same `PUT` call one more time with `{"status": "pending"}`,
still using its own credential - this is a status *reset*, not a revoke (see "Revoke is
owner-side only" under Risks and Edge Cases for why deletion deliberately doesn't call Glance's
member-`DELETE` at all).

## Risks and Edge Cases

- **Two-credential design is new for K-ORC, confirmed acceptable on review.** Every existing
  resource type reconciles against a single project via one `cloudCredentialsRef`. This
  enhancement's two halves (`Image` and `ImageMember`) each still only use one credential
  individually - achieving the *combined* desired state (image usable in the member project)
  requires both objects, in two different projects, to reconcile correctly, but neither object
  itself holds more than one credential. Raised during review as a possible precedent concern;
  maintainer feedback: this is fine, K-ORC will use as many CRDs as there are credentials needed.

- **Revoke is owner-side only - deleting `ImageMember` does not revoke access.** Glance's
  `DELETE .../members/{member_id}` (fully removing a member) is an owner-side operation - it
  requires the **image owner's** credential, not the member's. Initially this looked like a real
  asymmetry problem (the `ImageMember` object only ever holds the *member's* credential, which
  can't perform that call) requiring either borrowing the `Image`'s credential on delete or some
  other workaround. Resolved instead by not needing owner credentials on the `ImageMember`
  delete path at all: deleting the K8s object calls the *member's own* `PUT` once more with
  `{"status": "pending"}` - a status reset, not a Glance-side member removal. The member still
  technically has access (per Glance's own semantics, a `pending` member just doesn't list the
  image until re-accepted), and actual revocation is a deliberate, separate act: removing that
  project from `Image.spec.resource.members`, which the grant reconciler already handles with
  the owner's own credential, no asymmetry at all. **Implementation note**: the `ImageMember`
  controller must not add a finalizer on the referenced `Image` - an owner should always be able
  to delete an `Image` regardless of how many `ImageMember` objects reference it elsewhere.

- **Ordering is already handled by existing K-ORC machinery, not bespoke design.** `ImageMember`
  needs the referenced `Image` to be `Available`, and specifically to have already granted this
  project, before `PUT .../members/{member_id}` can succeed (Glance 404s a member that doesn't
  exist yet). Confirmed on review: this is exactly what ORC's existing dependency-wait machinery
  already does for any `ImageRef`/`ProjectRef`-style reference - no new pattern needed, just the
  standard one (`progress.WaitingOnObject`) applied here like everywhere else.

- **Idempotency.** Re-running the same `PUT .../members/{member_id}` with the same `status` is a
  no-op in Glance - matters for a member re-added after being removed and re-granted, or for a
  normal reconcile loop re-asserting an already-`accepted` status. Confirmed on review this is
  the expected, correct behavior for that case (and more generally, for re-adding a previously
  removed-then-regranted member) - the controller's own logic needs no special-casing for it.

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
- 2026-10-02: Revised per @mandre's review - `ProjectRef` instead of a raw project ID,
  `Members` as a `listType=map` keyed on `projectRef`, a `status` field on `ImageMember` for the
  consumer to declare accepted/rejected, member statuses surfaced on `Image.status`, and the
  revoke-asymmetry risk resolved (delete resets status to `pending` via the member's own
  credential rather than needing the owner's to fully remove the record - actual revocation stays
  an owner-side-only act via `Image.spec.resource.members`)
