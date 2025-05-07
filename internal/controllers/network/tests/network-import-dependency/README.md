# Check dependency handling for imported networks

## Step 00

Import a network, with a `projectRef`, that we created as an imported resource. The imported network has no matching resource yet.
Verify the network is waiting for the dependency to be ready.

## Step 01

Create a network matching the import filter, except for the project it belongs to, and verify that it's not being imported.

## Step 02

Create a project and a network matching the import filters.

Verify that the observed status on the imported network corresponds to the spec of the created network.

## Step 03

Delete the project and check that ORC does not prevent deletion. The OpenStack project still exists because it was an imported project and we only deleted the ORC representation of it.

## Step 04

Delete the network and validate that all resources are gone.
