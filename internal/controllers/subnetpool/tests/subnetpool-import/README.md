# Import SubnetPool

## Step 00

Import a subnetpool that matches all fields in the filter (name, description, minPrefixLength), and verify it is waiting for the external resource to be created.

## Step 01

Create a subnetpool whose name is a superstring of the one specified in the import filter, otherwise matching the filter, and verify that it's not being imported.

## Step 02

Create a subnetpool matching the filter and verify that the observed status on the imported subnetpool corresponds to the spec of the created subnetpool.
Also, confirm that it does not adopt any subnetpool whose name is a superstring of its own.

## Reference

https://k-orc.cloud/development/writing-tests/#import
