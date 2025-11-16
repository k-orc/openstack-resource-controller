# Writing tests

## Unit tests

### Mutability tests

For each controller, we have a file called `actuator_test.go` which handles
custom code and mutability operations. We require those tests to get good
coverage from all of our controllers. So, if your controller implements mutable
fields, you should add tests for those fields.

You can run all with:

```bash
$ make test
```

You can also specify which controller do you want to test using the `TEST_PATHS`
environment variable. This variable indicates a list of module paths, and you
can specify modules that you want to test by passing the package's path
separated by a blank space, for example:

```bash
$ TEST_PATHS=$(go list ./internal/controller/server ./internal/controller/image) make test
```

### API validation tests

All APIs are expected to have good API validation test coverage.

API validation tests ensure that any validations defined in the
API and included in the CRD perform as expected. Add API validation tests for
your controller in `test/apivalidations`.

### Controller-specific tests

Tests other than the ones above, that cover the functionality specific to
a single controller, should live in the controller's directory.

## E2E tests

All controllers are expected to have good test coverage. ORC uses
[kuttl](https://github.com/kudobuilder/kuttl) for end-to-end testing.

### General considerations when writing kuttl tests

#### Documentation

Kuttl tests can get very complicated very quickly and it can be difficult to
follow what each of them does.
We ask that each test contains a `README.md` file describing what it does.
Don't hesitate to sprinkle your test files with comments as well.

#### Naming convention

Each test runs in a separate Kubernetes namespace, however because they all run
in the same OpenStack tenant, we still need to be careful about name clashes when
referencing resources, and filters for importing resources:

- a test must only reference resources it created. Resources created on
  OpenStack must have a unique name among the whole test suite. Typically, you
  would use the name of the test wherever possible.
- a test must ensure their filters match resources uniquely

These constraints ensure that tests can safely run concurrently.

Also, to make things easier when debugging, we require that all tests are named
differently. In kuttl, the test name is the name of the directory containing
the test files. For this reason, we prefix all the test names with the name of
the resource being tested, to differentiate them.

#### Resource cleanup

At the end of each test, kuttl destroys the namespace. This deletes all
leftover ORC objects, and respective OpenStack resources, your test has
created.

However, if you have created OpenStack resources outside of ORC as part of the
test, you MUST clean them to avoid leaking resources. Keep in mind that if the
test fails before it had a chance to manually clean the resources, you would
still have a leak. To counter this, only create OpenStack resources
externally when it is absolutely necessary, and avoid writing tests that fail.

### Testing patterns

This section describes the expected tests for controllers. Use all the ones
that apply to your controller, to ensure the best coverage.

#### create-minimal

The `create-minimal` test consists of creating a resource, setting only the
required fields and validating that the observed state corresponds to the spec.

All controllers should implement this test.

When a resource doesn't have any required field, use an empty resource spec:

```yaml
apiVersion: openstack.k-orc.cloud/v1alpha1
kind: Network
metadata:
  name: network-create-minimal
spec:
  cloudCredentialsRef:
    cloudName: openstack
    secretName: openstack-clouds
  managementPolicy: managed
  resource: {}
```

Because this test creates a resource with minimal dependencies, we overload
it to validate the dependency on secrets. Every resource that interacts with
OpenStack should have a dependency on the credentials secret, meaning that we can't
delete the secret while the object exists.

Concretely, after trying to manually delete the secret with `kubectl delete
secret openstack-clouds --wait=false`, we check that the secret was flagged for
deletion but not deleted due to the presence of finalizers. We use a CEL assertion for this:
```yaml
apiVersion: kuttl.dev/v1beta1
kind: TestAssert
resourceRefs:
    - apiVersion: v1
      kind: Secret
      name: openstack-clouds
      ref: secret
assertAll:
    - celExpr: "secret.metadata.deletionTimestamp != 0"
    - celExpr: "'openstack.k-orc.cloud/flavor' in secret.metadata.finalizers"
```

The [`network-create-minimal`][network-create-minimal] test is a good example of this test.

[network-create-minimal]: https://github.com/k-orc/openstack-resource-controller/tree/main/internal/controllers/network/tests/network-create-minimal

#### create-full

Very similar to the `create-minimal` test, except that it sets all available fields
of the resource spec.

Sometimes you'll find options that are mutually exclusive. In that case you may
need to write separate tests to exercise all the options. Try to come up with
a meaningful name for the scenario, for example `create-sriov`.

All controllers should implement this test.

It can be seen with the [`network-create-full`][network-create-full] test for example.

[network-create-full]: https://github.com/k-orc/openstack-resource-controller/tree/main/internal/controllers/network/tests/network-create-full

#### dependency

Whenever your controller has dependencies on other resources, we want to ensure
it is able to wait for the resource to exist and be ready, and optionally have
a dependency guard on the resource to prevent its deletion while we depend on
it.

To know if your controller needs to implement this test, check the API and see
if it accepts any `<Resource>Ref` field in its resource spec. This should
almost always be the case; with few exceptional cases, every resource has
a dependency on the secret.
For additional dependencies, subnet for example has a dependency on
a [network][subnet-network-dep], and possibly a [router][subnet-router-dep].

The testing pattern goes like this:

1. Create the resources with all dependencies satisfied except for one
    1. Check that ORC is waiting on the missing dependency to exist
1. Create the missing dependencies
    1. Check that the resources are finally created
1. Delete all dependencies
    1. Validate that ORC prevented the deletion due to a finalizer. There might
       be exceptions (for example flavor for a server)
1. Delete the resources
    1. Verify that all dependencies are now deleted

See how the [`subnet-dependency`][subnet-dependency] test implements it.

[subnet-network-dep]: https://github.com/k-orc/openstack-resource-controller/blob/874c2174b097d4cefc092a5deac6b14d4e50ff3a/api/v1alpha1/subnet_types.go#L72
[subnet-router-dep]: https://github.com/k-orc/openstack-resource-controller/blob/874c2174b097d4cefc092a5deac6b14d4e50ff3a/api/v1alpha1/subnet_types.go#L128
[subnet-dependency]: https://github.com/k-orc/openstack-resource-controller/tree/main/internal/controllers/subnet/tests/subnet-dependency

#### import

The `import` test validates import filters, and adoption.

The testing pattern goes like this:

1. Create an unmanaged resource using all of the available filters for the resource
    1. Verify it is waiting on an external resource matching the filters to exist
1. Create a resource matching all of the import filter except for the name that
   is a superstring of the one specified in the import filter, e.g. if we are
   waiting on a resource called `foo`, create a resource called `foobar`.
    1. Verify that this resource is not being imported -- it validates that we
       don't perform regex-based name search (some OpenStack projects do it)
1. Create a resource matching all of the import filters, including the name.
    1. Verify that the imported resource is available and the observed status
       corresponds to that of the created resource.
    1. Validate that the previously created resource wasn't imported as the new
       one, again, because of regex-based name matching

All controllers should implement this test.

See the [`flavor-import`][flavor-import] test for an example.

[flavor-import]: https://github.com/k-orc/openstack-resource-controller/tree/main/internal/controllers/flavor/tests/flavor-import

#### import-error

In the `import-error` test, we verify that an import filter matching multiple
resources returns an error.

All controllers should implement this test.

See the [`flavor-import-error`][flavor-import-error] test for an example.

[flavor-import-error]: https://github.com/k-orc/openstack-resource-controller/tree/main/internal/controllers/flavor/tests/flavor-import-error

#### import-dependency

The `import-dependency` test is required for resources that allow
a `<Resource>Ref` field in their import filters.

We want to verify that ORC waits for the dependency to be created and available
when it creates the resource, but that it allows deleting the dependency
without the resource being deleted first. This ensures that unmanaged resources
don't add deletion guards on dependencies.

One such example is the [`port-import-dependency`][port-import-dependency] test.

The pattern for this test is:

1. Create two unmanaged resources: one for the resource being tested, another one for its dependency
    1. Verify that the resource under test is waiting for the dependency to be ready
1. Create a dummy resource matching the import filter except for the dependency
    1. Verify it is not being imported
1. Create resources matching the import filter for the resource being tested and for the dependency
    1. The resource under test must move to Available, and its observed status corresponds to the created resource
1. Delete the dependency
    1. Verify it's gone
1. Finally delete the resource
    1. Verify it's gone

[port-import-dependency]: https://github.com/k-orc/openstack-resource-controller/tree/main/internal/controllers/port/tests/port-import-dependency

#### update

The `update` test is required for resources that implement mutability. It
should test both setting and unsetting resource properties.

The testing pattern consists of:

1. Create a resource using only mandatory fields, similar to the `minimal` test
1. Update all fields that support mutability
    1. Verify that the changes are reflected in the observed status
1. Revert the changes
    1. Verify that the resource status is similar to the one we had in the first step

As support for mutability is still being worked on, we don't have tests that
implement this pattern yet. The closest we have is the
[`securitygroup-update`][securitygroup-update] test.

[securitygroup-update]: https://github.com/k-orc/openstack-resource-controller/tree/main/internal/controllers/securitygroup/tests/securitygroup-update

### Kuttl tips

It is possible to shell out and run `openstack` command:
```yaml
apiVersion: kuttl.dev/v1beta1
kind: TestStep
commands:
  - script: |
      cd $(dirname ${E2E_KUTTL_OSCLOUDS})
      export OS_CLOUD=openstack
      openstack port list
```

Similarly, you can run `kubectl` commands with:
```yaml
apiVersion: kuttl.dev/v1beta1
kind: TestStep
commands:
  - command: kubectl get port
    namespaced: true
```

### Running tests

We use environment variables to configure how the tests run.

| Variable      | Description | Default |
| ----------- | ----------- |----------- |
| `E2E_OSCLOUDS` | Path to a clouds.yaml to use for e2e tests | /etc/openstack/clouds.yaml |
| `E2E_CACERT`   | Path to a cacert file to use to connect to OpenStack | |
| `E2E_OPENSTACK_CLOUD_NAME` | Name of the openstack credentials to use | devstack |
| `E2E_OPENSTACK_ADMIN_CLOUD_NAME` | Name of the openstack admin credentials to use | devstack-admin-demo |
| `E2E_EXTERNAL_NETWORK_NAME` | Name of the external network to use | public |
| `E2E_KUTTL_DIR` | Run kuttl tests from a specific directory |  |
| `E2E_KUTTL_TEST` | Run a specific kuttl test |  |
| `E2E_KUTTL_FLAVOR` | Flavor name to use for tests | m1.tiny |

For example, to run the `import-dependency` test from the `subnet` controller:

```bash
E2E_KUTTL_DIR=internal/controllers/subnet/tests E2E_KUTTL_TEST=import-dependency make test-e2e
```
