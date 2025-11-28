# Installation

This page covers different methods for installing ORC in your Kubernetes cluster.

## Prerequisites

- Kubernetes v1.29 or later (required for CEL validations)
- `kubectl` configured to access your cluster

## Quick Install

The simplest way to install ORC is using the release manifest:

```bash
kubectl apply --server-side -f \
    https://github.com/k-orc/openstack-resource-controller/releases/latest/download/install.yaml
```

This installs:

- Custom Resource Definitions (CRDs) for all ORC resources
- The `orc-system` namespace
- The ORC controller deployment
- Required RBAC roles and bindings

Verify the installation:

```bash
kubectl get pods -n orc-system
```

Expected output:
```
NAME                                      READY   STATUS    RESTARTS   AGE
orc-controller-manager-5d4b8c9f7-xxxxx    1/1     Running   0          30s
```

## Install a Specific Version

To install a specific version:

```bash
export ORC_VERSION="v2.3.0"
kubectl apply --server-side -f \
    https://github.com/k-orc/openstack-resource-controller/releases/download/${ORC_VERSION}/install.yaml
```

See the [Releases page](https://github.com/k-orc/openstack-resource-controller/releases) for available versions.

## Install from Source

To install from the main branch (for development or testing):

```bash
git clone https://github.com/k-orc/openstack-resource-controller.git
cd openstack-resource-controller

make deploy IMG=quay.io/orc/openstack-resource-controller:branch-main
```

## Configuration Options

### Controller Flags

The controller supports several configuration flags:

| Flag | Description | Default |
|------|-------------|---------|
| `--namespace` | Namespace(s) to watch (repeatable) | All namespaces |
| `--scope-cache-max-size` | Maximum size of the credentials cache | 10 |
| `--default-ca-certs` | Path to CA certificates file | - |
| `--zap-log-level` | Log verbosity (0-5) | 0 |

To customize the deployment, edit the controller manager deployment:

```bash
kubectl edit deployment -n orc-system orc-controller-manager
```

### Watching Specific Namespaces

By default, ORC watches all namespaces. To restrict it to specific namespaces, add `--namespace` flags to the controller args:

```bash
kubectl patch deployment -n orc-system orc-controller-manager --type='json' -p='[
  {"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--namespace=namespace1"},
  {"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--namespace=namespace2"}
]'
```

The `--namespace` flag can be repeated to watch multiple namespaces.

### Resource Limits

The default memory limit is 256Mi. For large deployments, you may need to increase this:

```bash
kubectl set resources -n orc-system deployment/orc-controller-manager \
    --limits=memory=512Mi
```

## Upgrading

To upgrade to a new version:

```bash
export NEW_VERSION=<target_version> # For instance v2.4.0
kubectl apply --server-side -f \
    https://github.com/k-orc/openstack-resource-controller/releases/download/${NEW_VERSION}/install.yaml
```

The upgrade process:

1. Updates CRDs with any new fields
2. Rolls out the new controller version
3. Existing resources continue to be managed

!!! note

    Always check the [Changelog](changelog.md) for breaking changes before upgrading major versions.

## Uninstalling

To remove ORC from your cluster:

```bash
# 1. Delete all ORC resources first
kubectl delete openstack --all

# 2. Wait for resources to be cleaned up
kubectl get openstack

# 3. Delete credentials secrets
kubectl delete secret openstack-clouds

# 4. Uninstall ORC
kubectl delete -f \
    https://github.com/k-orc/openstack-resource-controller/releases/latest/download/install.yaml

# 4. Or, if you installed from source
make undeploy
```

!!! warning

    Deleting ORC before deleting ORC resources will leave orphaned resources in OpenStack that must be cleaned up manually. Alternatively, ORC can be re-installed to clean up the resources.

## Verifying the Installation

After installation, verify ORC is running:

```bash
# Check the controller pod
kubectl get pods -n orc-system

# Check CRDs are installed
kubectl get crd | grep openstack.k-orc.cloud

# Check controller logs
kubectl logs -n orc-system deployment/orc-controller-manager
```

## Next Steps

Once ORC is installed, follow the [Quick Start](getting-started.md) guide to create your first resources.
