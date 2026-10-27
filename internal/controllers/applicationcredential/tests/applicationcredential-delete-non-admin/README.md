# Delete an ApplicationCredential using non-admin credentials

Regression coverage for [#1022](https://github.com/k-orc/openstack-resource-controller/issues/1022).

## Step 00

Verify that the regular `openstack` cloud cannot list Keystone users: the
request must fail with HTTP 403. Other failures, or a successful listing,
fail the test rather than bypassing the regression scenario.

Read the authenticated user's ID from its token and import an unmanaged ORC
User through `openstack-admin`. No OpenStack user is created or deleted.

## Step 01

Create a managed ApplicationCredential through the regular user's cloud,
referencing that same User. Assert that it becomes Available, has an OpenStack
ID, and carries its controller finalizer.

## Step 02

Explicitly delete the ApplicationCredential and wait for the Kubernetes
object to disappear. This requires the controller to confirm the OpenStack
credential is gone and release its finalizer despite lacking permission to
list users.

The previous implementation falls back to listing users after the credential
lookup returns 404, and cannot complete this step.
