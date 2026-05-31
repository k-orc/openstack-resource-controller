# Terminal error resources don't resync

## Step 00

Create two networks with identical descriptions so that an import filter
matching on that description will find multiple results.

## Step 01

Attempt to import a network using a filter that matches both of the networks
created in step 00. This causes a terminal error (InvalidConfiguration) because
the import is ambiguous: the controller found more than one matching resource.

Also configure `resyncPeriod: 10s` on the failing resource to verify that the
terminal error state prevents the resync scheduler from enqueuing additional
reconciliations.

## Step 02

Wait 15 seconds (longer than the configured resyncPeriod) and verify that the
resource remains in the terminal error state. Specifically:
- Conditions still show InvalidConfiguration (terminal error unchanged).
- `lastSyncTime` is NOT set, because no successful reconciliation has occurred.
- The resource has NOT been re-reconciled (if resync fired, it might clear the
  error or change the condition message).

## Reference

Tests that resources in a terminal error state are excluded from the periodic
resync scheduler, as specified by acceptance criterion TS-008.
