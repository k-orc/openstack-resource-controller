# Create two servers to test adoption

## Step 00

Import a flavor, create an image,network,subnet and a port and then create a server with name 'create-adoption'.

## Step 01
Create another server with the name 'adoption'. The second server should have a resource name of 'adoption'

## Step 02

Validate we're able to delete resources.
Cleaning up resources also avoids a race where kuttl could delete the secret before the other resources.
