# Design decisions

## Protection against double creation

Before creating any resources, ORC checks that it doesn't already exist in the OpenStack cloud. If a matching resource exists, and it is not already associated with an ORC resource, that one is associated to the ORC resource.

Because the OpenStack API is not the same for every resource, ORC must use different strategies based on the available tools.

For some resources, the API allows the caller to set an arbitrary ID upon creation. When that is possible, ORC uses it for retrieval. If the ID is not set in the resource spec, ORC computes one deterministically based on the name of the Kubernetes resource.

For some resources, the name is guaranteed to be unique. When that is the case, ORC uses it for retrieval.

For floating IPs, ORC appens a marker to the resource description.

For the other resources, orphan are retrieved by their properties.

Here's how ORC retrieves orphan resources for each type:
* Image: id
* Flavor: id
* Keypair: name
* FloatingIP: description
* Network: properties
* Port: properties
* Router: properties
* Security group: properties
* Security group rule: properties
* Server: properties
* Subnet: properties
