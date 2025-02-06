# Create a network with all the options

## Step 00

Create a network using all available fields, and verify that the observed state corresponds to the spec.

Also validate that the OpenStack resource uses the name from the spec when it is specified.

## Step 01

Validate we're able to delete resources.
Cleaning up resoßurces also avoids a race where kuttl could delete the secret before the other resources.

## TODO
We may want to add in the future a test to check that a network with a dns domain that is not terminated by a dot will not be available