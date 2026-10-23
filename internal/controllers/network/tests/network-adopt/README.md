# Adopt an existing network

## Step 00

Create a project and a network in it with `onDelete: detach`.

## Step 01

Save the network's ID in a ConfigMap, then delete the network object. The OpenStack network is kept.

## Step 02

Remove the network's deletion guard from the project, as if it had never been added.

## Step 03

Create a network with the same spec. It adopts the existing network (same ID) and adds the deletion guard
to the project.

## Reference

https://github.com/k-orc/openstack-resource-controller/issues/1011
