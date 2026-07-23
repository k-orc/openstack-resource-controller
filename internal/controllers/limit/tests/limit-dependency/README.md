# Creation and deletion dependencies

## Step 00

Create Limits referencing non-existing resources. Each Limit is dependent on some other non-existing resource(project/domain/secret). Verify that the Limits are waiting for the needed resources to be created externally.

## Step 01

Create the missing dependencies and verify all the Limits are available.

## Step 02

Delete all the dependencies and check that ORC prevents deletion since there is still a resource that depends on them.

## Step 03

Delete the Limits and validate that all resources are gone.
