# Writing a new controller

## Define API

Add the new resource name to `cmd/resource-generator/main.go` and run:

```shell
make generate-resources
```

If your resource has dependency on other resources, you'll need to set
`SpecExtraType`.

Follow the [API contract](api-contracts.md) document.

Create a `api/v1alpha1/<resource>_types.go` file containing your API.

```shell
make generate
```

## Create controllers

Create the following files, based on files from an existing controller:

- `internal/controllers/<resource>/actuator.go`
- `internal/controllers/<resource>/controller.go`
- `internal/controllers/<resource>/reconcile.go`
- `internal/controllers/<resource>/status.go`

```shell
make generate
```

When the code compiles, enable the controller by adding it to `cmd/manager/main.go`.

## CRD

After running `make generate`, you should have your CRD in `config/crd/bases/openstack.k-orc.cloud_<resource>.yaml`.

Add this file to the CRD kustomize in `config/crd/kustomization.yaml`.

## Create example

Create your resource in one of the example directory in `examples/`.

Add your resource to `examples/components/kustomizeconfig/kustomizeconfig.yaml`
so that the resource name get prefixed with your username.

## e2e tests

Add e2e tests for your resource.
