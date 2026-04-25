# Enhancement: SubResourcesReady Condition

| Field | Value |
|-------|-------|
| **Status** | implementable |
| **Author(s)** | @eshulman |
| **Created** | 2026-02-05 |
| **Last Updated** | 2026-02-05 |
| **Tracking Issue** | TBD |

## Summary

This enhancement introduces a new `SubResourcesReady` condition for ORC resources that manage sub-resources (e.g., SecurityGroup→Rules, Router→Interfaces, LoadBalancer→Pools/Listeners/Members). This condition provides visibility into the status of sub-resources and enables users to define policies based on sub-resource readiness.

## Motivation

Currently, ORC resources with sub-resources have ambiguous status reporting:

- `Available=True` typically means the main resource exists and can be operated on
- `Progressing=True` indicates the controller is working, but doesn't distinguish between main resource operations and sub-resource operations

This creates confusion for users:

| Scenario | Available | Progressing | User can distinguish? |
|----------|-----------|-------------|----------------------|
| SG created, rules pending | True | True | No - could be tags updating |
| SG created, rule failed | True | True/False | No - unclear what failed |
| SG created, tag update in progress | True | True | No - same as above |
| SG fully ready | True | False | Yes |

Users need to know when sub-resources are fully reconciled to:
- Safely proceed with dependent resources (e.g., Port referencing SecurityGroup)
- Debug sub-resource failures without inspecting controller logs
- Build automation that waits for complete resource readiness

This pattern applies to multiple controllers, and having a consistent approach across the project improves predictability and user experience.

Related discussion: https://github.com/k-orc/openstack-resource-controller/pull/651

## Goals

- Add a `SubResourcesReady` condition to all ORC resources that manage sub-resources
- Provide clear visibility into sub-resource status for users
- Report detailed error information when sub-resources fail
- Establish a consistent pattern across all controllers with sub-resources
- Enable future policy-based behavior on sub-resource readiness

## Non-Goals

- Changing the semantics of `Available` or `Progressing` conditions (Phase 1)
- Implementing dependency management based on `SubResourcesReady` (Phase 2)
- Resource-specific conditions (e.g., `RulesReady`, `InterfacesReady`) - one consistent condition is preferred for simplicity

## Proposal

### New Condition: SubResourcesReady

Add a third standard condition `SubResourcesReady` to resources that manage sub-resources:

```yaml
status:
  conditions:
  - type: Available
    status: "True"
    reason: Success
    message: "OpenStack resource is available"
  - type: Progressing
    status: "False"
    reason: Success
    message: "OpenStack resource is up to date"
  - type: SubResourcesReady
    status: "True"
    reason: Success
    message: "All sub-resources are ready"
```

### Condition Semantics

| Condition | Meaning |
|-----------|---------|
| `Available` | Main resource exists and can be operated on (unchanged) |
| `Progressing` | Controller is actively reconciling something (unchanged) |
| `SubResourcesReady` | All sub-resources are in their desired state |

### Status Combinations

| Scenario | Available | Progressing | SubResourcesReady |
|----------|-----------|-------------|-------------------|
| Main resource creating | False | True | False |
| Main resource ready, sub-resources pending | True | True | False |
| Main resource ready, sub-resource failed | True | False | False (with error) |
| Fully ready | True | False | True |
| Tag update in progress | True | True | True |
| Sub-resource update in progress | True | True | False |

### Error Reporting

When a sub-resource fails, the condition should include details:

```yaml
- type: SubResourcesReady
  status: "False"
  reason: RuleCreationFailed
  message: "Rule 'allow-ssh' failed: invalid CIDR format for remoteIPPrefix"
```

For multiple failures, aggregate the messages:

```yaml
- type: SubResourcesReady
  status: "False"
  reason: MultipleFailures
  message: "2 sub-resources failed: Rule 'allow-ssh' (invalid CIDR), Rule 'allow-http' (port out of range)"
```

### Affected Resources

Resources that would gain the `SubResourcesReady` condition:

| Resource | Sub-resources |
|----------|---------------|
| SecurityGroup | Rules |
| Router | Interfaces, Routes |
| LoadBalancer | Pools, Listeners, Members, HealthMonitors |
| Server | Attached volumes, Network interfaces (if managed) |
| Network | Segments (if applicable) |
| Subnet | Allocation pools, Host routes |

Resources without sub-resources would not have this condition (e.g., Image, Flavor, FloatingIP).

### Implementation Approach

#### Phase 1: Visibility (Additive)

1. Define `SubResourcesReady` as a standard condition type in the API
2. Add the condition to status writers for affected resources
3. Implement sub-resource tracking in actuators
4. Report detailed error messages for failures
5. No changes to dependency management - existing behavior unchanged

Example implementation pattern for actuators:

```go
func (a actuator) reconcileSubResources(ctx context.Context, obj *orcv1alpha1.SecurityGroup, osResource *osResourceT) progress.ReconcileStatus {
    var failedRules []string

    for _, rule := range obj.Spec.Resource.Rules {
        if err := a.ensureRule(ctx, osResource, rule); err != nil {
            failedRules = append(failedRules, fmt.Sprintf("%s: %s", rule.Description, err))
        }
    }

    if len(failedRules) > 0 {
        return progress.SubResourcesFailed(failedRules)
    }
    return progress.SubResourcesReady()
}
```

#### Phase 2: Policy-Based Behavior (Future)

1. Add configuration option to control dependency behavior:
   ```yaml
   spec:
     managedOptions:
       waitForSubResources: true  # Wait for SubResourcesReady before dependents proceed
   ```

2. Update dependency management to optionally wait for `SubResourcesReady`
3. This may change existing behavior based on the policy selected

### API Changes

Add new condition type constant:

```go
const (
    ConditionAvailable        = "Available"
    ConditionProgressing      = "Progressing"
    ConditionSubResourcesReady = "SubResourcesReady"
)
```

Add new condition reasons:

```go
const (
    ConditionReasonSubResourcesReady   = "SubResourcesReady"
    ConditionReasonSubResourcesPending = "SubResourcesPending"
    ConditionReasonSubResourceFailed   = "SubResourceFailed"
    ConditionReasonMultipleFailures    = "MultipleFailures"
)
```

## Risks and Edge Cases

### Backwards Compatibility

**Risk**: Existing automation may not expect the new condition.

**Mitigation**: Phase 1 is purely additive. The new condition doesn't change existing `Available` or `Progressing` behavior. Users checking for `Available=True && Progressing=False` will continue to work.

### Performance Impact

**Risk**: Tracking sub-resource status adds complexity to reconciliation.

**Mitigation**: Sub-resource status is already computed during reconciliation. This change primarily affects how status is reported, not computed.

### Condition Bloat

**Risk**: Adding conditions to all resources with sub-resources could make status verbose.

**Mitigation**: Only resources with meaningful sub-resources get the condition. Resources like Image or Flavor don't need it.

### Phase 2 Behavior Changes

**Risk**: Enabling `waitForSubResources` policy could change existing workflows.

**Mitigation**:
- Phase 2 is opt-in via explicit configuration
- Default behavior remains unchanged
- Document clearly in release notes

### Error Aggregation

**Risk**: Resources with many sub-resources (e.g., SecurityGroup with 50 rules) could have very long error messages.

**Mitigation**: Limit message length and indicate "and X more failures" when truncated.

## Alternatives Considered

### Resource-Specific Conditions

Add conditions like `RulesReady`, `InterfacesReady`, `PoolsReady` for each resource type.

**Rejected because**:
- Increases complexity of dependency management (need to watch different conditions per resource type)
- Harder to build generic tooling
- Inconsistent user experience

### Modify Available Condition

Make `Available=False` when sub-resources aren't ready.

**Rejected because**:
- Changes semantics of `Available` (resource IS available for operations)
- Could break existing automation
- Loses the distinction between "can operate on" and "fully reconciled"

### Use Progressing Condition Only

Rely on `Progressing=False` to indicate everything is ready.

**Rejected because**:
- Doesn't distinguish between main resource and sub-resource issues
- Tag updates would show same status as rule failures
- No clear indication of what's still pending

## Implementation History

- 2026-02-05: Enhancement proposed
