# Import Trunk

## Step 00

Create the required dependencies (Network, Subnet, Port) and the external trunk to be imported.
Import the trunk that matches all fields in the filter and verify it is successfully imported.

## Step 01

Create a trunk whose name is a superstring of the one specified in the import filter, otherwise matching the filter, and verify that it's not being imported.

## Step 02

Verify that the observed status on the imported trunk corresponds to the spec of the created trunk.
Also, confirm that it does not adopt any trunk whose name is a superstring of its own.

## Reference

https://k-orc.cloud/development/writing-tests/#import
