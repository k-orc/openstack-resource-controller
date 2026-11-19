# Update RBACPolicy

## Step 00

Create a RBACPolicy using only mandatory fields (RBACPolicy has no optional
fields).

## Step 01

Update targetProjectID, the only field Neutron's RBAC policy API allows mutating
(networkRef and action are immutable - see rbacpolicy_types.go).

## Step 02

Revert the resource to its original targetProjectID and verify that the
resulting object matches its state when first created.

## Reference

https://k-orc.cloud/development/writing-tests/#update
