# Creation and deletion dependencies

## Step 00

Create a Lease referencing a non-existing credentials secret, and a Lease reserving instances of a non-existing Flavor. Verify that they are waiting for the needed resource to be created externally.

## Step 01

Create the missing dependencies: the secret, and the Flavor imported from the DevStack flavor. Verify that the Leases are created.

## Step 02

Delete all the dependencies. Check that ORC prevents the deletion of the secret since a Lease still depends on it. The Flavor has no deletion guard: Blazar copies its size when the lease is created, so it can be deleted.

## Step 03

Delete the Leases and validate that all resources are gone.

## Reference

https://k-orc.cloud/development/writing-tests/#dependency
