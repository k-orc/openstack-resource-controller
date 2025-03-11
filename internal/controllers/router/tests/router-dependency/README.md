# Creation and deletion dependencies

## Step 00

Create a router referencing a non-existing network, and verify that the router is waiting for the network to be created externally.

## Step 01

Create the network the router depends on, and verify that the router is now available.

## Step 02

Delete the network and check that ORC prevents deletion since there is still a resource that depends on it.

## Step 03

Delete the router and validate that all resources are gone.
