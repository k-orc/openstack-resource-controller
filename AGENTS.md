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
│   ├── osclients/             # OpenStack API client wrappers
│   ├── scope/                 # Cloud credentials & client factory
│   └── util/                  # Utilities (errors, dependency, tags)
├── cmd/
│   ├── manager/               # Main entry point
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

Controllers implement these methods (see `internal/controllers/flavor/` for a simple example):

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
GetResourceReconcilers(ctx, obj, osResource) ([]ResourceReconciler, error)
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

```go
nil                                    // Success, no reschedule
progress.WrapError(err)                // Wrap error for handling
reconcileStatus.WithRequeue(5*time.Second)  // Schedule reconcile after delay
reconcileStatus.WithProgressMessage("waiting...") // Add progress message
```

### Error Classification

- **Transient errors** (5xx, API unavailable): Default handling with exponential backoff
- **Terminal errors** (400, invalid config): Wrap with `orcerrors.Terminal()` - no retry

```go
// Terminal error example
if !orcerrors.IsRetryable(err) {
    err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration,
        "invalid configuration: "+err.Error(), err)
}
return nil, progress.WrapError(err)
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

## Common Patterns

### Resource Name Helper

```go
func getResourceName(orcObject *orcv1alpha1.Flavor) string {
    if orcObject.Spec.Resource.Name != nil {
        return *orcObject.Spec.Resource.Name
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

// Some resources are fully immutable (rare - e.g., Flavor, ServerGroup)
// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="FlavorResourceSpec is immutable"
type FlavorResourceSpec struct {
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

## Reference Controllers

- **Simple**: `internal/controllers/flavor/` - No dependencies, immutable
- **With dependencies**: `internal/controllers/securitygroup/` - Project dependency, rules reconciliation
- **Complex**: `internal/controllers/server/` - Multiple dependencies, reconcilers

## Documentation

Detailed documentation in `website/docs/development/`:
- `scaffolding.md` - Creating new controllers
- `controller-implementation.md` - Progressing condition, ReconcileStatus
- `interfaces.md` - Detailed interface descriptions
- `coding-standards.md` - Code style and conventions
- `writing-tests.md` - Testing patterns
