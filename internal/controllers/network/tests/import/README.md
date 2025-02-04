# Import Network

## Step 00

Import a network, matching all of the available filter's fields, and verify it is waiting for the external resource to be created.

## Step 01

Create a network matching the filter and verify that the observed status on the imported network corresponds to the spec of the created network.

## Step 02

Delete the created network and verify that the imported network is not available

## Step 03

Validate we're able to delete resources.
Cleaning up resources also avoids a race where kuttl could delete the secret before the other resources.

## TODO

Possibly check that adding a new network matching the import filter does not cause issues after it successfully imported the first one.
