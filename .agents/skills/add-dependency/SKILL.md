---
name: add-dependency
description: Add a dependency on another ORC resource to a controller. Use when a resource needs to reference or wait for another resource (e.g., Subnet depends on Network).
disable-model-invocation: true
---

# Add Dependency to Controller

Guide for adding a dependency on another ORC resource.

**Reference**: See `website/docs/development/controller-implementation.md` for detailed rationale on dependency patterns.

## When to Use Dependencies

Use a dependency when your controller needs to:
- Wait for another resource to be available before creating
- Reference another resource's OpenStack ID
- Optionally prevent deletion of a resource that's still in use (deletion guard)

## Key Principles

See also "Dependency Timing" in @.agents/skills/new-controller/patterns.md

### 1. Resolve Dependencies Late

Resolve dependencies as late as possible, as close to the point of use as possible. This reduces coupling and gives users flexibility when fixing failed deployments.

**Examples:**
- Subnet depends on Network for creation, but NOT for import by ID or after `status.ID` is set
- Don't require recreating a deleted Network just to delete a Subnet
- Add finalizers only immediately before the OpenStack create/update call

### 2. Choose the Right Dependency Type

| Type | Use When | Example |
|------|----------|---------|
| **Normal** (`NewDependency`) | Dependency is optional OR deletion is allowed by OpenStack | Import filter refs, Flavor ref |
| **Deletion Guard** (`NewDeletionGuardDependency`) | Deletion would fail or corrupt your resource | Subnet→Network, Port→Subnet |

### 3. Use Descriptive Names

When multiple dependencies of the same type exist, use descriptive prefixes:
- `vipSubnetDependency` not `subnetDependency` (when there could be other subnet refs)
- `sourcePortDependency` vs `destinationPortDependency`
- `memberNetworkDependency` vs `externalNetworkDependency`

## Dependency Types

### Normal Dependency
Wait for resource but don't prevent deletion:
```go
dependency.NewDependency[*orcv1alpha1.MyResourceList, *orcv1alpha1.DepResource](...)
```

### Deletion Guard Dependency
Wait for resource AND prevent its deletion:
```go
dependency.NewDeletionGuardDependency[*orcv1alpha1.MyResourceList, *orcv1alpha1.DepResource](...)
```

**Use Deletion Guard when**: Deleting the dependency would cause your resource to fail or become invalid (e.g., Subnet depends on Network, Port depends on SecurityGroup).

## Step 1: Add Reference Field to API

In `api/v1alpha1/<kind>_types.go`, add the reference field:

```go
type MyResourceSpec struct {
    // ...

    // projectRef is a reference to a Project.
    // +kubebuilder:validation:XValidation:rule="self == oldSelf",message="projectRef is immutable"
    // +optional
    ProjectRef *KubernetesNameRef `json:"projectRef,omitempty"`
}
```

For import filters, add to the Filter struct as well:
```go
type MyResourceFilter struct {
    // +optional
    ProjectRef *KubernetesNameRef `json:"projectRef,omitempty"`
}
```

## Step 2: Declare Dependency

In `internal/controllers/<kind>/controller.go`, add package-scoped variable:

```go
var (
    projectDependency = dependency.NewDeletionGuardDependency[*orcv1alpha1.MyResourceList, *orcv1alpha1.Project](
        "spec.resource.projectRef",  // Field path for indexing
        func(obj *orcv1alpha1.MyResource) []string {
            resource := obj.Spec.Resource
            if resource == nil || resource.ProjectRef == nil {
                return nil
            }
            return []string{string(*resource.ProjectRef)}
        },
        finalizer, externalObjectFieldOwner,
    )

    // For import filter dependencies (no deletion guard needed)
    projectImportDependency = dependency.NewDependency[*orcv1alpha1.MyResourceList, *orcv1alpha1.Project](
        "spec.import.filter.projectRef",
        func(obj *orcv1alpha1.MyResource) []string {
            imp := obj.Spec.Import
            if imp == nil || imp.Filter == nil || imp.Filter.ProjectRef == nil {
                return nil
            }
            return []string{string(*imp.Filter.ProjectRef)}
        },
    )
)
```

## Step 3: Setup Watches

In `SetupWithManager()` in `controller.go`:

```go
func (c myReconcilerConstructor) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
    log := ctrl.LoggerFrom(ctx)
    k8sClient := mgr.GetClient()

    // Create watch handlers
    projectWatchHandler, err := projectDependency.WatchEventHandler(log, k8sClient)
    if err != nil {
        return err
    }

    builder := ctrl.NewControllerManagedBy(mgr).
        WithOptions(options).
        For(&orcv1alpha1.MyResource{}).
        // Watch the dependency
        Watches(&orcv1alpha1.Project{}, projectWatchHandler,
            builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Project{})),
        )

    // Register dependencies with manager
    if err := errors.Join(
        projectDependency.AddToManager(ctx, mgr),
        credentialsDependency.AddToManager(ctx, mgr),
        credentials.AddCredentialsWatch(log, k8sClient, builder, credentialsDependency),
    ); err != nil {
        return err
    }

    r := reconciler.NewController(controllerName, k8sClient, c.scopeFactory, helperFactory{}, statusWriter{})
    return builder.Complete(&r)
}
```

## Step 4: Use Dependency in Actuator

In `actuator.go`, resolve the dependency before using it:

```go
func (actuator myActuator) CreateResource(ctx context.Context, obj *orcv1alpha1.MyResource) (*osResourceT, progress.ReconcileStatus) {
    resource := obj.Spec.Resource

    var projectID string
    if resource.ProjectRef != nil {
        project, reconcileStatus := projectDependency.GetDependency(
            ctx, actuator.k8sClient, obj,
            func(dep *orcv1alpha1.Project) bool {
                return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
            },
        )
        if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
            return nil, reconcileStatus
        }
        projectID = ptr.Deref(project.Status.ID, "")
    }

    createOpts := myresource.CreateOpts{
        ProjectID: projectID,
        // ...
    }
    // ...
}
```

For import filter dependencies:
```go
func (actuator myActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
    var reconcileStatus progress.ReconcileStatus

    project, rs := dependency.FetchDependency(
        ctx, actuator.k8sClient, obj.Namespace, filter.ProjectRef, "Project",
        func(dep *orcv1alpha1.Project) bool {
            return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
        },
    )
    reconcileStatus = reconcileStatus.WithReconcileStatus(rs)

    if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
        return nil, reconcileStatus
    }

    listOpts := myresource.ListOpts{
        ProjectID: ptr.Deref(project.Status.ID, ""),
    }
    return actuator.osClient.ListMyResources(ctx, listOpts), nil
}
```

## Step 5: Add k8sClient to Actuator

If not already present, add `k8sClient` to the actuator struct:

```go
type myActuator struct {
    osClient  osclients.MyResourceClient
    k8sClient client.Client  // Add this
}
```

Update `newActuator()`:
```go
func newActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (myActuator, progress.ReconcileStatus) {
    k8sClient := controller.GetK8sClient()  // Add this
    // ...
    return myActuator{
        osClient:  osClient,
        k8sClient: k8sClient,  // Add this
    }, nil
}
```

## Step 6: Add Tests

Create dependency tests in `internal/controllers/<kind>/tests/<kind>-dependency/`:
- Test that resource waits for dependency
- Test that dependency deletion is blocked (if using DeletionGuard)

Follow @.agents/skills/testing/SKILL.md for running unit tests, linting, and E2E tests.

## Checklist

- [ ] Reference field added to API types (with immutability validation)
- [ ] Dependency declared in controller.go
- [ ] Watch configured in SetupWithManager
- [ ] Dependency registered with manager (AddToManager)
- [ ] Dependency resolved in actuator before use
- [ ] k8sClient added to actuator struct
- [ ] `make generate` runs cleanly
- [ ] `make lint` passes
- [ ] Dependency tests added
