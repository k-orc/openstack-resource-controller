# Enhancement: Glance Image Member Sharing

| Field | Value |
|-------|-------|
| **Status** | implementable |
| **Author(s)** | @zunken1337 |
| **Created** | 2026-09-30 |
| **Last Updated** | 2026-10-09 |
| **Tracking Issue** | TBD |

## Summary

Add support for sharing a Glance `Image` with other OpenStack projects or users via Glance's
member API. Two new pieces are proposed: a `members` field on the existing `Image` resource for
the owner-side grant step, and a new `ImageMember` resource for the consumer-side accept step.
These are separate resources because Glance's accept operation must be performed with the
*member's* own credentials, which the owning `Image` object's `cloudCredentialsRef` cannot provide
- no existing K-ORC resource currently needs two different projects' credentials to reach a single
desired state (see "Two-credential design" under Risks and Edge Cases).

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

- Declaratively grant image access to one or more OpenStack projects or users from the owning
  `Image` object, using that object's own credentials.
- Declaratively accept a shared image on the consumer's side, using that consumer's own
  credentials - the step the owner's credentials cannot perform.
- Keep the declared member list in sync with Glance's real one, same full-sync approach
  `SecurityGroup`'s rule reconciliation already uses.
- Let the consumer declare its desired membership status (accepted/rejected), and let removing the
  `ImageMember` object revert that status to `pending` - revocation itself stays an owner-side
  operation, done by removing the entry from `Image.spec.resource.members` instead (see "Revoke is
  owner-side only" under Risks and Edge Cases).

## Non-Goals

- `community`-visibility images, which don't use the member list / accept flow at all - already
  fully expressible via the existing `visibility` field alone.
- Managing the lifecycle of the projects/users being shared with - members are expressed as a
  `ProjectRef`/`UserRef` (see Proposal below), so the referenced object can be created before or
  after, by this repo's own `Project`/`User` CRDs or externally. If a referenced object is deleted
  while still listed as a member, the grant reconciler keeps retrying the dependency wait rather
  than silently dropping the entry - matches how every other `*Ref` field in this codebase treats
  a missing-by-design dependency.
- Supporting OpenStack deployments whose Glance policy permits the image *owner* to accept on a
  member's behalf (some custom `policy.yaml` configurations allow this via an elevated role). The
  design below assumes Glance's documented default policy, where accept requires the member's own
  token - the safer, more portable assumption. Deployments with a relaxed policy could still use
  this design; they'd simply never need it.

## Proposal

### Grant: a `members` field on `Image`

Glance's member API (`POST /v2/images/{image_id}/members`) takes one opaque `member_id` per entry.
[Glance's own API reference](https://docs.openstack.org/api-ref/image/v2/index.html) documents
that id as the consumer's *project* id, and the default policy check is
`project_id:%(member_id)s`. Some deployments customize `policy.yaml` to check `user_id` instead -
same API call, same field, different id. `ImageMemberGrant` supports both, exactly one per entry:

```go
// ImageMemberGrant identifies a project or user an image is shared with. Exactly one of
// projectRef/userRef must be set - which one depends on whether the cloud's own Glance member
// policy checks the consumer's project id (Glance's documented default) or user id.
// +kubebuilder:validation:XValidation:rule="has(self.projectRef) != has(self.userRef)",message="exactly one of projectRef or userRef must be set"
type ImageMemberGrant struct {
    // projectRef is a reference to a Project ORC object to share this image with.
    // +optional
    ProjectRef *KubernetesNameRef `json:"projectRef,omitempty"`

    // userRef is a reference to a User ORC object to share this image with.
    // +optional
    UserRef *KubernetesNameRef `json:"userRef,omitempty"`
}
```

Added to `ImageResourceSpec`:

```go
// members specifies the list of projects/users this image is shared with.
// The image visibility must be set to "shared" for members to be effective.
// +kubebuilder:validation:MaxItems:=256
// +listType=atomic
// +optional
Members []ImageMemberGrant `json:"members,omitempty"`
```

`+listType=atomic`, not `map`: with two alternative reference fields there's no single field left
to serve as the SSA merge key, so the whole list is owned by one field manager - no different from
`SecurityGroup.spec.resource.rules`, which is also atomic.

Reconciled by a new single-concern reconciler (same shape as `SecurityGroup`'s `updateRules`,
registered in `GetResourceReconcilers` alongside the existing tag/property reconcilers): list the
image's current members from Glance, diff against `spec.resource.members`, `POST` new ones,
`DELETE` ones no longer declared.

### Status: member acceptance surfaced on `Image`

Added to `ImageResourceStatus`, so the owner side can see each member's real acceptance state
without needing cluster access to the member's own project/namespace:

```go
// ImageMemberStatus reports one member's real Glance acceptance state. Exactly one of
// projectID/userID is set, matching whichever ref granted this entry.
// +kubebuilder:validation:XValidation:rule="has(self.projectID) != has(self.userID)",message="exactly one of projectID or userID is set"
type ImageMemberStatus struct {
    // projectID is the Keystone project id this entry refers to, if granted via projectRef.
    // +optional
    ProjectID *string `json:"projectID,omitempty"`

    // userID is the Keystone user id this entry refers to, if granted via userRef.
    // +optional
    UserID *string `json:"userID,omitempty"`

    // status is this member's current acceptance status in Glance:
    //   - pending: Glance already grants access, but the image won't appear when the member
    //     lists images until they accept.
    //   - accepted: the member has access and the image appears in their listing.
    //   - rejected: the member explicitly declined; access is revoked.
    Status string `json:"status,omitempty"`
}
```

```go
// members reports the real Glance membership status of every project/user this image has been
// granted to, independent of whether each has accepted yet.
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

// MemberIDType selects which of the credential's own identity claims to present as the Glance
// member_id on accept - must match whichever ref type granted this member on the Image side.
// +kubebuilder:validation:Enum:=project;user
type MemberIDType string

const (
    MemberIDTypeProject MemberIDType = "project"
    MemberIDTypeUser    MemberIDType = "user"
)

type ImageMemberResourceSpec struct {
    // imageRef references the (owner's) Image object to accept a share for.
    // +required
    ImageRef KubernetesNameRef `json:"imageRef,omitempty"`

    // memberIDType selects whether this credential's project id or user id is the Glance
    // member_id to accept with. Defaults to "project", Glance's documented default; set to "user"
    // only if the Image side granted this member via userRef.
    // +kubebuilder:default:=project
    // +optional
    MemberIDType *MemberIDType `json:"memberIdType,omitempty"`

    // status is this member's desired membership status for the image.
    // Defaults to "accepted" - the common case, where a member just wants the shared image
    // usable. Set to "rejected" to explicitly decline a share while still recording that decision
    // as an ORC object (vs. simply never creating an ImageMember at all, which leaves the
    // membership "pending" and undecided).
    // +kubebuilder:default:=accepted
    // +optional
    Status *ImageMembershipStatus `json:"status,omitempty"`
}
```

Example usage, project-scoped (the common case):

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
    # to the member's own token.
    secretName: acme-cloud-config
    cloudName: acme
  resource:
    imageRef: x2s-rhcos-image
    # memberIdType omitted - defaults to "project".
    # status omitted - defaults to "accepted".
```

User-scoped, on a cloud whose Glance member policy checks `user_id`:

```yaml
apiVersion: openstack.k-orc.cloud/v1alpha1
kind: ImageMember
metadata:
  name: acme-alice-rhcos-accept
  namespace: openstack-customers
spec:
  cloudCredentialsRef:
    secretName: acme-alice-cloud-config
    cloudName: acme-alice
  resource:
    imageRef: x2s-rhcos-image
    memberIdType: user
```

The controller resolves `imageRef` to the owning `Image` object (same-namespace reference, the
existing `dependency.FetchDependency` pattern used throughout the codebase) to get its
`status.id`, resolves its own credential's project or user id per `memberIdType` (from the auth
token's scope, the same mechanism already used for `ProjectRef` resolution elsewhere), and calls
Glance's member-status endpoint (`PUT .../members/{member_id}`, `{"status": "<spec.status>"}`)
using its **own** credential - the same single API call sets either `accepted` or `rejected`, so
`Status` drives it directly with no branching logic needed in the reconciler itself.

On delete, the controller makes the same `PUT` call one more time with `{"status": "pending"}`,
still using its own credential - this is a status *reset*, not a revoke (see "Revoke is
owner-side only" under Risks and Edge Cases for why deletion deliberately doesn't call Glance's
member-`DELETE` at all).

## Risks and Edge Cases

- **Two-credential design is new for K-ORC, confirmed acceptable on review.** Every existing
  resource type reconciles against a single project via one `cloudCredentialsRef`. This
  enhancement's two halves (`Image` and `ImageMember`) each still only use one credential
  individually - achieving the *combined* desired state (image usable by the member) requires both
  objects, in two different projects, to reconcile correctly, but neither object itself holds more
  than one credential. Raised during review as a possible precedent concern; maintainer feedback:
  this is fine, K-ORC will use as many CRDs as there are credentials needed.

- **`memberIdType` is explicit, not auto-detected.** The controller could instead try `project_id`
  first and fall back to `user_id` on a 404/403. Rejected: that hides which identity claim is
  actually in effect, turns a config mistake into a silent extra API round-trip instead of an
  immediate, legible error, and departs from this codebase's general preference for explicit
  declared state over inference. The field defaults to `project`, Glance's own documented default,
  so the common case needs no extra configuration.

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
  entry from `Image.spec.resource.members`, which the grant reconciler already handles with the
  owner's own credential, no asymmetry at all. **Implementation note**: the `ImageMember`
  controller must not add a finalizer on the referenced `Image` - an owner should always be able
  to delete an `Image` regardless of how many `ImageMember` objects reference it elsewhere.

- **Ordering is already handled by existing K-ORC machinery, not bespoke design.** `ImageMember`
  needs the referenced `Image` to be `Available`, and specifically to have already granted this
  member, before `PUT .../members/{member_id}` can succeed (Glance 404s a member that doesn't
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
  this enhancement (the image still isn't usable by the member without a separate, non-declarative
  accept step). Noted as the minimal fallback if the two-credential design below isn't accepted.

- **A single `members` field with a per-member accept-credential reference, all reconciled inside
  `Image`'s own controller** - would require a single `Image` object's reconcile loop to hold and
  use credentials for N *other* projects/users simultaneously. Rejected in favor of the
  two-resource design above: it's a bigger deviation from "one object, one credential, one
  project" than adding a second resource type, and mixes the owner's and every member's
  failure/permission domains into one reconcile loop instead of keeping them independently owned
  and independently failing.

- **Project-only (`projectRef`), documenting user-keyed Glance policies as an unsupported
  limitation** - simpler CRD (no CEL xor-validation, no `memberIdType`), but leaves clouds running
  a user-keyed member policy unable to use this enhancement at all. Rejected since both ref types
  resolve to the same single Glance `member_id` field regardless - supporting both costs one extra
  optional field on each side, not a second implementation.

## Implementation History

- 2026-09-30: Enhancement proposed
