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

Every ORC resource reports its status through conditions:

| Condition | Meaning |
|-----------|---------|
| `Available=True` | The OpenStack resource is ready for use |
| `Available=False` | The resource is not yet ready or has an error |
| `Progressing=True` | ORC is still working on reconciling the resource |
| `Progressing=False` | Reconciliation is complete (success or terminal error) |

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
