# Check dependency handling for imported RoleAssignment

## Step 00

Create an unmanaged Role importing by filter (name that doesn't exist yet),
managed User and Project dependencies, and an unmanaged RoleAssignment
importing by filter with roleRef pointing to the unmanaged Role.
Verify the RoleAssignment is waiting for the Role dependency to be ready.

## Step 01

Create a trap RoleAssignment with a different role but the same user and
project, and verify that it is not being imported.

## Step 02

Create a managed Role matching the unmanaged Role's import filter and a
managed RoleAssignment matching the import filter. Verify the imported
RoleAssignment is available with correct component IDs.

## Step 03

Delete the import dependency (the unmanaged Role) and verify ORC does not
prevent deletion. Import dependencies should not have deletion guards.

## Step 04

Delete the imported RoleAssignment and verify it's gone.

## Reference

https://k-orc.cloud/development/writing-tests/#import-dependency
