# Test import making sure only a router with the exact name matches

## Step 00

Import a router, matching all of the available filter's fields, and verify it is waiting for the external resource to be created.

## Step 01

Create a router which name is a superstring of the one specified in the import filter, and otherwise matching the filter, and verify that it's not being imported.

## Step 02

Create a router matching the filter and verify that the observed status on the imported router corresponds to the spec of the created router.
Also verify that the created router didn't adopt the one which name is a superstring of it.