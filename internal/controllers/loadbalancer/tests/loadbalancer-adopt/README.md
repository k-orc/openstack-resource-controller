# Adopt an existing load balancer

## Step 00

Create a load balancer with `onDelete: detach`.

## Step 01

Save the load balancer's ID in a ConfigMap, then delete the load balancer object. The OpenStack
load balancer is kept.

## Step 02

Remove the load balancer's deletion guard from the subnet, as if it had never been added.

## Step 03

Create a load balancer with the same spec. It adopts the existing load balancer (same ID) and adds
the deletion guard to the subnet.

## Reference

https://github.com/k-orc/openstack-resource-controller/issues/1011
