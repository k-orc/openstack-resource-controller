# Creation and deletion dependencies

## Step 00

Create a routerinterface referencing a non-existing subnet, and verify that it is waiting for the subnet to be created externally.

## Step 01

Create a routerinterface referencing a non-existing router, and verify that it is waiting for the router to be created externally.

## Step 02

Create the missing dependency, and verify that the routerinterface is now available.

## Step 03

Delete all the dependencies and check that ORC prevents deletion since there is still a resource that depends on them.

## Step 04

Delete the routerinterface and validate that all resources are gone.
