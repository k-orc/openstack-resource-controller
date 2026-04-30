# Create a load balancer using a port reference for the VIP

## Step 00

Create a load balancer specifying `spec.resource.vipPortRef` to use an existing
ORC Port as the VIP port, and verify that the observed state reflects the
referenced port's identity.

This test validates SC-003 (vipPortRef VIP port assignment):
- Creates prerequisite ORC Network, Subnet, and Port resources
- Creates a LoadBalancer with only `vipPortRef` set (mutually exclusive with `subnetRef` and `networkRef`)
- Asserts `Available=True` once OpenStack provisioning_status reaches ACTIVE
- Asserts `Progressing=False` once the resource is fully reconciled
- Asserts `status.id` is populated with a valid UUID
- Validates `status.resource.vipPortID` matches the referenced Port's `status.id` using CEL resourceRefs

When `vipPortRef` is used, OpenStack assigns the specified port as the VIP port
for the load balancer. The resulting port ID is reflected in
`status.resource.vipPortID` and must match the Port object's `status.id`.

## Reference

https://k-orc.cloud/development/writing-tests/#create-minimal
