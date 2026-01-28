# Import User

## Step 00

Import a user that matches all fields in the filter, and verify it is waiting for the external resource to be created.

## Step 01

Create a user whose name is a superstring of the one specified in the import filter, otherwise matching the filter, and verify that it's not being imported.

## Step 02

Disable the first domain dependency so it can be deleted without issue by KUTTL during cleanup.

## Step 03

Create a user matching the filter and verify that the observed status on the imported user corresponds to the spec of the created user.
Also, confirm that it does not adopt any user whose name is a superstring of its own.

## Step 04

Disable the second domain dependency so it can be deleted without issue by KUTTL during cleanup.

## Reference

https://k-orc.cloud/development/writing-tests/#import
