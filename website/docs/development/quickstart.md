# Local development quickstart

This guide covers setting up a local development environment for ORC, running the controller, and executing tests.

## Setting up kind

[kind](https://kind.sigs.k8s.io/) (Kubernetes in Docker) provides a local Kubernetes cluster for development.

### Install kind

Follow the [official installation instructions](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) for your platform.

### Create a cluster

Create a default kind cluster:

```bash
kind create cluster
```

Verify the cluster is running:

```bash
kubectl get nodes
```

```
NAME                 STATUS   ROLES           AGE     VERSION
kind-control-plane   Ready    control-plane   1m      v1.30.0
```

## Setting up DevStack

ORC's end-to-end tests require an OpenStack environment. We recommend [DevStack](https://docs.openstack.org/devstack/latest/) for local development because several e2e tests require admin-level access to OpenStack APIs, and it's easy enough to restart fresh if anything goes wrong with the OpenStack environment while testing.

### Install DevStack

Follow the [DevStack Quick Start Guide](https://docs.openstack.org/devstack/latest/guides/single-machine.html) to set up a single-machine installation.

For ORC development, the default services are currently enough.

### Configure credentials

After DevStack installation, locate your `clouds.yaml` file (typically at
`/etc/openstack/clouds.yaml` or in your DevStack directory). You'll need two
cloud entries:

- `devstack`, for regular user credentials on the demo project
- `devstack-admin-demo`, for admin credentials on the demo project

See [Set Up Cloud Credentials](../howto/cloud-credentials.md) for how to
create the Kubernetes secret from your `clouds.yaml`.

## Loading the CRDs

Install the ORC Custom Resource Definitions into your cluster:

```bash
kubectl apply -k config/crd --server-side
```

Verify the CRDs are installed:

```bash
kubectl get crds | grep openstack
```

```
flavors.openstack.k-orc.cloud         2024-11-11T12:00:00Z
images.openstack.k-orc.cloud          2024-11-11T12:00:00Z
networks.openstack.k-orc.cloud        2024-11-11T12:00:00Z
...
```

!!! tip "After changing API types"

    Run `make generate` to regenerate CRDs and other generated code, then
    re-apply them to the cluster with the command above.

## Running ORC locally

Run the ORC manager directly from source:

```bash
go run ./cmd/manager -zap-log-level 5
```

The manager will start all controllers and begin watching for ORC resources. Use ++ctrl+c++ to stop the manager.

!!! tip "After changing source code"

    Re-run the same command to recompile and restart the controller.

## Running tests

At this point, you're ready to run both the [unit tests](writing-tests.md#running-tests) and the [end-to-end tests](writing-tests.md#running-tests_1).

```bash
# Run all unit tests
make test

# Run all end-to-end tests
make test-e2e
```

See [Writing Tests](writing-tests.md) for details on running individual
controller tests and writing new tests.
