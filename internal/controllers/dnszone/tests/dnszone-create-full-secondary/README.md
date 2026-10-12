# Create a SECONDARY DNSZone with all the options

## Step 00

Create a SECONDARY DNSZone using all the fields that apply to that type (masters, no email/ttl),
and verify that the observed state corresponds to the spec. See dnszone-create-full-primary for
the PRIMARY-only fields.

Also validate that the OpenStack resource uses the name from the spec when it is specified.

## Reference

https://k-orc.cloud/development/writing-tests/#create-full
