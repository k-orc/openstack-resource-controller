# Creation and deletion dependencies

## Step 00

Create health monitors referencing a secret and a pool that don't exist yet, and verify that they wait.

## Step 01

Create the missing dependencies and verify that the health monitors become available.

## Step 02

Delete a pool and a secret in use, and verify that the finalizers stop their deletion.

## Step 03

Delete the health monitors and verify that the dependencies are deleted too.

## Reference

https://k-orc.cloud/development/writing-tests/
