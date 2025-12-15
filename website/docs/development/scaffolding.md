# Scaffolding a new controller

The first step in writing a new controller is to generate the scaffolding. ORC provides an interactive scaffolding tool that generates most of the boilerplate code required for a new controller.

## Running the scaffolding tool

To scaffold a new controller, run:

```bash
go run ./cmd/scaffold-controller
```

By default, the tool runs interactively, prompting you for each required value. For automation or reproducibility, use non-interactive mode with flags:

```bash
go run ./cmd/scaffold-controller -interactive=false \
    -kind=VolumeBackup \
    -gophercloud-client=NewBlockStorageV3 \
    -gophercloud-module=github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/backups \
    ...
```

After the scaffolding tool returned successfully, generate the files and commit your changes:

```bash
# Run code generation
make generate

# Commit the scaffolding output with the command as the message
git add .
git commit -m "$(cat <<'EOF'
Scaffolding for the VolumeBackup controller

$ go run ./cmd/scaffold-controller -interactive=false \
    -kind=VolumeBackup \
    -gophercloud-client=NewBlockStorageV3 \
    -gophercloud-module=github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/backups \
    ...
EOF
)"
```

This step is important as it makes it a lot easier to review your changes. Reviewers can skip the scaffolding commit (it's generated code) and focus on your actual implementation changes. Also, having the commit message document exactly how the scaffolding was generated helps ensure reproducibility.

## Generated files

The scaffolding tool generates the following files:

### API types

- `api/v1alpha1/<kind>_types.go` - API type definitions with TODO markers

### Controller implementation

- `internal/controllers/<kind>/actuator.go` - Actuator implementation stubs
- `internal/controllers/<kind>/controller.go` - Controller setup and registration
- `internal/controllers/<kind>/status.go` - Status writer implementation

### OpenStack client

- `internal/osclients/<kind>.go` - OpenStack client wrapper

### Tests

- `internal/controllers/<kind>/tests/<kind>-create-minimal/` - Minimal creation test
- `internal/controllers/<kind>/tests/<kind>-create-full/` - Full creation test
- `internal/controllers/<kind>/tests/<kind>-import/` - Import test
- `internal/controllers/<kind>/tests/<kind>-import-error/` - Import error test
- `internal/controllers/<kind>/tests/<kind>-update/` - Mutability test
- Additional test directories based on dependencies

### Samples

- `config/samples/openstack_v1alpha1_<kind>.yaml` - Example resource manifest

## Post-scaffolding steps

After the scaffolding tool completes, you need to perform several manual integration steps:

### Register with the resource generator

Add the new resource to `cmd/resource-generator/main.go`:

```go
var resources []templateFields = []templateFields{
    // ... existing resources ...
    {
        Name: "YourResource",
    },
}
```

Then regenerate the supporting code:

```bash
make generate
```

This generates additional files:

- `api/v1alpha1/zz_generated.<kind>-resource.go` - Generated API helpers
- `internal/controllers/<kind>/zz_generated.adapter.go` - Generated adapter
- `internal/controllers/<kind>/zz_generated.controller.go` - Generated controller wrapper

This generator covers functionality common to all controllers. Its purpose is not only to reduce boilerplate, but also to guarantee consistency of behaviour across APIs.

!!! note

    While it is possible to write this code manually, any controller requiring this potentially stretches assumptions made throughout the project. If this is required, consider whether changes can be made to avoid it, or whether further design work is needed in the scaffolding or generic controller code.

### Add the OpenStack client to scope

Update three files in `internal/scope/`:

- `scope.go`, add the client interface:
```go
type Scope interface {
    // ... existing methods ...
    NewYourResourceClient() (osclients.YourResourceClient, error)
}
```

- `provider.go`, implement the client constructor:
```go
func (s *providerScope) NewYourResourceClient() (osclients.YourResourceClient, error) {
    return osclients.NewYourResourceClient(s.provider)
}
```

- `mock.go`, add mock support:
```go
type MockScopeFactory struct {
	// ... existing clients ...
	YourResourceClient *mock.MockYourResourceClient
}

func NewMockScopeFactory(mockCtrl *gomock.Controller) *MockScopeFactory {
	// ... existing clients ...
	yourResourceClient := mock.NewMockServiceClient(mockCtrl)

	return &MockScopeFactory{
		// ... existing clients ...
		YourResourceClient: yourResourceClient,
	}
}

func (s *mockScope) NewYourResourceClient() (osclients.YourResourceClient, error) {
    return s.yourResourceClient, nil
}
```

### Register the controller

Add the controller to `cmd/manager/main.go`:

```go
import (
    // ... existing imports ...
    yourresourcecontroller "github.com/k-orc/openstack-resource-controller/internal/controllers/yourresource"
)

// In the controllers slice:
controllers := []interfaces.Controller{
    // ... existing controllers ...
    yourresourcecontroller.New(scopeFactory),
}
```

### Implement the TODOs

Search the generated code for `TODO(scaffolding)` markers and implement each one:

```bash
grep -r "TODO(scaffolding)" api/ internal/controllers/<kind>/
```

Key areas requiring implementation:

- [API types](api-contracts.md): Define `Filter`, `ResourceSpec`, and `ResourceStatus` structs
- [Actuator](interfaces.md#actuator): Implement `CreateResource`, `DeleteResource`, and optionally `GetResourceReconcilers`
- [Status writer](interfaces.md#resourcestatuswriter): Implement `ResourceAvailableStatus` and `ApplyResourceStatus`
- [Tests](writing-tests.md): Ensure the tests for your controller are complete

!!! note

    There is a high chance that the API for the resource you're working on differs from the stub the scaffolding tool created. For example, some resources don't have name, id, or description. If that's the case, adapt the generated code to match your resource.

### Generate the OLM bundle

After implementation is complete:

```bash
make generate-bundle
```

### Update documentation

- Add the new controller to the README.md
- Update any relevant user documentation
