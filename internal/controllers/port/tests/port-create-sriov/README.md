# Create a port with vnic type direct

## Step 00

Create two ports: one with vnic type direct and port security disabled, and another with admin credentials, so that we can use fields which are enforced by policies, and verify that the observed state corresponds to the spec.

Also validate that the OpenStack resource uses the name from the spec when it is specified.

## Reference

https://k-orc.cloud/development/writing-tests/#create-full
