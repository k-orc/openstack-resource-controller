# Import security group

## Step 00

Import a security group using non-admin credentials, matching all of the available filter's fields, and verify it is waiting for the external resource to be created.

## Step 01

Create a security group which name is a superstring of the one specified in the import filter, and otherwise matching the filter, and verify that it's not being imported.

## Step 02

Create a security group matching the filter and verify that the observed status on the imported security group corresponds to the spec of the created security group.
Also verify that the created security group didn't adopt the one which name is a superstring of it.

## TODO

Possibly check that adding a new security group matching the import filter does not cause issues after it successfully imported the first one.
