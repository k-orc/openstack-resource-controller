# Creation and deletion dependencies

## Step 00

Create router interfaces referencing non-existing resources. Each router interface is dependent on other non-existing resource. Verify that the router interfaces are waiting for the needed resources to be created externally.

## Step 01

Create the missing dependencies and make and verify all the router interfaces are available.

## Step 02

Delete all the dependencies and check that ORC prevents deletion since there is still a resource that depends on them.

## Step 03

Delete the router interfaces and validate that all resources are gone.
