# Create a load balancer using a network reference for VIP allocation

## Step 00

Create a load balancer specifying `spec.resource.networkRef` to allocate the
VIP address from a network (without pinning a specific subnet), and verify that
the observed state reflects auto-allocation.

This test validates SC-002 (networkRef VIP allocation):
- Creates a prerequisite ORC Network
- Creates a LoadBalancer with only `networkRef` set (mutually exclusive with `subnetRef` and `vipPortRef`)
- Asserts `Available=True` once OpenStack provisioning_status reaches ACTIVE
- Asserts `Progressing=False` once the resource is fully reconciled
- Asserts `status.id` is populated with a valid UUID
- Asserts `status.resource.vipAddress` is non-empty (auto-allocated by OpenStack from the network)
- Validates `status.resource.vipNetworkID` matches the referenced Network's `status.id` using CEL resourceRefs

When `networkRef` is used (instead of `subnetRef`), OpenStack selects a subnet
automatically. The resulting VIP address is reflected in `status.resource.vipAddress`.

## Reference

https://k-orc.cloud/development/writing-tests/#create-minimal
