# ORC API Contracts

## General

* Do not define API fields which are not implemented. We will define them when we implement them.
* All strings must have a maximum length, even if we have to guess.
* All lists must have a maximum length, even if we have to guess.
* Do not reference `tenant_id` anywhere, either in spec or status. Use `project_id` instead.

## ProjectID

In general, use of ProjectID in a spec requires admin credentials unless the ProjectID matches the project of the current session. This is also the default, so in general non-admin users do not require ProjectID. CAPO does not focus on use cases which require admin credentials, so CAPO does not allow ProjectID to be specified. However, ORC does intend to support use cases which require admin credentials. Therefore, consider allowing ProjectID to be specified in resource specs. It should always be reported in resource statuses when it is available.

## Resource-specific conventions

After scaffolding, each resource will require 3 custom structs:

* `CustomFilter`
* `CustomResourceSpec`
* `CustomResourceStatus`

where `Custom` is the name of the specific resource.

### Filter

This is located at `spec.import.filter` in the base object. It is used when importing a pre-existing OpenStack resource into ORC when the resource's ID is not already known.

* Filter must not contain an ID field. This is handled separately by `spec.import.id`.
* Where an equivalent filter exists in CAPO, consider copying it where possible.
* Neutron types should include FilterByNeutronTags inline

### ResourceSpec

This is located at `spec.resource` is the base object. It is only defined for managed objects (`spec.managementPolicy == 'managed'`).

* Where relevant, the ResourceSpec should include a `name` field to allow object name to be overridden
* All fields should use pre-defined validated types where possible, e.g. `OpenStackName`, `OpenStackDescription`, `IPvAny`.
* Lists should have type `set` or `map` where possible, but `atomic` lists may be necessary where a struct has no merge key.

### ResourceStatus

This is located at `status.resource` in the base object. It contains the observed state of the OpenStack object.

* ID must not be included. It is stored separately at `status.ID`.
* ResourceStatus fields should not be validated: we should store any value returned by OpenStack, even invalid ones.
  * This may require implementing separate `Spec` and `Status` variants of structs.
* Lists should be `atomic`.

## Dependencies

Dependencies are at the core of what ORC does. At the lowest level ORC performs CRUD operations OpenStack resources using the REST API. However, one of the principal benefits of using ORC rather than just making REST calls is that it automatically does this:
* In the correct order
* As soon as possible
* In parallel if possible

It achieves this through dependency management. We must keep these high level goals in mind as we continue to evolve ORC.

For the sake of the user experience we should aim to keep the number of dependency models to a minimum. Always re-use an existing dependency model where possible.

### API

* Dependencies are always by name between ORC objects (i.e. never on a resource specified by OpenStack UUID).
* Dependencies are always between objects in the same kubernetes namespace.
* Dependencies should follow [the kubernetes naming convention on references](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#naming-of-the-reference-field) (e.g. `fooRef`)
* There should only be one way to specify a particular dependency.

### Behaviour

* Controllers must handle a dependency which is missing or not yet ready by waiting
* Controllers should react immediately to the creation or readiness of a dependency

### Specific models

#### Deletion guard

The purpose of the deletion guard is to prevent the deletion of an OpenStack resource until all objects which depend on it have been deleted first. It works by adding an additional controller which reconciles a finalizer on the `guarded` object on behalf of `dependency` objects.

e.g. A subnet cannot exist without its network. The subnet controller adds a deletion guard on network objects. `guarded` is the network. `dependency` is the subnet.

The deletion guard controller watches `guarded` and automatically adds its finalizer to any `guarded` object which does not have it. If the `guarded` object is deleted, the deletion guard controller checks that there are no `dependency` objects before removing it.

Note that for this model to work in practice, the `guarded` controller must not do any resource cleanup until the only remaining finalizer is its own. Otherwise, while the kubernetes object will not be deleted until all finalizers are removed, we will still attempt to delete the OpenStack resource while it is still in use.

#### One-way ordering dependency

Meta: need a better name for this. 'normal' dependency?

We expect most dependencies to use this model. This is where one object refers to another object which must exist and be ready before reconciliation can continue.

e.g. Networks and subnets. Subnet refers to a network. Subnet cannot be created until the referenced network exists and is available.

Note that in principal this dependency could be specified in either direction. We could, for example, specify a list of subnets on a network. However, it is a convention that we specify the dependency in the direction that we must wait for the referenced object to exist and be available first.

The reconcile loop must fetch the referenced object to check that it exists and is ready, and most likely to get the UUID of the OpenStack resource from its status. If the referenced object either does not exist or is not ready, the reconcile loop MUST NOT return an error, which would result in an exponential backoff loop and errors in the logs for an expected situation. It should indicate which object it is waiting on in its `Progressing` condition, and return no error. The reconcile loop relies on being called again when the referenced object is ready for use.

In concert with the above behaviour, during initialisation of the controller (i.e. in SetupWithManager) we must also add a watch on the referenced type. When this watch observes an event on the referenced resource, it must trigger a reconcile of every resource resource that references it. e.g. When a network becomes available, we should immediately trigger a reconcile of every subnet which references it. When we observe an event on the referenced object, the query we need to execute is: list all objects managed by this controller which reference the event object.

However, because dependencies are only specified in the direction of 'waits on', there is no 'back reference' to all waiting objects. We must create one. We do this by adding a custom FieldIndexer for the managed type.

In the network/subnet case, the subnet controller adds a field indexer to subnets which returns the name of the referenced network. This allows us to list subnets and filter on referenced network using the custom index. Note that we don't need to worry about namespace collisions, because we additionally filter by namespace.

A controller adding this kind of guard will also typically add a deletion guard.

Controllers should only add field indexers on their own managed type.
