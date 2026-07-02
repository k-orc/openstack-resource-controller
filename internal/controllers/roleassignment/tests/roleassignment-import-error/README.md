# Import RoleAssignment Error

## Step 00

Create dependencies (User, two Roles, a Project) as managed resources, and
two managed RoleAssignments assigning each role to the same user on the same
project.

## Step 01

Import an unmanaged RoleAssignment using a filter that specifies only userRef
and projectRef. Both role assignments match the filter, causing a terminal
error because more than one matching resource was found.

## Reference

https://k-orc.cloud/development/writing-tests/#import-error
