# sharetype-import-error

Tests error handling when import filter doesn't match any existing resource.

The test:
1. Creates an unmanaged ShareType with import filter for non-existent name
2. Verifies it enters and stays in Progressing state waiting for the resource
3. Verifies no ID is assigned (resource not found)

This ensures graceful handling when import criteria don't match anything.
