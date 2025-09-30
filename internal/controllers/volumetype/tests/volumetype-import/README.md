# Import VolumeType

## Step 00

Import a volumetype, matching all of the available filter's fields, and verify it is waiting for the external resource to be created.

## Step 01

Create a volumetype which name is a superstring of the one specified in the import filter, and otherwise matching the filter, and verify that it's not being imported.

## Step 02

Create a volumetype matching the filter and verify that the observed status on the imported volumetype corresponds to the spec of the created volumetype.
Also verify that the created volumetype didn't adopt the one which name is a superstring of it.

## Reference

https://k-orc.cloud/development/writing-tests/#import
