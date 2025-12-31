# Creation and deletion dependencies

## Step 00

Create Volumes referencing non-existing resources. Each Volume is dependent on other non-existing resource. Verify that the Volumes are waiting for the needed resources to be created externally.

## Step 01

Create the missing dependencies and make and verify all the Volumes are available.

## Step 02

Delete all the dependencies and check:
- VolumeType and Secret have finalizers preventing deletion (hard dependencies)
- Image is deleted immediately (soft dependency - no finalizer)

## Step 03

Delete the Volumes and validate that VolumeType and Secret are now gone.

## Reference

https://k-orc.cloud/development/writing-tests/#dependency
