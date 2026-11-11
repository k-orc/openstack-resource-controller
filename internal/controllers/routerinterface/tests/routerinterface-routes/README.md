# Remove an interface used by a route

## Step 00

Create a router with a route whose next hop is reachable through a router interface.

## Step 01

Delete the router interface and verify that it is not removed while the router's route uses it.

## Step 02

Delete the router and verify that both the router and its interface are deleted.

## Reference

https://k-orc.cloud/development/writing-tests/#dependency
