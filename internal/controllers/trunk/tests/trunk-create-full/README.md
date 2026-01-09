# Create a Trunk with all the options

## Step 00

Create a Trunk using all available fields, and verify that the observed state corresponds to the spec.

Also validate that the OpenStack resource uses the name from the spec when it is specified.

## Step 01

By default neutron refuses to delete disabled trunks. This step sets the `AdminStateUp` accordingly so that we can delete the trunk.

## Reference

https://k-orc.cloud/development/writing-tests/#create-full
