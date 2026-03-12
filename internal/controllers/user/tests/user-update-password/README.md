# Update User Password

This test verifies that a User's password can be updated by changing the referenced Secret.

## Step 00

Create a User with a password Secret containing "InitialPassword123".

## Step 01

Update the password Secret to contain "UpdatedPassword456". Verify that the User reconciles and remains Available.

## Step 02

Revert the password Secret back to "InitialPassword123". Verify that the User reconciles and remains Available.

## What This Tests

- Password is set during creation
- Password can be updated by changing the Secret
- Password updates trigger reconciliation
- User remains Available throughout password updates
- Password can be changed multiple times

## Note

The password value itself is write-only in OpenStack, so we cannot verify the actual password value in the status. This test only verifies that password updates don't cause errors and the User remains Available.
