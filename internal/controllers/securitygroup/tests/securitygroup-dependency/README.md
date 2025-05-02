# Creation and deletion dependencies

## Step 00

Create securitygroups referencing non-existing resources. Each securitygroup is dependent on other non-existing resource. Verify that the securitygroups are waiting for the needed resources to be created externally.

## Step 01

Create the missing dependencies and make and verify all the securitygroups are available.

## Step 02

Delete all the dependencies and check that ORC prevents deletion since there is still a resource that depends on them.

## Step 03

Delete the securitygroups and validate that all resources are gone.
