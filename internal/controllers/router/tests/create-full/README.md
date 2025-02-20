# Create a router with all the options

## Step 00

Create a router using all available fields, and verify that the observed state corresponds to the spec.

Also validate that the OpenStack resource uses the name from the spec when it is specified.

## Step 01

Validate we're able to delete resources.
Cleaning up resources also avoids a race where kuttl could delete the secret before the other resources.
