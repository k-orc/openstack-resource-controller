# Create a load balancer with the minimum options

## Step 00

Create a minimal load balancer, setting only the required fields, and verify
that the observed state corresponds to the spec.

This test validates SC-001 (minimal creation path):
- Creates a LoadBalancer with only `subnetRef` set (the minimum required field)
- Asserts `Available=True` once OpenStack provisioning_status reaches ACTIVE
- Asserts `Progressing=False` once the resource is fully reconciled
- Asserts `status.id` is populated with a valid UUID
- Asserts `status.resource.vipSubnetID` matches the referenced Subnet's `status.id`

Also validates that the OpenStack resource uses the name of the ORC object when
it is not specified.

## Step 01

Try deleting the secret and ensure that it is not deleted thanks to the
finalizer.

## Reference

https://k-orc.cloud/development/writing-tests/#create-minimal
