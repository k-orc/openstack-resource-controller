# Check dependency handling for imported LBHealthMonitor

## Step 00

Import a health monitor by the pool it checks, where the pool is itself imported and doesn't exist yet. Verify that it waits for the pool.

## Step 01

Create a trap health monitor on another pool. Verify that it is not imported.

## Step 02

Create the pool the import filter matches and a health monitor on it. Verify that it is imported.

## Step 03

Delete the imported pool object and verify that nothing stops it.

## Step 04

Delete the health monitor object.

## Reference

https://k-orc.cloud/development/writing-tests/
