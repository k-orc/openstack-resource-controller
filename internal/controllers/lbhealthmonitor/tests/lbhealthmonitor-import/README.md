# Import LBHealthMonitor

## Step 00

Import a health monitor that matches a filter on name and type, before it exists. Verify that it waits.

## Step 01

Create a health monitor whose name is a superstring of the filter's on another pool. Verify that it is not imported.

## Step 02

Create a health monitor matching the filter and verify that it is imported.

## Reference

https://k-orc.cloud/development/writing-tests/
