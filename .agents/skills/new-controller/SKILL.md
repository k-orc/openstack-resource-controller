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

**Before scaffolding**, research the resource to understand field names, types, and mutability:

1. **Read the gophercloud struct** to get exact field names and types:
```bash
go doc <gophercloud-module>.<Type>
go doc <gophercloud-module>.CreateOpts
```
   - Note field types precisely: `int32` in gophercloud → `*int32` in status (not `*int`)

2. **Check OpenStack API documentation** for update capabilities:
   - URL: `https://docs.openstack.org/api-ref/<service>/`
   - Which fields can be updated? (determines mutability in spec)

3. **Look at a similar existing controller** for patterns (if user provided one):
   - Check their `*_types.go` for API structure
   - Check their `actuator.go` for implementation patterns

4. **Field naming rules for API types**:
   - **Status**: Preserve gophercloud names exactly, just camelCase for JSON
     - Example: gophercloud `ProjectID` → status `projectID`
     - **Expose ALL fields from gophercloud struct** unless they contain sensitive data or are redundant
     - Include: core fields (IDs, names), observability fields (state, network details), metadata (timestamps like `createdAt`, `updatedAt`)
   - **Spec**: Convert OpenStack ID fields to ORC references using `*KubernetesNameRef` with `Ref` suffix
     - **Naming rule**: Use the ORC resource name (e.g., `networkRef`, `subnetRef`, `flavorRef`)
     - **Keep prefixes only if they have semantic meaning or prevent collision**:
       - Keep semantic prefixes: `vipSubnetRef` (VIP has meaning), `floatingNetworkRef` (floating has meaning), `sourceSecurityGroupRef` (distinguishes from destination)
       - Drop service prefixes: `NeutronNetID` → `networkRef` (not `neutronNetRef` - users think in terms of ORC Network resource)
       - Keep distinguishing prefixes: If resource has both `NetworkID` and `ExternalNetworkID` → `networkRef` and `externalNetworkRef`
     - **Only if ORC has a controller for that resource** - otherwise omit the field and add a TODO comment
     - Example: `// TODO(controller): Add AvailabilityZoneRef when AvailabilityZone controller is implemented`

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
| `-import-dependency` | Dependency for import filter (can repeat flag for multiple). Prefer enabling all available dependencies that can be used for filtering. | none |
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
- `ListOSResourcesForAdoption()` - Match by spec fields. **Include all available fields** that the OpenStack List API supports for filtering (e.g., name, description, and other immutable configuration fields)
- `GetResourceReconcilers()` - (if resource supports updates)

### Implementation Patterns

Follow the patterns in patterns.md when implementing the actuator and API types.

### Status Writer (internal/controllers/<kind>/status.go)

Implement:
- `ResourceAvailableStatus()` - When is resource available?
- `ApplyResourceStatus()` - Map OpenStack fields to status
  - **Expose ALL fields from the gophercloud struct** unless they contain sensitive data (passwords, tokens) or are redundant with ORC object metadata
  - Include core fields (IDs, names, primary configuration)
  - Include observability fields (state, status, network details, capacity information)
  - Include metadata fields (timestamps like `createdAt`, `updatedAt` if available)
  - Use conditional writes for optional/zero-value fields (see existing controllers for patterns)

## Step 7: Write and Run Tests

**This step is required** - do not skip it.

The scaffolding tool creates test directory stubs in `internal/controllers/<kind>/tests/` with README files explaining each test suite's purpose.

**Read the README files** in each test directory to understand:
- What each test suite validates
- The test structure and steps
- What assertions are needed

Complete the test stubs and run tests following the testing skill.

**Test Assertion Pattern**: In create-full tests, validate **all status fields** including optional ones (observability fields, timestamps), not just basic fields like name and description.

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
  - [ ] Status field names match gophercloud exactly
  - [ ] All gophercloud fields exposed in status (core, observability, metadata/timestamps)
  - [ ] Field types match gophercloud precisely (int32 vs int)
  - [ ] Spec ID fields converted to *KubernetesNameRef with Ref suffix using ORC resource names
  - [ ] Semantic prefixes preserved (vip, floating, source, destination, external)
  - [ ] Service prefixes dropped (neutron → network/subnet)
  - [ ] Stricter types where appropriate (IPvAny, custom tag types)
  - [ ] Status constants in types.go (if resource has provisioning states)
- [ ] Actuator methods implemented:
  - [ ] DeleteResource: no cascade unless explicitly requested
  - [ ] DeleteResource: handles pending states and 409 Conflict (if resource has intermediate states)
  - [ ] ListOSResourcesForAdoption: includes all available filterable fields (name, description, immutable config)
  - [ ] CreateResource includes tags with sorting (if applicable)
  - [ ] Proper error classification (Terminal vs retryable)
- [ ] Status writer implemented:
  - [ ] All gophercloud fields mapped to status
  - [ ] Conditional writes for optional/zero-value fields
  - [ ] Timestamps included (createdAt, updatedAt if available)
- [ ] Update reconciler includes tags update (if tags are mutable)
- [ ] All TODOs resolved
- [ ] `make generate` runs cleanly
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] E2E tests written:
  - [ ] Read README files in test directories
  - [ ] All test stubs completed
  - [ ] Create-full test validates ALL status fields (not just name/description)
- [ ] E2E tests passing
