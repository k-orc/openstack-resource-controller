# Import DNSZone (Unmanaged ID Scenario)

## Step 00

- Create the credentials secret `openstack-clouds`.
- Pre-create a DNS zone in Designate using the `openstack` CLI.
- Dynamically generate `01-import-resource.yaml` with the ID of the pre-created zone.

## Step 01

- Apply the unmanaged `DNSZone` CR with `spec.import.id` pointing to the pre-created zone.
- Assert that ORC imports the zone successfully and populates status correctly.

## Step 02

- Delete the `DNSZone` CR.
- Verify that the `DNSZone` CR is deleted from Kubernetes, but the actual zone still exists in Designate.
- Clean up the pre-created zone in Designate.
