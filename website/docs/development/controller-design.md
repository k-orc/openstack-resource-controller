# Controller design considerations

## The Progressing condition

All ORC objects publish a Progressing condition. This condition has a strict definition:

True means that after the previous reconcile:

* The object status did not yet reflect the desired spec.
* The controller expects the object to be reconciled again.

False means:

* The object will not be reconciled again unless the spec changes.

Strict adherence to this definition enables some important use cases:

* Controllers can filter reconciles of objects which are up to date. This is especially useful after a controller restart when a reconcile would otherwise be triggered for every object present. This can present a 'thundering herd' problem, particularly when they result in many OpenStack API calls.
* A consumer of the API can easily determine when to stop waiting on an object to be reconciled. If Progressing is False then either the resource is available, or any currently reported error is not one the controller can resolve without intervention.

In particular, the generic controller's reconcile loop filters objects with an up-to-date Progressing status of False. This means that if your controller exits a reconcile without correctly setting the Progressing condition it may result in the object never successfully reconciling.

### Setting the Progressing condition

The Progressing condition is set by common code invoked from the generic controller based on 2 values:

* A slice of `ProgressStatus` objects
* An error

Several functions in the actuator interface your controller implements will return these 2 values. If the operation did not complete fully, these MUST return either:

* at least one `ProgressStatus`
* an error

As noted above, failure to do this will likely result in the reconciliation of your object hanging.

#### ProgressStatus

There are currently 3 types of ProgressStatus:

* Waiting for an OpenStack operation to complete: this will involve polling OpenStack, so these will also require a polling interval to be specified.
* Waiting for a dependent ORC object to be available: this requires that an appropriate dependency has been configured on the dependent object. If it has not, the reconciliation will hang.
* The operation completed, but we modified the resource and its reported status is now out of date: another reconcile will be scheduled immediately.

#### Transient and Terminal errors

Controllers will perform many operations which can fail. We split these errors into transient and terminal errors. A transient error is one which may eventually resolve itself without the object spec being updated. A terminal error is one which we don't expect to ever succeed unless the object spec is updated.

Example transient errors:

* Failure to contact an API endpoint
* An API call returned a 5xx (internal error)
* A kubernetes read or write operation failed for any reason

Example terminal errors:

* The spec is invalid
* OpenStack returned a non-retryable status, e.g. invalid request when creating a resource

By default, all errors should be treated as transient. No special handling is required for transient errors. If your method returns an error it will eventually be passed to the status writer. A transient error results in a Progressing status of True. The condition's reason will be set to TransientError, and the error message itself will be reported to the user via the condition's message. The controller will enter a default exponential backoff loop, so the object will continue to be reconciled indefinitely until the error no longer occurs.

> **NOTE**: we currently report *all* error messages to the user. At some point we may restrict this to only OpenStack errors to avoid potentially leaking internal configuration details.

When you are confident that an error will never be resolved we can instead return a terminal error. As well as not wasting resources by continuing to attempt an operation which will never succeed, this will clearly communicate to any API user waiting for the object to be reconciled that they can stop waiting.

To return a terminal error, wrap the error in an `orcerrors.TerminalError`. The status writer will observe this and set Progressing to False. Additionally, the error will not be returned by the reconcile loop, so we will not enter the error handling exponential backoff loop.

## Dependencies

Dependencies are at the core of what ORC does. At the lowest level ORC performs CRUD operations OpenStack resources using the REST API. However, one of the principal benefits of using ORC rather than just making REST calls is that it automatically does this:

* In the correct order
* As soon as possible
* In parallel if possible

It achieves this through dependency management.

> **NOTE**: in ORC dependencies can *only* be expressed between ORC objects. Therefore if one OpenStack resource depends on another, that relationship can only be expressed in ORC if both resources have corresponding ORC objects. Resoruces which a user may depend on but cannot create, like a flavor or a provider network, can be expressed by importing an existing resource.

A dependency is used anywhere that the controller must reference another object to complete an action. The dependency has features that enable us to:

* wait until the dependency is satisfied before continuing
* communicate to the user that we are waiting on other resources, and what those resources are
* ensure we are reconciled as soon as possible once the dependency is satisfied

Dependencies can be used anywhere in the resource actuator that can return `[]ProgressStatus` and `error`, which is most places. In general, try to resolve dependencies as late as possible, as close to the point of use as possible. Do this to avoid accidentally injecting a dependency requirement where it is not strictly required. For example, Subnet depends on Network. Subnet requires its network to be present:

* For creation, because a Subnet cannot be created without a network
* For import by filter, because the filter has an implicit constraint to the Subnet's Network

However, Network is not required for import by ID, or once `status.ID` has been set, so we should try to avoid requiring it in those circumstances. The reason for this is that it reduces coupling, so gives a user attempting to fix a failed deployment greater freedom. This is especially important when deleting resources. In a situation where a user has, for whatever reason, force deleted a network resource, we should not require them to recreate it before deleting a subnet resource whose `status.ID` is already set, or to propagate the requirement for manually removing finalizers.

To illustrate how this works we'll use a concrete example: a Server has a FlavorRef. When reconciling a Server, the controller will attempt to fetch the Flavor named by FlavorRef. If that Flavor doesn't exist, or is not yet ready, the operation will return an appropriate ProgressStatus indicating that we are waiting on another object. This will set the Progressing condition to True, but will not directly trigger another reconcile.

We must therefore ensure that we are triggered again when the Flavor named by FlavorRef becomes available. To achieve this, the Server controller must watch Flavors. For each Flavor which becomes Available, the watch must lookup every Server object which has a reference to it, and trigger a reconcile for that object. However, as a Flavor does not have a reference to all the Servers which reference it we need some way to perform the 'reverse' lookup. We do this by adding an index of FlavorRefs to Servers.

Note that we only need to check this dependency on creation: we would not update a Server because the Flavor it references was modified or deleted (even if Flavors were not immutable). If we implement Server resize, this will be by modifying the Server object's FlavorRef to reference a different Flavor.

Both of these actions are handled by `dependency.Dependency`. Dependencies are idiomatically defined as package-scoped variables in `controller.go`, the same file where `SetupWithManager` is defined. Methods on `dependency.Dependency` allow it to be configured appropriately for the controller. Refer to the godoc for details.

### Reconcile dependencies

A reconcile dependency is a normal dependency with the additional property that it may cause a fully reconciled object to restart its reconciliation. For example, a Server references one or more ports. The Server must wait for all of these Ports to be ready before creating the Server. When the Server is created, the Progressing status will be set to False with `observedGeneration` matching the object's current generation, indicating that the object is fully up to date. We check this at the start of every reconcile, and will skip reconciliation of a fully up to date object.

However, a Port may be deleted independently of the Server. When a Port is deleted, OpenStack will automatically detach it from the Server. This would mean that our fully reconciled Server whose spec has not changed is now out of date, as it has a reference to a Port that no longer exists. However, as the Server object is fully reconciled we will never check this, so even if the Port is recreated we will not re-add it.

There are 2 ways we can address this:

* A [deletion dependency](#deletion-dependencies) would prevent deletion of the Port while it is still in use
* A reconcile dependency allows the Port to be deleted, but resets the Server's Progressing condition so that the Server can be reconciled again

A deletion dependency works, and this is how this is currently implemented. The disadvantage is that it makes it harder to reconfigure an existing deployment, and reduces a user's flexibility to resolve deployment issues without resorting to manually removing finalizers, or having to delete more resources than strictly necessary. Where possible, a reconcile dependency should be favoured.

> **NOTE**: Reconcile dependencies are not currently implemented.

### Deletion dependencies

Deletion dependencies have all the same properties as a regular dependency, but additionally prevent the dependency object from being deleted while the object which depends on it still exists. This is achieved by:

* adding a finalizer to dependency object
* adding a small controller, called a deletion guard, to the dependency object which removes the finalizer if the dependency is marked deleted and no longer has any references managed by the deletion guard.

Deletion dependencies must be used where deleting the dependency object would either fail, or cause the reconciled object to fail. An example of each:

* Port has a SubnetRef. Attempting to delete a Subnet which still has ports will fail with a 409. Port adds a deletion dependency on Subnet so ORC will not attempt to delete a Subnet until all Ports referencing it have been deleted.
* Almost all objects reference a Secret containing cloud credentials. Deleting this Secret would cause any object using it to fail. Objects with a cloudCredentialsRef add a deletion dependency on Secret so Secrets may not be deleted while they are still in use.

Deletion dependencies should be avoided where the deletion of an object is allowed by OpenStack, even when it may cause another object to become out of date. Prefer a [reconcle dependency](#reconcile-dependencies) in this case.

Deletion dependencies are handled by `dependency.DeletionGuardDependency`. Refer to the godoc for details.

Methods on the deletion dependency will add the deletion guard to the manager, and therefore ensure that any added finalizers are removed at the correct time. However, the deletion guard is not responsible for adding finalizers to dependencies. This is the responsibility of the controller. `DeletionGuardDependency` provides the convenience method `GetDependencies` which will return all managed dependencies, ensuring the required finalizer has been added.

Finalizers should be added at the last possible moment, namely immediately before their OpenStack resource is about to be referenced in an OpenStack create/update operation. The reason for this is, again, usability. If a user is provisioning a large number of resources and a failure happens somewhere, we want them to be able to delete, e.g. a failed Network. We don't want them to have to manually remove the Subnet finalizer from the Network for a Subnet which was never created because of the failure to provision its Network. Therefore we should not add a finalizer to Network until immediately before calling Create on the Subnet.