# Create a router with the minimum options

## Step 00

Create a minimal router, that sets only the required fields, and verify that the observed state corresponds to the spec.

Also validate that the OpenStack resource uses the name of the ORC object when it is not specified.

## Step 01

Validate we're able to delete resources.
Cleaning up resources also avoids a race where kuttl could delete the secret before the other resources.
