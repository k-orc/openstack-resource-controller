# Create a PRIMARY DNSZone with all the options

## Step 00

Create a PRIMARY DNSZone using all the fields that apply to that type, and verify that the
observed state corresponds to the spec. See dnszone-create-full-secondary for the fields that
only apply to a SECONDARY zone.

Also validate that the OpenStack resource uses the name from the spec when it is specified.

## Reference

https://k-orc.cloud/development/writing-tests/#create-full
