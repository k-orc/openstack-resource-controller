# Running example resources

This tutorial walks through creating ORC resources using the provided examples. This is useful for manually testing your changes during development.

## Prerequisites

Before you begin, ensure you have:

- A running Kubernetes cluster (see [Development Quickstart](quickstart.md))
- ORC CRDs installed
- ORC manager running locally or deployed to the cluster
- Access to an OpenStack environment with valid credentials

## Create OpenStack credentials

Create a `clouds.yaml` file in `examples/local-config/`:

```yaml
clouds:
  openstack:
    auth:
      auth_url: https://your-openstack:5000
      project_name: your-project
      username: your-username
      password: your-password
      user_domain_name: Default
      project_domain_name: Default
    region_name: RegionOne
```

!!! warning "Security"

    The `examples/local-config/` directory is in both `.gitignore` and `.dockerignore`, so credentials should not be accidentally committed or included in container builds.

!!! note

    The cloud name **must** be `openstack` to match the examples.

## Define an external network

Create `examples/local-config/external-network-filter.yaml` with a filter that uniquely identifies an external network in your cloud:

```yaml
apiVersion: openstack.k-orc.cloud/v1alpha1
kind: Network
spec:
  import:
    filter:
      name: public
```

See `examples/local-config/external-network-filter.yaml.example` as a template.

## Initialize the kustomize environment

From the `examples/` directory, run:

```bash
make
```

This performs two setup tasks:

1. **Generates dev-settings**: Creates `components/dev-settings/kustomization.yaml` with your username as a `namePrefix`. This avoids naming conflicts when multiple developers share the same OpenStack cloud.

2. **Loads local configuration**: Creates a secret with your credentials and ORC resources referencing your external network.

Example output:

```
echo "$KUSTOMIZATION" > components/dev-settings/kustomization.yaml
kubectl apply -k apply/local-config --server-side
secret/jdoe-cloud-config-g4ckbm986f serverside-applied
network.openstack.k-orc.cloud/jdoe-external-network serverside-applied
subnet.openstack.k-orc.cloud/jdoe-external-subnet-ipv4 serverside-applied
```

!!! note "Troubleshooting"

    If you see this error, you need to run `make` in the examples directory first:

    ```
    Error: accumulating components: accumulateDirectory: "couldn't make target for path
    '.../examples/components/dev-settings': unable to find one of 'kustomization.yaml',
    'kustomization.yml' or 'Kustomization' in directory '.../examples/components/dev-settings'"
    ```

## Create an example resource

The `examples/apply/` directory contains various example configurations. To create a managed network:

```bash
kubectl apply -k examples/apply/managed-network --server-side
```

See the resource being created with:

```bash
kubectl get network -w
```

## Clean up

Delete the example resources:

```bash
kubectl delete -k examples/apply/managed-network
```

!!! tip "Credentials persist"

    We intentionally create credentials separately from example resources. This allows you to delete examples without removing the credentials secret, which would block deletion until the secret's finalizers are removed.
