# Import Group

## Step 00

Import a group, matching all of the available filter's fields, and verify it is waiting for the external resource to be created.

## Step 01

Create a group which name is a superstring of the one specified in the import filter, and otherwise matching the filter, and verify that it's not being imported.

## Step 02

Disable the first domain dependency so it can be deleted without issue by KUTTL during cleanup. 

## Step 03

Create a group matching the filter and verify that the observed status on the imported group corresponds to the spec of the created group.
Also verify that the created group didn't adopt the one which name is a superstring of it.

## Step 04

Disable the second domain dependency so it can be deleted without issue by KUTTL during cleanup.

## Reference

https://k-orc.cloud/development/writing-tests/#import
