# Update a trunk

This test verifies that mutable fields of a trunk can be updated.

## Step 00

Create a minimal trunk and verify its initial state.

## Step 01

Update the trunk with:
- New name
- Description
- Admin state (false)
- Tags

Verify that the changes are reflected in the observed status.

## Step 02

Revert the changes back to the minimal configuration and verify that the resource status is similar to the one we had in step 00.

## Reference

https://k-orc.cloud/development/writing-tests/#update

