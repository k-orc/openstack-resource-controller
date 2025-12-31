# Boot from Volume Test

This test creates a server that boots from a Cinder volume instead of an
image. This is the boot-from-volume (BFV) pattern where:

1. An image is created
2. A bootable volume is created from that image
3. A server is created booting from the volume (no imageRef)

The test verifies:
- Server reaches ACTIVE state
- Volume is marked as bootable
- Volume is attached to the server
- Port is attached to the server
