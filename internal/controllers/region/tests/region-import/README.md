# Import Region

## Step 00

Import a region that matches all fields in the filter, and verify it is waiting for the external resource to be created.

## Step 01

Create a region whose name is a superstring of the one specified in the import filter, otherwise matching the filter, and verify that it's not being imported.

## Step 02

Create a region matching the filter and verify that the observed status on the imported region corresponds to the spec of the created region.
Also, confirm that it does not adopt any region whose name is a superstring of its own.

## Reference

https://k-orc.cloud/development/writing-tests/#import
