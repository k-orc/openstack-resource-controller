# Remove a reference to a dependency being deleted

## Step 00

Create two ports referencing the same security group, one of them also referencing a second security group.

## Step 01

Delete the security group referenced by both ports and verify that ORC prevents its deletion.

## Step 02

Remove the security group from the first port and verify that its deletion is still prevented, since the second port references it.

## Step 03

Remove the security group from the second port and verify that it is deleted.

## Reference

https://k-orc.cloud/development/writing-tests/#dependency
