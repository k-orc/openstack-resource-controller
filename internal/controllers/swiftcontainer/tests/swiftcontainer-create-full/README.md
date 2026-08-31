# Create a SwiftContainer with all the options

## Step 00

Create a Swift container using all available fields, and verify that the
observed state corresponds to the spec.

Validates that:
- The OpenStack resource uses the name from `spec.resource.name` when it is
  specified, rather than the ORC object name.
- Custom metadata key-value pairs are applied and reflected in
  `status.resource.metadata`.
- Read ACL (`containerRead`) and write ACL (`containerWrite`) are configured
  and reflected in `status.resource`.
- Storage policy (`storagePolicy`) is configured and reflected in
  `status.resource`.
- `Available=True` and `Progressing=False` conditions are set.

## Reference

https://k-orc.cloud/development/writing-tests/#create-full
