# Coding Standards

This page documents the coding standards and patterns used throughout ORC.

## Package structure

Each controller lives in its own package under `internal/controllers/<resource>/`:

| File | Purpose |
|------|---------|
| `controller.go` | Controller setup, name, and `SetupWithManager` |
| `actuator.go` | OpenStack CRUD operations |
| `status.go` | Status writer implementation |
| `zz_generated.*.go` | Generated code (do not edit) |

Package-scoped variables for the controller name, finalizers, and dependencies are defined in `controller.go` or `zz_generated.controller.go`.

## Type aliases

Use type aliases at the top of `actuator.go` to improve readability:

```go
// OpenStack resource types
type (
    osResourceT = flavors.Flavor

    createResourceActuator = generic.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
    deleteResourceActuator = generic.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
    helperFactory          = generic.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)
```

## Interface compliance

Use compile-time interface assertions to verify actuators implement required interfaces:

```go
var _ createResourceActuator = flavorActuator{}
var _ deleteResourceActuator = flavorActuator{}
```

## Naming conventions

### Controller names

Controller names must be:

- Unique among all controllers
- Contain only lowercase letters and hyphens

```go
const controllerName = "flavor"
```

### OpenStack resource names

By default, use the ORC object name as the OpenStack resource name. Provide a helper function:

```go
func getResourceName(orcObject *orcv1alpha1.Flavor) string {
    if orcObject.Spec.Resource.Name != nil {
        return *orcObject.Spec.Resource.Name
    }
    return orcObject.Name
}
```

## Error handling

For detailed information on error handling patterns, including transient vs terminal errors and `ReconcileStatus`, see [Controller Implementation](controller-implementation.md#transient-and-terminal-errors).

## Logging

ORC uses four logging levels defined in `internal/logging/levels.go`:

### Status level

Always shown. Reserved for operational logs about the service itself:

- Startup and shutdown messages
- Runtime conditions indicating service state

```go
log.V(logging.Status).Info("Failed to reach API server")
```

### Info level

Default for most deployments. Log principal actions:

- Resource creation and deletion
- Reconcile completion (Progressing=False)

```go
log.V(logging.Info).Info("OpenStack resource created", "id", resource.ID)
```

### Verbose level

Additional context for administrators. Logs on every reconcile:

```go
log.V(logging.Verbose).Info("web-download is not supported", "reason", reason)
```

### Debug level

Very verbose, for development and debugging:

```go
log.V(logging.Debug).Info("Got resource", "resource", resource)
```

## Import ordering

Organize imports in three groups, separated by blank lines:

1. Standard library
2. Third-party packages
3. Internal packages

```go
import (
    "context"
    "errors"

    "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
    ctrl "sigs.k8s.io/controller-runtime"

    orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
    "github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
)
```

## RBAC markers

Define RBAC requirements using kubebuilder markers in `controller.go`:

```go
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=flavors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=flavors/status,verbs=get;update;patch
```

## Pointer handling

Use `k8s.io/utils/ptr` for pointer operations:

```go
import "k8s.io/utils/ptr"

// Dereference with default
value := ptr.Deref(optionalField, defaultValue)

// Create pointer to value
ptr.To(someValue)
```

## Testing

### Test file naming

- Unit tests: `*_test.go` alongside source files
- API validation tests: `test/apivalidations/<resource>_test.go`

### Mocking

Use generated mocks from `internal/osclients/mock/` for unit tests. Mocks are generated via `//go:generate` directives.

## Generated code

Files matching these patterns are generated and should not be edited:

- `zz_generated.*.go`
- Files in `pkg/clients/`

Always run `make generate` after modifying API types.
