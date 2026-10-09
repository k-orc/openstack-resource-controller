# Create a Router with static routes

## Step 00

Create a Router with an interface on its own subnet, and a static route whose next hop is
reachable via that interface. Verify the route is reflected in status - Neutron only accepts
routes on update, never at creation time, so this also exercises that the update reconciler
picks it up on a later pass rather than only at creation.

## Reference

https://k-orc.cloud/development/writing-tests/#create-full
