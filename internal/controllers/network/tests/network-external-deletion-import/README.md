# External deletion of an imported (unmanaged) Network produces a terminal error

## Step 00

Create an external managed Network that will be used as the import target,
and wait for it to become available in OpenStack.

## Step 01

Import the external network into ORC as an unmanaged resource (using an import
filter). Verify the import succeeds and the network is available.

## Step 02

Delete the external OpenStack network directly (bypassing ORC). On the next
reconcile, ORC detects that the network referenced by `status.id` no longer
exists in OpenStack. Because the resource was originally imported (unmanaged),
ORC cannot recreate it - instead it sets a terminal error condition
(`UnrecoverableError`) with the message "resource has been deleted from
OpenStack". No further reconciliation occurs.

## Reference

Tests the external deletion handling for imported/unmanaged resources as
described in `resource_actions.go`: when a resource was originally imported
and is found to be missing from OpenStack, ORC returns a terminal error instead
of attempting recreation.
