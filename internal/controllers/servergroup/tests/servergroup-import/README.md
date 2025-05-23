# Import ServerGroup

## Step 00

Import a server group using non-admin credentials, matching all of the available filter's fields, and verify it is waiting for the external resource to be created.

## Step 01

Create a server group which name is a superstring of the one specified in the import filter, and otherwise matching the filter, and verify that it's not being imported.

## Step 02

Create a server group matching the filter and verify that the observed status on the imported server group corresponds to the spec of the created server group.
Also verify that the created server group didn't adopt the one which name is a superstring of it.
