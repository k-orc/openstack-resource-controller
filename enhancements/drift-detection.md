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
- **Hierarchical configuration**: Support configuration at ORC-wide, resource-type, and per-resource levels
- **Minimal API impact**: Avoid excessive OpenStack API calls that could trigger rate limiting

## Non-Goals

- **Real-time drift detection**: Event-driven detection of changes (would require OpenStack webhooks or very short polling intervals)
- **Drift reporting without correction**: Alerting on drift without taking corrective action. This applies to both mutable fields (which are corrected, not just reported) and immutable fields (which are ignored, not reported). May be considered as a future enhancement.
- **Selective field reconciliation**: Allowing some fields to drift while correcting others
- **Conflict resolution with merge semantics**: Merging external changes with desired state
- **Drift detection for unmanaged resources**: Unmanaged resources are explicitly not modified by ORC

## Proposal

### Periodic Resync Mechanism

The drift detection mechanism works by periodically triggering a full reconciliation of managed resources. Unlike event-driven reconciles (triggered by Kubernetes spec/status changes), drift detection uses a time-based trigger to catch changes made directly in OpenStack:

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

**Default value**: Disabled (`0`) - drift detection is opt-in initially until the design and implementation are proven. Users can enable it by setting a positive value (e.g., `10h` aligns with ACK's default).

**Enable resync**: Set `resyncPeriod` to a positive duration (e.g., `1h`, `10h`) to enable periodic drift detection for a resource.

### Configuration Hierarchy

Drift detection supports a three-level configuration hierarchy, with more specific configurations taking precedence:

| Level | Scope | Configuration Location | Precedence |
|-------|-------|----------------------|------------|
| ORC-wide | All resources across all types | See options below | Lowest |
| Resource-type | All resources of a specific type (e.g., all Networks) | Strategy CR `resourceOverrides` section | Medium |
| Per-resource | Individual resource instance | `spec.managedOptions.resyncPeriod` on the CR | Highest |

**Resolution order**: Per-resource → Strategy CR resourceOverrides → ORC-wide → Built-in default (disabled)

#### ORC-wide Configuration Options

The following are possible implementations to consider for global configuration:

1. **CLI flag**: `--default-resync-period=10h`
2. **Environment variable**: `DEFAULT_RESYNC_PERIOD=10h` (similar to ASO's `AZURE_SYNC_PERIOD`)
3. **Strategy CR**: A dedicated CR for organizational policies (see below)

#### Strategy CR (Future Consideration)

For sophisticated deployments, a Strategy CR could define both global defaults and per-resource-type overrides in a Kubernetes-native, auditable format:

```yaml
apiVersion: openstack.k-orc.cloud/v1alpha1
kind: ORCStrategy
metadata:
  name: production
  namespace: orc-system
spec:
  # Global defaults
  defaults:
    drift:
      resyncPeriod: 10h
    subResources:
      waitForReady: false  # See PR #673 for SubResourcesReady condition

  # Per-resource-type overrides
  resourceOverrides:
    Network:
      drift:
        resyncPeriod: 1h
    SecurityGroup:
      subResources:
        waitForReady: true
```

Benefits of Strategy CR:
- Kubernetes-native, versioned, auditable, RBAC-controlled
- Supports multiple strategies (e.g., `production`, `development`)
- Can be extended to include other configurable behaviors (e.g., sub-resource policies from [PR #673](https://github.com/k-orc/openstack-resource-controller/pull/673))

Alternatively, platform teams can use [kro (Kube Resource Orchestrator)](https://kro.run/) to wrap ORC resources with organizational defaults without changes to ORC itself.

### Resource Recreation on External Deletion

When a managed resource is deleted from OpenStack but the ORC object still exists:

1. On the next reconciliation, ORC attempts to fetch the resource by the ID stored in `status.id`
2. If not found and the resource was originally created by ORC (not imported), ORC recreates it
3. The new resource ID is stored in `status.id`

**Behavior when drift detection is disabled** (`resyncPeriod: 0`): External deletion remains a terminal error (current behavior preserved). Resource recreation only occurs when drift detection is enabled and a periodic resync discovers the missing resource. This maintains backwards compatibility.

For **imported resources** that are deleted externally, this is always a terminal error regardless of drift detection settings, because the resource was not created by ORC and recreating it would not restore the original resource.

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
- Implement hash-based comparison as an optimization: on resync, compute a hash of the fetched OpenStack resource state and compare it to `status.observedStateHash` from the last reconciliation. If identical, skip the full field-by-field comparison (no drift detected). If different, proceed with full reconciliation. This avoids expensive value comparisons when nothing has changed.
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
