# SwiftContainer credentials dependency

00. Create a SwiftContainer without the referenced Secret and verify it waits.
01. Create the Secret and verify the container becomes available.
02. Delete the Secret and verify the credential finalizer prevents deletion.
03. Delete the container and verify the Secret is then removed.
