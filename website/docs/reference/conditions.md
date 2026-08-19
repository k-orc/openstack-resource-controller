# Status Conditions

Every ORC resource reports its state through two conditions: **Available** and
**Progressing**. This page is a reference for all possible condition states and
reasons.

## Conditions

| Condition | Status | Meaning |
|-----------|--------|---------|
| `Available` | `True` | Resource is ready for use |
| `Available` | `False` | Resource is not ready (check `message` for details) |
| `Progressing` | `True` | ORC is still working on the resource |
| `Progressing` | `False` | ORC has finished (either success or terminal error) |

A resource is **healthy** when `Available=True` and `Progressing=False`.

## Condition Reasons

The `reason` field categorizes what is happening:

| Reason | Progressing | Available | Description | Action |
|--------|-------------|-----------|-------------|--------|
| `Success` | `False` | `True` | Resource is reconciled successfully | None needed |
| `Progressing` | `True` | `False` | Normal operation in progress (creation, update, waiting for dependency) | Wait for completion |
| `TransientError` | `True` | `False` | Temporary error, will retry automatically | Check if it persists; see [Troubleshooting](../troubleshooting.md) |
| `InvalidConfiguration` | `False` | `False` | Spec has invalid values; terminal error | Fix the resource spec |
| `UnrecoverableError` | `False` | `False` | Permanent error, won't retry; terminal error | Fix the underlying issue and see [Troubleshooting](../troubleshooting.md) |

## Reading conditions

```bash
# Quick overview: the AVAILABLE and PROGRESSING columns come from conditions
kubectl get openstack

# Full conditions for a specific resource
kubectl get network my-network -o jsonpath='{.status.conditions}' | jq

# Detailed output including status.resource
kubectl get network my-network -o yaml
```

## Terminal errors

When `Progressing=False` and `Available=False`, the resource is in a **terminal
error** state. ORC will not retry until the spec is changed. Common causes:

- `InvalidConfiguration`: The spec references something that doesn't exist or
  contains invalid values. Fix the spec.
- `UnrecoverableError`: An operation failed permanently (e.g. an imported
  resource was deleted from OpenStack, or a filter matched multiple results).
  See [Troubleshooting](../troubleshooting.md) for resolution steps.

## The `status.resource` field

In addition to conditions, every ORC resource has a `.status.resource` field
containing the observed state from OpenStack. This includes fields that
OpenStack assigns, such as `projectID`, `createdAt`, and `revisionNumber`,
that are not part of the spec.

## The `status.lastSyncTime` field

Records when ORC last successfully read the resource state from OpenStack. See
[How to Enable Drift Detection](../howto/drift-detection.md) for details on
periodic resync.
