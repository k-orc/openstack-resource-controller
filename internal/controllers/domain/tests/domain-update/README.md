# Update Domain

## Step 00

Create a Domain using only mandatory fields.

## Step 01

Update all mutable fields.

## Step 02

Revert the resource to its original value and verify the resulting object is similar to when if was first created.

## Step 03

Disable the enabled field so the Domain can be deleted successfully.

Disabling the Domain is required before deletion in Openstack.

## Reference

https://k-orc.cloud/development/writing-tests/#update