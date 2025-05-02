# Creation and deletion dependencies

## Step 00

Create a server referencing a non-existing flavor, and verify that it is waiting for the flavor to be created externally.

## Step 01

Create a server referencing a non-existing image, and verify that it is waiting for the image to be created externally.

## Step 02

Create a server referencing a non-existing port, and verify that it is waiting for the port to be created externally.

## Step 03

Create a server referencing a non-existing user-data secret, and verify that it is waiting for the secret to be created externally.

## Step 04

Create the missing dependency, and verify that the server is now available.

## Step 05

Delete all the dependencies and check that ORC prevents deletion of image and port since there is still a resource that depends on them.
Verify that flavor and user-data secrets are deleted.

## Step 06

Delete the server and validate that all resources are gone.


## Note

The server-dependency test has a different pattern to the other resources
dependency tests to limit the risks of getting an instance in ERROR state on
smaller devstacks. We create the instances serially rather than in parallel.
