## ORC examples directory

This directory contains kustomize modules for generating some example resource
deployments.

Before executing any modules in this directory you must:
* Populate the dev-settings component by executing `make dev-settings`
* Provide an appropriate clouds.yaml in bases/credentials/clouds.yaml with a
  cloud name of 'openstack'
* Create a secret containing your clouds.yaml by executing `make
  load-credentials`.

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