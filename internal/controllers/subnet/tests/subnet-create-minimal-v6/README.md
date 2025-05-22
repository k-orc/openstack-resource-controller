# Create an IPv6 subnet with the minimum options

## Step 00

Create a minimal subnet, that sets only the required fields, and verify that the observed state corresponds to the spec.

Also validate that the OpenStack resource uses the name of the ORC object when it is not specified.

## Step 01

Try deleting the secret and ensure that it is not deleted thanks to the finalizer.

## Reference

https://k-orc.cloud/development/writing-tests/#create-minimal
