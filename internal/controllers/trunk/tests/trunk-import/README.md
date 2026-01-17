# Import Trunk

## Step 00

Create Network and Subnet prerequisites, then create an import resource that
matches all fields in the filter (name, description, adminStateUp, tags).
Verify it is waiting for the external resource to be created.

## Step 01

Create a trunk whose name is a superstring of the one specified in the import
filter (`trunk-import-external-not-this-one`), otherwise matching the filter,
and verify that it's not being imported.

## Step 02

Create a trunk matching the filter (`trunk-import-external`) and verify that
the observed status on the imported trunk corresponds to the spec of the
created trunk. Also, confirm that it does not adopt the trap trunk whose name
is a superstring of its own.

## Reference

https://k-orc.cloud/development/writing-tests/#import
