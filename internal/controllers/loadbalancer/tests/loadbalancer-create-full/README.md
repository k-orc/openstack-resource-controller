# Create a load balancer with all the options

## Step 00

Create a load balancer using all available optional fields, and verify that the
observed state corresponds to the spec.

This test validates SC-004 (full creation path):
- Creates a LoadBalancer with all optional fields populated:
  - `name`: overrides the default name with `loadbalancer-create-full-override`
  - `description`: human-readable description
  - `subnetRef`: references a prerequisite Subnet for VIP allocation
  - `adminStateUp`: set to `false` (non-default value)
  - `tags`: two tags (`tag1`, `tag2`)
  - `projectRef`: references a prerequisite Project
- Asserts `Available=True` once OpenStack provisioning_status reaches ACTIVE
- Asserts `Progressing=False` once the resource is fully reconciled
- Asserts `status.id` is populated with a valid UUID
- Validates `status.resource.name` matches `spec.resource.name` using CEL
- Validates `status.resource.description` matches `spec.resource.description` using CEL
- Validates `status.resource.adminStateUp` is `false` using CEL
- Validates `status.resource.vipSubnetID` matches the referenced Subnet's `status.id` using CEL resourceRefs
- Validates `status.resource.projectID` matches the referenced Project's `status.id` using CEL resourceRefs
- Validates `status.resource.tags` contains both expected tags using CEL

We omit `flavor` because it is provider-specific and not reliably available in
all test environments.

## Reference

https://k-orc.cloud/development/writing-tests/#create-full
