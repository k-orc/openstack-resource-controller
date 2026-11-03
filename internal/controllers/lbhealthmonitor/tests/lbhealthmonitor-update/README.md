# Update LBHealthMonitor

## Step 00

Create a LBHealthMonitor using only mandatory fields.

## Step 01

Update all mutable fields.

## Step 02

Revert the resource to its original value. The required fields and adminStateUp go back; the optional fields keep their values, since Octavia has defaults for them and gophercloud can't reset them or clear the tags.

## Reference

https://k-orc.cloud/development/writing-tests/
