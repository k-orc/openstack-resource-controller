# Primary DNSZone Lifecycle E2E Test

This test verifies the end-to-end lifecycle of a primary managed `DNSZone` (create, assert status, update, delete).

## Step 00

Creates a primary managed `DNSZone` CR with:
- `name`: `lifecycle-primary.example.com.`
- `email`: `admin@example.com`
- `description`: `"Primary DNS Zone lifecycle test"`
- `ttl`: `3600`
- `type`: `PRIMARY`

And verifies that:
- It transition to status `ACTIVE`
- It generates `status.id`
- It populates the observed fields in `status.resource` matching the real Designate values
- Both `Available` and `Progressing` conditions are successfully reconciled

## Step 01

Modifies the mutable fields on the created `DNSZone` CR:
- `email`: `newadmin@example.com`
- `description`: `"Primary DNS Zone lifecycle test - Updated"`
- `ttl`: `7200`

And asserts that the updates successfully propagate to Designate and the CR's `status.resource` is updated accordingly.

## Step 02

Deletes the `DNSZone` CR and verifies that the clean deletion is triggered, the finalizers execute, and the zone is completely removed.
