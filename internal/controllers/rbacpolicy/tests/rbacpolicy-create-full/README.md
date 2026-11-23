# Create a RBACPolicy with all the options

## Step 00

Create an RBACPolicy using all the fields: the target project by reference (`targetProjectRef`,
`create-minimal` uses `targetProjectID`) and the other `action`, `access_as_external`, which
Neutron restricts to admins.

Verify that the observed state corresponds to the spec.

## Reference

https://k-orc.cloud/development/writing-tests/#create-full
