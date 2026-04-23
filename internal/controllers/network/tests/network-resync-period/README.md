# Network resync after configured period

## Step 00

Create a network with a short `resyncPeriod` (10s) and verify that:
- The network becomes available with correct conditions.
- `lastSyncTime` is set in the status after the first successful reconciliation.

## Step 01

Wait for the resync period to elapse, then verify that `lastSyncTime` is updated
to a newer timestamp. This confirms that the controller re-reconciles the network
after the configured period and writes a fresh `lastSyncTime`.

## Reference

Tests the resync scheduling feature: a resource with a configured `resyncPeriod`
should be periodically re-reconciled and `lastSyncTime` updated accordingly.
