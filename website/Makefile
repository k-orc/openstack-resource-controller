.PHONY: default
default: generated

CRD_REF_DOCS?=github.com/elastic/crd-ref-docs@v0.1.0

websitedir := $(dir $(lastword $(MAKEFILE_LIST)))
topdir := $(websitedir)/..

.PHONY: generated
generated: crd_ref_docs

.PHONY: crd_ref_docs
crd_ref_docs:
	go run $(CRD_REF_DOCS) --config $(websitedir)/crd-ref-docs-config.yaml --output-path=$(websitedir)/docs/crd-reference.md --source-path=$(topdir)/api/v1alpha1 --renderer=markdown
