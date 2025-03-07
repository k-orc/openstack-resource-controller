# Writing tests

## API validation tests

All APIs are expected to have good API validation test coverage.

API validation tests are tests which ensure that any validations defined in the
API and included in the CRD perform as expected. Add API validation tests for
your controller in `test/apivalidations`.

## E2E tests

All controllers are expected to have good e2e test coverage.

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

## Controller-specific tests

Tests other than the above, for example tests covering functionality specific to
a single controller, should live in the controller's directory.