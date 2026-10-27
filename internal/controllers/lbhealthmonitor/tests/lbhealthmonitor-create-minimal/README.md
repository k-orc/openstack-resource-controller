# Create a LBHealthMonitor with the minimum options

## Step 00

Create a TCP health monitor on a pool with only the required fields, and verify the observed state, including Octavia's default for maxRetriesDown.

## Step 01

Try deleting the secret and ensure that it is not deleted thanks to the finalizer.

## Reference

https://k-orc.cloud/development/writing-tests/
