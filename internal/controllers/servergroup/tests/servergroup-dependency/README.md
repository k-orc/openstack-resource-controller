# Creation and deletion dependencies

## Step 00

Create server group referencing non-existing resources. Verify that the server group is waiting for the needed resource to be created externally.

## Step 01

Create the missing dependencies and make and verify the server group is available.

## Step 02

Delete all the dependencies and check that ORC prevents deletion since there is still a resource that depends on them.

## Step 03

Delete the server group and validate that all resources are gone.
