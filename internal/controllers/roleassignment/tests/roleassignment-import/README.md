# Import RoleAssignment

## Step 00

Create dependencies (Role, User, Project) as managed resources, and an
unmanaged RoleAssignment importing by filter that references all three.
Verify that the import RoleAssignment is waiting for the external resource
to be created in OpenStack.

## Step 01

Create a trap RoleAssignment using a different role but the same user and
project, and verify that it is not being imported by the filter.

## Step 02

Create a managed RoleAssignment matching the import filter and verify that
the imported RoleAssignment picks it up with the correct component IDs.
Also verify that the imported RoleAssignment didn't pick the trap.

## Reference

https://k-orc.cloud/development/writing-tests/#import
