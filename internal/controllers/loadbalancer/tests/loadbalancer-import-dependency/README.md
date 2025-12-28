# Check dependency handling for imported LoadBalancer

## Step 00

Import a LoadBalancer that references other imported resources. The referenced imported resources have no matching resources yet.
Verify the LoadBalancer is waiting for the dependency to be ready.

## Step 01

Create a LoadBalancer matching the import filter, except for referenced resources, and verify that it's not being imported.

## Step 02

Create the referenced resources and a LoadBalancer matching the import filters.

Verify that the observed status on the imported LoadBalancer corresponds to the spec of the created LoadBalancer.

## Step 03

Delete the referenced resources and check that ORC does not prevent deletion. The OpenStack resources still exist because they
were imported resources and we only deleted the ORC representation of it.

## Step 04

Delete the LoadBalancer and validate that all resources are gone.

## Reference

https://k-orc.cloud/development/writing-tests/#import-dependency
