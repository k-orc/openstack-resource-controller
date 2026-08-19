# Controller initialisation

Controllers are initialised and added to the controller manager from `cmd/manager/main.go`. Each controller must implement the [`Controller` interface](godoc/generic-interfaces.md#Controller):

```golang
type Controller interface {
	SetupWithManager(context.Context, ctrl.Manager, controller.Options) error
	GetName() string
}
```

This interface is typically defined in `controller.go` in the controller's package directory.

## Generated helper variables

The scaffolding automatically generated some helper code in `zz_generated.controller.go`. This code includes the following package-scoped variables:

* `finalizer`: the string used to identify this controller when adding a finalizer to another controller's objects
* `externalObjectFieldOwner`: the string used to identify this controller in server-side apply transactions when adding fields to another controller's objects
* `credentialsDependency`: a deletion dependency on the credential secret

## Dependencies

Dependencies are defined as package-scoped variables in `controller.go`. Use
`NewDeletionGuardDependency` for resources that should not be deleted while
referenced (the common case), or `NewDependency` for import filter lookups
that don't need deletion guards:

```go
var (
    // Deletion guard: prevents the Project from being deleted while
    // this SecurityGroup references it.
    projectDependency = dependency.NewDeletionGuardDependency[
        *orcv1alpha1.SecurityGroupList, *orcv1alpha1.Project,
    ](
        "spec.resource.projectRef",
        func(sg *orcv1alpha1.SecurityGroup) []string {
            resource := sg.Spec.Resource
            if resource == nil || resource.ProjectRef == nil {
                return nil
            }
            return []string{string(*resource.ProjectRef)}
        },
        finalizer, externalObjectFieldOwner,
    )

    // Import dependency: no deletion guard, just an index for
    // import filter lookups.
    projectImportDependency = dependency.NewDependency[
        *orcv1alpha1.SecurityGroupList, *orcv1alpha1.Project,
    ](
        "spec.import.filter.projectRef",
        func(sg *orcv1alpha1.SecurityGroup) []string {
            imp := sg.Spec.Import
            if imp == nil || imp.Filter == nil || imp.Filter.ProjectRef == nil {
                return nil
            }
            return []string{string(*imp.Filter.ProjectRef)}
        },
    )
)
```

See the [securitygroup controller](https://github.com/k-orc/openstack-resource-controller/blob/main/internal/controllers/securitygroup/controller.go)
for a complete example with dependencies.

## Controller name

`GetName` must return the name of the controller, which must be:

* unique among all controllers
* contain only lower case letters and '-'

This name is used variously anywhere the controller must be identified, including:

* all structured logs
* the name of any associated deletion guard controllers

## SetupWithManager

This method is responsible for:

* Initialising the controller's reconciler data structure
* Registering dependencies and their field indexers
* Adding watches so the controller reconciles when dependencies change
* Adding the reconciler to the manager

Here is a minimal example (no dependencies beyond credentials):

```go
func (c *myReconcilerConstructor) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
    log := ctrl.LoggerFrom(ctx)

    builder := ctrl.NewControllerManagedBy(mgr).
        WithOptions(options).
        For(&orcv1alpha1.MyResource{})

    if err := errors.Join(
        credentialsDependency.AddToManager(ctx, mgr),
        credentials.AddCredentialsWatch(log, mgr.GetClient(), builder, credentialsDependency),
    ); err != nil {
        return err
    }

    r := reconciler.NewController(controllerName, mgr.GetClient(), c.scopeFactory, myHelperFactory{}, myStatusWriter{}, c.defaultResyncPeriod)
    return builder.Complete(&r)
}
```

When adding a dependency, you need to:

1. Call `AddToManager` to register the field indexer (and deletion guard if applicable)
2. Create a watch event handler with `WatchEventHandler`
3. Add a `Watches` call on the builder so changes to the dependency trigger reconciliation

See the [securitygroup controller](https://github.com/k-orc/openstack-resource-controller/blob/main/internal/controllers/securitygroup/controller.go)
for a complete example with dependency watches.
