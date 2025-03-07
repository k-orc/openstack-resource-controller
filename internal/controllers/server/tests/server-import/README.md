# Test import making sure only a server with the exact name matches

## Step 00

Import a server, matching all of the available filter's fields, and verify it is waiting for the external resource to be created.

## Step 01

Create a server which name is a superstring of the one specified in the import filter, and otherwise matching the filter, and verify that it's not being imported.

## Step 02

Create a server matching the filter and verify that the observed status on the imported server corresponds to the spec of the created server.
Also verify that the created server didn't adopt the one which name is a superstring of it.
