# ORC Controller Implementation Patterns

Follow these principles when implementing controllers. See `website/docs/development/` for detailed rationale.

## 1. Defensive Operations

Avoid destructive defaults - require explicit user intent for dangerous operations.

**Examples:**
- Never use cascade delete unless the user explicitly requests it (cascade removes all child resources)
- Don't auto-correct invalid states that might cause data loss
- Ask the user additional questions if required
- Prefer failing safely over making assumptions

## 2. Resource Lifecycle Management

Handle all states a resource can be in throughout its lifecycle.

**For resources with intermediate provisioning states** (PENDING_CREATE, BUILD, PENDING_DELETE, etc.):
- Check the current state before attempting operations
- Wait for stable states before making changes
- Handle race conditions where state changes between check and action

```go
// Example: Handle all states before deletion
switch resource.ProvisioningStatus {
case ProvisioningStatusPendingDelete:
    return progress.WaitingOnOpenStack(progress.WaitingOnReady, deletingPollingPeriod)
case ProvisioningStatusPendingCreate, ProvisioningStatusPendingUpdate:
    // Can't delete in pending state, wait for ACTIVE
    return progress.WaitingOnOpenStack(progress.WaitingOnReady, availablePollingPeriod)
}

// Example: Handle 409 Conflict (state changed between check and API call)
err := actuator.osClient.DeleteResource(ctx, resource.ID)
if orcerrors.IsConflict(err) {
    return progress.WaitingOnOpenStack(progress.WaitingOnReady, deletingPollingPeriod)
}
```

**Note**: Resources without intermediate states (e.g., Flavor, Keypair) are created/deleted synchronously and don't need this handling.

## 3. Deterministic State

Ensure consistent, comparable state to enable reliable drift detection.

**Principle**: Data should be normalized before storage and comparison so equivalent states produce identical representations.

**Examples:**
- Sort lists before creation and comparison (tags, security group rules, allowed address pairs)
- Normalize strings (trim whitespace, consistent casing where appropriate)
- Use canonical forms for complex types

```go
// Example: Sort tags for consistent comparison
tags := make([]string, len(resource.Tags))
for i := range resource.Tags {
    tags[i] = string(resource.Tags[i])
}
slices.Sort(tags)
createOpts.Tags = tags

// Example: Compare with sorting (copy before sorting to avoid mutation)
desiredTags := make([]string, len(resource.Tags))
copy(desiredTags, resource.Tags)
slices.Sort(desiredTags)

currentTags := make([]string, len(osResource.Tags))
copy(currentTags, osResource.Tags)
slices.Sort(currentTags)

if !slices.Equal(desiredTags, currentTags) {
    updateOpts.Tags = &desiredTags
}
```

**Note**: Import `"slices"` when using sorting/comparison functions.

## 4. Error Classification

Distinguish between errors that can be retried vs those requiring user action.

| Error Type | When to Use | Behavior |
|------------|-------------|----------|
| **Retryable** (default) | Transient issues (network, API unavailable) | Automatic retry with backoff |
| **Terminal** | Invalid configuration, bad input, permission denied | No retry until spec changes |

```go
// Terminal: User must fix the spec
if !orcerrors.IsRetryable(err) {
    err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration,
        "invalid configuration: "+err.Error(), err)
}

// Conflict on update: Treat as terminal (spec likely conflicts with existing state)
// unless resource has intermediate states that could cause transient conflicts
if orcerrors.IsConflict(err) {
    err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration,
        "invalid configuration updating resource: "+err.Error(), err)
}
```

## 5. Dependency Timing

Resolve dependencies as late as possible, as close to the point of use as possible.

**Rationale**: Avoid injecting dependency requirements where not strictly required. This reduces coupling and gives users greater flexibility when fixing failed deployments.

**Examples:**
- A Subnet depends on Network for creation, but not for import by ID or deletion
- Don't require recreating a deleted Network just to delete a Subnet whose `status.ID` is already set
- Add finalizers to dependencies only immediately before the OpenStack create/update call that references them

```go
// Good: Only fetch dependency when needed for creation
if resource.VipSubnetRef != nil {
    subnet, depRS := subnetDependency.GetDependency(ctx, ...)
    reconcileStatus = reconcileStatus.WithReconcileStatus(depRS)
}

// Bad: Fetching dependency unconditionally even when not needed
subnet, depRS := subnetDependency.GetDependency(ctx, ...)  // Wrong if subnet is optional
```

For detailed dependency implementation: @.agents/skills/add-dependency/SKILL.md

## 6. Code Clarity

Write self-documenting code through naming and organization.

**Naming**: Use descriptive names that prevent ambiguity:
- `vipSubnetDependency` not `subnetDependency` (when multiple subnet types possible)
- `sourcePortDependency` vs `destinationPortDependency`
- `memberNetworkDependency` vs `externalNetworkDependency`

**Organization**: Define constants and types where they're most accessible:
- Status constants: prefer using constants from gophercloud if available
- Only define constants in ORC's `types.go` if gophercloud doesn't provide them
- Internal helpers in `actuator.go`

```go
// Prefer gophercloud constants when available:
import "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
if osResource.Status == ports.StatusActive { ... }

// Only define in types.go if gophercloud doesn't have them:
const (
    <Kind>ProvisioningStatusActive        = "ACTIVE"
    <Kind>ProvisioningStatusPendingCreate = "PENDING_CREATE"
    <Kind>ProvisioningStatusError         = "ERROR"
)
```

## 7. API Safety

Design APIs that prevent invalid states through types and validation.

**Use stricter types** where OpenStack provides specific formats:
- `IPvAny` for IP addresses (validates format)
- `OpenStackName` for resource names - but check the specific OpenStack project for exact limits (e.g., Keystone names max 64 chars, Neutron names max 255 chars)
- Custom types with validation (e.g., tag types with length limits)

**Note**: Always check how fields are defined in the related OpenStack project to determine correct validation constraints.

**Add validation markers** to catch errors early:
- `+kubebuilder:validation:MinLength`, `MaxLength`
- `+kubebuilder:validation:Pattern` for format constraints
- `+kubebuilder:validation:XValidation` for cross-field rules
