# Create a Domain with the minimum options

## Step 00

Create a minimal Domain, that sets only the required fields, and verify that the observed state corresponds to the spec.

Also validate that the OpenStack resource uses the name of the ORC object when it is not specified.

## Step 01
By default the enabled field is set to true, the enabled field needs to be disabled.

Disabling the Domain is required before deletion in Openstack.

## Step 02

Try deleting the secret and ensure that it is not deleted thanks to the finalizer.

## Reference

https://k-orc.cloud/development/writing-tests/#create-minimal