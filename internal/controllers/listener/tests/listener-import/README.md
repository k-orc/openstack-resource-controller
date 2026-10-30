# Import Listener

## Step 00

Import a listener that matches all fields in the filter, and verify it is waiting for the external resource to be created.

## Step 01

Create a listener whose name is a superstring of the one specified in the import filter, otherwise matching the filter, and verify that it's not being imported.

## Step 02

Create a listener matching the filter and verify that the observed status on the imported listener corresponds to the spec of the created listener.
Also, confirm that it does not adopt any listener whose name is a superstring of its own.

## Reference

https://k-orc.cloud/development/writing-tests/#import
