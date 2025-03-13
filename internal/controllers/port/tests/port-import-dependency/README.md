# Check dependency handling for imported port

## Step 00

Import a port, with a `networkRef`, and verify it is waiting for the external resource to be created.

## Step 01

Create a port matching the import filter, except for the network it belongs to, and verify that it's not being imported.

## Step 02

Create the network the port depends on.
Create a port matching the filter and verify that the observed status on the imported port corresponds to the spec of the created port.

## Step 03

Delete the network and check that ORC prevents deletion since there is still a resource that depends on it.

## Step 04

Delete the port and validate that all resources are gone.
