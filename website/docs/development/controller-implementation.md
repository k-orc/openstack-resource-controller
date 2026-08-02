# Controller Implementation

## The Progressing condition

All ORC objects publish a Progressing condition. This condition has a strict definition:

`True` means that after the previous reconcile:

* The object status did not yet reflect the desired spec.
* The controller expects the object to be reconciled again.

`False` means:

* The object will not be reconciled again unless the spec changes.

Strict adherence to this definition enables some important use cases:

* Controllers can filter reconciles of objects which are up to date. This is especially useful after a controller restart when a reconcile would otherwise be triggered for every object present. This can present a 'thundering herd' problem, particularly when they result in many OpenStack API calls.
* A consumer of the API can easily determine when to stop waiting on an object to be reconciled. If Progressing is False then either the resource is available, or any currently reported error is not one the controller can resolve without intervention.

In particular, the generic controller's reconcile loop filters objects with an up-to-date Progressing status of False. This means that if your controller exits a reconcile without correctly setting the Progressing condition it may result in the object never successfully reconciling.

## ReconcileStatus

When a reconcile completes, we should have enough context to know:

* If we need to be reconciled again
* If so, whether we expect it to be:
    * Event triggered: on creation/update of another kubernetes object
    * Polling: we schedule another reconcile after a particular amount of time, for example because we're waiting on OpenStack
    * Immediate: we schedule another reconcile immediately after this one finishes, for example to force a refresh of status
    * With exponential backoff due to an error
* If we should never reconcile this spec again because it is invalid

This is handled internally by `ReconcileStatus`. Most methods in the actuator interface return a `ReconcileStatus` in place of an error.

As noted above, failure to return a correct `ReconcileStatus` from an actuator method will likely result in the reconciliation of your object hanging.

!!! note

    `nil` is a valid `ReconcileStatus`, and is the preferred representation of an empty `ReconcileStatus`. It is permitted to call methods on a `nil` `ReconcileStatus`.

!!! warning

    `ReconcileStatus` methods which modify the `ReconcileStatus` do so by returning a `ReconcileStatus` containing the modification. Similar to how `append()` works, this modification may or may not be in-place. Consequently, the return value of `ReconcileStatus` **MUST ALWAYS** be used. e.g.:

    ```golang
    reconcileStatus = reconcileStatus.WithError(err)
    ```

Refer to [the `ReconcileStatus documentation`](godoc/reconcile-status.md) for details of available methods.

## Transient and Terminal errors

Controllers will perform many operations which can fail. We split these errors into transient and terminal errors.

A **transient error** is one which may eventually resolve itself without the object spec being updated. Example transient errors:

* Failure to contact an API endpoint
* An API call returned a 5xx (internal error)
* A kubernetes read or write operation failed for any reason

A **terminal error** is one which we don't expect to ever succeed unless the object spec is updated. Example terminal errors:

* The spec is invalid
* OpenStack returned a non-retryable status, e.g. invalid request when creating a resource

By default, all errors should be treated as transient. No special handling is required for transient errors. If your method returns an error it will eventually be passed to the status writer. A transient error results in a Progressing status of True. The condition's reason will be set to TransientError, and the error message itself will be reported to the user via the condition's message. The controller will enter a default exponential backoff loop, so the object will continue to be reconciled indefinitely until the error no longer occurs.

!!! note

    We currently report *all* error messages to the user. At some point we may restrict this to only OpenStack errors to avoid potentially leaking internal configuration details.

When you are confident that an error will never be resolved we can instead return a terminal error. As well as not wasting resources by continuing to attempt an operation which will never succeed, this will clearly communicate to any API user waiting for the object to be reconciled that they can stop waiting.

To return a terminal error, wrap the error in an `orcerrors.TerminalError`. The status writer will observe this and set Progressing to False. Additionally, the error will not be returned by the reconcile loop, so we will not enter the error handling exponential backoff loop.

## Using dependencies in the actuator

See [Controller Initialisation](controller-init.md#dependencies) for how to
declare and register dependencies. This section covers how to use them in the
actuator.

### Resolve dependencies late

Resolve dependencies as late as possible, as close to the point of use as
possible. This avoids injecting a dependency requirement where it is not
strictly required.

For example, Subnet depends on Network. It requires the Network:

* For creation, because a Subnet cannot be created without a Network
* For import by filter, because the filter has an implicit constraint to the Subnet's Network

However, a Network is not required for import by ID, or once `status.ID` has
been set. Avoiding the dependency in those cases gives users greater freedom to
fix a failed deployment. This is especially important for deletion: if a user
has force-deleted a Network, they should not have to recreate it before
deleting a Subnet whose `status.ID` is already set.

### Lightweight lookups with FetchDependency

For one-off lookups that don't need finalizers (e.g. resolving refs in
`ListOSResourcesForAdoption` or import filters), use `dependency.FetchDependency`
instead of a declared dependency:

```go
project, rs := dependency.FetchDependency(
    ctx, actuator.k8sClient, obj.Namespace, filter.ProjectRef, "Project",
    func(dep *orcv1alpha1.Project) bool {
        return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
    },
)
reconcileStatus = reconcileStatus.WithReconcileStatus(rs)
```

Unlike `GetDependency`, this does not add a finalizer to the referenced object,
so it should not be used when the dependency must be prevented from deletion.

### When to add finalizers

Finalizers (via `GetDependency`/`GetDependencies`) should be added at the last
possible moment, immediately before the OpenStack resource is about to be
created or updated. If a user is provisioning many resources and a failure
happens, we want them to be able to delete a failed resource without having to
manually remove finalizers from dependencies that were never actually used.
