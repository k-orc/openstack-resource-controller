# LoadBalancer dependency and waiting behavior

This test validates the edge case from SC-001: "Referenced Subnet does not exist".

## Step 00

Create a LoadBalancer referencing a Subnet (`loadbalancer-dependency`) that does not yet exist.
Verify that the LoadBalancer waits gracefully:

- `Progressing=True` with reason `Progressing`
- Message indicates it is waiting for the missing Subnet: `Waiting for Subnet/loadbalancer-dependency to be created`
- `Available=False` with the same reason and message

The LoadBalancer is created *before* its referenced Subnet exists, which is the key edge case
being validated: ORC must not error out, but must wait and reconcile once the dependency appears.

## Step 01

Create the referenced Network and Subnet resources. Once the Subnet becomes Available, the
LoadBalancer controller detects the dependency is satisfied and reconciles the LoadBalancer.

Verify that the LoadBalancer transitions to:

- `Available=True` with reason `Success`
- `Progressing=False` with reason `Success`
- `status.id` is populated with a valid UUID
- `status.resource.vipSubnetID` matches the Subnet's `status.id`

## Reference

https://k-orc.cloud/development/writing-tests/#dependency
