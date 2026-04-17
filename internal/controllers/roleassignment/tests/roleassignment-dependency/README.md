# Test RoleAssignment dependency handling

## Step 00

Create a RoleAssignment that references Role, User, and Project that don't exist yet.
Verify that it enters Progressing state waiting for dependencies.

## Step 01

Create the dependencies and verify the RoleAssignment becomes Available.

## Step 02

Try to delete a dependency (Project) while it's still referenced by the RoleAssignment.
Verify the deletion is blocked by the finalizer.

## Step 03

Delete the RoleAssignment first, then verify dependencies can be deleted.

## Reference

https://k-orc.cloud/development/writing-tests/#dependencies
