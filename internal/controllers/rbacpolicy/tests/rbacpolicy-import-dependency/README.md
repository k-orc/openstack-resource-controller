# Check dependency handling for imported RBAC policies

## Step 00

Import an RBAC policy whose filter references an imported project that has no matching resource yet.
Verify the RBAC policy is waiting for the dependency to be ready.

## Step 01

Create an RBAC policy matching the import filter, except for the referenced project, and verify that it's not being imported.

## Step 02

Create the referenced project and an RBAC policy matching the import filter.

Verify that the observed status on the imported RBAC policy corresponds to the spec of the created one.

## Step 03

Delete the referenced project and check that ORC does not prevent deletion.

## Step 04

Delete the RBAC policy and validate that all resources are gone.

## Reference

https://k-orc.cloud/development/writing-tests/#import-dependency
