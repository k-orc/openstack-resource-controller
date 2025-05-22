# Check dependency handling for imported router

## Step 00

Import a router that references other imported resources. The referenced imported resources have no matching resources yet.
Verify the router is waiting for the dependency to be ready.

## Step 01

Create a router matching the import filter, except for the referenced resources, and verify that it's not being imported.

## Step 02

Create the referenced resources and a router matching the import filters.

Verify that the observed status on the imported router corresponds to the spec of the created router.

## Step 03

Delete the referenced resources and check that ORC does not prevent deletion. The OpenStack resources still exists because they
were imported resources and we only deleted the ORC representation of it.

## Step 04

Delete the router and validate that all resources are gone.

## Reference

https://k-orc.cloud/development/writing-tests/#import-dependency
