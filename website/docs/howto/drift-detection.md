# How to Enable Drift Detection

Drift detection lets ORC periodically re-check OpenStack resources and correct
any changes made outside of ORC. This page covers how to enable and monitor it.
For a deeper explanation of how drift detection works, including external
deletion handling and implications for dependent resources, see
[Drift Detection Explained](../concepts/drift-detection.md).

## Enable per-resource

Set `spec.resyncPeriod` on any ORC resource:

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

The field accepts any Go duration string: `10m`, `1h`, `24h`, etc.

Set it to `0` (or omit it) to disable periodic resync for that resource. When
disabled, ORC only reconciles in response to spec changes or controller
restarts.

!!! note

    Conservative resync periods (e.g. `1h` or `10h`) are recommended in
    production to avoid excessive OpenStack API calls.

## Set a global default

To enable drift detection for all resources without setting `resyncPeriod` on
each one, add the `--default-resync-period` flag to the controller:

```yaml
spec:
  containers:
  - name: manager
    args:
    - --default-resync-period=10h
```

Per-resource `spec.resyncPeriod` takes precedence over this default when set.
See [Controller Configuration](../reference/configuration.md#default-resync-period)
for more details.

## Check when a resource was last synced

Every ORC resource has a `status.lastSyncTime` field:

```bash
kubectl get network critical-network -o jsonpath='{.status.lastSyncTime}'
# 2026-02-03T10:30:00Z
```

## Verify that drift was corrected

After a resync, check the resource status:

```bash
kubectl get network critical-network -o yaml
```

If drift was detected on a **managed** resource, ORC updates OpenStack to match
the spec. The `.status.resource` field reflects the corrected state.

For **unmanaged** resources, ORC refreshes `.status.resource` to reflect the
current OpenStack state but makes no changes.
