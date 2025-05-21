# Check dependency handling for imported floating IP

## Step 00

Import a floating IP that references other imported resources. The referenced imported resources has no matching resource yet.
Verify the floating IP is waiting for the dependency to be ready.

## Step 01

Create a floating IP matching the import filter, except for referenced resources, and verify that it's not being imported.

## Step 02

Create a the referenced resources and a floating IP matching the import filters.

Verify that the observed status on the imported floating IP corresponds to the spec of the created floating IP.

## Step 03

Delete the referenced resources and check that ORC does not prevent deletion. The OpenStack resource still exists because they
were imported resources and we only deleted the ORC representation of it.

## Step 04

Delete the floating IP and validate that all resources are gone.

## Reference

https://k-orc.cloud/development/writing-tests/#import-dependency
