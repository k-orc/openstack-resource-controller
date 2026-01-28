# User Guide

This guide covers the core concepts and common usage patterns for ORC (openstack-resource-controller). It assumes you have already completed the [Quick Start](../getting-started.md) guide and have ORC deployed to your cluster.

## Core Concepts

### Management Policies

Every ORC resource has a `managementPolicy` that determines how ORC manages the OpenStack resource:

| Policy | Description |
|--------|-------------|
| `managed` | ORC creates, updates, and deletes the OpenStack resource. This is the default. |
| `unmanaged` | ORC imports an existing OpenStack resource but will not modify or delete it. |

**When to use `managed`:**

- You want ORC to have full control over the resource lifecycle
- The resource doesn't exist yet and should be created
- You want changes to the Kubernetes object to be reflected in OpenStack

**When to use `unmanaged`:**

- The resource already exists and is managed by another system
- You need to reference shared infrastructure (external networks, public flavors)
- You want to prevent accidental deletion of critical resources

### Import vs Create

ORC can either create new OpenStack resources or import existing ones:

**Creating a new resource** (managed):
```yaml
apiVersion: openstack.k-orc.cloud/v1alpha1
kind: Network
metadata:
  name: my-network
spec:
  cloudCredentialsRef:
    cloudName: openstack
    secretName: openstack-clouds
  managementPolicy: managed
  resource:
    description: My application network
    tags:
      - my-app
```

**Importing an existing resource by ID** (unmanaged):
```yaml
apiVersion: openstack.k-orc.cloud/v1alpha1
kind: Network
metadata:
  name: external-network
spec:
  cloudCredentialsRef:
    cloudName: openstack
    secretName: openstack-clouds
  managementPolicy: unmanaged
  import:
    id: "12345678-1234-1234-1234-123456789abc"
```

**Importing an existing resource by filter** (unmanaged):
```yaml
apiVersion: openstack.k-orc.cloud/v1alpha1
kind: Network
metadata:
  name: external-network
spec:
  cloudCredentialsRef:
    cloudName: openstack
    secretName: openstack-clouds
  managementPolicy: unmanaged
  import:
    filter:
      name: public
      external: true
```

!!! note

    When importing by filter, the filter must match exactly one resource. If no resources match, ORC will keep retrying. If multiple resources match, ORC will report an error.

### Deletion Behavior

For managed resources, the `managedOptions.onDelete` field controls what happens when the Kubernetes object is deleted:

| Value | Description |
|-------|-------------|
| `delete` | Delete the OpenStack resource when the Kubernetes object is deleted. This is the default. |
| `detach` | Keep the OpenStack resource when the Kubernetes object is deleted. |

```yaml
spec:
  managementPolicy: managed
  managedOptions:
    onDelete: detach  # Keep the OpenStack resource on deletion
  resource:
    # ...
```

### Drift Detection

ORC periodically reconciles managed resources to detect and correct drift from the desired state. This ensures that if someone modifies an OpenStack resource outside of ORC (e.g., through the OpenStack CLI or dashboard), ORC will detect the change and restore the resource to match the Kubernetes specification.

**By default, drift detection is enabled with a resync period of 10 hours (`10h`).** The `managedOptions.resyncPeriod` field controls how often ORC checks for drift using standard duration format (e.g., `1h`, `30m`, `24h`):

| Value | Description |
|-------|-------------|
| `10h` | Check for drift every 10 hours. This is the default. |
| `1h` | Check for drift every hour. |
| `30m` | Check for drift every 30 minutes. |
| `0` | Disable periodic drift detection. Resources are only reconciled when their spec changes. |

```yaml
spec:
  managementPolicy: managed
  managedOptions:
    resyncPeriod: 1h  # Check for drift every hour
  resource:
    # ...
```

!!! note

    Drift detection only applies to managed resources. Unmanaged resources are never modified by ORC, so drift detection does not apply to them.

!!! tip

    For resources that are frequently modified outside of ORC, consider using a shorter resync period. For stable resources, the default of 10 hours is usually sufficient.

!!! warning

    Be aware of the side effects of drift detection, especially with a low `resyncPeriod`:

    - **Increased OpenStack API load**: Each drift detection cycle queries the OpenStack API. A low resyncPeriod across many resources can generate significant API traffic and may trigger rate limiting.
    - **Controller resource consumption**: Frequent reconciliation increases CPU and memory usage on the ORC controller.
    - **Potential conflicts**: If resources are actively being modified by external systems (other controllers, automation scripts, or manual operations), frequent drift correction can cause conflicts or unexpected behavior.
    - **Network overhead**: Each reconciliation involves network calls to OpenStack, which adds latency and bandwidth usage.

    Consider your environment's scale and requirements when configuring resyncPeriod. For most use cases, the default of 10 hours provides a good balance between drift detection and resource efficiency.

### Resource References

ORC resources reference each other using `*Ref` fields. These references:

- Are resolved by Kubernetes object name (in the same namespace)
- Create automatic dependencies - ORC waits for referenced resources to be ready
- Prevent deletion of resources that are still in use

For example, a Subnet references a Network:
```yaml
apiVersion: openstack.k-orc.cloud/v1alpha1
kind: Subnet
metadata:
  name: my-subnet
spec:
  # ...
  resource:
    networkRef: my-network  # References the Network named "my-network"
    cidr: 192.168.1.0/24
    ipVersion: 4
```

### Understanding Status and Conditions

Every ORC resource reports its status through two conditions: `Available` (whether the resource is ready for use) and `Progressing` (whether ORC is still working on it). For detailed information about conditions and their meanings, see [Troubleshooting: Status Conditions Explained](../troubleshooting.md#status-conditions-explained).

Check resource status with:
```bash
kubectl get networks
kubectl get network my-network -o yaml
```

The `.status.resource` field contains the observed state of the OpenStack resource, including fields set by OpenStack (like `projectID`, `createdAt`, etc.).

## Cloud Credentials

All ORC resources require a `cloudCredentialsRef` that points to a Kubernetes Secret containing OpenStack credentials.

### Creating the Credentials Secret

The secret must contain a `clouds.yaml` file:

```bash
kubectl create secret generic openstack-clouds \
    --from-file=clouds.yaml=/path/to/your/clouds.yaml
```

Example `clouds.yaml`:
```yaml
clouds:
  openstack:
    auth:
      auth_url: https://keystone.example.com:5000/v3
      project_name: my-project
      username: my-user
      password: my-password
      user_domain_name: Default
      project_domain_name: Default
    region_name: RegionOne
```

### Using Custom CA Certificates

If your OpenStack deployment uses a custom CA, include it in the secret:

```bash
kubectl create secret generic openstack-clouds \
    --from-file=clouds.yaml=/path/to/clouds.yaml \
    --from-file=cacert=/path/to/ca-bundle.crt
```

### Referencing Credentials

Every ORC resource references credentials like this:
```yaml
spec:
  cloudCredentialsRef:
    secretName: openstack-clouds  # Name of the secret
    cloudName: openstack          # Name of the cloud in clouds.yaml
```

!!! warning

    ORC prevents deletion of credential secrets while they are still referenced by ORC resources. Delete the ORC resources first before deleting the credentials secret.
