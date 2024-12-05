# Local development quickstart

We will:
* Run ORC locally:
  * Create a local kind cluster
  * Load the ORC CRDs
  * Run the ORC manager locally directly from source
* Create an example ORC resource:
  * Create some credentials
  * Generate `dev-settings` containing our own username as a name prefix
  * Create an ORC resource using the above

## Run ORC locally

### Create a kind cluster

Obtain `kind` from https://kind.sigs.k8s.io/.

Create a default kind cluster:
```bash
$ kind create cluster
Creating cluster "kind" ...
 ✓ Ensuring node image (kindest/node:v1.30.0) 🖼c
 ✓ Preparing nodes 📦
 ✓ Writing configuration 📜
 ✓ Starting control-plane 🕹️ 
 ✓ Installing CNI 🔌
 ✓ Installing StorageClass 💾
Set kubectl context to "kind-kind"
You can now use your cluster with:

kubectl cluster-info --context kind-kind

Have a question, bug, or feature request? Let us know! https://kind.sigs.k8s.io/#community 🙂
```

Ensure your local context is set to the newly created kind cluster:
```bash
$ kubectl get node
NAME                 STATUS   ROLES           AGE     VERSION
kind-control-plane   Ready    control-plane   4m22s   v1.30.0
```

### Load the ORC CRDs

From the `orc` directory:
```bash
$ kubectl kustomize config/crd | kubectl apply -f - --server-side
customresourcedefinition.apiextensions.k8s.io/images.openstack.k-orc.cloud serverside-applied
customresourcedefinition.apiextensions.k8s.io/networks.openstack.k-orc.cloud serverside-applied
customresourcedefinition.apiextensions.k8s.io/subnets.openstack.k-orc.cloud serverside-applied
```

Ensure you run this from the `orc` directory, not the base CAPO directory. If you run it from CAPO it will load all the CAPO CRDs. It's not a problem if you do this accidentally, but they're not used and you still need to load the ORC CRDs from the `orc` directory.

### Run the ORC manager locally

```bash
$ go run ./cmd/manager -zap-log-level 5
2024-11-11T12:09:30Z    INFO    setup   starting manager
2024-11-11T12:09:30Z    INFO    starting server {"name": "health probe", "addr": "[::]:8081"}
2024-11-11T12:09:30Z    INFO    Starting EventSource    {"controller": "network", "controllerGroup": "openstack.k-orc.cloud", "controllerKind": "Network", "source": "kind source: *v1alpha1.Network"}
2024-11-11T12:09:30Z    INFO    Starting Controller     {"controller": "network", "controllerGroup": "openstack.k-orc.cloud", "controllerKind": "Network"}
2024-11-11T12:09:30Z    INFO    Starting EventSource    {"controller": "image", "controllerGroup": "openstack.k-orc.cloud", "controllerKind": "Image", "source": "kind source: *v1alpha1.Image"}
2024-11-11T12:09:30Z    INFO    Starting Controller     {"controller": "image", "controllerGroup": "openstack.k-orc.cloud", "controllerKind": "Image"}
2024-11-11T12:09:30Z    INFO    Starting workers        {"controller": "image", "controllerGroup": "openstack.k-orc.cloud", "controllerKind": "Image", "worker count": 1}
2024-11-11T12:09:30Z    INFO    Starting workers        {"controller": "network", "controllerGroup": "openstack.k-orc.cloud", "controllerKind": "Network", "worker count": 1}
```

To recompile, kill the process with ctrl-C and re-run it.

## Create an example ORC resource

### Define and create OpenStack credentials

Create a `clouds.yaml` file in `orc/examples/credentials`. The name of the cloud in this clouds.yaml must be `openstack`.

This file is in both `.gitignore` and `.dockerignore`, so should not be accidentally added to the git repo or a container build.

Create a credentials secret in your development cluster by loading the
`credentials-only` kustomize resource:

```bash
$ kubectl apply -k orc/examples/credentials-only --server-side
secret/mbooth-dev-test-cloud-config-g4ckbm986f serverside-applied
```

Note that we intentionally create credentials separately from other modules.
This allows us to delete an entire example kustomize module without also
deleting the credentials, which would prevent the deletion from completing.

### Generate `dev-settings`

The examples depend on a kustomize component called `dev-settings` which by default contains only a `namePrefix` with the current user's name. The purpose of this is to avoid naming conflicts between developers when generating resources in shared clouds, and also to identify culprits if the resources are not cleaned up.

To generate this file, change to the `examples/dev-settings` directory and run `make`:
```bash
$ cd orc/examples/dev-settings/
$ make
echo "$KUSTOMIZATION" > kustomization.yaml
```

Note that failing to do this will result in an error trying to generate an example resource like:
```
Error: accumulating components: accumulateDirectory: "couldn't make target for path '.../cluster-api-provider-openstack/orc/examples/dev-settings': unable to find one of 'kustomization.yaml', 'kustomization.yml' or 'Kustomization' in directory '.../cluster-api-provider-openstack/orc/examples/dev-settings'"
```

### Create an ORC resource

To generate the `managed-network` example:
```bash
$ cd orc/examples/managed-network
$ kustomize build . | kubectl apply -f - --server-side
network.openstack.k-orc.cloud/mbooth-orc-managed-network serverside-applied
```

NOTE: This will not create the required secret! To do this, see the section
above on creating credentials.

To cleanup the `managed-network` example:
```bash
$ kubectl delete -k .
```
