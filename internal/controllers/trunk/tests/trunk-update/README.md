# Update Trunk

## Step 00

Create a Trunk using only mandatory fields (portRef only).

## Step 01

Update all mutable fields: name, description, tags, and add subports.

## Step 02

Update subports: remove subport2, add subport3, change subport1 segmentation.

## Step 03

Revert the resource to its original value and verify that the resulting object matches its state when first created (no description, no tags, no subports, adminStateUp true).

## Reference

https://k-orc.cloud/development/writing-tests/#update
