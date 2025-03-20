# Controller initialisation

Controllers are initialised and added to the controller manager from `cmd/manager/main.go`. Each controller must implement the [`Controller` interface](../godoc/generic-interfaces/#Controller):

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

Dependencies are idiomatically defined as package-scoped variables in `controller.go`.

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
* Adding dependencies, deletion guards, and other watches to the reconciler
* Adding the reconciler to manager