# Drift Detection and External Deletion Handling

ORC can periodically reconcile resources to detect and correct configuration drift — changes made to OpenStack resources outside of ORC's control. This feature also detects when managed resources have been deleted directly from OpenStack and recreates them automatically.

## Enabling Drift Detection

Drift detection is disabled by default. Enable it per-resource by setting `spec.resyncPeriod`:

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
  resyncPeriod: 1h   # Re-check OpenStack every hour
  resource:
    description: Critical application network
```

The `resyncPeriod` field accepts any Go duration string: `10m`, `1h`, `24h`, etc. Very low values can create a high volume of OpenStack API calls.

**Default:** `0` (disabled). When disabled, ORC only reconciles resources in response to spec changes or controller restarts.

### Setting a Global Default

To enable drift detection for all resources without setting `resyncPeriod` on each one, configure the manager's `--default-resync-period` flag:

```yaml
spec:
  containers:
  - name: manager
    args:
    - --default-resync-period=10h
```

Per-resource `spec.resyncPeriod` takes precedence over this default when set.

!!! note

    Conservative resync periods (e.g., `1h` or `10h`) are recommended in production to avoid excessive OpenStack API calls.

## How It Works

After a resource reaches a stable state (`Progressing=False`), ORC schedules a reconciliation after the configured `resyncPeriod`. On each resync:

1. ORC fetches the current state of the OpenStack resource.
2. For **managed** resources: if drift is detected, ORC updates the resource to match the Kubernetes spec.
3. For **unmanaged** resources: ORC refreshes `status.resource` to reflect the current OpenStack state, but makes no changes.
4. The next resync is scheduled.

A small random jitter ([0%, +20%]) is applied to `resyncPeriod` to spread reconciliations and avoid thundering-herd effects.

!!! note

    Resources in a terminal error state (`Progressing=False` with reason `InvalidConfiguration` or `UnrecoverableError`) are **not** periodically resynced. Terminal errors require manual intervention to resolve.

## Tracking Sync Status

Every ORC resource has a `status.lastSyncTime` field that records when ORC last successfully reconciled with OpenStack:

```bash
kubectl get network critical-network -o jsonpath='{.status.lastSyncTime}'
# 2026-02-03T10:30:00Z
```

ORC persists this timestamp in the Kubernetes status. After a controller restart, it uses `lastSyncTime` to determine when the next resync should occur, preventing a thundering herd of reconciliations on startup.

## External Deletion Handling

When a resource is deleted directly from OpenStack (bypassing ORC), the behavior depends on how ORC originally obtained the resource.

### ORC-Created Resources (Managed, Not Imported)

If you created the resource through ORC's `spec.resource` field, ORC **recreates** it automatically:

1. ORC detects the resource is missing from OpenStack (the ID stored in `status.id` no longer exists).
2. ORC clears `status.id`.
3. On the next reconcile, ORC creates a new OpenStack resource.
4. The new resource ID is stored in `status.id`.

The ORC object continues to exist and becomes `Available=True` again once the resource is recreated.

```yaml
# This type of resource will be recreated if deleted from OpenStack
spec:
  managementPolicy: managed
  resyncPeriod: 10m  # Enable resync to detect deletion quickly
  resource:          # Resource was created by ORC
    description: My application network
```

!!! warning

    Recreation produces a new OpenStack resource with a **new ID**. Any OpenStack resources (outside ORC) that referenced the old ID will need to be updated manually.

### Imported Resources (Terminal Error)

If you imported an existing resource using `spec.import`, ORC reports a **terminal error** when the resource is deleted from OpenStack:

- `Available=False`
- `Progressing=False`
- Condition reason: `UnrecoverableError`
- Message: `resource has been deleted from OpenStack`

ORC does **not** recreate imported resources because it did not create them originally, and recreating a new empty resource would not restore what was lost.

```yaml
# Unmanaged resources enter terminal error if deleted from OpenStack
spec:
  managementPolicy: unmanaged
  import:
    filter:
      name: public  # Was imported by filter
```

To recover: delete and recreate the ORC object pointing at a newly created OpenStack resource.

### Summary Table

| Resource Type | How Obtained | External Deletion Behavior |
|--------------|--------------|---------------------------|
| Managed | `spec.resource` | **Recreated** automatically |
| Unmanaged | `spec.import.*` | **Terminal error** |

## Implications for Dependent Resources

OpenStack enforces referential integrity for most resource relationships (e.g., a Network cannot be deleted while Subnets exist). If an external deletion manages to bypass these constraints (e.g., direct database manipulation), the behavior of dependent ORC resources follows these rules:

### If a Parent Resource Is Recreated

When a parent resource (e.g., Network) is recreated by ORC, dependent resources that reference it (e.g., Subnets) detect the parent as available again but may encounter errors when OpenStack rejects operations referencing the old parent ID. **Manual intervention may be required** to recreate dependent resources against the new parent.

### If a Parent Resource Enters Terminal Error

When a parent resource enters terminal error:

- **Dependent resources waiting on it** (e.g., a Subnet waiting for its Network): ORC will not proceed — it waits until the parent becomes available again. The dependent is not itself in an error state; it is just waiting.
- **Dependent resources already created**: ORC continues managing them normally. If ORC attempts to update a dependent resource that references a deleted parent in OpenStack, the behavior depends on what OpenStack returns for that operation.

!!! warning

    If a parent resource is externally deleted in a way that bypasses OpenStack's referential integrity checks, the resulting state may require manual cleanup of both the parent and dependent resources. This is an unusual operational scenario and not specific to drift detection.
