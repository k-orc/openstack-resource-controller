# Check dependency handling for imported router

## Step 00

Import a router, with a `networkRef`, that we created as an imported resource. The imported network has no matching resource yet.
Verify the router is waiting for the dependency to be ready.

## Step 01

Create a router matching the import filter, except for the network it belongs to, and verify that it's not being imported.

## Step 02

Create a network and a router matching the import filters.

Verify that the observed status on the imported router corresponds to the spec of the created router.

## Step 03

Delete the network and check that ORC does not prevent deletion. The OpenStack network still exists because it was an imported network and we only deleted the ORC representation of it.

## Step 04

Delete the router and validate that all resources are gone.
