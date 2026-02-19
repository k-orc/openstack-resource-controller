# Import AddressScope

## Step 00

Import a addressscope that matches all fields in the filter, and verify it is waiting for the external resource to be created.

## Step 01

Create a addressscope whose name is a superstring of the one specified in the import filter, otherwise matching the filter, and verify that it's not being imported.

## Step 02

Create a addressscope matching the filter and verify that the observed status on the imported addressscope corresponds to the spec of the created addressscope.
Also, confirm that it does not adopt any addressscope whose name is a superstring of its own.

## Reference

https://k-orc.cloud/development/writing-tests/#import
