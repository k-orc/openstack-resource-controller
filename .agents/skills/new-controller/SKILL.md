---
name: new-controller
description: Create a new ORC controller for an OpenStack resource. Use when adding support for a new OpenStack resource type (e.g., LoadBalancer, FloatingIP).
disable-model-invocation: true
---

# Create New Controller

Create a new ORC controller for an OpenStack resource.

**IMPORTANT**: Complete ALL steps in order. Do not stop after implementing TODOs - you must also write E2E tests and run them. Ask the user for `E2E_OSCLOUDS` path if needed to run tests.

## Prerequisites

Ask the user one by one about:
1. What OpenStack resource to create (e.g., "VolumeBackup")
2. Which service it belongs to (compute, network, blockstorage, identity, image)
3. Does it need polling for availability or deletion? (i.e., does the resource have intermediate provisioning states like PENDING_CREATE, BUILD, etc.)
4. Any dependencies on other ORC resources (required, optional, or import-only)?
5. Is there a similar existing controller to use as reference? (e.g., Listener for LoadBalancer)
6. Do they have `E2E_OSCLOUDS` path to a clouds.yaml for running E2E tests locally? (If not, local E2E testing will be skipped)
7. Any additional requirements or constraints? (e.g., cascade delete support, special validation rules, immutability requirements)

## Step 1: Research the OpenStack Resource

**Before scaffolding**, research the resource to understand the exact field names:

1. **Read the gophercloud struct** to get exact field names:
```bash
go doc <gophercloud-module>.<Type>
go doc <gophercloud-module>.CreateOpts
```

2. **Look at a similar existing controller** for patterns (if user provided one):
   - Check their `*_types.go` for API structure
   - Check their `actuator.go` for implementation patterns

3. **Note the exact field names** from gophercloud - use these when defining API types:
   - If gophercloud has `VipSubnetID`, name the ORC field `VipSubnetRef` (not just `SubnetRef`)
   - If gophercloud has `FlavorID`, name the ORC field `FlavorRef`
   - Preserve prefixes like `Vip`, `Source`, `Destination` etc.

## Step 2: Run Scaffolding Tool

**IMPORTANT**: Build a single scaffolding command using the user's answers and the flags reference below. Run it exactly ONCE (user will be prompted to approve).

Use the field names discovered in Step 1 to inform your implementation later.

### Scaffolding Flags Reference

**Required flags:**

| Flag | Description | Example |
|------|-------------|---------|
| `-kind` | The Kind of the new resource (PascalCase) | `VolumeBackup`, `FloatingIP` |
| `-gophercloud-client` | The gophercloud function to instantiate a client | `NewBlockStorageV3`, `NewNetworkV2` |
| `-gophercloud-module` | Full gophercloud module import path | `github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/backups` |

**Optional flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `-gophercloud-type` | The gophercloud struct type name | Same as `-kind` |
| `-openstack-json-object` | Object name in OpenStack JSON responses | snake_case of kind (e.g., `volume_backup`) |
| `-available-polling-period` | Polling period in seconds while waiting for resource to become available | `0` (available immediately) |
| `-deleting-polling-period` | Polling period in seconds while waiting for resource to be deleted | `0` (deleted immediately) |
| `-required-create-dependency` | Required dependency for creation (can repeat flag for multiple) | none |
| `-optional-create-dependency` | Optional dependency for creation (can repeat flag for multiple) | none |
| `-import-dependency` | Dependency for import filter (can repeat flag for multiple) | none |
| `-interactive` | Run in interactive mode | `true` (set to `false` for scripted use) |

### Common Gophercloud Clients

| Service | Client Function | Module Path Prefix |
|---------|-----------------|-------------------|
| Compute | `NewComputeV2` | `github.com/gophercloud/gophercloud/v2/openstack/compute/v2/...` |
| Network | `NewNetworkV2` | `github.com/gophercloud/gophercloud/v2/openstack/networking/v2/...` |
| Block Storage | `NewBlockStorageV3` | `github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/...` |
| Identity | `NewIdentityV3` | `github.com/gophercloud/gophercloud/v2/openstack/identity/v3/...` |
| Image | `NewImageV2` | `github.com/gophercloud/gophercloud/v2/openstack/image/v2/...` |

### Example Command (for reference only - build your own based on user input)

```bash
# Example with dependencies - adapt based on user's answers
go run ./cmd/scaffold-controller -interactive=false \
    -kind=Port \
    -gophercloud-client=NewNetworkV2 \
    -gophercloud-module=github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports \
    -required-create-dependency=Network \
    -optional-create-dependency=Subnet \
    -optional-create-dependency=SecurityGroup \
    -import-dependency=Network
```

After scaffolding completes, run code generation:

```bash
make generate
```

Commit the scaffolding with the command used:

```bash
git add .
git commit -m "$(cat <<'EOF'
Scaffolding for the VolumeBackup controller

$ go run ./cmd/scaffold-controller -interactive=false \
    -kind=VolumeBackup \
    -gophercloud-client=NewBlockStorageV3 \
    -gophercloud-module=github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/backups
EOF
)"
```

## Step 3: Register with Resource Generator

Add the new resource to `cmd/resource-generator/main.go` in the `resources` slice:

```go
var resources []templateFields = []templateFields{
    // ... existing resources (keep alphabetically sorted) ...
    {
        Name: "VolumeBackup",
    },
}
```

Then regenerate to create the `zz_generated.*.go` files:

```bash
make generate
```

## Step 4: Add OpenStack Client to Scope

Update these files in `internal/scope/`:

### scope.go
Add interface method:
```go
NewYourResourceClient() (osclients.YourResourceClient, error)
```

### provider.go
Implement the constructor:
```go
func (s *providerScope) NewYourResourceClient() (osclients.YourResourceClient, error) {
    return osclients.NewYourResourceClient(s.provider)
}
```

### mock.go
Add mock client field and implementation for testing.

## Step 5: Register Controller

Add to `cmd/manager/main.go`:

```go
import (
    yourresourcecontroller "github.com/k-orc/openstack-resource-controller/v2/internal/controllers/yourresource"
)

// In controllers slice:
controllers := []interfaces.Controller{
    // ...
    yourresourcecontroller.New(scopeFactory),
}
```

## Step 6: Implement TODOs

**Reference Documentation**: For detailed patterns and rationale, see:
- `website/docs/development/controller-implementation.md` - Progressing condition, ReconcileStatus, error handling, dependencies
- `website/docs/development/api-design.md` - Filter, ResourceSpec, ResourceStatus conventions
- `website/docs/development/coding-standards.md` - Code organization, naming, logging

Find all scaffolding TODOs:

```bash
grep -r "TODO(scaffolding)" api/v1alpha1/ internal/controllers/<kind>/
```

### API Types (api/v1alpha1/<kind>_types.go)

Use the exact field names from gophercloud discovered in Step 1.

Define:
- `<Kind>ResourceSpec` - Creation parameters with validation markers
- `<Kind>Filter` - Import filter with `MinProperties:=1`
- `<Kind>ResourceStatus` - Observed state fields

### Actuator (internal/controllers/<kind>/actuator.go)

Implement:
- `CreateResource()` - Build CreateOpts, call OpenStack API
- `DeleteResource()` - Call delete API
- `ListOSResourcesForImport()` - Apply filter to list results
- `ListOSResourcesForAdoption()` - Match by spec fields
- `GetResourceReconcilers()` - (if resource supports updates)

### Implementation Patterns

Follow the patterns in @.agents/skills/new-controller/patterns.md when implementing the actuator and API types.

### Status Writer (internal/controllers/<kind>/status.go)

Implement:
- `ResourceAvailableStatus()` - When is resource available?
- `ApplyResourceStatus()` - Map OpenStack fields to status

## Step 7: Write and Run Tests

**This step is required** - do not skip it.

Complete the test stubs in `internal/controllers/<kind>/tests/` and run tests following @.agents/skills/testing/SKILL.md

## Checklist

- [ ] Gophercloud struct researched (field names noted)
- [ ] Similar controller reviewed (if applicable)
- [ ] Scaffolding complete
- [ ] First `make generate` run
- [ ] Scaffolding committed
- [ ] Registered in resource-generator
- [ ] Second `make generate` run (creates zz_generated files)
- [ ] OpenStack client added to scope
- [ ] Controller registered in main.go
- [ ] API types implemented:
  - [ ] Correct field names (matching OpenStack conventions)
  - [ ] Stricter types where appropriate (IPvAny, custom tag types)
  - [ ] Status constants in types.go (if resource has provisioning states)
- [ ] Actuator methods implemented:
  - [ ] DeleteResource: no cascade unless explicitly requested
  - [ ] DeleteResource: handles pending states and 409 Conflict (if resource has intermediate states)
  - [ ] CreateResource includes tags with sorting (if applicable)
  - [ ] Proper error classification (Terminal vs retryable)
  - [ ] Descriptive dependency variable names
- [ ] Status writer implemented
- [ ] Update reconciler includes tags update (if tags are mutable)
- [ ] All TODOs resolved
- [ ] `make generate` runs cleanly
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] E2E tests written (including dependency tests if applicable)
- [ ] E2E tests passing
