# Create a LoadBalancer with all the options

## Step 00

Create a LoadBalancer using all available fields, and verify that the observed state corresponds to the spec.

Also validate that the OpenStack resource uses the name from the spec when it is specified.

projectRef needs admin credentials, so it is tested on a separate LoadBalancer; the main one uses
the regular credentials.

## Reference

https://k-orc.cloud/development/writing-tests/#create-full
