# Creation and deletion dependencies

## Step 00

Create a subnet referencing a non-existing network, and verify that the subnet is waiting for the network to be created externally.

## Step 01

Create the network the subnet depends on, and verify that the subnet is now available.

## Step 02

Delete the network and check that ORC prevents deletion since there is still a resource that depends on it.

## Step 03

Delete the subnet and validate that all resources are gone.

## TODO

- Validate `routerRef`
