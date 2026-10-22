# Import RBACPolicy

## Step 00

Import a RBACPolicy, matching all of the available filter's fields (action,
targetProjectID), and verify it is waiting for the external resource to be
created.

## Step 01

Create a RBACPolicy with a different targetProjectID than the one specified in
the import filter, and verify that it's not being imported.

## Step 02

Create a RBACPolicy matching the filter and verify that the observed status on
the imported RBACPolicy corresponds to the spec of the created RBACPolicy.
Also verify that it didn't adopt the RBACPolicy with the different
targetProjectID.

## Reference

https://k-orc.cloud/development/writing-tests/#import
