# RouterInterface status when Router is not Available

## Step 00

Create a Network, Subnet, and a Router which exists but is not Available (the
Router is waiting on a missing external network). Verify that the Router is
waiting for its dependency.

## Step 01

Create a RouterInterface referencing that Router. Verify that the
RouterInterface status reports that it is waiting for the Router to be ready,
rather than leaving status empty.

## Reference

https://github.com/k-orc/openstack-resource-controller/issues/838
