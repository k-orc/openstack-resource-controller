# Create a RBACPolicy with the minimum options

## Step 00

Create a minimal RBACPolicy, that sets only the required fields (RBACPolicy has no
optional fields - networkRef, action and targetProjectID are all required), and
verify that the observed state corresponds to the spec.

## Step 01

Try deleting the secret and ensure that it is not deleted thanks to the finalizer.

## Reference

https://k-orc.cloud/development/writing-tests/#create-minimal
