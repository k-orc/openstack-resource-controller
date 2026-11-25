# SwiftContainer safe deletion test

00. Create two empty containers and wait for Swift to report zero objects in each.
01. Upload an object to one container and wait for a periodic resync to report it.
02. Delete both CRs. The empty container is deleted; the non-empty container retains its finalizer and reports why deletion is waiting.
03. Remove the object from Swift and verify ORC completes deletion without another Kubernetes update.
