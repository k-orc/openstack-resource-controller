# External deletion and recreation of a managed Network

## Step 00

Create a managed Network resource with a short `resyncPeriod` (10s) and wait
for ORC to create it in OpenStack and report it as available. Record the
OpenStack network ID assigned by ORC.

The `resyncPeriod` ensures ORC checks the network state periodically, allowing
it to detect external deletion without requiring a manual trigger or watch event.

## Step 01

Delete the OpenStack network directly (bypassing ORC). ORC detects the deletion
on the next periodic resync (within 10s), clears `status.id`, and recreates
the network in OpenStack on the following reconcile.

Verify that:
- The network is available again with correct conditions and resource status.
- The OpenStack ID in `status.id` has changed (a new network was created,
  confirming ORC detected the external deletion and recreated the resource).

## Reference

Tests the external deletion handling for managed resources as described in
`internal/controllers/generic/reconciler/resource_actions.go`: when a managed,
non-imported resource is found to be missing from OpenStack (the ID in
`status.id` no longer exists), ORC clears `status.id` and recreates the
resource on the next reconcile.
