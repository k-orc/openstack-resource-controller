---
name: update-controller
description: Update an existing ORC controller. Use when adding fields, making fields mutable, adding tag support, or improving error handling.
disable-model-invocation: true
---

# Update Existing Controller

Guide for modifying an existing ORC controller.

**Reference**: See `website/docs/development/` for detailed patterns and rationale.

## Before Making Changes

Research the resource before implementing changes:

1. **Check gophercloud** for the resource's API:
   ```bash
   go doc <gophercloud-module>.UpdateOpts
   go doc <gophercloud-module>.CreateOpts
   ```

2. **Check existing controller** patterns:
   - How are similar fields handled?
   - Does the resource have intermediate provisioning states?
   - How are tags updated (standard Update API or separate tags API)?

3. **Check OpenStack API documentation** for:
   - Field constraints (max lengths, allowed values)
   - Mutability (can the field be updated after creation?)

## Key Principles

When updating controllers, follow the patterns in [patterns.md](../new-controller/patterns.md)

## Common Update Scenarios

### Adding a New Field to Spec

1. **Update API types** in `api/v1alpha1/<kind>_types.go`:
   - Add field to `<Kind>ResourceSpec`
   - Add corresponding field to `<Kind>ResourceStatus`
   - Add validation markers (`+kubebuilder:validation:*`)

2. **Update actuator** in `internal/controllers/<kind>/actuator.go`:
   - Add field to `CreateOpts` in `CreateResource()`
   - If mutable, add update logic in reconciler

3. **Update status writer** in `internal/controllers/<kind>/status.go`:
   - Add field mapping in `ApplyResourceStatus()`

4. **Regenerate**:
   ```bash
   make generate
   ```

5. **Update tests** to cover the new field (add only what's relevant to your change):
   - Unit tests in `internal/controllers/<kind>/actuator_test.go` (if complex logic)
   - E2E tests in `internal/controllers/<kind>/tests/`:
     - `create-full`: Set new field to non-default value and verify
     - `create-minimal`: Verify default value behavior (if field has defaults)
     - `update`: Test setting and unsetting the field (only if field is mutable)
     - `*-dependency`: Test dependency behavior (only if adding a new dependency)
     - `*import*`: Test import filtering (only if adding a new filter field)

### Adding a New Filter Field

1. Add field to `<Kind>Filter` in `api/v1alpha1/<kind>_types.go`

2. Update `ListOSResourcesForImport()` in actuator to apply the filter

3. Add import test case

### Making a Field Mutable

1. Remove immutability validation from the field:
   ```go
   // Remove or update this validation
   // +kubebuilder:validation:XValidation:rule="self == oldSelf"
   ```

2. Implement `GetResourceReconcilers()` if not already present

3. Add update handling to the `updateResource()` reconciler (or create it if not present). Follow the pattern in `internal/controllers/securitygroup/actuator.go` (or `trunk/actuator.go`):
   - Build an `UpdateOpts` struct using `handleXXXUpdate()` helpers for each mutable field
   - Use a `needsUpdate()` helper that serializes the opts to a map and checks `len() > 0`
   - Call the Update API only if something changed, return `progress.NeedsRefresh()`
   - Return terminal error if `spec.resource` is nil

   **Note**: Only use `updateResource` when the field is updated via the resource's standard Update API. If the field requires a different API (e.g., extra specs, subports, tags on networking resources), create a separate single-concern reconciler instead. See [Adding a Single-Concern Reconciler](#adding-a-single-concern-reconciler) below.

4. Register in `GetResourceReconcilers()`:
   ```go
   return []resourceReconciler{
       actuator.updateResource,
   }, nil
   ```

### Adding a Single-Concern Reconciler

When a mutable field uses a separate OpenStack API (not the resource's Update API), create a dedicated reconciler with a descriptive verb+noun name instead of adding logic to `updateResource`.

**Examples**: `reconcileExtraSpecs` (flavor, volumetype), `reconcileSubports` (trunk), `reconcilePassword` (user), `updateRules` (securitygroup).

Key differences from `updateResource`:

- **Naming**: Use a descriptive name (e.g., `reconcileExtraSpecs`), not `updateResource`.
- **Nil guard**: Return `nil` when `spec.resource` is nil (not a terminal error). The terminal error pattern is reserved for `updateResource`.
- **Multiple API calls**: A single-concern reconciler may make multiple API calls (e.g., create some extra specs, delete others). This is an established pattern (see `reconcileSubports`, `updateRules`).
- **Idempotency**: Operations must be idempotent. If the reconciler fails partway through, the next reconciliation recomputes the diff from the current OpenStack state and retries only what's still needed.

```go
func (actuator myActuator) reconcileExtraSpecs(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
    resource := obj.Spec.Resource
    if resource == nil {
        return nil  // Not a terminal error (unlike updateResource)
    }

    // Compute desired vs current diff
    // Make API calls (creates, updates, deletes)
    // Return progress.NeedsRefresh() if any changes were made
}
```

Register alongside other reconcilers in `GetResourceReconcilers()`:
```go
return []resourceReconciler{
    actuator.updateResource,       // general field updates via Update API
    actuator.reconcileExtraSpecs,  // single-concern: separate API
}, nil
```

**Do NOT duplicate work in `CreateResource`**. If a reconciler handles a concern (e.g., extra specs), do not also set that data in `CreateResource`. The `CreateResource` contract forbids actions that can fail after creating the primary resource. The reconciler will handle it on the first reconciliation after creation.

### Adding a Dependency

See [add-dependency](../add-dependency/SKILL.md) for detailed steps.

### Improving DeleteResource

For resources with intermediate provisioning states, ensure robust deletion:

```go
func (actuator myActuator) DeleteResource(ctx context.Context, _ orcObjectPT, resource *osResourceT) progress.ReconcileStatus {
    // Handle intermediate states
    switch resource.ProvisioningStatus {
    case ProvisioningStatusPendingDelete:
        return progress.WaitingOnOpenStack(progress.WaitingOnReady, deletingPollingPeriod)
    case ProvisioningStatusPendingCreate, ProvisioningStatusPendingUpdate:
        // Can't delete in pending state, wait for ACTIVE
        return progress.WaitingOnOpenStack(progress.WaitingOnReady, availablePollingPeriod)
    }

    err := actuator.osClient.DeleteResource(ctx, resource.ID)
    // Handle 409 (state changed between check and API call)
    if orcerrors.IsConflict(err) {
        return progress.WaitingOnOpenStack(progress.WaitingOnReady, deletingPollingPeriod)
    }
    return progress.WrapError(err)
}
```

**Important**: Never use cascade delete unless explicitly requested by the user.

### Adding Tag Support

**Note**: Tag handling varies by OpenStack service. Some services (e.g., block storage) include tags in the standard Update API, while others (e.g., networking) require a separate tags API and a dedicated reconciler. Check gophercloud for the specific resource.

1. Add `Tags` field to spec and status:
   ```go
   // In ResourceSpec
   // +kubebuilder:validation:MaxItems:=64
   // +listType=set
   Tags []NeutronTag `json:"tags,omitempty"`

   // In ResourceStatus
   // +listType=atomic
   Tags []string `json:"tags,omitempty"`
   ```

2. Sort tags before creation and comparison (use `slices.Sort` — see `patterns.md` §3 Deterministic State).

3. Add a `handleTagsUpdate()` helper that sorts both desired and current tags, compares with `slices.Equal`, and sets `updateOpts.Tags` only if different. Copy before sorting to avoid mutating the original.

4. Register `updateResource` (which calls `handleTagsUpdate`) in `GetResourceReconcilers()`.

### Adding Status Constants

For resources with provisioning states, prefer using constants from gophercloud when available. Only define constants in ORC's `types.go` if gophercloud doesn't provide them.

```go
// Prefer gophercloud constants when available:
import "github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/loadbalancers"
if osResource.ProvisioningStatus == loadbalancers.ProvisioningStatusActive { ... }

// Only define in types.go if gophercloud doesn't have them:
const (
    MyResourceProvisioningStatusActive        = "ACTIVE"
    MyResourceProvisioningStatusPendingCreate = "PENDING_CREATE"
    MyResourceProvisioningStatusError         = "ERROR"
)
```

See also [patterns.md](../new-controller/patterns.md) for more details on this pattern.

### Improving Error Handling

See [patterns.md](../new-controller/patterns.md) §4 Error Classification. Wrap non-retryable errors with `orcerrors.Terminal`; leave transient errors as-is for automatic retry.

## Testing Changes

Follow [testing](../testing/SKILL.md) for running unit tests, linting, and E2E tests.

## Checklist

- [ ] API types updated with proper validation
- [ ] Actuator updated (create/update logic)
- [ ] Status writer updated
- [ ] `make generate` runs cleanly
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] E2E tests updated/added
- [ ] E2E tests passing
- [ ] Unit tests added (if complex logic)
