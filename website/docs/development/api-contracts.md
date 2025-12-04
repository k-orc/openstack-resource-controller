# ORC API Contracts

## Kubernetes API conventions

In general we try to follow the [Kubernetes API conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md). Note that these API conventions are written for Kubernetes itself. As ORC is not part of Kubernetes we are not 'bound' by them, and have occasionally chosen to deviate from them. However, we treat these conventions as the distilled experience of the authors of the API itself, so we don't deviate from them lightly.

## Resource-specific conventions

After scaffolding, each resource will require 3 custom structs:

* `<ResourceName>Filter`
* `<ResourceName>ResourceSpec`
* `<ResourceName>ResourceStatus`

where `Custom` is the name of the specific resource.

These structs, along with any supporting resource-specific code, should be defined in `api/v1alpha1/<resourcename>_types.go`.

### Filter

This is located at `spec.import.filter` in the base object. It is used when
importing a pre-existing OpenStack resource into ORC when the resource's ID is
not already known.

* Filter must not contain an ID field. This is handled separately by `spec.import.id`.
* Where an equivalent filter exists in [Cluster API Provider OpenStack](https://github.com/kubernetes-sigs/cluster-api-provider-openstack/tree/main/api/v1beta1), consider copying it where possible as these filters are already widely used.
* Neutron types should include FilterByNeutronTags inline.

### ResourceSpec

This is located at `spec.resource` in the base object. It is only defined for managed objects (`spec.managementPolicy == 'managed'`).

* Where relevant, the ResourceSpec should include a `name` field to allow object name to be overridden.
* All fields should use pre-defined validated types where possible, e.g. `OpenStackName`, `NeutronDescription`, `IPvAny`.
* Lists should have type `set` or `map` where possible, but `atomic` lists may be necessary where a struct has no merge key.

### ResourceStatus

This is located at `status.resource` in the base object. It contains the observed state of the OpenStack object.

* ID of the resource must not be included. It is stored separately at `status.ID`.
* The content of ResourceStatus fields should not be validated: we should store any value returned by OpenStack, even invalid ones.
    * This requires implementing separate `Spec` and `Status` variants of structs.
* Lists should be `atomic`.
* All fields should be optional and have the `omitempty` tag.
* strings do not need to be pointers.

Note that although we don't validate the content of status fields, we must still add maximum length validation for strings and lists. This both defensively constrains the size of the object in the etcd database, and limits the maximum size of the object computed by kube-apiserver for the purposes of CEL validation complexity. Strings should ideally be constrained to the length of the field on the source service's database, or failing that some value in the order of kilobytes large enough to be improbable to occur in practice. Lists should similarly be constrained to a value large enough to be improbable to occur in practice.

## Generating CRDs

The CRDs are automatically generated from the Go API. To generate them, run:

```bash
make generate
```

However, note that this will not add a new CRD to the list of CRDs to load. This must be done manually by adding the new CRD to the list in `config/crd/kustomization.yaml`.

This will also generate all other artifacts, including the apply configurations in `pkg/clients/applyconfiguration`.

## Kustomize references

You should update `examples/components/kustomizeconfig/kustomizeconfig.yaml` with any references defined by your API. This will ensure that kustomize handles them correctly during transformations.

!!! note

    `kustomizeconfig.yaml` is currently used only by the examples, but we intend to eventually publish it as a release artifact.

## Generating API artifacts

There are a number of artifacts which are automatically generated from the API. Of primary interest are:

* The CRDs
* The apply configurations in `pkg/clients`

Every time you make a change to the API you should ensure these are up to date by running:

```
make generate
```

## General API guidelines

* Do not define API fields which are not implemented. We will define them when we implement them.
* All strings must have a maximum length, even if we have to guess.
* All lists must have a maximum length, even if we have to guess.
* Constants coming from OpenStack should be copied verbatim.
* All fields should have a godoc comment, beginning with the json name of the field.
* Use sized integers, `int32` or `int64`, instead of `int`.
* Do not use unsigned integers: use `intN` with a kubebuilder marker validating for a minimum of 0.
* Optional fields should have the `omitempty` tag.
* Optional fields should be pointers, unless their zero-value is also the OpenStack default, or we can be very confident that we will never need to distinguish between empty and unset values. e.g. Will we ever want to set a value explicitly to the empty string?
