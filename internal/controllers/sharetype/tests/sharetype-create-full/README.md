# sharetype-create-full

Creates a ShareType with all available configuration options:
- Custom name override
- isPublic set to false (private share type)
- driverHandlesShareServers set to false
- snapshotSupport set to true

Validates that:
- The ShareType is created with the overridden name
- isPublic is false
- All extra specs are correctly set
- Resource becomes Available
- Resource is deleted when credentials are removed
