# Enhancement: Drift Detection and Automatic Reconciliation

| Field | Value |
|-------|-------|
| **Status** | implementable |
| **Author(s)** | @eshulman |
| **Created** | 2026-02-03 |
| **Last Updated** | 2026-02-03 |
| **Tracking Issue** | TBD |

## Summary

This enhancement introduces drift detection and automatic reconciliation for ORC managed resources. The feature enables ORC to periodically check OpenStack resources for changes made outside of ORC (via CLI, dashboard, or other tools) and automatically restore them to match the desired state defined in the Kubernetes specification.

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

## Goals

- **Ensure state consistency**: Managed resources in OpenStack should match the desired state declared in Kubernetes
- **Detect external modifications**: Identify when OpenStack resources are modified outside of ORC
- **Automatic correction**: Restore drifted resources to their desired state without manual intervention
- **Resource recreation**: Recreate managed resources that are deleted externally from OpenStack
- **Configurable frequency**: Allow operators to tune the resync interval based on their requirements
- **Hierarchical configuration**: Support configuration at ORC-wide, resource-type, and per-resource levels
- **Minimal API impact**: Avoid excessive OpenStack API calls that could trigger rate limiting

## Non-Goals

- **Real-time drift detection**: Event-driven detection of changes (would require OpenStack webhooks or very short polling intervals)
- **Drift reporting without correction**: Alerting on drift without taking corrective action (future enhancement)
- **Selective field reconciliation**: Allowing some fields to drift while correcting others
- **Conflict resolution with merge semantics**: Merging external changes with desired state
- **Drift detection for unmanaged resources**: Unmanaged resources are explicitly not modified by ORC

## Proposal

### Periodic Resync Mechanism

The drift detection mechanism works by periodically triggering a full reconciliation of managed resources:

1. **Trigger**: After a resource reaches a stable state (Progressing=False), ORC schedules a resync after `resyncPeriod` duration
2. **Fetch**: On resync, ORC fetches the current state of the OpenStack resource
3. **Compare**: The current state is compared against the desired state in the Kubernetes spec
4. **Update**: If drift is detected, ORC updates the OpenStack resource to match the desired state
5. **Reschedule**: After successful reconciliation, the next resync is scheduled

### API Changes

The `ManagedOptions` struct is extended with a `resyncPeriod` field:

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

**Default value**: 10 hours (`10h`) - chosen to balance drift detection responsiveness with API load, aligning with ACK's default.

**Disable resync**: Set `resyncPeriod: 0` to disable periodic drift detection for a resource.

### Configuration Hierarchy

Drift detection supports a three-level configuration hierarchy, with more specific configurations taking precedence:

| Level | Scope | Configuration Location | Precedence |
|-------|-------|----------------------|------------|
| ORC-wide | All resources across all types | Controller deployment configuration | Lowest |
| Resource-type | All resources of a specific type (e.g., all Networks) | CRD-level configuration or controller flags | Medium |
| Per-resource | Individual resource instance | `spec.managedOptions.resyncPeriod` on the CR | Highest |

**Resolution order**: Per-resource → Resource-type → ORC-wide → Default (10h)

### Resource Recreation on External Deletion

When a managed resource is deleted from OpenStack but the ORC object still exists:

1. On the next reconciliation, ORC attempts to fetch the resource by the ID stored in `status.id`
2. If not found and the resource was originally created by ORC (not imported), ORC recreates it
3. The new resource ID is stored in `status.id`

For **imported resources** that are deleted externally, this is a terminal error because the resource was not created by ORC and recreating it would not restore the original resource.

### Field Coverage

Drift detection covers all **mutable fields** that ORC actuators implement update operations for. Before this feature is considered stable, all actuator implementations must be audited to ensure they cover all mutable fields.

## Risks and Edge Cases

### Split-Brain Scenarios

**Risk**: Multiple controllers or systems may be managing the same OpenStack resources, leading to conflicts where changes are repeatedly overwritten.

**Mitigation**:
- Implement retry with exponential backoff when update conflicts are detected
- Document that ORC should be the sole manager of resources it creates
- Report conflicts in resource conditions for observability

### API Rate Limiting

**Risk**: Frequent resync across many resources could trigger OpenStack API rate limiting.

**Mitigation**:
- Conservative 10-hour default resync period
- Add random jitter to resync times to avoid thundering herd
- Allow operators to disable or lengthen resync for stable resources

### Controller Resource Consumption

**Risk**: Frequent reconciliation increases CPU and memory usage on the ORC controller.

**Mitigation**:
- Implement hash-based comparison: compute a hash of the OpenStack resource state and store it in `status.observedStateHash`. Only proceed with update operations if the hash differs from the previous reconciliation.
- Conservative default limits reconciliation frequency

### Conflicts with External Systems

**Risk**: If resources are intentionally managed by external systems (e.g., autoscalers, other controllers), drift correction can cause unexpected behavior.

**Mitigation**:
- Allow `resyncPeriod: 0` to disable drift detection
- Use `managementPolicy: unmanaged` for externally managed resources
- Document the implications clearly in the user guide

### Upgrade/Downgrade Considerations

**Risk**: Users upgrading to a version with drift detection may experience unexpected reconciliations.

**Mitigation**: The 10-hour default is conservative enough that most users won't notice immediate impact. Document the new behavior in release notes.

## Alternatives Considered

### Event-Driven Drift Detection

Use OpenStack notifications (Oslo messaging) to detect changes in real-time.

**Rejected because**: Requires OpenStack notification infrastructure, complex to implement, not all deployments have notifications enabled.

### Drift Detection Without Correction

Detect and report drift without automatically correcting it.

**Rejected because**: Adds operational burden requiring human intervention. Could be added as a separate management policy option in the future.

### Watch-Based Detection

Implement a watcher that periodically lists all resources from OpenStack and compares.

**Rejected because**: List operations can be expensive, harder to implement with proper filtering, and per-resource reconciliation integrates naturally with controller-runtime.

## Implementation History

- 2026-02-03: Enhancement proposed
