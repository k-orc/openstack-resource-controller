# Import Volume Type

## Step 00

Import a volume type, matching all of the available filter's fields, and verify it is waiting for the external resource to be created.

## Step 01

Create a volume type which name is a superstring of the one specified in the import filter, and otherwise matching the filter, and verify that it's not being imported.

## Step 02

Create a volume type matching the filter and verify that the observed status on the imported volume type corresponds to the spec of the created volume type.
Also verify that the created volume type didn't adopt the one which name is a superstring of it.

## TODO

Possibly check that adding a new volume type matching the import filter does not cause issues after it successfully imported the first one.

## Reference

https://k-orc.cloud/development/writing-tests/#import
