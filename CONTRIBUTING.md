# Contributing to Gopherkube

## Development environment
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Run the controllers locally
1. Run code generation and compile your code:

```sh
make
```

2. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

3. Once done, to delete the CRDs from the cluster:

```sh
make uninstall
```

## Design choices

### Naming conventions
For most of the resources in the OpenStack API, names are not unique. OpenStack relies in UUIDs for differentiating between resources. This won't be the case for Gopherkube, in which we rely on name uniqueness to differentiate between resources. For such purpose, we create mapping between the resource representation in Gopherkube and the resource in OpenStack.

The API of each OpenStack resource in Gopherkube is based on the corresponding OpenStack REST API. The name of the `snake_case` fields in the REST API are replaced with `camelCase`.

### References
Some resource definitions must refer to other resources. For example, a subnet definition refers to a network. In Gopherkube, a resource refers to another by the name of the corresponding Gopherkube manifest. For example: given an OpenStackNetwork with `metadata.name: network-1`, the OpenStackSubnet spec will have `spec.network: network-1` regardless of the UUID or the name of the network in the OpenStack cloud.

### How to write a new CRD
The `spec` of each OpenStack resource is based on the OpenStack REST API for creation (the POST call), with these modifications:
* `snake_case` names must be replaced with their `camelCase` version;
* fields that reference other OpenStack resources must reference the corresponding CRD by name. For example: in a subnet, the [REST API](https://docs.openstack.org/api-ref/network/v2/#create-subnet) requires the field `network_id`; the OpenStackSubnet CRD requires instead the field `network` that expects the name of the corresponding OpenStackNetwork.
* optional fields are be represented as pointers. One exception is arrays, that are represented as bare slices even when optional.
* some resources can be further modified in the OpenStack API with additional calls REST API calls after creation. Gopherkube CRDs can embed the information needed to perform that second call, directly in the `spec`. For example, an OpenStackImage `spec` contains a `properties` array that the controller will use to set image properties with a separate PATCH call.
* static validation is enforced when possible, in accordance with the REST API. For this purpose, we rely on [Kubebuilder validations](https://book.kubebuilder.io/reference/markers/crd-validation.html).

The `status` of each OpenStack resource is based on the response of REST API upon resource creation. Data in `status` is reported as-is, complete with the bare OpenStack UUID identifiers of the other resources. Controllers shouldn't attempt to resolve OpenStack UUIDs with their relative Gopherkube representations.
