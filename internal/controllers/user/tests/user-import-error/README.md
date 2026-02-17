# Import User with more than one matching resources

## Step 00

Create two Users with identical specs.

## Step 01

Ensure that an imported User with a filter matching the resources returns an error.

## Step 02

Disable the domain dependency so KUTTL can clean the resources without failing.

## Reference

https://k-orc.cloud/development/writing-tests/#import-error