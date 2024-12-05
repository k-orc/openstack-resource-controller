default: kustomization.yaml

define KUSTOMIZATION
apiVersion: kustomize.config.k8s.io/v1alpha1
kind: Component
namePrefix: $(USER)-
endef
export KUSTOMIZATION

kustomization.yaml: Makefile
	echo "$$KUSTOMIZATION" > $@
