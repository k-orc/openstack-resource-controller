# Create a floating IP with the minimum options

## Step 00

Create a minimal floating IP, that sets only the required fields, and verify that the observed state corresponds to the spec.

## Step 01

Try deleting the secret and ensure that it is not deleted thanks to the finalizer.

## Reference

https://k-orc.cloud/development/writing-tests/#create-minimal
