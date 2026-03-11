# Creation and deletion dependencies

## Step 00

Create AddressScopes referencing non-existing resources. Each AddressScope is dependent on other non-existing resource. Verify that the AddressScopes are waiting for the needed resources to be created externally.

## Step 01

Create the missing dependencies and verify all the AddressScopes are available.

## Step 02

Delete all the dependencies and check that ORC prevents deletion since there is still a resource that depends on them.

## Step 03

Delete the AddressScopes and validate that all resources are gone.

## Reference

https://k-orc.cloud/development/writing-tests/#dependency
