# Create a Group with all the options

## Step 00

Create a Group using all available fields, and verify that the observed state corresponds to the spec.

Also validate that the OpenStack resource uses the name from the spec when it is specified.

## Step 01

By default the enabled field is set to true, the enabled field needs to be disabled.

Disabling the Domain is required before deletion in Openstack.

## Reference

https://k-orc.cloud/development/writing-tests/#create-full
