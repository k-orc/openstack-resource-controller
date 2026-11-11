# Listener default pool

## Step 00

Create a load balancer, a pool on it (`loadBalancerRef`) and a listener with that pool as its
`defaultPoolRef`. Verify that the listener's default pool is the pool, and that the pool has the
listener's deletion guard.
