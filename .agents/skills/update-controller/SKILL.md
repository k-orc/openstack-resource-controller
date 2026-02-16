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

When updating controllers, follow the patterns in @.agents/skills/new-controller/patterns.md

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

3. Add update handling to the `updateResource()` reconciler (or create it if not present):
   ```go
   func (actuator myActuator) updateResource(...) progress.ReconcileStatus {
       var updateOpts resources.UpdateOpts
       // Add a handleXXXUpdate() call for each mutable field
       handleMyFieldUpdate(&updateOpts, resource, osResource)
       // Call API only if something changed
       if updateOpts != (resources.UpdateOpts{}) {
           _, err := actuator.osClient.UpdateResource(ctx, *obj.Status.ID, updateOpts)
           // ...
       }
   }

   func handleMyFieldUpdate(updateOpts *resources.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
       if resource.MyField != nil && *resource.MyField != osResource.MyField {
           updateOpts.MyField = resource.MyField
       }
   }
   ```

   **Note**: Only create a separate reconciler method if the field requires a different API call (e.g., tags on networking resources use a separate tags API).

4. Register in `GetResourceReconcilers()`:
   ```go
   return []resourceReconciler{
       actuator.updateResource,
   }, nil
   ```

### Adding a Dependency

See @.agents/skills/add-dependency/SKILL.md for detailed steps.

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

2. Sort tags before creation (deterministic state):
   ```go
   tags := make([]string, len(resource.Tags))
   for i := range resource.Tags {
       tags[i] = string(resource.Tags[i])
   }
   slices.Sort(tags)
   createOpts.Tags = tags
   ```

3. Add tag update handler with sorting:
   ```go
   func handleTagsUpdate(updateOpts *resources.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
       desiredTags := make([]string, len(resource.Tags))
       for i := range resource.Tags {
           desiredTags[i] = string(resource.Tags[i])
       }
       slices.Sort(desiredTags)

       currentTags := make([]string, len(osResource.Tags))
       copy(currentTags, osResource.Tags)  // Don't mutate original
       slices.Sort(currentTags)

       if !slices.Equal(desiredTags, currentTags) {
           updateOpts.Tags = &desiredTags
       }
   }
   ```

4. Register in `GetResourceReconcilers()`:
   ```go
   return []resourceReconciler{
       actuator.updateResource,  // includes handleTagsUpdate
   }, nil
   ```

**Note**: Import `"slices"` for sorting/comparison functions.

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

See also `@.agents/skills/new-controller/patterns.md` for more details on this pattern.

### Improving Error Handling

Ensure proper error classification:

```go
// Terminal: Invalid configuration - user must fix spec
if !orcerrors.IsRetryable(err) {
    err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration,
        "invalid configuration: "+err.Error(), err)
}
return nil, progress.WrapError(err)

// Conflict on update: Treat as terminal
if orcerrors.IsConflict(err) {
    err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration,
        "invalid configuration updating resource: "+err.Error(), err)
}
```

## Testing Changes

Follow @.agents/skills/testing/SKILL.md for running unit tests, linting, and E2E tests.

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
