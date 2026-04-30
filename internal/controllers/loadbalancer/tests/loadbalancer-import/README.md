# Import LoadBalancer

## Step 00

Create the network and subnet infrastructure needed as VIP resources for the
external load balancers, and submit an import resource using all available
filter fields (name, description, tags). Verify the import is waiting for an
external resource matching the filters to exist.

## Step 01

Create a load balancer whose name is a superstring of the one specified in the
import filter (`loadbalancer-import-external-not-this-one` vs
`loadbalancer-import-external`), but otherwise matching the filter. Verify
that this resource is not being imported -- it validates that we don't perform
regex-based name search.

## Step 02

Create a load balancer matching all of the import filters, including the exact
name. Verify that the imported resource is available and the observed status
corresponds to that of the created resource. Also verify that the previously
created "trap" resource wasn't imported as the new one, again, because of
regex-based name matching.

## Reference

https://k-orc.cloud/development/writing-tests/#import
