# Import FloatingIP

## Step 00

Import a floating ip credentials, matching all of the available filter's fields, and verify it is waiting for the external resource to be created.

## Step 01

Create a floating ip which value is slightly different of the one specified in the import filter, and otherwise matching the filter, and verify that it's not being imported.

## Step 02

Create a floating ip matching the filter and verify that the observed status on the imported floating ip corresponds to the spec of the created floating ip.
Also verify that the created floating ip didn't adopt the one which name is a superstring of it.

