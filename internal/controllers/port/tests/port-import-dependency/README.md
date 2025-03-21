# Check dependency handling for imported port

## Step 00

Import a port, with a `networkRef`, that we created as an imported resource. The imported network has no matching resource yet.
Verify the port is waiting for the dependency to be ready.

## Step 01

Create a port matching the import filter, except for the network it belongs to, and verify that it's not being imported.

## Step 02

Create a network and a port matching the import filters.

Verify that the observed status on the imported port corresponds to the spec of the created port.

## Step 03

Delete the network and check that ORC does not prevent deletion. The OpenStack network still exists because it was an imported network and we only deleted the ORC representation of it.

## Step 04

Delete the port and validate that all resources are gone.
