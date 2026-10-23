# Adopt an existing subnet

## Step 00

Create a network and a subnet with `onDelete: detach`.

## Step 01

Save the subnet's ID in a ConfigMap, then delete the subnet object. The OpenStack subnet is kept.

## Step 02

Remove the subnet's deletion guard from the network, as if it had never been added.

## Step 03

Create a subnet with the same spec. It adopts the existing subnet (same ID; a new one with the same CIDR
would be rejected) and adds the deletion guard to the network.

## Reference

https://github.com/k-orc/openstack-resource-controller/issues/1011
