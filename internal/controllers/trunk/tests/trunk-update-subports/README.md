# Update Trunk Subports

This test verifies that subports can be updated after trunk creation:
- Adding new subports
- Removing existing subports
- Updating subport segmentation (VLAN ID/type)

## Step 00
Create prerequisites: network, subnet, parent port, and three subport ports.

## Step 01
Create a trunk with two initial subports (subport1 with VLAN 100, subport2 with VLAN 200).

## Step 02
Update the trunk to:
- Change subport1 segmentation from VLAN 100 to 150
- Remove subport2
- Add subport3 with VLAN 300

## Step 03
Verify the trunk now has only subport1 (VLAN 150) and subport3 (VLAN 300).

