## ORC examples directory

This directory contains kustomize modules for generating some example resource
deployments.

Before executing any modules in this directory you must:
* Provide an appropriate clouds.yaml in local-config/clouds.yaml with a cloud
  name of 'openstack'
* Define an external network to use by creating
  local-config/external-network-filter.yaml
* Populate the dev-settings component and load the above credentials by running `make`

### Layout

* components
  - Contains utility components common to several modules
* bases
  - Contains modules containing example ORC resource definitions
  - Modify these if you want to change the OpenStack resource which are created by an example module
  - Not indended to be loaded directly
* apply
  - Contains modules which apply local namespacing to base modules and combine them with local credentials
  - These are the modules indended to be loaded directly
* local-config
  - Contains configuration local to the current environment

### Binary dependencies

The Makefile requires that appropriately up-to-date versions of the following binaries are available in `$PATH`:
* kubectl
