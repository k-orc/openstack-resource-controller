# Image URL to use all building/pushing image targets
IMG ?= controller:latest
BUNDLE_IMG ?= bundle:latest
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.29.0
TRIVY_VERSION = 0.49.1
GO_VERSION ?= 1.23.10

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# Enables shell script tracing. Enable by running: TRACE=1 make <target>
TRACE ?= 0

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: modules
modules:
	go mod tidy

.PHONY: generate
generate: generate-resources generate-controller-gen generate-codegen generate-go generate-docs modules manifests

.PHONY: generate-resources
generate-resources:
	go run ./cmd/resource-generator

.PHONY: generate-controller-gen
generate-controller-gen: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: generate-codegen
generate-codegen: generate-controller-gen ## codegen requires DeepCopy etc
	./hack/update-codegen.sh

.PHONY: generate-go
generate-go: mockgen
	go generate ./...

.PHONY: generate-docs
generate-docs:
	$(MAKE) -C website generated

.PHONY: verify-generated
verify-generated: generate
	@if test -n "`git status --porcelain`"; then \
		git status; \
		git diff; \
		echo "generated files are out of date, run make generate"; exit 1; \
	fi

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: verify-fmt
verify-fmt: SRC := $(shell find . -path './.git' -prune -o -type f -name '*.go' -print)
verify-fmt: ## Errors if the code is not go-fmt'd.
	@UNFORMATTED="$$(gofmt -s -l $(SRC))"; \
	if [ -n "$$UNFORMATTED" ]; then \
		echo "Run go fmt ./... to fix these files:"; \
		echo "$$UNFORMATTED"; \
		exit 1; \
	fi

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
TEST_PATHS ?= ./...
test: envtest
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test $$(go list $(TEST_PATHS) | grep -v /e2e) -coverprofile cover.out

# Utilize Kind or modify the e2e tests to load the image locally, enabling compatibility with other vendors.
# The kuttl tests executed by test-e2e support the following environment
# variables:
# - E2E_OSCLOUDS: if set, the path to a clouds.yaml to use for the test run. If
#   not set, defaults to /etc/openstack/clouds.yaml.
# - E2E_KUTTL_DIR: if set, the path to a directory containing kuttl tests, e.g.
#   ./internal/controllers/flavor/tests. Only tests from this directory will
#   run. If not set, all discovered kuttl tests will run.
# - E2E_KUTTL_TEST: if set, only run the specific named test, e.g.
#   'create-full-v4'. If not set, all tests will run.
.PHONY: test-e2e  # Run the e2e tests against a Kind k8s instance that is spun up.
test-e2e: kuttl
	# go test ./test/e2e/ -v -ginkgo.v
	./hack/e2e.sh

.PHONY: test-examples
test-examples:
	./hack/run_examples.sh

.PHONY: lint
lint: golangci-kal ## Run golangci-kal linter
	$(GOLANGCI_KAL) run

.PHONY: lint-fix
lint-fix: golangci-kal ## Run golangci-lint linter and perform fixes
	$(GOLANGCI_KAL) run --fix

##@ Build

.PHONY: build
build: manifests generate fmt vet build-manager

.PHONY: build-manager
# Set build time variables including version details
build-manager: LDFLAGS ?= $(shell source ./hack/version.sh; version::get_git_vars; version::get_build_date; version::ldflags)
build-manager:
	go build -ldflags "${LDFLAGS}" -o bin/manager cmd/manager/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/manager/main.go

DOCKER_BUILD_ARGS ?=

# If you wish to build the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	source hack/version.sh && version::get_git_vars && version::get_build_date && \
	$(CONTAINER_TOOL) build \
		--tag ${IMG} \
		--build-arg "GO_VERSION=$(GO_VERSION)" \
		--build-arg "BUILD_DATE=$${BUILD_DATE}" \
		--build-arg "GIT_COMMIT=$${GIT_COMMIT}" \
		--build-arg "GIT_RELEASE_COMMIT=$${GIT_RELEASE_COMMIT}" \
		--build-arg "GIT_TREE_STATE=$${GIT_TREE_STATE}" \
		--build-arg "GIT_VERSION=$${GIT_VERSION}" \
		${DOCKER_BUILD_ARGS} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	$(CONTAINER_TOOL) push ${IMG}

# PLATFORMS defines the target platforms for the manager image be built to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - be able to use docker buildx. More info: https://docs.docker.com/build/buildx/
# - have enabled BuildKit. More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image to your registry (i.e. if you do not set a valid value via IMG=<myregistry/image:<tag>> then the export will fail)
# To adequately provide solutions that are compatible with multiple platforms, you should consider using this option.
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
.PHONY: docker-buildx
docker-buildx: ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	# Note that get_git_vars and get_build_date both honour environment
	# variable which are already set
	- source hack/version.sh && version::get_git_vars && version::get_build_date && \
	  $(CONTAINER_TOOL) buildx build \
		--platform=$(PLATFORMS) \
		--tag ${IMG} --push \
		--build-arg "GO_VERSION=$(GO_VERSION)" \
		--build-arg "BUILD_DATE=$${BUILD_DATE}" \
		--build-arg "GIT_COMMIT=$${GIT_COMMIT}" \
		--build-arg "GIT_RELEASE_COMMIT=$${GIT_RELEASE_COMMIT}" \
		--build-arg "GIT_TREE_STATE=$${GIT_TREE_STATE}" \
		--build-arg "GIT_VERSION=$${GIT_VERSION}" \
		-f Dockerfile.cross \
		$(DOCKER_BUILD_ARGS) .
	rm Dockerfile.cross

.PHONY: build-installer
build-installer: manifests generate kustomize ## Generate a consolidated YAML with CRDs and deployment.
	mkdir -p dist
	$(MAKE) custom-deploy IMG=${IMG}
	$(KUSTOMIZE) build $(CUSTOMDEPLOY) > dist/install.yaml

.PHONY: build-bundle-image
build-bundle-image: kustomize operator-sdk
	bash hack/bundle.sh
	$(CONTAINER_TOOL) build -f bundle.Dockerfile -t ${BUNDLE_IMG} .

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

# custom-deploy initialises a new kustomize module in $(CUSTOMDEPLOY) (deleting
# any existing directory first). This kustomize module:
# - includes config/default as a resource
# - overrides the controller image with the value of $(IMG)
# - adds an argument to the controller to set a custom log level if $(LOGLEVEL)
#   is set

define args_patch
- op: add
  path: /spec/template/spec/containers/0/args/-
  value: "-zap-log-level=$(LOGLEVEL)"
endef
export args_patch

CUSTOMDEPLOY ?= $(shell pwd)/.custom-deploy

.PHONY: custom-deploy
custom-deploy: customdeploy_relative = $(shell realpath -m --relative-to $(CUSTOMDEPLOY) $(shell pwd))
custom-deploy: kustomize
	if [ -d $(CUSTOMDEPLOY) ]; then rm -f $(CUSTOMDEPLOY)/kustomization.yaml && rmdir $(CUSTOMDEPLOY); fi
	mkdir -p $(CUSTOMDEPLOY)
	cd $(CUSTOMDEPLOY); $(KUSTOMIZE) create --resources $(customdeploy_relative)/config/default
	cd $(CUSTOMDEPLOY); $(KUSTOMIZE) edit set image controller=$(IMG)
	if [ -n "$(LOGLEVEL)" ]; then \
	  cd $(CUSTOMDEPLOY) && \
	  $(KUSTOMIZE) edit add patch --kind Deployment --namespace orc-system --name orc-controller-manager --patch "$$args_patch"; \
	fi

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	$(MAKE) custom-deploy IMG=${IMG}
	$(KUSTOMIZE) build $(CUSTOMDEPLOY) | $(KUBECTL) apply -f -

.PHONY: undeploy
undeploy: kustomize ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

##@ Security

.PHONY: verify-container-images
verify-container-images: ## Verify container images
	TRACE=$(TRACE) ./hack/verify-container-images.sh $(TRIVY_VERSION)

.PHONY: verify-govulncheck
verify-govulncheck: govulncheck ## Verify code for vulnerabilities
	$(GOVULNCHECK) ./... && R1=$$? || R1=$$?; \
	if [ "$$R1" -ne "0" ]; then \
		exit 1; \
	fi

.PHONY: verify-security
verify-security: ## Verify code and images for vulnerabilities
	$(MAKE) verify-container-images && R1=$$? || R1=$$?; \
	$(MAKE) verify-govulncheck && R2=$$? || R2=$$?; \
	if [ "$$R1" -ne "0" ] || [ "$$R2" -ne "0" ]; then \
	  echo "Check for vulnerabilities failed! There are vulnerabilities to be fixed"; \
		exit 1; \
	fi

##@ Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

export PATH := $(LOCALBIN):$(PATH)

## Tool Binaries
KUBECTL ?= kubectl
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint
GOLANGCI_KAL = $(LOCALBIN)/golangci-kube-api-linter
MOCKGEN = $(LOCALBIN)/mockgen
KUTTL = $(LOCALBIN)/kubectl-kuttl
GOVULNCHECK = $(LOCALBIN)/govulncheck
OPERATOR_SDK = $(LOCALBIN)/operator-sdk

## Tool Versions
KUSTOMIZE_VERSION ?= v5.6.0
CONTROLLER_TOOLS_VERSION ?= v0.17.1
ENVTEST_VERSION ?= release-0.19
GOLANGCI_LINT_VERSION ?= v2.0.1
KAL_VERSION ?= v0.0.0-20250531094218-f86bf7bd4b19
MOCKGEN_VERSION ?= v0.5.0
KUTTL_VERSION ?= v0.22.0
GOVULNCHECK_VERSION ?= v1.1.4
OPERATOR_SDK_VERSION ?= v1.41.1

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

define custom-gcl
version:  $(GOLANGCI_LINT_VERSION)
name: golangci-kube-api-linter
destination: $(LOCALBIN)
plugins:
- module: 'sigs.k8s.io/kube-api-linter'
  version: $(KAL_VERSION)
endef
export custom-gcl

CUSTOM_GCL_FILE ?= $(shell pwd)/.custom-gcl.yml

.PHONY: golangci-kal
golangci-kal: $(GOLANGCI_KAL)
$(GOLANGCI_KAL): $(LOCALBIN) $(GOLANGCI_LINT)
	$(file >$(CUSTOM_GCL_FILE),$(custom-gcl))
	$(GOLANGCI_LINT) custom

.PHONY: mockgen
mockgen: $(MOCKGEN) ## Download mockgen locally if necessary.
$(MOCKGEN): $(LOCALBIN)
	$(call go-install-tool,$(MOCKGEN),go.uber.org/mock/mockgen,$(MOCKGEN_VERSION))

.PHONY: kuttl
kuttl: $(KUTTL) ## Download kuttl locally if necessary.
$(KUTTL): $(LOCALBIN)
	$(call go-install-tool,$(KUTTL),github.com/kudobuilder/kuttl/cmd/kubectl-kuttl,$(KUTTL_VERSION))

.PHONY: govulncheck
govulncheck: $(GOVULNCHECK) ## Download govulncheck locally if necessary.
$(GOVULNCHECK): $(LOCALBIN)
	$(call go-install-tool,$(GOVULNCHECK),golang.org/x/vuln/cmd/govulncheck,$(GOVULNCHECK_VERSION))

.PHONY: operator-sdk
operator-sdk: $(OPERATOR_SDK) ## Download operator-sdk locally if necessary.
$(OPERATOR_SDK): $(LOCALBIN)
	$(call go-install-tool,$(OPERATOR_SDK),github.com/operator-framework/operator-sdk/cmd/operator-sdk,$(OPERATOR_SDK_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef

##@ helpers:

go-version: ## Print the go version we use to compile our binaries and images
	@echo $(GO_VERSION)
