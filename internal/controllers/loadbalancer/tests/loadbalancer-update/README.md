# LoadBalancer Update Test

This test validates that mutable fields of a LoadBalancer resource are correctly
reconciled when the spec is updated, and reverted when the spec is restored to
its original values.

## Mutable Fields Tested

- `description`: human-readable description (mutable, no immutability validation)
- `adminStateUp`: administrative state (mutable, no immutability validation)
- `tags`: list of tags (mutable, no immutability validation)

## Test Steps

### Step 00: Create LoadBalancer

Creates a Network and Subnet as VIP infrastructure prerequisites, then creates a
LoadBalancer with initial values:
- `description: initial description`
- `adminStateUp: true` (explicit default)
- No tags

Asserts:
- `Available=True` and `Progressing=False` once OpenStack provisioning reaches ACTIVE
- `status.id` is populated (non-empty)
- `status.resource.description` is `initial description`
- `status.resource.adminStateUp` is `true`
- `status.resource.tags` is absent (not set)

### Step 01: Update Mutable Fields

Patches the LoadBalancer spec to update all mutable fields:
- `description: updated description`
- `adminStateUp: false`
- `tags: [tag1, tag2]`

Asserts:
- `Available=True` and `Progressing=False` after reconciliation
- `status.resource.description` reflects `updated description`
- `status.resource.adminStateUp` is `false`
- `status.resource.tags` contains `tag1` and `tag2`

### Step 02: Revert to Original Values

Uses `kubectl replace` (not a kuttl patch) to restore the original spec, because
kuttl's default merge-patch cannot remove keys (e.g., `tags` would remain if
patched). The original `00-create-loadbalancer.yaml` is used as the source of
truth.

Asserts:
- `Available=True` and `Progressing=False` after reconciliation
- `status.resource.description` is restored to `initial description`
- `status.resource.adminStateUp` is restored to `true`
- `status.resource.tags` is absent (cleared)

## Notes

- Immutable fields (`subnetRef`, `networkRef`, `vipPortRef`, `flavor`,
  `projectRef`) are not tested here; they cannot be changed after creation.
- The `name` field is also mutable but is not tested here since the default name
  (object name) is used throughout and changes to `name` are validated in
  `loadbalancer-create-full`.
- CIDR `192.168.109.0/24` is used for the VIP subnet to avoid conflicts with
  other loadbalancer tests.

## Reference

https://k-orc.cloud/development/writing-tests/#update
