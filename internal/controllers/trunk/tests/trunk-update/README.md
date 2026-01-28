# Update Trunk

## Step 00

Create a disabled Trunk (adminStateUp: false) using only mandatory fields.

## Step 01

Update all mutable fields: name, description, tags.
Enable the trunk by setting adminStateUp to true.
Assert that the trunk is enabled with status: ACTIVE.

## Step 02

Revert the resource to minimal state (no description, no tags) but keep it enabled (adminStateUp: true) so it can be deleted.

## Step 03

Delete all resources (Trunk, Port, Subnet, Network).

## Reference

https://k-orc.cloud/development/writing-tests/#update
