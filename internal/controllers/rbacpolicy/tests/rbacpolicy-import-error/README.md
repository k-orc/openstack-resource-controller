# Import RBACPolicy with more than one matching resources

## Step 00

Create two RBACPolicies sharing the same action and targetProjectID (RBACPolicyFilter
only matches on those two fields, not on the network), each attached to a different
network since Neutron rejects an exact duplicate policy on the same network.

## Step 01

Ensure that an imported RBACPolicy with a filter matching both resources returns an
error.

## Reference

https://k-orc.cloud/development/writing-tests/#import-error
