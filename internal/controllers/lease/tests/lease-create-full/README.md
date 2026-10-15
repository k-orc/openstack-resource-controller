# Create a Lease with all the options

## Step 00

Create a Lease using all available fields of an instance reservation, and verify that the observed state corresponds to the spec.

The lease starts in the future, so it stays PENDING: it is not Available and is still Progressing.

Also validate that the OpenStack resource uses the name from the spec when it is specified.

## Reference

https://k-orc.cloud/development/writing-tests/#create-full
