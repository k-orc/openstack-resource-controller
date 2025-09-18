.PHONY: default
default: generated

CRD_REF_DOCS?=github.com/elastic/crd-ref-docs@v0.2.0
GOMARKDOC?=github.com/princjef/gomarkdoc/cmd/gomarkdoc@v1.1.0

websitedir := $(dir $(lastword $(MAKEFILE_LIST)))
topdir := $(websitedir)/..
godocdir := $(websitedir)/docs/development/godoc

.PHONY: generated
generated: crd_ref_docs interfaces

.PHONY: crd_ref_docs
crd_ref_docs:
	go run $(CRD_REF_DOCS) --config $(websitedir)/crd-ref-docs-config.yaml --output-path=$(websitedir)/docs/crd-reference.md --source-path=$(topdir)/api/v1alpha1 --renderer=markdown

.PHONY: interfaces
interfaces: | $(godocdir)
	go run $(GOMARKDOC) --repository.url "https://github.com/k-orc/openstack-resource-controller" --repository.default-branch main --repository.path / \
		$(topdir)/internal/controllers/generic/interfaces > $(godocdir)/generic-interfaces.md
	go run $(GOMARKDOC) --repository.url "https://github.com/k-orc/openstack-resource-controller" --repository.default-branch main --repository.path / \
		$(topdir)/internal/controllers/generic/progress > $(godocdir)/reconcile-status.md

$(godocdir):
	mkdir -p $(godocdir)
