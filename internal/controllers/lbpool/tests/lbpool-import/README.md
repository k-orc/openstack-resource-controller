# Import LBPool

## Step 00

Import a lbpool that matches all fields in the filter, and verify it is waiting for the external resource to be created.

## Step 01

Create a lbpool whose name is a superstring of the one specified in the import filter, otherwise matching the filter, and verify that it's not being imported.

## Step 02

Create a lbpool matching the filter and verify that the observed status on the imported lbpool corresponds to the spec of the created lbpool.
Also, confirm that it does not adopt any lbpool whose name is a superstring of its own.

## Reference

https://k-orc.cloud/development/writing-tests/#import
