# ORC API Contracts

## General

* Do not define API fields which are not implemented. We will define them when we implement them.
* All strings must have a maximum length, even if we have to guess.
* All lists must have a maximum length, even if we have to guess.
* Constants coming from OpenStack should be copied verbatim.
* All fields should have a godoc comment, beginning with the json name of the field.
* Use sized integers, `int32` or `int64`, instead of `int`.
* Do not use unsigned integers: use `int` with a kubebuilder marker validating for a minimum of 0.
* Optional fields should have the `omitempty` tag.
* Optional fields should be pointers, unless their zero-value is also the OpenStack default.

## Resource-specific conventions

After scaffolding, each resource will require 3 custom structs:

* `CustomFilter`
* `CustomResourceSpec`
* `CustomResourceStatus`

where `Custom` is the name of the specific resource.

These structs, along with any supporting resource-specific code, should be defined in `api/v1alpha1/<resource>_types.go`.

### Filter

This is located at `spec.import.filter` in the base object. It is used when
importing a pre-existing OpenStack resource into ORC when the resource's ID is
not already known.

* Filter must not contain an ID field. This is handled separately by `spec.import.id`.
* Where an equivalent filter exists in CAPO, consider copying it where possible.
* Neutron types should include FilterByNeutronTags inline.

### ResourceSpec

This is located at `spec.resource` in the base object. It is only defined for managed objects (`spec.managementPolicy == 'managed'`).

* Where relevant, the ResourceSpec should include a `name` field to allow object name to be overridden.
* All fields should use pre-defined validated types where possible, e.g. `OpenStackName`, `NeutronDescription`, `IPvAny`.
* Lists should have type `set` or `map` where possible, but `atomic` lists may be necessary where a struct has no merge key.

### ResourceStatus

This is located at `status.resource` in the base object. It contains the observed state of the OpenStack object.

* ID of the resource must not be included. It is stored separately at `status.ID`.
* ResourceStatus fields should not be validated: we should store any value returned by OpenStack, even invalid ones.
    * This requires implementing separate `Spec` and `Status` variants of structs.
* Lists should be `atomic`.
* All fields should be optional and have the `omitempty` tag.
* strings do not need to be pointers.