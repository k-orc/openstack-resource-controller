# Check dependency handling for imported port

## Step 00

Import a port that references other imported resources. The referenced imported resources have no matching resources yet.
Verify the port is waiting for the dependency to be ready.

## Step 01

Create a port matching the import filter, except for the referenced resources, and verify that it's not being imported.

## Step 02

Create the referenced resources and a port matching the import filters.

Verify that the observed status on the imported port corresponds to the spec of the created port.

## Step 03

Delete the referenced resources and check that ORC does not prevent deletion. The OpenStack resources still exists because they
were imported resources and we only deleted the ORC representation of it.

## Step 04

Delete the port and validate that all resources are gone.

## Reference

https://k-orc.cloud/development/writing-tests/#import-dependency
