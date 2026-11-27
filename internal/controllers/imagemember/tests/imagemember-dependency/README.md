# Creation and deletion dependencies

## Step 00

Create an ImageMember before its image and verify that it waits for the image.

## Step 01

Create the image without members and verify that the ImageMember waits for the owner to add the project. Verify that the image has no finalizer from the ImageMember.

## Step 02

Add the project as a member of the image and verify that the membership is accepted.

## Step 03

Delete the image and verify that the ImageMember does not prevent it.

## Reference

https://k-orc.cloud/development/writing-tests/#dependency
