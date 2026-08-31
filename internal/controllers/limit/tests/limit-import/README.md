# Import Limit

## Step 00

Import a limit that matches all fields in the filter, and verify it is waiting for the external resource to be created.

## Step 01

Create a limit whose resource name is a superstring of the one specified in the import filter, otherwise matching the filter, and verify that it's not being imported.

## Step 02

Create a limit matching the filter and verify that the observed status on the imported limit corresponds to the spec of the created limit.
Also, confirm that it does not adopt any limit whose name is a superstring of its own.
