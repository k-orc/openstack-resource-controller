# Import SwiftContainer

## Step 00

Import a Swift container using a prefix filter, and verify it is waiting for
the external resource to be created.

## Step 01

Create a Swift container whose name does not match the import prefix, and
verify that it is not being imported.

## Step 02

Create a Swift container matching the prefix filter and verify that the
observed status on the imported container corresponds to the spec of the
created container. Also verify that the created container didn't adopt the
unrelated trap container.

Validates that:
- The import filter matches the container by prefix.
- `status.id` is populated with the imported container name.
- Adoption from unmanaged import to available state works without recreating
  the container.

## Reference

https://k-orc.cloud/development/writing-tests/#import
