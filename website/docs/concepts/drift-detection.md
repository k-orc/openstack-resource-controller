# Drift Detection and External Deletion

This page explains how ORC's periodic resync works, what happens when resources
are changed or deleted outside of ORC, and how dependent resources are affected.
For practical setup steps, see
[How to Enable Drift Detection](../howto/drift-detection.md).

## How periodic resync works

After a resource reaches a stable state (`Progressing=False`), ORC schedules a
reconciliation after the configured `resyncPeriod`. On each resync:

1. ORC fetches the current state of the OpenStack resource.
2. For **managed** resources: if drift is detected, ORC updates the resource to
   match the Kubernetes spec.
3. For **unmanaged** resources: ORC refreshes `status.resource` to reflect the
   current OpenStack state, but makes no changes.
4. The next resync is scheduled.

A small random jitter ([0%, +20%]) is applied to `resyncPeriod` to spread
reconciliations and avoid thundering-herd effects.

!!! note

    Resources in a terminal error state (`Progressing=False` with reason
    `InvalidConfiguration` or `UnrecoverableError`) are **not** periodically
    resynced. Terminal errors require manual intervention to resolve.

### Restart behavior

ORC persists `status.lastSyncTime` in the Kubernetes status. After a controller
restart, it uses this timestamp to determine when the next resync should occur,
preventing a thundering herd of reconciliations on startup.

## External deletion handling

When a resource is deleted directly from OpenStack (bypassing ORC), the behavior
depends on how ORC originally obtained the resource.

### Managed resources are recreated

If you created the resource through ORC's `spec.resource` field, ORC
**recreates** it automatically:

1. ORC detects the resource is missing from OpenStack (the ID stored in
   `status.id` no longer exists).
2. ORC clears `status.id`.
3. On the next reconcile, ORC creates a new OpenStack resource.
4. The new resource ID is stored in `status.id`.

The ORC object continues to exist and becomes `Available=True` again once the
resource is recreated.

!!! warning

    Recreation produces a new OpenStack resource with a **new ID**. Any
    OpenStack resources (outside ORC) that referenced the old ID will need to be
    updated manually.

### Imported resources enter terminal error

If you imported an existing resource using `spec.import`, ORC reports a
**terminal error** when the resource is deleted from OpenStack:

- `Available=False`
- `Progressing=False`
- Condition reason: `UnrecoverableError`
- Message: `resource has been deleted from OpenStack`

ORC does **not** recreate imported resources because it did not create them
originally, and recreating a new empty resource would not restore what was lost.

To recover: delete and recreate the ORC object pointing at a new or restored
OpenStack resource.

## Implications for dependent resources

OpenStack enforces referential integrity for most resource relationships. For
example, a Network cannot be deleted while Subnets exist on it. This means
externally deleting a parent resource is normally prevented by OpenStack itself.

!!! warning

    If a parent resource is externally deleted in a way that bypasses
    OpenStack's referential integrity (e.g. direct database manipulation),
    manual cleanup of both the parent and dependent resources may be required.
    This is an unusual operational scenario.
