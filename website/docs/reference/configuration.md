# Controller Configuration

This page documents all configuration flags and options for the ORC controller.

## Controller Flags

The controller accepts the following flags, set as arguments on the controller
manager container:

| Flag | Description | Default |
|------|-------------|---------|
| `--namespace` | Namespace(s) to watch (repeatable) | All namespaces |
| `--scope-cache-max-size` | Maximum size of the credentials cache | 10 |
| `--default-ca-certs` | Path to CA certificates file | - |
| `--default-resync-period` | Global default resync period for drift detection (e.g. `10h`) | `0` (disabled) |
| `--zap-log-level` | Log verbosity (0-5) | 0 |

To customize the deployment, edit the controller manager deployment:

```bash
kubectl edit deployment -n orc-system orc-controller-manager
```

## Namespace Scoping

By default, ORC watches all namespaces. To restrict it to specific namespaces,
add `--namespace` flags to the controller args:

```bash
kubectl patch deployment -n orc-system orc-controller-manager --type='json' -p='[
  {"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--namespace=namespace1"},
  {"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--namespace=namespace2"}
]'
```

The `--namespace` flag can be repeated to watch multiple namespaces.

## Resource Limits

The default memory limit is 256Mi. For large deployments, you may need to
increase this:

```bash
kubectl set resources -n orc-system deployment/orc-controller-manager \
    --limits=memory=512Mi
```

## Default Resync Period

By default, ORC only reconciles resources in response to spec changes or
controller restarts. To enable periodic drift detection globally, set
`--default-resync-period`:

```bash
kubectl patch deployment -n orc-system orc-controller-manager --type='json' -p='[
  {"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--default-resync-period=10h"}
]'
```

Per-resource `spec.resyncPeriod` takes precedence over this default when set.
See [How to Enable Drift Detection](../howto/drift-detection.md) for per-resource
configuration and [Drift Detection Explained](../concepts/drift-detection.md)
for how resync works.

## Log Levels

| Level | Description |
|-------|-------------|
| 0 | Status messages: startup, shutdown |
| 1 | Info: resource creation/deletion, reconcile completion |
| 2 | Verbose: fires every reconcile |
| 3+ | Debug: detailed internal state |

Set log level via the `--zap-log-level` flag:

```bash
kubectl patch deployment -n orc-system orc-controller-manager --type='json' -p='[
  {"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--zap-log-level=2"}
]'
```
