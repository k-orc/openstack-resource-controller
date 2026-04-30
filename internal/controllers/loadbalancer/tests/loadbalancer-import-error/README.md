# Import LoadBalancer with more than one matching resource

## Step 00

Create two load balancers with identical tags (`tag1`, `tag2`). Each load
balancer is created on its own network and subnet. Verify that both are
available before proceeding.

## Step 01

Ensure that an import resource with a filter matching both load balancers (by
tags) returns an error with `InvalidConfiguration` reason and
`Available=False`, `Progressing=False`.

## Reference

https://k-orc.cloud/development/writing-tests/#import-error
