# Create a DNSZone with the minimum options

## Step 00

Create a minimal DNSZone, that sets only the required fields (`name` and `email` - a zone name must
be explicit and end with a period per Designate's own convention, so unlike most other resources
the ORC object's own name can't be used as a fallback here), and verify that the observed state
corresponds to the spec.

## Step 01

Try deleting the secret and ensure that it is not deleted thanks to the finalizer.

## Reference

https://k-orc.cloud/development/writing-tests/#create-minimal
