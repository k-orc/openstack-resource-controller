# Create Limits with the minimum options

## Step 00

Create two minimal Limits that set only the required fields (serviceRef/projectRef/resourceName/resourceLimit and serviceRef/domainRef/resourceName/resourceLimit), and verify that the observed state corresponds to the spec. `projectRef` and `domainRef` are mutual exclusive, so the only field that is really optional is the description field.

## Step 01

Try deleting the secret and other dependencies ensure that they are not deleted thanks to the finalizer.
