# DNSZone Errors (Terminal Error with Non-Existent Zone ID)

## Description

Verify that importing a non-existent zone ID produces a terminal error condition in status.

## Steps

### Step 00

- Create the credentials secret `openstack-clouds`.
- Apply a `DNSZone` CR with `spec.managementPolicy="unmanaged"` and `spec.import.id` pointing to a non-existent zone ID.
- Assert that the `DNSZone` resource transitions to a terminal error state (`UnrecoverableError` reason and `"referenced resource does not exist in OpenStack"` message).
