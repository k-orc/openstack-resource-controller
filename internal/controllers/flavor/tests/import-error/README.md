# Import Flavor with more than one matching resources

## Step 00

Create two flavors with identical specs.

## Step 01

Ensure that an imported flavor with a filter matching the resources returns an error.

## Step 02

Validate we're able to delete resources.
Cleaning up resources also avoids a race where kuttl could delete the secret before the other resources.
