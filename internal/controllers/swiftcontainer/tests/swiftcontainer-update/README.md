# Update SwiftContainer

## Step 00

Create a SwiftContainer using only the required fields (no metadata, no ACLs),
and verify that the observed state corresponds to the spec.

## Step 01

Update all mutable fields:
- Add custom metadata key-value pairs
- Set read and write ACLs (`containerRead` and `containerWrite`)

Verify that all updated properties are reflected in the resource status.

## Step 02

Revert the resource to its original value (no metadata, no ACLs) and verify
the resulting object matches the initial creation state.

Validates that:
- Clearing `containerRead` and `containerWrite` (by removing the fields) removes the ACLs from the container.
- Removing metadata entries removes them from the container.
- `Available=True` and `Progressing=False` conditions are set after each step.

## Reference

https://k-orc.cloud/development/writing-tests/#update
