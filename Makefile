IMG_REPO ?= jkremser/log2rbac

# Image URL to use all building/pushing image targets
IMG ?= $(IMG_REPO):latest

# version
LAST_VERSION ?= $(shell git describe --tags --abbrev=0)
GIT_SHA ?= $(shell git rev-parse --short HEAD)
VERSION ?= $(LAST_VERSION)-$(GIT_SHA)
OTEL_VERSION ?= v0.48.0

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

GOLANG_VERSION ?= 1.19.1

.PHONY: all
all: build

##@ General

check_defined = \
    $(strip $(foreach 1,$1, \
        $(call __check_defined,$1,$(strip $(value 2)))))
__check_defined = \
    $(if $(value $1),, \
      $(error Undefined $1$(if $2, ($2))))

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-17s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen kustomize ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=log2rbac-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	#$(KUSTOMIZE) build config/default > deploy/all-in-one.yaml

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out

##@ Build

.PHONY: build
build: generate fmt vet ## Build log2rbac (manager) binary.
	go build -ldflags "-X main.gitSha=$(GIT_SHA) -X main.version=$(VERSION)" -o bin/log2rbac main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

GIT_COMMIT := $(shell git rev-list -1 HEAD)

.PHONY: container-img
container-img: ## Build container image with the manager.
	docker build --build-arg GOLANG_VERSION=$(GOLANG_VERSION) --build-arg GIT_SHA=$(GIT_SHA) --build-arg VERSION=$(VERSION) -f Dockerfile.multistage -t $(IMG) .

##@ Deployment

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=false -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image docker.io/jkremser/log2rbac:latest=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=false -f -

.PHONY: deploy-otel
deploy-otel: ## Deploy OpenTelemetry collector
	$(KUSTOMIZE) build config/otel | kubectl apply -f -
#	kubectl apply -n log2rbac -f https://raw.githubusercontent.com/open-telemetry/opentelemetry-collector/$(OTEL_VERSION)/examples/k8s/otel-config.yaml
#	kubectl -n log2rbac set env deploy/log2rbac -c manager OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector.log2rbac:4318 TRACING_ENABLED=true

.PHONY: undeploy-otel
undeploy-otel: ## Undeploy OpenTelemetry collector
	$(KUSTOMIZE) build config/otel | kubectl delete --ignore-not-found=false -f -
#	kubectl delete -n log2rbac --ignore-not-found=false -f https://raw.githubusercontent.com/open-telemetry/opentelemetry-collector/$(OTEL_VERSION)/examples/k8s/otel-config.yaml
#	kubectl -n log2rbac set env deploy/log2rbac -c manager OTEL_EXPORTER_OTLP_ENDPOINT- TRACING_ENABLED-

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest

## Tool Versions
KUSTOMIZE_VERSION ?= v3.8.7
CONTROLLER_TOOLS_VERSION ?= v0.10.0

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -s $(LOCALBIN)/kustomize || { curl -Ss $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest


##@ SLSA utils
.PHONY: container-digest
container-digest: ## retrieves the container digest from the given tag
	@:$(call check_defined, GITHUB_REF)
	@docker inspect ghcr.io/$(IMG_REPO):$(subst refs/tags/,,$(GITHUB_REF)) --format '{{ index .RepoDigests 0 }}' | cut -d '@' -f 2

.PHONY: manifest-digest
manifest-digest: ## retrieves the container digest from the given tag
	@:$(call check_defined, GITHUB_REF)
	@docker manifest inspect ghcr.io/$(IMG_REPO):$(subst refs/tags/,,$(GITHUB_REF)) | grep digest | cut -d '"' -f 4

.PHONY: container-tags
container-tags: ## retrieves the container tags applied to the image with a given digest
	@:$(call check_defined, CONTAINER_DIGEST)
	@docker inspect ghcr.io/$(IMG_REPO)@$(CONTAINER_DIGEST) --format '{{ join .RepoTags "\n" }}' | sed 's/.*://' | awk '!_[$$0]++'

.PHONY: container-repos
container-repos: ## retrieves the container repos applied to the image with a given digest
	@:$(call check_defined, CONTAINER_DIGEST)
	@docker inspect ghcr.io/$(IMG_REPO)@$(CONTAINER_DIGEST) --format '{{ join .RepoTags "\n" }}' | sed 's/:.*//' | awk '!_[$$0]++'
