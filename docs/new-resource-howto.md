# Add new resource HOWTO

## Define API

Add the new resource name to `cmd/resource-generator/main.go` and run:

```shell
make generate-resources
```

If your resource has dependency on other resources, you'll need to set
`SpecExtraType` and `StatusExtraType`.

Follow the API contract document.

Create a `api/v1alpha1/<resource>_types.go` file containing your API.

```shell
make generate
```

## Create controllers

Create the following files, based on existing files:
- `internal/controllers/<resource>/controller.go`
- `internal/controllers/<resource>/reconcile.go`
- `internal/controllers/<resource>/status.go`

```shell
make generate
```

When the code compiles, enable the controller by starting it from `cmd/manager/main.go`.
Add the controller's name to `pkg/controllers/alias.go`.

## CRD

After runing `make generate`, you should have your CRD in `config/crd/bases/openstack.k-orc.cloud_<resource>.yaml`.

Add this file to the CRD kustomize in `config/crd/kustomization.yaml`.

## Create example

Create your resource in one of the example directory in `examples/`.

Add your resource to `examples/kustomizeconfig/kustomizeconfig.yaml` so that the resource name get prefixed with your username.
