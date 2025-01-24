# Import Flavor

## Step 00

Import a flavor using non-admin credentials, matching all of the available filter's fields, and verify it is waiting for the external resource to be created.

## Step 01

Create a flavor matching the filter and verify that the observed status on the imported flavor corresponds to the spec of the created flavor.

## Step 02

Validate we're able to delete resources.
Cleaning up resources also avoids a race where kuttl could delete the secret before the other resources.

## TODO

Possibly check that adding a new flavor matching the import filter does not cause issues after it successfully imported the first one.
