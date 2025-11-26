# Import KeyPair

## Step 00

Import a keypair, matching all of the available filter's fields, and verify it is waiting for the external resource to be created.

## Step 01

Create a keypair which name is a superstring of the one specified in the import filter, and otherwise matching the filter, and verify that it's not being imported.

## Step 02

Create a keypair matching the filter and verify that the observed status on the imported keypair corresponds to the spec of the created keypair.
Also verify that the created keypair didn't adopt the one which name is a superstring of it.

## Reference

https://k-orc.cloud/development/writing-tests/#import
