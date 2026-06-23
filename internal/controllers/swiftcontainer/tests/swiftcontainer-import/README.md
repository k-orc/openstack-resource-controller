# Import SwiftContainer

## Step 00

Import a swiftcontainer that matches all fields in the filter, and verify it is waiting for the external resource to be created.

## Step 01

Create a swiftcontainer whose name is a superstring of the one specified in the import filter, otherwise matching the filter, and verify that it's not being imported.

## Step 02

Create a swiftcontainer matching the filter and verify that the observed status on the imported swiftcontainer corresponds to the spec of the created swiftcontainer.
Also, confirm that it does not adopt any swiftcontainer whose name is a superstring of its own.

## Reference

https://k-orc.cloud/development/writing-tests/#import
