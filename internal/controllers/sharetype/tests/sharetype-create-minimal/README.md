# sharetype-create-minimal

Creates a ShareType with the minimal required configuration (only driverHandlesShareServers set to true).

Validates that:
- The ShareType is created with the correct name
- Default isPublic is true
- The required driver_handles_share_servers extra spec is set
- Resource becomes Available
- Resource is deleted when credentials are removed
