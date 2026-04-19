# Enhancement: Drift Detection and Automatic Reconciliation

| Field | Value |
|-------|-------|
| **Status** | implemented |
| **Author(s)** | @eshulman |
| **Created** | 2026-02-03 |
| **Last Updated** | 2026-02-10 |
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

- **AWS Controllers for Kubernetes (ACK)**: Drift detection is **enabled by default** with a 10-hour resync period. Uses a detect-then-correct approach: periodically describes the AWS resource and only updates if drift is found. Configuration is set per-controller by authors, not configurable per-resource by users. No per-resource opt-out mechanism documented. ([ACK Drift Recovery docs](https://aws-controllers-k8s.github.io/community/docs/user-docs/drift-recovery/))

- **Azure Service Operator (ASO)**: Drift detection is **enabled by default** with a 1-hour resync period. Uses a PUT-on-every-reconcile approach rather than detect-then-correct. Provides **per-resource opt-out** via `reconcile-policy` annotation for adopted resources users don't want fully managed. **Global configuration** via `AZURE_SYNC_PERIOD` environment variable. Rate limiting via token-bucket algorithm and `MAX_CONCURRENT_RECONCILES` for parallelism control. ([ASO Controller Settings](https://azure.github.io/azure-service-operator/guide/aso-controller-settings-options/), [ASO Change Detection ADR](https://azure.github.io/azure-service-operator/design/adr-2022-11-change-detection/))

**Key design observations:**
- Both projects enable drift detection by default
- ASO provides more user-facing configuration options (global and per-resource)
- Neither project documents behavior for externally-deleted resources

## Goals

- **Ensure state consistency**: Managed resources in OpenStack should match the desired state declared in Kubernetes
- **Detect external modifications**: Identify when OpenStack resources are modified outside of ORC
- **Automatic correction**: Restore drifted resources to their desired state without manual intervention
- **Resource recreation**: Recreate managed resources that are deleted externally from OpenStack
- **Configurable frequency**: Allow operators to tune the resync interval based on their requirements
- **Hierarchical configuration**: Support configuration at ORC-wide and per-resource levels, at minimum
- **Minimal API impact**: Avoid excessive OpenStack API calls that could trigger rate limiting

## Non-Goals

- **Real-time drift detection**: Event-driven detection of changes (would require OpenStack webhooks or very short polling intervals)
- **Drift reporting without correction**: Alerting on drift without taking corrective action. This applies to both mutable fields (which are corrected, not just reported) and immutable fields (which are ignored, not reported). May be considered as a future enhancement.
- **Selective field reconciliation**: Allowing some fields to drift while correcting others
- **Conflict resolution with merge semantics**: Merging external changes with desired state
- **Drift correction for unmanaged resources**: Unmanaged resources are not modified by ORC; however, periodic resync will refresh their status to reflect the current OpenStack state

## Proposal

### Periodic Resync Mechanism

The drift detection mechanism works by periodically triggering reconciliation of resources. Unlike event-driven reconciles (triggered by Kubernetes spec/status changes), drift detection uses a time-based trigger to catch changes made directly in OpenStack. For managed resources, this includes drift correction; for unmanaged resources, this refreshes the status only.

1. **Trigger**: After a resource reaches a stable state (Progressing=False), ORC schedules a resync after `resyncPeriod` duration
2. **Fetch**: On resync, ORC fetches the current state of the OpenStack resource
3. **Compare**: The current state is compared against the desired state in the Kubernetes spec
4. **Update**: If drift is detected, ORC updates the OpenStack resource to match the desired state
5. **Reschedule**: After successful reconciliation, the next resync is scheduled

#### Implementation Details

At the end of a successful reconciliation (when no other reschedule is pending), the controller schedules the next resync:

```go
// If periodic resync is enabled and we're not already rescheduling for
// another reason, schedule the next resync to detect drift.
if resyncPeriod > 0 {
    needsReschedule, _ := reconcileStatus.NeedsReschedule()
    if !needsReschedule {
        reconcileStatus = reconcileStatus.WithRequeue(resyncPeriod)
    }
}
```

This ensures the controller automatically triggers reconciliation after the configured period.

Additionally, `shouldReconcile` must be updated to allow periodic resync. Currently it returns `false` when `Progressing=False` and generation is current, which would discard resync requests. The updated logic checks the last sync timestamp:

```go
func shouldReconcile(obj orcv1alpha1.ObjectWithConditions, resyncPeriod time.Duration) bool {
    // ... existing checks ...
    
    // At this point, Progressing is False and generation is up to date.
    // For periodic resync, check if enough time has passed since the last sync.
    if resyncPeriod > 0 {
        if lastSync := obj.GetLastSyncTime(); lastSync != nil {
            return time.Since(lastSync.Time) >= resyncPeriod
        }
        return true // First sync after feature enablement
    }
    return false
}
```

**Note**: Using `Progressing.LastTransitionTime` is not suitable because it only updates when the condition value changes, not on every reconcile. A dedicated `LastSyncTime` status field is required (see Status Changes below).

**Resources in terminal error are not resynced**: When a resource is in a terminal error state (e.g., invalid configuration, unrecoverable OpenStack error), periodic resync is not scheduled. Terminal errors indicate issues that cannot be resolved through automatic retry and require manual intervention to fix the underlying problem. This prevents wasted reconciliation cycles on resources that are known to be in an unrecoverable state.

### API Changes

A `resyncPeriod` field is added at the spec level, making it available to both managed and unmanaged resources:

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
  resyncPeriod: 1h  # Periodic resync every hour
  resource:
    description: Critical application network
```

**Default**: Disabled (`0`). Set a positive duration like `10h` to enable.

### Status Changes

A new `lastSyncTime` field is added to the status of all ORC resources:

```yaml
status:
  lastSyncTime: "2026-02-03T10:30:00Z"  # Last successful reconciliation with OpenStack
  id: "abc123"
  # ... other status fields
```

This field is updated at the end of every successful reconciliation that fetches the resource from OpenStack. It is required because:

1. **Controller restarts**: Without persisted state, the controller would lose track of when resources were last synced, potentially causing a thundering herd of reconciliations on restart.
2. **Accurate timing**: The `Progressing.LastTransitionTime` only updates when the condition value changes, not on every reconcile, making it unsuitable for tracking sync intervals.

The `shouldReconcile` function uses this field to determine if enough time has passed since the last sync to trigger a periodic resync.

### Behavior by Management Policy

The periodic resync behavior differs based on `managementPolicy`:

| Policy | On Resync |
|--------|-----------|
| `managed` | Fetch from OpenStack → correct drift → update status |
| `unmanaged` | Fetch from OpenStack → update status only (no writes to OpenStack) |

This allows unmanaged/imported resources to keep their `status.resource` in sync with the actual OpenStack state without ORC modifying the resource.

### Configuration Hierarchy

Drift detection supports a two-level configuration hierarchy:

| Level | Scope | Configuration Location | Precedence |
|-------|-------|----------------------|------------|
| ORC-wide | All resources across all types | CLI flag | Lowest |
| Per-resource | Individual resource instance | `spec.resyncPeriod` on the CR | Highest |

**Resolution order**: Per-resource → ORC-wide → Built-in default (disabled)

#### ORC-wide Configuration Options

A CLI flag sets the global default:

```
--default-resync-period=10h
```

For per-resource-type configuration, platform teams can use [kro (Kube Resource Orchestrator)](https://kro.run/) to wrap ORC resources with organizational defaults without changes to ORC itself.

### Resource Recreation on External Deletion

When a resource with `managementPolicy=managed` is deleted from OpenStack but the ORC object still exists:

1. On the next reconciliation, ORC attempts to fetch the resource by the ID stored in `status.id`
2. If not found and the resource was originally created by ORC (not imported), ORC recreates it
3. The new resource ID is stored in `status.id`

#### Implementation Changes

Currently, `GetOrCreateOSResource` returns a terminal error when fetching a resource by `status.id` results in a 404. To support resource recreation, this logic must be updated to:

1. Check if `managementPolicy == managed` and the resource was not imported (no `importID` or `importFilter`)
2. If both conditions are met, clear `status.id` and proceed to the creation path instead of returning an error
3. If the resource was imported or is unmanaged, retain the existing terminal error behavior

This ensures that managed resources created by ORC are automatically recreated, while imported or unmanaged resources correctly fail with a terminal error when deleted externally.

**Behavior when drift detection is disabled** (`resyncPeriod: 0`): Periodic resyncs do not occur, so discovery of external deletion depends on other triggers (spec change, controller restart). When discovered, ORC will still recreate managed resources (not a terminal error). The difference is timing of discovery, not the recreation behavior itself.

For **imported resources** that are deleted externally, this is always a terminal error regardless of drift detection settings, because the resource was not created by ORC and recreating it would not restore the original resource.

**Note on dependent resources**: OpenStack enforces referential integrity for most resources (e.g., Networks cannot be deleted while Subnets exist). If resources are deleted through means that bypass these checks (direct database manipulation, OpenStack bugs), drift detection preserves ORC's existing reconciliation behavior:

- **Parent resource (e.g., Network)**: On next reconciliation, `GetOSResourceByID` returns 404 → terminal error ("resource has been deleted from OpenStack").
- **Dependent resource update path (e.g., Subnet update)**: The controller doesn't check if its parent dependency is in terminal error. It fetches the resource by `status.id`, and if successful, proceeds with the update. The result depends on what OpenStack returns for that specific operation and would preserve the existing error handling behavior.
- **Dependent resource create/recreate path**: The controller checks `IsAvailable(parent)` before proceeding. If the parent is in terminal error, the dependent waits on the dependency (not terminal, just waiting).

These behaviors exist regardless of drift detection—drift detection only changes scheduling, not reconciliation logic. Resolving such inconsistencies requires manual intervention.

### Field Coverage

Drift detection covers all **mutable fields** that ORC actuators implement update operations for. Before this feature is considered stable, all actuator implementations must be audited to ensure they cover all mutable fields.

## Risks and Edge Cases

### Split-Brain Scenarios

**Risk**: Multiple controllers or systems may be managing the same OpenStack resources, leading to conflicts where changes are repeatedly overwritten.

**Mitigation**:
- Document that ORC should be the sole manager of resources it creates
- Report conflicts in resource conditions for observability

### API Rate Limiting

**Risk**: Frequent resync across many resources could trigger OpenStack API rate limiting.

**Mitigation**:
- Disabled by default; when enabled, recommend conservative intervals (e.g., 10 hours)
- Add random jitter to resync times to avoid thundering herd: since reconciliation already uses "requeue after X duration", jitter simply adds a random offset (e.g., ±10%) to the resync period, spreading resyncs over time rather than having them fire simultaneously
- Allow operators to disable or lengthen resync for stable resources

### Controller Resource Consumption

**Risk**: Frequent reconciliation increases CPU and memory usage on the ORC controller.

**Mitigation**:
- Disabled by default; when enabled, conservative intervals limit reconciliation frequency

### Conflicts with External Systems

**Risk**: If resources are intentionally managed by external systems (e.g., autoscalers, other controllers), drift correction can cause unexpected behavior.

**Mitigation**:
- Allow `resyncPeriod: 0` to disable drift detection
- Use `managementPolicy: unmanaged` for externally managed resources
- Document the implications clearly in the user guide

### Upgrade/Downgrade Considerations

**Risk**: Users upgrading to a version with drift detection may experience unexpected reconciliations.

**Mitigation**: Drift detection is disabled by default (opt-in), so users upgrading will not experience any behavior change unless they explicitly enable it. Document the new feature in release notes.

## Alternatives Considered

### Event-Driven Drift Detection

Use OpenStack notifications (Oslo messaging) to detect changes in real-time.

**Rejected because**: Requires OpenStack notification infrastructure, complex to implement, not all deployments have notifications enabled.

### Drift Detection Without Correction

Detect and report drift without automatically correcting it.

**Out of scope for this enhancement**: While drift notification has value for observability, it is better addressed as a separate alerting effort. This enhancement focuses on drift correction; reporting-only mode could be added as a future management policy option.

### Watch-Based Detection

Implement a watcher that periodically lists all resources from OpenStack and compares.

**Rejected because**: List operations can be expensive, harder to implement with proper filtering, and per-resource reconciliation integrates naturally with controller-runtime.

## Implementation History

- 2026-02-03: Enhancement proposed
- 2026-02-03: Implemented — all tasks completed

### Implemented Components

The following have been implemented:

**API Changes**
- Added `spec.resyncPeriod` field (`*metav1.Duration`) to all ORC resource types
- Added `status.lastSyncTime` field (`*metav1.Time`) to all ORC resource types

**Periodic Resync**
- `shouldReconcile` updated to check `lastSyncTime` against `resyncPeriod` for time-based resync
- Jitter (±10%) applied to resync scheduling via `resync.CalculateJitteredDuration`
- `status.lastSyncTime` written on every successful reconciliation cycle
- Resources in terminal error state are not rescheduled

**External Deletion Handling**
- `IsImported()` method added to `APIObjectAdapter` interface (all resource adapters)
- `GetOrCreateOSResource` branches on management policy and import status when 404 is received:
  - Managed, non-imported resources → `(nil, nil)` to trigger recreation
  - Unmanaged or imported resources → terminal error
- `status.ClearStatusID` clears `status.id` before recreation (using JSON merge patch with explicit `null`)
- `reconcileNormal` handles the `(nil, nil)` recreation signal from `GetOrCreateOSResource`

**E2E Tests**
- `network-resync-period`: verifies `lastSyncTime` is updated after configured period
- `network-resync-disabled`: verifies `lastSyncTime` is not updated when `resyncPeriod: 0`
- `network-resync-terminal-error`: verifies terminal errors are not rescheduled
- `network-resync-jitter`: verifies independent jitter-based scheduling for multiple resources
- `network-external-deletion`: verifies managed ORC-created network is recreated with new ID after external deletion
- `network-external-deletion-import`: verifies imported network enters terminal error state after external deletion

**Documentation**
- `website/docs/user-guide/drift-detection.md`: user-facing documentation covering external deletion behavior, resync configuration, verification steps, and implications for dependent resources
