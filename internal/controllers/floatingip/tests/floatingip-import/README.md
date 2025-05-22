# Import Floating IP

## Step 00

Import a floating IP, matching all of the available filter's fields, and verify it is waiting for the external resource to be created.

## Step 01

Create a floating IP which value is slightly different of the one specified in the import filter, and otherwise matching the filter, and verify that it's not being imported.

## Step 02

Create a floating IP matching the filter and verify that the observed status on the imported floating IP corresponds to the spec of the created floating IP.
Also verify that the created floating IP didn't adopt the one which name is a superstring of it.

## Reference

https://k-orc.cloud/development/writing-tests/#import
