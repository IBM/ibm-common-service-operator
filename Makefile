# Copyright 2020 IBM Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

.DEFAULT_GOAL:=help

# Dependence tools
KUBECTL ?= $(shell which kubectl)
OPERATOR_SDK ?= $(shell which operator-sdk)
CONTROLLER_GEN ?= $(shell which controller-gen)
KUSTOMIZE ?= $(shell which kustomize)

# Specify whether this repo is build locally or not, default values is '1';
# If set to 1, then you need to also set 'DOCKER_USERNAME' and 'DOCKER_PASSWORD'
# environment variables before build the repo.
BUILD_LOCALLY ?= 1

VCS_URL ?= https://github.com/IBM/ibm-common-service-operator
VCS_REF ?= $(shell git rev-parse HEAD)
VERSION ?= $(shell git describe --exact-match 2> /dev/null || \
                git describe --match=$(git rev-parse --short=8 HEAD) --always --dirty --abbrev=8)
RELEASE_VERSION ?= $(shell cat ./version/version.go | grep "Version =" | awk '{ print $$3}' | tr -d '"')

LOCAL_OS := $(shell uname)
ifeq ($(LOCAL_OS),Linux)
    TARGET_OS ?= linux
    XARGS_FLAGS="-r"
	STRIP_FLAGS=
else ifeq ($(LOCAL_OS),Darwin)
    TARGET_OS ?= darwin
    XARGS_FLAGS=
	STRIP_FLAGS="-x"
else
    $(error "This system's OS $(LOCAL_OS) isn't recognized/supported")
endif

ARCH := $(shell uname -m)
LOCAL_ARCH := "amd64"
ifeq ($(ARCH),x86_64)
    LOCAL_ARCH="amd64"
else ifeq ($(ARCH),ppc64le)
    LOCAL_ARCH="ppc64le"
else ifeq ($(ARCH),s390x)
    LOCAL_ARCH="s390x"
else
    $(error "This system's ARCH $(ARCH) isn't recognized/supported")
endif

# Default image repo
IMAGE_REPO ?= quay.io/opencloudio

ifeq ($(BUILD_LOCALLY),0)
REGISTRY ?= "hyc-cloud-private-integration-docker-local.artifactory.swg-devops.com/ibmcom"
else
REGISTRY ?= "hyc-cloud-private-scratch-docker-local.artifactory.swg-devops.com/ibmcom"
endif

# Current Operator image name
OPERATOR_IMAGE_NAME ?= common-service-operator
# Current Operator bundle image name
BUNDLE_IMAGE_NAME ?= common-service-operator-bundle
# Current Operator version
OPERATOR_VERSION ?= 3.5.0

# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

ifeq ($(BUILD_LOCALLY),0)
    export CONFIG_DOCKER_TARGET = config-docker
    export CONFIG_DOCKER_TARGET_QUAY = config-docker-quay
endif

include common/Makefile.common.mk

##@ Development

check: lint-all ## Check all files lint error
	./common/scripts/lint-csv.sh

code-dev: ## Run the default dev commands which are the go tidy, fmt, vet then execute the $ make code-gen
	@echo Running the common required commands for developments purposes
	- make generate-all
	- make code-tidy
	- make code-fmt
	- make code-vet
	@echo Running the common required commands for code delivery
	- make check

build: ## Build manager binary
	go build -o bin/manager main.go

run: generate code-fmt code-vet manifests ## Run against the configured Kubernetes cluster in ~/.kube/config
	OPERATOR_NAME=ibm-common-service-operator go run ./main.go -v=2

install: manifests ## Install CRDs into a cluster
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests ## Uninstall CRDs from a cluster
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: manifests ## Deploy controller in the configured Kubernetes cluster in ~/.kube/config
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMAGE_REPO)/$(OPERATOR_IMAGE_NAME):$(OPERATOR_VERSION)
	$(KUSTOMIZE) build config/default | kubectl apply -f -

##@ Generate code and manifests

manifests: ## Generate manifests e.g. CRD, RBAC etc.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=common-service-operator webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: ## Generate code e.g. API etc.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

bundle-manifests:
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle \
	-q --overwrite --version $(OPERATOR_VERSION) $(BUNDLE_METADATA_OPTS)
	$(OPERATOR_SDK) bundle validate ./bundle

package-manifests:
ifeq (, $(wildcard deploy/olm-catalog/ibm-common-service-operator/$(OPERATOR_VERSION)))
	@mkdir deploy/olm-catalog/ibm-common-service-operator/$(OPERATOR_VERSION)
endif
	@cp -rf bundle/manifests/ibm-common-service-operator.clusterserviceversion.yaml deploy/olm-catalog/ibm-common-service-operator/$(OPERATOR_VERSION)/ibm-common-service-operator.v$(OPERATOR_VERSION).clusterserviceversion.yaml
	@cp -rf bundle/manifests/operator.ibm.com_commonservices.yaml deploy/olm-catalog/ibm-common-service-operator/$(OPERATOR_VERSION)/operator.ibm.com_commonservices_crd.yaml

generate-all: generate manifests ## Generate bundle manifests, metadata and package manifests
	$(OPERATOR_SDK) generate kustomize manifests -q
	- make bundle-manifests CHANNELS=dev DEFAULT_CHANNEL=dev
	- make package-manifests

##@ Test

test: ## Run unit test on prow
	@echo good

e2e-test: ## Run e2e test
	@echo "Running e2e tests for the controllers."
	@USE_EXISTING_CLUSTER=true \
	OPERATOR_NAME=ibm-common-service-operator \
	OPERATOR_NAMESPACE=ibm-common-services \
	go test ./controllers/... -coverprofile cover.out

coverage: ## Run code coverage test
	@common/scripts/codecov.sh ${BUILD_LOCALLY} "controllers"

##@ Build

build-operator-image: ## Build the operator image.
	@echo "Building the $(OPERATOR_IMAGE_NAME) docker image for $(LOCAL_ARCH)..."
	@docker build -t $(REGISTRY)/$(OPERATOR_IMAGE_NAME)-$(LOCAL_ARCH):$(VERSION) \
	--build-arg VCS_REF=$(VCS_REF) --build-arg VCS_URL=$(VCS_URL) \
	--build-arg GOARCH=$(LOCAL_ARCH) -f Dockerfile .

build-bundle-image: ## Build the operator bundle image.
	docker build -f bundle.Dockerfile -t $(IMAGE_REPO)/$(BUNDLE_IMAGE_NAME)-$(LOCAL_ARCH):$(VERSION) .

##@ Release

build-push-image: $(CONFIG_DOCKER_TARGET) build-operator-image  ## Build and push the operator images.
	@echo "Pushing the $(OPERATOR_IMAGE_NAME) docker image for $(LOCAL_ARCH)..."
	@docker push $(REGISTRY)/$(OPERATOR_IMAGE_NAME)-$(LOCAL_ARCH):$(VERSION)

build-push-bundle-image: $(CONFIG_DOCKER_TARGET_QUAY) build-bundle-image ## Build and push the bundle images.
	@echo "Pushing the $(BUNDLE_IMAGE_NAME) docker image for $(LOCAL_ARCH)..."
	@docker push $(IMAGE_REPO)/$(BUNDLE_IMAGE_NAME)-$(LOCAL_ARCH):$(VERSION)

multiarch-image: $(CONFIG_DOCKER_TARGET) ## Generate multiarch images for operator image.
	@MAX_PULLING_RETRY=20 RETRY_INTERVAL=30 common/scripts/multiarch_image.sh $(REGISTRY) $(OPERATOR_IMAGE_NAME) $(VERSION) $(RELEASE_VERSION)

multiarch-bundle-image: $(CONFIG_DOCKER_TARGET_QUAY) ## Generate multiarch images for bundle image.
	@MAX_PULLING_RETRY=20 RETRY_INTERVAL=30 common/scripts/multiarch_image.sh $(IMAGE_REPO) $(BUNDLE_IMAGE_NAME) $(VERSION) $(RELEASE_VERSION)

##@ Help
help: ## Display this help
	@echo "Usage:\n  make \033[36m<target>\033[0m"
	@awk 'BEGIN {FS = ":.*##"}; \
		/^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
