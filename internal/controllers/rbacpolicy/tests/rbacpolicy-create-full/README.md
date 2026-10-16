# Create a RBACPolicy with all the options

## Step 00

RBACPolicy has no optional fields - networkRef, action and targetProjectID are
all required, so there's nothing extra to add on top of `create-minimal`.
Instead, this test exercises the other `action` enum value
(`access_as_external`, vs. `access_as_shared` in `create-minimal`) to get full
coverage of the resource spec's possible values.

Verify that the observed state corresponds to the spec.

## Reference

https://k-orc.cloud/development/writing-tests/#create-full
