.PHONY: default
default: load-credentials

KUBECTL ?= kubectl

##
## dev-settings
##
## Populates the dev-settings kustomization component, which prefixes all
## emitted objects with the current user's username. This is intended to
## prevent resource naming conflicts when using a shared tenant for development.

dev_settings_kustomization=components/dev-settings/kustomization.yaml

.PHONY: dev-settings
dev-settings: $(dev_settings_kustomization)

define KUSTOMIZATION
apiVersion: kustomize.config.k8s.io/v1alpha1
kind: Component

namePrefix: $(USER)-
endef
export KUSTOMIZATION

$(dev_settings_kustomization): Makefile
	echo "$$KUSTOMIZATION" > $@

##
## check-credentials
##
## Emits an error message if clouds.yaml has not been populated.

credentials=bases/credentials/clouds.yaml

.PHONY: check-credentials
check-credentials: $(credentials)

$(credentials):
	$(error You must copy an appropriate clouds.yaml to $(PWD)/$(credentials). The name of the contained cloud must be 'openstack'.)

##
## Utility targets
##

.PHONY: load-credentials
load-credentials: check-credentials dev-settings
	$(KUBECTL) apply -k apply/credentials --server-side