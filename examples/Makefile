.PHONY: default
default: local-config

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
## check-prereqs
##
## Emits an error message if clouds.yaml has not been populated.

prereqs=local-config/clouds.yaml local-config/external-network-filter.yaml

.PHONY: check-prereqs
check-local-config: $(prereqs)

$(prereqs):
	$(error You must pre-create $(prereqs). See README.md for details.)

##
## Utility targets
##

.PHONY: local-config
local-config: check-prereqs dev-settings
	$(KUBECTL) apply -k apply/local-config --server-side
