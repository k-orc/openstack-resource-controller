# OEP-001: Drift Detection and Automatic Reconciliation

## Table of Contents

- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Design Details](#design-details)
    - [Periodic Resync Mechanism](#periodic-resync-mechanism)
    - [Configuration Hierarchy](#configuration-hierarchy)
    - [Resource Recreation on External Deletion](#resource-recreation-on-external-deletion)
    - [Field Coverage](#field-coverage)
  - [API Changes](#api-changes)
- [Risks and Mitigations](#risks-and-mitigations)
  - [Split-Brain Scenarios](#split-brain-scenarios)
  - [API Rate Limiting](#api-rate-limiting)
  - [Controller Resource Consumption](#controller-resource-consumption)
  - [Conflicts with External Systems](#conflicts-with-external-systems)
- [Implementation](#implementation)
  - [Phase 1: Per-Resource Configuration (Current)](#phase-1-per-resource-configuration-current)
  - [Phase 2: Resource-Type Level Configuration](#phase-2-resource-type-level-configuration)
  - [Phase 3: ORC-Wide Configuration](#phase-3-orc-wide-configuration)
- [Test Plan](#test-plan)
- [Graduation Criteria](#graduation-criteria)
- [Production Readiness Checklist](#production-readiness-checklist)
- [Alternatives Considered](#alternatives-considered)
- [References](#references)

## Summary

This proposal introduces drift detection and automatic reconciliation for ORC managed resources. The feature enables ORC to periodically check OpenStack resources for changes made outside of ORC (via CLI, dashboard, or other tools) and automatically restore them to match the desired state defined in the Kubernetes specification.

Additionally, managed resources that are deleted externally from OpenStack will be automatically recreated by ORC, ensuring the declared state is maintained.

## Motivation

In production environments, OpenStack resources may be modified outside of ORC through various means:

- Direct OpenStack CLI/SDK operations
- OpenStack Horizon dashboard
- Other automation tools or controllers
- Manual emergency interventions
- Third-party integrations

Without drift detection, these changes go unnoticed until they cause issues, leading to configuration drift between the declared Kubernetes state and the actual OpenStack state. This undermines the declarative model that ORC provides.

Similar Kubernetes controllers for cloud resources have implemented drift detection:

- **AWS Controllers for Kubernetes (ACK)**: Uses a 10-hour default resync period for drift recovery
- **Azure Service Operator (ASO)**: Uses a 1-hour default resync period with configurable intervals

### Goals

1. **Ensure state consistency**: Managed resources in OpenStack should match the desired state declared in Kubernetes
2. **Detect external modifications**: Identify when OpenStack resources are modified outside of ORC
3. **Automatic correction**: Restore drifted resources to their desired state without manual intervention
4. **Resource recreation**: Recreate managed resources that are deleted externally from OpenStack
5. **Configurable frequency**: Allow operators to tune the resync interval based on their requirements
6. **Hierarchical configuration**: Support configuration at ORC-wide, resource-type, and per-resource levels
7. **Minimal API impact**: Avoid excessive OpenStack API calls that could trigger rate limiting

### Non-Goals

1. **Real-time drift detection**: Event-driven detection of changes (this would require OpenStack webhooks or polling at very short intervals)
2. **Drift reporting without correction**: Alerting on drift without taking corrective action (future enhancement)
3. **Selective field reconciliation**: Allowing some fields to drift while correcting others
4. **Conflict resolution with merge semantics**: Merging external changes with desired state
5. **Drift detection for unmanaged resources**: Unmanaged resources are explicitly not modified by ORC

## Proposal

### User Stories

#### Story 1: Automatic State Restoration

As a platform operator, I want ORC to automatically restore OpenStack resources to their declared state when someone modifies them through the OpenStack CLI, so that my infrastructure remains consistent with my GitOps repository.

#### Story 2: Resource Recreation

As a developer, I want ORC to automatically recreate a managed network if it's accidentally deleted from OpenStack, so that my applications don't experience prolonged outages.

#### Story 3: Configurable Resync Frequency

As a platform operator managing critical resources, I want to configure a shorter resync period for high-priority resources while using a longer period for stable infrastructure, so that I can balance between drift detection responsiveness and API usage.

#### Story 4: Disable Drift Detection

As a platform operator, I want to disable periodic resync for specific resources that are frequently modified by external systems (with full awareness of the implications), so that ORC doesn't constantly revert expected changes.

### Design Details

#### Periodic Resync Mechanism

The drift detection mechanism works by periodically triggering a full reconciliation of managed resources:

1. **Trigger**: After a resource reaches a stable state (Progressing=False), ORC schedules a resync after `resyncPeriod` duration
2. **Fetch**: On resync, ORC fetches the current state of the OpenStack resource
3. **Compare**: The current state is compared against the desired state in the Kubernetes spec
4. **Update**: If drift is detected, ORC updates the OpenStack resource to match the desired state
5. **Reschedule**: After successful reconciliation, the next resync is scheduled

The resync check is implemented in the `shouldReconcile` function:

```go
func shouldReconcile(obj orcv1alpha1.ObjectWithConditions, resyncPeriod time.Duration) bool {
    progressing := meta.FindStatusCondition(obj.GetConditions(), orcv1alpha1.ConditionProgressing)
    if progressing == nil {
        return true
    }
    if progressing.Status == metav1.ConditionTrue {
        return true
    }
    if progressing.ObservedGeneration != obj.GetGeneration() {
        return true
    }
    // For periodic resync, check if enough time has passed since the last sync
    if resyncPeriod > 0 {
        return time.Since(progressing.LastTransitionTime.Time) >= resyncPeriod
    }
    return false
}
```

#### Configuration Hierarchy

Drift detection supports a three-level configuration hierarchy, with more specific configurations taking precedence:

| Level | Scope | Configuration Location | Precedence |
|-------|-------|----------------------|------------|
| ORC-wide | All resources across all types | Controller deployment configuration | Lowest |
| Resource-type | All resources of a specific type (e.g., all Networks) | CRD-level configuration or controller flags | Medium |
| Per-resource | Individual resource instance | `spec.managedOptions.resyncPeriod` on the CR | Highest |

**Resolution order**: Per-resource → Resource-type → ORC-wide → Default (10h)

Example per-resource configuration:

```yaml
apiVersion: openstack.k-orc.cloud/v1alpha1
kind: Network
metadata:
  name: critical-network
spec:
  cloudCredentialsRef:
    secretName: openstack-clouds
    cloudName: openstack
  managementPolicy: managed
  managedOptions:
    resyncPeriod: 1h  # Check for drift every hour
  resource:
    description: Critical application network
```

**Default value**: 10 hours (`10h`)

This default was chosen to balance between:
- Detecting drift in a reasonable timeframe
- Avoiding excessive OpenStack API load
- Aligning with similar controllers (ACK uses 10h default)

**Disable resync**: Set `resyncPeriod: 0` to disable periodic drift detection for a resource.

#### Resource Recreation on External Deletion

When a managed resource is deleted from OpenStack but the ORC object still exists:

1. **Detection**: On the next reconciliation (triggered by resync or spec change), ORC attempts to fetch the resource by the ID stored in `status.id`
2. **Not Found**: If the resource is not found and the ORC object:
   - Has `managementPolicy: managed`
   - Was NOT imported by ID or filter (i.e., was originally created by ORC)
3. **Recreation**: ORC creates a new OpenStack resource matching the desired spec
4. **Status Update**: The new resource ID is stored in `status.id`

For **imported resources** that are deleted externally, this is a terminal error because:
- The resource was not created by ORC
- Recreating it would not restore the original resource
- The import reference (ID or filter) would become invalid

```go
if objAdapter.GetManagementPolicy() == orcv1alpha1.ManagementPolicyManaged &&
   objAdapter.GetImportID() == nil &&
   objAdapter.GetImportFilter() == nil {
    log.V(logging.Info).Info("Resource has been deleted from OpenStack, will recreate", "ID", *resourceID)
    // Fall through to creation
} else {
    return osResource, progress.WrapError(
        orcerrors.Terminal(orcv1alpha1.ConditionReasonUnrecoverableError,
            "resource has been deleted from OpenStack"))
}
```

#### Field Coverage

Drift detection covers all **mutable fields** that ORC actuators implement update operations for. This includes:

- Resource names and descriptions
- Tags and metadata
- Network/subnet configuration (where mutable)
- Security group rules
- Other resource-specific mutable attributes

**Immutable fields** are not subject to drift correction because:
- OpenStack does not allow modifying them
- Changing them would require resource recreation (which is a separate consideration)

**Implementation requirement**: Before this feature graduates to stable, all actuator implementations must be audited to ensure they cover all mutable fields in their update reconcilers.

### API Changes

The `ManagedOptions` struct is extended with a `resyncPeriod` field:

```go
type ManagedOptions struct {
    // onDelete specifies the behaviour of the controller when the ORC
    // object is deleted. Options are `delete` - delete the OpenStack resource;
    // `detach` - do not delete the OpenStack resource. If not specified, the
    // default is `delete`.
    // +kubebuilder:default:=delete
    // +optional
    OnDelete OnDelete `json:"onDelete,omitempty"`

    // resyncPeriod specifies the interval after which a successfully
    // reconciled resource will be reconciled again to detect drift from the
    // desired state. Set to 0 to disable periodic resync. If not specified,
    // the default is 10 hours.
    // +kubebuilder:default:="10h"
    // +optional
    ResyncPeriod *metav1.Duration `json:"resyncPeriod,omitempty"`
}
```

## Risks and Mitigations

### Split-Brain Scenarios

**Risk**: Multiple controllers or systems may be managing the same OpenStack resources, leading to conflicts where changes are repeatedly overwritten.

**Mitigations**:

1. **Retry with backoff**: Implement exponential backoff when update conflicts are detected
2. **Conflict detection**: Detect when a resource is being modified rapidly and log warnings
3. **Documentation**: Clearly document that ORC should be the sole manager of resources it creates
4. **Eventual consistency**: Accept that in conflict scenarios, the most recent writer wins, and ORC will eventually restore desired state
5. **Condition reporting**: Report conflicts in resource conditions for observability

### API Rate Limiting

**Risk**: Frequent resync across many resources could trigger OpenStack API rate limiting.

**Mitigations**:

1. **Conservative default**: 10-hour default resync period minimizes API calls
2. **Jittering**: Add random jitter to resync times to avoid thundering herd
3. **Batching**: Consider batching status checks where OpenStack APIs support it
4. **Configurable per-resource**: Allow operators to disable or lengthen resync for stable resources
5. **Resource-type configuration**: Allow operators to set defaults per resource type

### Controller Resource Consumption

**Risk**: Frequent reconciliation increases CPU and memory usage on the ORC controller.

**Mitigations**:

1. **Hash-based comparison**: Before triggering a full reconciliation, compute a hash of the OpenStack resource state (using deterministic sorted JSON serialization) and compare it against the previous hash stored in `status.observedStateHash`. Only proceed with update operations if the hashes differ. This avoids expensive field-by-field comparison and update API calls when no drift has occurred.

   ```yaml
   status:
     id: "12345678-1234-1234-1234-123456789abc"
     observedStateHash: "sha256:a1b2c3d4..."  # Hash of last observed OpenStack state
   ```

2. **Conservative default**: 10-hour default limits reconciliation frequency
3. **Monitoring**: Expose metrics for reconciliation frequency and duration
4. **Horizontal scaling**: ORC already supports running multiple controller replicas with leader election (`--leader-elect` flag), distributing controller load across nodes

### Conflicts with External Systems

**Risk**: If resources are intentionally managed by external systems (e.g., autoscalers, other controllers), drift correction can cause unexpected behavior.

**Mitigations**:

1. **Disable option**: Allow `resyncPeriod: 0` to disable drift detection
2. **Unmanaged policy**: Use `managementPolicy: unmanaged` for externally managed resources
3. **Documentation**: Clearly document the implications of drift detection
4. **Warning in user guide**: Warn about side effects of low resyncPeriod values

## Implementation

### Phase 1: Per-Resource Configuration (Current)

- [x] Add `resyncPeriod` field to `ManagedOptions`
- [x] Implement `shouldReconcile` function with resync check
- [x] Schedule resync after successful reconciliation
- [x] Recreate managed resources deleted externally
- [x] Terminal error for imported resources deleted externally
- [x] Documentation in user guide
- [ ] Audit all actuators for mutable field coverage
- [ ] Add jitter to resync scheduling
- [ ] Implement hash-based state comparison with `status.observedStateHash`

### Phase 2: Resource-Type Level Configuration

- [ ] Design resource-type configuration mechanism (CRD annotations, controller flags, or ConfigMap)
- [ ] Implement configuration inheritance (per-resource overrides resource-type)
- [ ] Add validation for configuration values
- [ ] Update documentation

### Phase 3: ORC-Wide Configuration

- [ ] Design ORC-wide configuration mechanism (controller deployment flags or ConfigMap)
- [ ] Implement full configuration hierarchy resolution
- [ ] Add configuration reload without controller restart (if ConfigMap-based)
- [ ] Update documentation

## Test Plan

### Unit Tests

- [ ] `shouldReconcile` returns true when resync period has elapsed
- [ ] `shouldReconcile` returns false before resync period has elapsed
- [ ] `shouldReconcile` respects `resyncPeriod: 0` (disabled)
- [ ] `GetResyncPeriod` returns default when not specified
- [ ] Configuration hierarchy resolution

### Integration Tests

- [ ] Resource is reconciled after resync period elapses
- [ ] Resource drift is corrected on resync
- [ ] Deleted managed resource is recreated
- [ ] Imported resource deletion results in terminal error
- [ ] Resync scheduling includes jitter

### E2E Tests

- [ ] Modify OpenStack resource via CLI, verify ORC corrects drift
- [ ] Delete OpenStack resource, verify ORC recreates it
- [ ] Verify resync period is honored (with shortened test value)

## Graduation Criteria

This feature should graduate alongside other ORC APIs:

### Alpha (Current)

- Feature implemented behind default configuration
- Basic documentation available
- Unit and integration tests passing

### Beta

- All actuators audited for mutable field coverage
- Resource-type level configuration implemented
- E2E tests covering main scenarios
- User feedback incorporated
- Jitter implementation complete

### Stable

- ORC-wide configuration implemented
- Full documentation including best practices
- Observability improvements (metrics, if beneficial)
- Production usage validated
- No outstanding drift detection issues

## Production Readiness Checklist

- [ ] All mutable fields covered by actuator update reconcilers
- [ ] Hash-based state comparison implemented to minimize unnecessary updates
- [ ] Jitter added to prevent thundering herd
- [ ] Documentation includes guidance on resyncPeriod selection
- [ ] Warnings documented for low resyncPeriod values
- [ ] Resource-type and ORC-wide configuration available
- [ ] Conflict detection and logging implemented

## Alternatives Considered

### Event-Driven Drift Detection

**Description**: Use OpenStack notifications (Oslo messaging) to detect changes in real-time.

**Pros**:
- Immediate drift detection
- No polling overhead

**Cons**:
- Requires OpenStack notification infrastructure
- Complex to implement and maintain
- Not all OpenStack deployments have notifications enabled
- Additional infrastructure dependency

**Decision**: Not pursued for initial implementation. May be considered as future enhancement.

### Drift Detection Without Correction

**Description**: Detect and report drift without automatically correcting it.

**Pros**:
- Safer for mixed-management scenarios
- Allows human review before changes

**Cons**:
- Requires additional alerting/monitoring setup
- Delays remediation
- Adds operational burden

**Decision**: Not included in initial implementation. Could be added as a separate management policy option in the future.

### Watch-Based Detection

**Description**: Implement a watcher that periodically lists all resources from OpenStack and compares.

**Pros**:
- Single API call can check multiple resources
- Potentially more efficient for large numbers of resources

**Cons**:
- List operations can be expensive
- Harder to implement with proper filtering
- Different resource types require different list calls

**Decision**: Not pursued. Per-resource reconciliation is simpler and integrates naturally with controller-runtime.

## References

- [AWS Controllers for Kubernetes - Drift Recovery](https://aws-controllers-k8s.github.io/community/docs/user-docs/drift-recovery/)
- [Azure Service Operator - ADR-2022-11: Resource Reconciliation](https://azure.github.io/azure-service-operator/design/ADR-2022-11-Reconcile-Interval/)
- [Kubernetes Enhancement Proposal Template](https://github.com/kubernetes/enhancements/tree/master/keps/NNNN-kep-template)
- ORC User Guide: [Drift Detection](../../../user-guide/index.md#drift-detection)
