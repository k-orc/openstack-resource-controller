# Import DNSZoneShare with more than one matching resources

## Step 00

Create two DNSZoneShares sharing the same targetProjectID (DNSZoneShareFilter only matches on
that field, not the zone), each on a different zone since Designate rejects an exact duplicate
share on the same zone.

## Step 01

Ensure that an imported DNSZoneShare with a filter matching both resources returns an error.

## Reference

https://k-orc.cloud/development/writing-tests/#import-error
