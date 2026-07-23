# Check dependency handling for imported Limit

## Step 00

Import a Limit that references other imported resource(projectRef). The referenced imported resource(project) has no matching resources yet.
Verify the Limit is waiting for the dependency to be ready.

## Step 01

Create the referenced resource(projectRef) and a Limit matching the import filters.

Verify that the observed status on the imported Limit corresponds to the spec of the created Limit.

## Step 02

Delete the referenced imported resources and check that ORC does not prevent deletion. The OpenStack resources still exist because they
were imported resources and we only deleted the ORC representation of it.

## Step 03

Delete the Limit and validate that all resources are gone.
