# Creation and deletion dependencies

## Step 00

Create Roles referencing non-existing resources. Each Role is dependent on other non-existing resource. Verify that the Roles are waiting for the needed resources to be created externally.

## Step 01

Create the missing dependencies and make and verify all the Roles are available.

## Step 02

Disable the domain dependency to allow KUTTL to cleanup resources without any issues.

## Step 03

Delete all the dependencies and check that ORC prevents deletion since there is still a resource that depends on them.

## Step 04

Delete the Roles and validate that all resources are gone.

## Reference

https://k-orc.cloud/development/writing-tests/#dependency
