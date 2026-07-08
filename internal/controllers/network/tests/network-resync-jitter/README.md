# Network resync with jitter

## Step 00

Create three networks that all share the same `resyncPeriod` (10s). Once all
three become available, each will have a `lastSyncTime` that records when the
controller last successfully reconciled them.

## Step 01

Record the initial `lastSyncTime` for all three networks.

## Step 02

After the resync period elapses, verify that the three recorded timestamps have
advanced and that at least one elapsed interval differs. This confirms both
periodic resync and jittered scheduling for multiple resources using the same
period without requiring every random jitter sample to be unique.

## Reference

Tests periodic resync scheduling and jitter: resources with the same
`resyncPeriod` should be independently scheduled rather than all reconciling
simultaneously.
