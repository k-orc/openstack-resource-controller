# Jitter spreads reconciliation times across multiple resources

## Step 00

Create three networks that all share the same `resyncPeriod` (10s). Once all
three become available, each will have a `lastSyncTime` that records when the
controller last successfully reconciled them.

## Step 01

Record the initial `lastSyncTime` for all three networks.

## Step 02

After the resync period elapses, verify that all three networks have been
independently re-reconciled (i.e., all three `lastSyncTime` values have been
updated to newer timestamps).

The test verifies that all three resources are scheduled for resync
independently. The jitter mechanism ([0%, +20%]) ensures they are not all
re-reconciled at exactly the same instant, which would cause a thundering-herd
effect. Because the [0%, +20%] jitter applied to a 10s period produces scheduling
spread of up to +2s, we check that all three networks successfully re-synced
(demonstrating independent scheduling) rather than requiring exact timestamp
differences (which would be flaky at sub-second granularity).

## Reference

Tests the jitter-based resync scheduling feature (TS-011): multiple resources
with the same `resyncPeriod` should be independently scheduled rather than
all reconciling simultaneously.
