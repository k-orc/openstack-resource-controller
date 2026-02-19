# Update AddressScope

## Step 00

Create two AddressScopes using only mandatory fields, but one of them
will be used to update the `shared` field.

## Step 01

Update all mutable fields.

## Step 02

Revert the resource to its original value and verify that the resulting object matches its state when first created, except the resource with the shared field.

## Reference

https://k-orc.cloud/development/writing-tests/#update
