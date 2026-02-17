# Creation and deletion dependencies

## Step 00

Create Users referencing non-existing resources. Each User is dependent on other non-existing resource. Verify that the Users are waiting for the needed resources to be created externally.

## Step 01

Create the missing dependencies and verify all the Users are available.

## Step 02

Disable the domain dependency to allow KUTTL to cleanup resources without any issues.

## Step 03

Delete all the dependencies and check that ORC prevents deletion since there is still a resource that depends on them.

## Step 04

Delete the Users and validate that all resources are gone.

## Reference

https://k-orc.cloud/development/writing-tests/#dependency