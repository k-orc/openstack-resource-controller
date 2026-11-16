# Import DNSZoneShare

## Step 00

Import a DNSZoneShare, matching all of the available filter's fields (targetProjectID), and
verify it is waiting for the external resource to be created.

## Step 01

Create a DNSZoneShare with a different targetProjectID than the one specified in the import
filter, and verify that it's not being imported.

## Step 02

Create a DNSZoneShare matching the filter and verify that the observed status on the imported
DNSZoneShare corresponds to the spec of the created DNSZoneShare. Also verify that it didn't
adopt the DNSZoneShare with the different targetProjectID.

## Reference

https://k-orc.cloud/development/writing-tests/#import
