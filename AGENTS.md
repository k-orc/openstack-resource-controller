# OpenStack Resource Controller (ORC) - Development Guide

This document provides instructions for AI agents to develop controllers in the ORC project.

## Project Overview

ORC is a Kubernetes operator that manages OpenStack resources declaratively. Each OpenStack resource (Flavor, Server, Network, etc.) has a corresponding Kubernetes Custom Resource and controller.

**Key Principle**: ORC objects only reference other ORC objects, never OpenStack resources directly. OpenStack resource IDs appear only in status fields.

## Project Structure

```
openstack-resource-controller/
├── api/v1alpha1/              # CRD type definitions (*_types.go)
├── internal/
│   ├── controllers/           # Controller implementations
│   │   └── <resource>/        # Each controller in its own package
│   │       ├── controller.go  # Setup, dependencies, SetupWithManager
│   │       ├── actuator.go    # OpenStack CRUD operations
│   │       ├── status.go      # Status writer implementation
│   │       ├── zz_generated.*.go  # Generated code (DO NOT EDIT)
│   │       └── tests/         # KUTTL E2E tests
│   ├── logging/               # Log level constants
│   ├── osclients/             # OpenStack API client wrappers
│   ├── scope/                 # Cloud credentials & client factory
│   └── util/
│       ├── applyconfigs/      # SSA apply config helpers
│       ├── credentials/       # Credential watch & dependency setup
│       ├── dependency/        # Dependency framework
│       ├── errors/            # Error classification (Terminal, IsRetryable)
│       ├── finalizers/        # Finalizer helpers
│       ├── result/            # Result helpers
│       ├── strings/           # Finalizer/field-owner name generation
│       └── tags/              # Tag reconciliation utilities
├── cmd/
│   ├── manager/               # Main entry point
│   ├── models-schema/         # OpenAPI schema generation
│   ├── resource-generator/    # Code generation
│   └── scaffold-controller/   # New controller scaffolding
└── website/docs/development/  # Detailed documentation
```

## Architecture

### Generic Reconciler Framework

All controllers use a generic reconciler that handles the reconciliation loop. Controllers implement interfaces:

- **CreateResourceActuator**: Create and import operations
- **DeleteResourceActuator**: Delete operations
- **ReconcileResourceActuator**: Post-creation updates (optional)
- **ResourceStatusWriter**: Status and condition management

### Key Interfaces

Controllers implement these methods (see `internal/controllers/servergroup/` for a simple example):

```go
// Required by all actuators
GetResourceID(osResource) string
GetOSResourceByID(ctx, id) (*osResource, ReconcileStatus)
ListOSResourcesForAdoption(ctx, obj) (iterator, bool)

// For creation/import
ListOSResourcesForImport(ctx, obj, filter) (iterator, ReconcileStatus)
CreateResource(ctx, orcObject) (*osResource, ReconcileStatus)

// For deletion
DeleteResource(ctx, orcObject, osResource) ReconcileStatus

// Optional - for updates after creation
GetResourceReconcilers(ctx, obj, osResource, controller) ([]ResourceReconciler, ReconcileStatus)
```

### Two Critical Conditions

Every ORC object has these conditions:

1. **Progressing**
   - `True`: Spec doesn't match status; controller expects more reconciles
   - `False`: Either available OR terminal error (no more reconciles until spec changes)

2. **Available**
   - `True`: Resource is ready for use
   - Determined by `ResourceStatusWriter.ResourceAvailableStatus()`

### ReconcileStatus Pattern

Methods return `ReconcileStatus` instead of `error`:

`ReconcileStatus` is a type alias for a pointer (`type ReconcileStatus = *reconcileStatus`). `nil` is a valid value meaning "success, no reschedule", and all methods are safe to call on a nil receiver.

```go
nil                                          // Success, no reschedule
progress.WrapError(err)                      // Wrap error for handling
reconcileStatus.WithRequeue(5*time.Second)   // Schedule reconcile after delay
reconcileStatus.WithProgressMessage("...")   // Add progress message
progress.NeedsRefresh()                      // Immediate re-reconcile to refresh status after mutation
progress.WaitingOnOpenStack(progress.WaitingOnReady, 15*time.Second) // Poll for OpenStack state change
progress.WaitingOnObject("Network", name, progress.WaitingOnCreation) // Wait for a k8s object
reconcileStatus.WithReconcileStatus(other)   // Merge two ReconcileStatuses
```

### Error Classification

- **Transient errors** (5xx, API unavailable): Default handling with exponential backoff
- **Non-recoverable errors** (409 Conflict, non-HTTP gophercloud errors): Wrap with `orcerrors.Terminal()` - no retry

```go
// Non-recoverable error example
if err != nil {
    if !orcerrors.IsRetryable(err) {
        err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration,
            "invalid configuration: "+err.Error(), err)
    }
    return nil, progress.WrapError(err)
}
```

## Dependencies

Dependencies are core to ORC - they ensure resources are created in order.

### Types of Dependencies

1. **Normal Dependency**: Wait for object to exist and be available
2. **Deletion Guard Dependency**: Normal + prevents deletion of dependency while in use

### Declaring Dependencies (in controller.go)

```go
var projectDependency = dependency.NewDeletionGuardDependency[*orcv1alpha1.SecurityGroupList, *orcv1alpha1.Project](
    "spec.resource.projectRef",           // Field path for indexing
    func(sg *orcv1alpha1.SecurityGroup) []string {
        if sg.Spec.Resource != nil && sg.Spec.Resource.ProjectRef != nil {
            return []string{string(*sg.Spec.Resource.ProjectRef)}
        }
        return nil
    },
    finalizer, externalObjectFieldOwner,
)
```

### Using Dependencies (in actuator.go)

```go
project, reconcileStatus := projectDependency.GetDependency(
    ctx, actuator.k8sClient, orcObject,
    func(dep *orcv1alpha1.Project) bool {
        return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
    },
)
if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
    return nil, reconcileStatus
}
// project is now guaranteed available
projectID := ptr.Deref(project.Status.ID, "")
```

### Lightweight Dependency Lookup (FetchDependency)

For one-off lookups that don't need finalizers (e.g., resolving refs in `ListOSResourcesForAdoption` or import filters), use `dependency.FetchDependency` instead of a declared dependency:

```go
import "github.com/k-orc/openstack-resource-controller/v2/internal/util/dependency"

project, rs := dependency.FetchDependency(
    ctx, actuator.k8sClient, obj.Namespace, filter.ProjectRef, "Project",
    func(dep *orcv1alpha1.Project) bool {
        return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
    },
)
reconcileStatus = reconcileStatus.WithReconcileStatus(rs)
```

### Credentials Dependency (generated)

Every controller has a `credentialsDependency` auto-generated in `zz_generated.controller.go`. It is a `DeletionGuardDependency` on `corev1.Secret` that ensures the cloud credentials secret exists and carries the controller's finalizer. It is checked in `newActuator()` before creating an OpenStack client:

```go
_, reconcileStatus := credentialsDependency.GetDependencies(
    ctx, controller.GetK8sClient(), orcObject,
    func(*corev1.Secret) bool { return true },
)
if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
    return myActuator{}, reconcileStatus
}
```

The credential watch is registered in `SetupWithManager` via `credentials.AddCredentialsWatch()`.

## Common Patterns

### Resource Name Helper (generated)

`getResourceName` is auto-generated in `zz_generated.adapter.go` — do not write it manually. It returns `spec.resource.name` if set, otherwise falls back to the ORC object's Kubernetes name:

```go
// In zz_generated.adapter.go (DO NOT EDIT)
func getResourceName(orcObject orcObjectPT) string {
    if orcObject.Spec.Resource.Name != nil {
        return string(*orcObject.Spec.Resource.Name)
    }
    return orcObject.Name
}
```

### Type Aliases (top of actuator.go)

```go
type (
    osResourceT = flavors.Flavor
    createResourceActuator = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
    deleteResourceActuator = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
    helperFactory = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)
```

### Interface Assertions

```go
var _ createResourceActuator = flavorActuator{}
var _ deleteResourceActuator = flavorActuator{}
```

### Actuator Factory (newActuator)

Every controller defines a `newActuator()` function that resolves credentials, creates the OpenStack client scope, and returns the actuator. This is called by the `helperFactory` methods `NewCreateActuator` and `NewDeleteActuator`:

```go
func newActuator(ctx context.Context, orcObject *orcv1alpha1.Flavor, controller generic.ResourceController) (flavorActuator, progress.ReconcileStatus) {
    log := ctrl.LoggerFrom(ctx)

    // Ensure credential secrets exist and have our finalizer
    _, reconcileStatus := credentialsDependency.GetDependencies(
        ctx, controller.GetK8sClient(), orcObject,
        func(*corev1.Secret) bool { return true },
    )
    if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
        return flavorActuator{}, reconcileStatus
    }

    clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(
        ctx, controller.GetK8sClient(), log, orcObject,
    )
    if err != nil {
        return flavorActuator{}, progress.WrapError(err)
    }
    osClient, err := clientScope.NewComputeClient()  // or NewNetworkClient, etc.
    if err != nil {
        return flavorActuator{}, progress.WrapError(err)
    }

    return flavorActuator{osClient: osClient}, nil
}
```

### Tag Reconciliation (Neutron resources)

Neutron resources use a separate tags API instead of the resource's Update API. The `internal/util/tags` package provides a reusable reconciler:

```go
import "github.com/k-orc/openstack-resource-controller/v2/internal/util/tags"

func (actuator myActuator) GetResourceReconcilers(...) ([]resourceReconciler, progress.ReconcileStatus) {
    return []resourceReconciler{
        tags.ReconcileTags[orcObjectPT, osResourceT](
            orcObject.Spec.Resource.Tags,
            osResource.Tags,
            tags.NewNeutronTagReplacer(actuator.osClient, "security-groups", osResource.ID),
        ),
        actuator.updateRules,
    }, nil
}
```

`ReconcileTags` computes the diff between desired and observed tags and replaces them atomically. For resources whose tags are set via the standard Update API (e.g., block storage), use a `handleTagsUpdate()` helper in `updateResource` instead.

### Pointer Handling

```go
import "k8s.io/utils/ptr"

ptr.Deref(optionalPtr, defaultValue)  // Dereference with default
ptr.To(value)                          // Create pointer
```

## API Types Structure

### ResourceSpec (creation parameters)

```go
// Most resources have a mix of immutable and mutable fields.
// Immutability is typically applied per-field, not on the whole struct.
type ServerResourceSpec struct {
    // +optional
    Name *OpenStackName `json:"name,omitempty"`

    // +required
    // +kubebuilder:validation:XValidation:rule="self == oldSelf",message="imageRef is immutable"
    ImageRef KubernetesNameRef `json:"imageRef,omitempty"`

    // tags is mutable (no immutability validation)
    // +optional
    Tags []ServerTag `json:"tags,omitempty"`
}

// Some resources are fully immutable (rare - e.g., ServerGroup)
// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="ServerGroupResourceSpec is immutable"
type ServerGroupResourceSpec struct {
    // ...
}
```

### Filter (import parameters)

```go
// +kubebuilder:validation:MinProperties:=1
type FlavorFilter struct {
    Name *OpenStackName `json:"name,omitempty"`
    RAM  *int32         `json:"ram,omitempty"`
}
```

### ResourceStatus (observed state)

```go
type FlavorResourceStatus struct {
    Name string `json:"name,omitempty"`
    RAM  *int32 `json:"ram,omitempty"`
}
```

## Logging Levels

```go
import "github.com/k-orc/openstack-resource-controller/v2/internal/logging"

log.V(logging.Status).Info("...")   // Always shown: startup, shutdown
log.V(logging.Info).Info("...")     // Default: creation/deletion, reconcile complete
log.V(logging.Verbose).Info("...")  // Admin: fires every reconcile
log.V(logging.Debug).Info("...")    // Development: detailed debugging
```

## Key Make Targets

```bash
make generate      # Generate all code (run after API type changes)
make build         # Build manager binary
make lint          # Run linters
make test          # Run unit tests
make test-e2e      # Run KUTTL E2E tests (requires E2E_OSCLOUDS)
make fmt           # Format code
```

## Reconciler Naming Conventions

`GetResourceReconcilers` returns a list of reconciler functions. There are two types:

1. **`updateResource`**: Handles general mutable field updates via the resource's Update API. Uses `handleXXXUpdate()` helpers to build an `UpdateOpts` struct, then makes a single API call. Returns a terminal error when `spec.resource` is nil. Examples: securitygroup, volumetype, trunk, router.

2. **Single-concern reconcilers**: Handle a specific aspect of the resource using a separate API (not the resource's Update API). Named with a descriptive verb+noun (e.g., `reconcileExtraSpecs`, `reconcileSubports`, `reconcilePassword`, `updateRules`). Return `nil` (not a terminal error) when `spec.resource` is nil. May make multiple API calls within a single reconciler.

```go
// updateResource pattern - general mutable fields via Update API
func (actuator myActuator) updateResource(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
    resource := obj.Spec.Resource
    if resource == nil {
        // Terminal error: updateResource is only registered for managed resources
        return progress.WrapError(
            orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Update requested, but spec.resource is not set"))
    }
    // ... build UpdateOpts, make single Update API call
}

// Single-concern reconciler pattern - separate API
func (actuator myActuator) reconcileExtraSpecs(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
    resource := obj.Spec.Resource
    if resource == nil {
        return nil  // Not a terminal error
    }
    // ... compute diff, make API calls (creates, deletes, etc.)
}
```

Both types are registered in `GetResourceReconcilers`:
```go
func (actuator myActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller generic.ResourceController) ([]resourceReconciler, progress.ReconcileStatus) {
    return []resourceReconciler{
        actuator.updateResource,       // general field updates
        actuator.reconcileExtraSpecs,  // single-concern reconciler
    }, nil
}
```

## Reference Controllers

- **Simple**: `internal/controllers/servergroup/` - No dependencies, fully immutable
- **Single-concern reconciler**: `internal/controllers/flavor/` - No dependencies, immutable except extra specs (`reconcileExtraSpecs`)
- **With dependencies**: `internal/controllers/securitygroup/` - Project dependency, rules reconciliation
- **Multiple reconcilers**: `internal/controllers/trunk/` - `updateResource` + `reconcileSubports` + tags
- **Complex**: `internal/controllers/server/` - Multiple dependencies, many reconcilers

## Documentation

Detailed documentation in `website/docs/development/`:
- `scaffolding.md` - Creating new controllers
- `controller-implementation.md` - Progressing condition, ReconcileStatus
- `interfaces.md` - Detailed interface descriptions
- `coding-standards.md` - Code style and conventions
- `writing-tests.md` - Testing patterns
