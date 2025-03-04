# Import Network

## Step 00

Import a network, matching all of the available filter's fields, and verify it is waiting for the external resource to be created.

## Step 01

Create a network which name is a superstring of the one specified in the import filter, and otherwise matching the filter, and verify that it's not being imported.

## Step 02

Create a network matching the filter and verify that the observed status on the imported network corresponds to the spec of the created network.
Also verify that the created network didn't adopt the one which name is a superstring of it.

## Step 03

Validate we're able to delete resources.
Cleaning up resources also avoids a race where kuttl could delete the secret before the other resources.

## TODO

Possibly check that adding a new network matching the import filter does not cause issues after it successfully imported the first one.
