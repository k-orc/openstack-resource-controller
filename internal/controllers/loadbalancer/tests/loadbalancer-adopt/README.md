# Adopt an existing load balancer

## Step 00

Create a load balancer with `onDelete: detach`.

## Step 01

Save the load balancer's ID in a ConfigMap, then delete the load balancer object. The OpenStack
load balancer is kept.

## Step 02

Create a load balancer with the same spec. It adopts the existing load balancer (same ID).
