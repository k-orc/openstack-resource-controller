# Import Router

## Step 00

Import a Router using non-admin credentials, matching all of the available filter's fields, and verify it is waiting for the external resource to be created.

## Step 01

Create a Router which name is a superstring of the one specified in the import filter, and otherwise matching the filter, and verify that it's not being imported.

## Step 02

Create a Router matching the filter and verify that the observed status on the imported Router corresponds to the spec of the created Router.
Also verify that the created Router didn't adopt the one which name is a superstring of it.

## Step 03

Validate we're able to delete resources.
Cleaning up resources also avoids a race where kuttl could delete the secret before the other resources.

