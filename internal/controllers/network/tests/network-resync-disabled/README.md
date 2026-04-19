# Network with resyncPeriod=0 disables periodic resync

## Step 00

Create a network with `resyncPeriod: 0s` (disabled periodic resync) and verify that:
- The network becomes available with correct conditions.
- `lastSyncTime` is set in the status after the first successful reconciliation.
  (Even with resync disabled, the controller always records the initial sync time.)

## Step 01

Wait for a period longer than the minimum resync period and verify that
`lastSyncTime` has NOT changed. When `resyncPeriod` is 0 (disabled), the
controller does not schedule additional reconciliations, so `lastSyncTime`
should remain stable after the initial reconciliation.

## Reference

Tests that setting `resyncPeriod: 0s` (or omitting resyncPeriod) disables
periodic resync scheduling. The resource is still reconciled on events (spec
changes, dependency updates) but not on a timer.
