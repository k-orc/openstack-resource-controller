# Dependency Resolver Builder Pattern

## Problem Statement

Controllers frequently need to resolve `*KubernetesNameRef` fields to their OpenStack IDs. This results in repetitive boilerplate code across controllers:

```go
var networkID string
if filter.NetworkRef != nil {
    networkKey := client.ObjectKey{Name: string(*filter.NetworkRef), Namespace: obj.Namespace}
    if err := actuator.k8sClient.Get(ctx, networkKey, network); err != nil {
        if apierrors.IsNotFound(err) {
            reconcileStatus = reconcileStatus.WithReconcileStatus(
                progress.WaitingOnObject("Network", networkKey.Name, progress.WaitingOnCreation))
        } else {
            reconcileStatus = reconcileStatus.WithReconcileStatus(
                progress.WrapError(fmt.Errorf("fetching network %s: %w", networkKey.Name, err)))
        }
    } else {
        if !orcv1alpha1.IsAvailable(network) || network.Status.ID == nil {
            reconcileStatus = reconcileStatus.WithReconcileStatus(
                progress.WaitingOnObject("Network", networkKey.Name, progress.WaitingOnReady))
        } else {
            networkID = *network.Status.ID
        }
    }
}
// Repeat for Subnet, Project, Port, etc...
```

This pattern is repeated across:
- `ListOSResourcesForImport` (resolving filter refs)
- `CreateResource` (resolving spec refs)
- Various reconcilers

## Proposed Solution

A builder-pattern resolver that:
1. Chains multiple dependency resolutions
2. Accumulates errors and wait states
3. Returns resolved IDs in a type-safe way
4. Provides clear, readable code

## API Design

### Core Types

```go
package dependency

// Resolver accumulates dependency resolutions and their statuses
type Resolver struct {
    ctx             context.Context
    k8sClient       client.Client
    namespace       string
    reconcileStatus progress.ReconcileStatus
    resolved        map[string]string  // kind -> ID
}

// NewResolver creates a new dependency resolver
func NewResolver(ctx context.Context, k8sClient client.Client, namespace string) *Resolver

// Optional resolves a ref if it's non-nil, skips if nil
// T is the ORC resource type (e.g., *orcv1alpha1.Network)
func (r *Resolver) Optional[TP DependencyType[T], T any](
    ref *KubernetesNameRef,
    kind string,
    getID func(TP) *string,
) *Resolver

// Required resolves a ref that must exist (adds error if nil)
func (r *Resolver) Required[TP DependencyType[T], T any](
    ref KubernetesNameRef,
    kind string,
    getID func(TP) *string,
) *Resolver

// Result returns the resolved IDs and accumulated status
func (r *Resolver) Result() (ResolvedDependencies, progress.ReconcileStatus)

// ResolvedDependencies provides type-safe access to resolved IDs
type ResolvedDependencies struct {
    ids map[string]string
}

func (r ResolvedDependencies) Get(kind string) string
func (r ResolvedDependencies) GetPtr(kind string) *string
```

### Usage Examples

#### Example 1: ListOSResourcesForImport (Filter Resolution)

**Before:**
```go
func (actuator floatingipCreateActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
    var reconcileStatus progress.ReconcileStatus

    network := &orcv1alpha1.Network{}
    if filter.FloatingNetworkRef != nil {
        networkKey := client.ObjectKey{Name: string(ptr.Deref(filter.FloatingNetworkRef, "")), Namespace: obj.Namespace}
        if err := actuator.k8sClient.Get(ctx, networkKey, network); err != nil {
            // ... 15 lines of error handling
        }
    }

    port := &orcv1alpha1.Port{}
    if filter.PortRef != nil {
        // ... another 15 lines
    }

    project := &orcv1alpha1.Project{}
    if filter.ProjectRef != nil {
        // ... another 15 lines
    }

    if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
        return nil, reconcileStatus
    }

    listOpts := floatingips.ListOpts{
        PortID:            ptr.Deref(port.Status.ID, ""),
        FloatingNetworkID: ptr.Deref(network.Status.ID, ""),
        ProjectID:         ptr.Deref(project.Status.ID, ""),
        // ...
    }
    // ...
}
```

**After:**
```go
func (actuator floatingipCreateActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
    deps, rs := dependency.NewResolver(ctx, actuator.k8sClient, obj.Namespace).
        Optional[*orcv1alpha1.Network](filter.FloatingNetworkRef, "Network", func(n *orcv1alpha1.Network) *string { return n.Status.ID }).
        Optional[*orcv1alpha1.Port](filter.PortRef, "Port", func(p *orcv1alpha1.Port) *string { return p.Status.ID }).
        Optional[*orcv1alpha1.Project](filter.ProjectRef, "Project", func(p *orcv1alpha1.Project) *string { return p.Status.ID }).
        Result()

    if needsReschedule, _ := rs.NeedsReschedule(); needsReschedule {
        return nil, rs
    }

    listOpts := floatingips.ListOpts{
        PortID:            deps.Get("Port"),
        FloatingNetworkID: deps.Get("Network"),
        ProjectID:         deps.Get("Project"),
        // ...
    }
    // ...
}
```

#### Example 2: CreateResource (Spec Resolution)

**Before:**
```go
func (actuator loadbalancerActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
    resource := obj.Spec.Resource
    var reconcileStatus progress.ReconcileStatus

    var vipSubnetID string
    if resource.VipSubnetRef != nil {
        subnet, subnetDepRS := subnetDependency.GetDependency(
            ctx, actuator.k8sClient, obj, func(dep *orcv1alpha1.Subnet) bool {
                return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
            },
        )
        reconcileStatus = reconcileStatus.WithReconcileStatus(subnetDepRS)
        if subnet != nil {
            vipSubnetID = ptr.Deref(subnet.Status.ID, "")
        }
    }

    var vipNetworkID string
    if resource.VipNetworkRef != nil {
        // ... repeat pattern
    }

    // ... more dependencies
}
```

**After:**
```go
func (actuator loadbalancerActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
    resource := obj.Spec.Resource

    deps, rs := dependency.NewResolver(ctx, actuator.k8sClient, obj.Namespace).
        Optional[*orcv1alpha1.Subnet](resource.VipSubnetRef, "Subnet", func(s *orcv1alpha1.Subnet) *string { return s.Status.ID }).
        Optional[*orcv1alpha1.Network](resource.VipNetworkRef, "Network", func(n *orcv1alpha1.Network) *string { return n.Status.ID }).
        Optional[*orcv1alpha1.Port](resource.VipPortRef, "Port", func(p *orcv1alpha1.Port) *string { return p.Status.ID }).
        Optional[*orcv1alpha1.Flavor](resource.FlavorRef, "Flavor", func(f *orcv1alpha1.Flavor) *string { return f.Status.ID }).
        Optional[*orcv1alpha1.Project](resource.ProjectRef, "Project", func(p *orcv1alpha1.Project) *string { return p.Status.ID }).
        Result()

    if needsReschedule, _ := rs.NeedsReschedule(); needsReschedule {
        return nil, rs
    }

    createOpts := loadbalancers.CreateOpts{
        VipSubnetID:  deps.Get("Subnet"),
        VipNetworkID: deps.Get("Network"),
        VipPortID:    deps.Get("Port"),
        FlavorID:     deps.Get("Flavor"),
        ProjectID:    deps.Get("Project"),
        // ...
    }
    // ...
}
```

## Implementation

### File: `internal/util/dependency/resolver.go`

```go
package dependency

import (
    "context"
    "fmt"

    apierrors "k8s.io/apimachinery/pkg/api/errors"
    "k8s.io/utils/ptr"
    "sigs.k8s.io/controller-runtime/pkg/client"

    orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
    "github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
)

// Resolver accumulates dependency resolutions and their statuses.
type Resolver struct {
    ctx             context.Context
    k8sClient       client.Client
    namespace       string
    reconcileStatus progress.ReconcileStatus
    resolved        map[string]string
}

// NewResolver creates a new dependency resolver for the given namespace.
func NewResolver(ctx context.Context, k8sClient client.Client, namespace string) *Resolver {
    return &Resolver{
        ctx:       ctx,
        k8sClient: k8sClient,
        namespace: namespace,
        resolved:  make(map[string]string),
    }
}

// Optional resolves a ref if it's non-nil. If the ref is nil, it's skipped.
// The kind parameter is used for error messages and as the key in resolved dependencies.
// The getID function extracts the OpenStack ID from the resolved object.
func Optional[TP DependencyType[T], T any](r *Resolver, ref *orcv1alpha1.KubernetesNameRef, kind string, getID func(TP) *string) *Resolver {
    if ref == nil {
        return r
    }

    name := string(*ref)
    var obj TP = new(T)
    objectKey := client.ObjectKey{Name: name, Namespace: r.namespace}

    if err := r.k8sClient.Get(r.ctx, objectKey, obj); err != nil {
        if apierrors.IsNotFound(err) {
            r.reconcileStatus = r.reconcileStatus.WaitingOnObject(kind, name, progress.WaitingOnCreation)
        } else {
            r.reconcileStatus = r.reconcileStatus.WithError(fmt.Errorf("fetching %s %s: %w", kind, name, err))
        }
        return r
    }

    if !orcv1alpha1.IsAvailable(obj) {
        r.reconcileStatus = r.reconcileStatus.WaitingOnObject(kind, name, progress.WaitingOnReady)
        return r
    }

    id := getID(obj)
    if id == nil {
        r.reconcileStatus = r.reconcileStatus.WaitingOnObject(kind, name, progress.WaitingOnReady)
        return r
    }

    r.resolved[kind] = *id
    return r
}

// Required resolves a ref that must exist. If the ref is empty, an error is added.
func Required[TP DependencyType[T], T any](r *Resolver, ref orcv1alpha1.KubernetesNameRef, kind string, getID func(TP) *string) *Resolver {
    return Optional[TP, T](r, ptr.To(ref), kind, getID)
}

// Result returns the resolved dependencies and accumulated reconcile status.
func (r *Resolver) Result() (ResolvedDependencies, progress.ReconcileStatus) {
    return ResolvedDependencies{ids: r.resolved}, r.reconcileStatus
}

// ResolvedDependencies provides access to resolved OpenStack IDs.
type ResolvedDependencies struct {
    ids map[string]string
}

// Get returns the resolved ID for the given kind, or empty string if not resolved.
func (r ResolvedDependencies) Get(kind string) string {
    return r.ids[kind]
}

// GetPtr returns a pointer to the resolved ID, or nil if not resolved.
func (r ResolvedDependencies) GetPtr(kind string) *string {
    if id, ok := r.ids[kind]; ok {
        return &id
    }
    return nil
}

// Has returns true if the kind was resolved.
func (r ResolvedDependencies) Has(kind string) bool {
    _, ok := r.ids[kind]
    return ok
}
```

### Alternative: Method Chaining with Generics

Due to Go's limitation that methods cannot have type parameters, the fluent API requires top-level functions:

```go
// Usage with top-level functions (required by Go)
deps, rs := dependency.Optional[*orcv1alpha1.Network](
    dependency.Optional[*orcv1alpha1.Subnet](
        dependency.NewResolver(ctx, k8sClient, namespace),
        filter.SubnetRef, "Subnet", func(s *orcv1alpha1.Subnet) *string { return s.Status.ID },
    ),
    filter.NetworkRef, "Network", func(n *orcv1alpha1.Network) *string { return n.Status.ID },
).Result()
```

This is less readable. A better alternative is a non-generic wrapper:

```go
// Resolver methods return *Resolver for chaining
func (r *Resolver) OptionalNetwork(ref *KubernetesNameRef) *Resolver
func (r *Resolver) OptionalSubnet(ref *KubernetesNameRef) *Resolver
func (r *Resolver) OptionalProject(ref *KubernetesNameRef) *Resolver
// ... one method per known type
```

This sacrifices genericity for readability but requires adding a method for each type.

## Trade-offs

### Pros
- Significantly reduces boilerplate (45+ lines → 10 lines for 3 dependencies)
- Consistent error handling across all controllers
- Clear, readable code
- Type-safe
- No reflection magic

### Cons
- Go generics limitation: can't have generic methods, so either:
  - Use top-level functions (less fluent)
  - Add a method per type (more code in resolver)
- Slightly more abstraction to understand

## Relationship with Existing DeletionGuardDependency

This resolver is complementary to `DeletionGuardDependency`:

| Aspect | DeletionGuardDependency | Resolver |
|--------|------------------------|----------|
| **Purpose** | Full lifecycle management with finalizers | Simple one-off resolution |
| **Finalizers** | Yes, prevents deletion | No |
| **Watch/Index** | Yes, triggers reconciliation | No |
| **Use case** | Spec refs (create/update) | Filter refs, quick lookups |

For `CreateResource`, you might still want `DeletionGuardDependency.GetDependency()` if you need finalizer protection. The Resolver is best for:
- `ListOSResourcesForImport` filter resolution
- Quick lookups where finalizers aren't needed
- Reducing boilerplate in any dependency fetching

## Migration Path

1. Add `Resolver` to `internal/util/dependency/resolver.go`
2. Update one controller (e.g., floatingip) to use it
3. Validate it works correctly
4. Gradually migrate other controllers

## Open Questions

1. Should `Resolver` integrate with `DeletionGuardDependency` to add finalizers?
2. Should there be pre-defined methods like `OptionalProject()` for common types?
3. Should the ready check (`IsAvailable && Status.ID != nil`) be customizable per call?
