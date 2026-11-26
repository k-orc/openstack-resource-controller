# SwiftContainer import test

00. Import a container by name. Verify that ORC waits for it to be created.
01. Create a container whose name starts with the name in the import filter. Verify that ORC does not import it.
02. Create the container with the exact name. Verify that ORC imports it and reports its attributes.
