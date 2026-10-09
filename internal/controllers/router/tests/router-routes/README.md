# Create a Router with static routes

## Step 00

Create a Router with an interface on its own subnet, and a static route whose next hop is
reachable via that interface. Verify the route is reflected in status - Neutron only accepts
routes on update, never at creation time, so this also exercises that the update reconciler
picks it up on a later pass rather than only at creation.

## Step 01

Clear the router's routes before the test ends. Neutron refuses to remove a router interface
while a route still depends on its subnet (RouterInterfaceInUseByRoute) - and deleting the
Router itself doesn't sidestep this, since K-ORC won't start that delete until the
RouterInterface referencing it is gone first, which creates a deadlock. Clearing just the route
breaks the cycle: the interface removal in the implicit end-of-test cleanup that follows has
nothing left blocking it. Confirmed live - the first two attempts at this teardown step both
failed in exactly this way before landing on clearing the route specifically.

## Reference

https://k-orc.cloud/development/writing-tests/#create-full
