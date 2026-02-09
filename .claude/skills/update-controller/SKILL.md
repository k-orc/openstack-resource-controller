---
name: update-controller
description: Update an existing ORC controller. Use when adding fields, making fields mutable, adding tag support, or improving error handling.
disable-model-invocation: true
---

# Update Existing Controller

Guide for modifying an existing ORC controller.

**Reference**: See `website/docs/development/` for detailed patterns and rationale.

## Key Principles

When updating controllers, follow the patterns in @.claude/skills/new-controller/patterns.md

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

5. **Update tests** to cover the new field

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

3. Add a reconciler function to handle updates:
   ```go
   func (actuator myActuator) updateMyField(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
       // Compare spec vs osResource
       // Call OpenStack update API if different
       // Return progress.NeedsRefresh() if updated
   }
   ```

4. Add the reconciler to `GetResourceReconcilers()`:
   ```go
   func (actuator myActuator) GetResourceReconcilers(...) ([]resourceReconciler, progress.ReconcileStatus) {
       return []resourceReconciler{
           actuator.updateMyField,
       }, nil
   }
   ```

### Adding a Dependency

See @.claude/skills/add-dependency/SKILL.md for detailed steps.

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

For resources with provisioning states, define constants in `types.go`:

```go
const (
    MyResourceProvisioningStatusActive        = "ACTIVE"
    MyResourceProvisioningStatusPendingCreate = "PENDING_CREATE"
    MyResourceProvisioningStatusPendingUpdate = "PENDING_UPDATE"
    MyResourceProvisioningStatusPendingDelete = "PENDING_DELETE"
    MyResourceProvisioningStatusError         = "ERROR"
)
```

Use these constants in actuator and status writer for consistency.

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

Follow @.claude/skills/testing/SKILL.md for running unit tests, linting, and E2E tests.

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
