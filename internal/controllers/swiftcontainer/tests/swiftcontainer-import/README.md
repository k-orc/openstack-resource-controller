# Import SwiftContainer

## Step 00

Import a Swift container using a name filter, and verify it is waiting for
the external resource to be created.

## Step 01

Create a Swift container whose name is a superstring of the one specified in
the import filter, and otherwise matching the filter, and verify that it is
not being imported (trap pattern).

## Step 02

Create a Swift container matching the filter and verify that the observed
status on the imported container corresponds to the spec of the created
container. Also verify that the created container didn't adopt the one whose
name is a superstring of it (filter specificity).

Validates that:
- The import filter matches the container with exact name (not a superstring).
- `status.id` is populated with the imported container name.
- Adoption from unmanaged import to available state works without recreating
  the container (SC-003).

## Reference

https://k-orc.cloud/development/writing-tests/#import
