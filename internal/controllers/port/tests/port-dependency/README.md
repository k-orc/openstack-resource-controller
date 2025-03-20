# Creation and deletion dependencies

## Step 00

Create a port referencing a non-existing network, and verify that it is waiting for the network to be created externally.

## Step 01

Create a port referencing a non-existing subnet, and verify that it is waiting for the subnet to be created externally.

## Step 02

Create a port referencing a non-existing security group, and verify that it is waiting for the security group to be created externally.

## Step 03

Create the missing dependency, and verify that the port is now available.

## Step 04

Delete all the dependencies and check that ORC prevents deletion since there is still a resource that depends on them.

## Step 05

Delete the port and validate that all resources are gone.
