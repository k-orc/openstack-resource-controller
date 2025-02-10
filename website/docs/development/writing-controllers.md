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

## Unit tests

Add API validation tests for your controller in `test/apivalidations`.

## e2e tests

ORC uses [kuttl](https://kuttl.dev/) for end-to-end testing. Add tests for the
controllers you're writing. Check out the [flavor][flavor-tests] and the
[subnet][subnet-tests] that should cover all the patterns we're expecting to
see tested.

Each test runs in a separate kubernetes namespace, however because they all run
in the same openstack tenant, we still need to be careful about name clash when
referencing resources, and filters for importing resources:

- a test must only reference resources it created. Resources created on
  OpenStack must have a unique name among the whole test suite.
- a test must ensure their filters match resources uniquely

This is the condition to run tests concurrently.

Each test contains a `README.md` file describing what the test does.

[flavor-tests]: https://github.com/k-orc/openstack-resource-controller/tree/main/internal/controllers/flavor/tests
[subnet-tests]: https://github.com/k-orc/openstack-resource-controller/tree/main/internal/controllers/subnet/tests
