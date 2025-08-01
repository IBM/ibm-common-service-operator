# Copyright 2022 IBM Corporation
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
YQ_VERSION=v4.27.3
KUSTOMIZE_VERSION=v5.0.0
OPERATOR_SDK_VERSION=v1.38.0
CONTROLLER_TOOLS_VERSION ?= v0.14.0
OPENSHIFT_VERSIONS ?= v4.12-v4.17

CSV_PATH=bundle/manifests/ibm-common-service-operator.clusterserviceversion.yaml

# Specify whether this repo is build locally or not, default values is '1';
# If set to 1, then you need to also set 'DOCKER_USERNAME' and 'DOCKER_PASSWORD'
# environment variables before build the repo.
BUILD_LOCALLY ?= 1

VCS_URL ?= https://github.com/IBM/ibm-common-service-operator
VCS_REF ?= $(shell git rev-parse HEAD)
VERSION ?= $(shell git describe --exact-match 2> /dev/null || \
                git describe --match=$(git rev-parse --short=8 HEAD) --always --dirty --abbrev=8)
RELEASE_VERSION ?= $(shell cat ./version/version.go | grep "Version =" | awk '{ print $$3}' | tr -d '"')
PREVIOUS_VERSION := 3.23.0
LATEST_VERSION ?= 4.14.0

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
QUAY_REGISTRY ?= quay.io/opencloudio

ifeq ($(BUILD_LOCALLY),0)
ARTIFACTORYA_REGISTRY ?= "docker-na-public.artifactory.swg-devops.com/hyc-cloud-private-integration-docker-local/ibmcom"
else
ARTIFACTORYA_REGISTRY ?= "docker-na-public.artifactory.swg-devops.com/hyc-cloud-private-scratch-docker-local/ibmcom"
endif

REGISTRY ?= "docker-na-public.artifactory.swg-devops.com/hyc-cloud-private-scratch-docker-local/ibmcom"

# Current Operator image name
OPERATOR_IMAGE_NAME ?= common-service-operator
# Current Operator bundle image name
BUNDLE_IMAGE_NAME ?= common-service-operator-bundle
# Current Operator image with registry
IMG ?= icr.io/cpopen/common-service-operator:$(LATEST_VERSION)

CHANNELS := v4.14
DEFAULT_CHANNEL := v4.14

# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

ifeq ($(BUILD_LOCALLY),0)
    export CONFIG_DOCKER_TARGET = config-docker
    export CONFIG_DOCKER_TARGET_QUAY = config-docker-quay
endif

include common/Makefile.common.mk
include hack/keycloak-themes/Makefile

##@ Development

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

clis: yq kustomize operator-sdk

yq: ## Install yq, a yaml processor
ifneq ($(shell yq -V | cut -d ' ' -f 3 | cut -d '.' -f 1 ), 4)
	@{ \
	if [ v$(shell ./bin/yq --version | cut -d ' ' -f3) != $(YQ_VERSION) ]; then\
		set -e ;\
		mkdir -p bin ;\
		echo "Downloading yq ...";\
		curl -sSLO https://github.com/mikefarah/yq/releases/download/$(YQ_VERSION)/yq_$(LOCAL_OS)_$(LOCAL_ARCH);\
		mv yq_$(LOCAL_OS)_$(LOCAL_ARCH) ./bin/yq ;\
		chmod +x ./bin/yq ;\
	fi;\
	}
YQ=$(realpath ./bin/yq)
else
YQ=$(shell which yq)
endif

kustomize: ## Install kustomize
ifeq (, $(shell which kustomize 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p bin ;\
	echo "Downloading kustomize ...";\
	curl -sSLo - https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize/$(KUSTOMIZE_VERSION)/kustomize_$(KUSTOMIZE_VERSION)_$(LOCAL_OS)_$(LOCAL_ARCH).tar.gz | tar xzf - -C bin/ ;\
	}
KUSTOMIZE=$(realpath ./bin/kustomize)
else
KUSTOMIZE=$(shell which kustomize)
endif

operator-sdk:
ifneq ($(shell operator-sdk version | cut -d ',' -f1 | cut -d ':' -f2 | tr -d '"' | xargs | cut -d '.' -f1), v1)
	@{ \
	if [ "$(shell ./bin/operator-sdk version | cut -d ',' -f1 | cut -d ':' -f2 | tr -d '"' | xargs)" != $(OPERATOR_SDK_VERSION) ]; then \
		set -e ; \
		mkdir -p bin ;\
		echo "Downloading operator-sdk..." ;\
		curl -sSLo ./bin/operator-sdk "https://github.com/operator-framework/operator-sdk/releases/download/$(OPERATOR_SDK_VERSION)/operator-sdk_$(LOCAL_OS)_$(LOCAL_ARCH)" ;\
		chmod +x ./bin/operator-sdk ;\
	fi ;\
	}
OPERATOR_SDK=$(realpath ./bin/operator-sdk)
else
OPERATOR_SDK=$(shell which operator-sdk)
endif

check: lint-all ## Check all files lint error
	./common/scripts/lint-csv.sh

code-dev: ## Run the default dev commands which are the go tidy, fmt, vet then execute the $ make code-gen
	@echo Running the common required commands for developments purposes
	- make code-tidy
	- make code-fmt
	- make code-vet
	@echo Running the common required commands for code delivery
	- make check

build: ## Build manager binary
	go build -o bin/manager cmd/main.go

run: generate code-fmt code-vet manifests ## Run against the configured Kubernetes cluster in ~/.kube/config
	ENABLE_WEBHOOKS=false OPERATOR_NAMESPACE="$${OPERATOR_NAMESPACE:-ibm-common-services}" OPERATOR_NAME=ibm-common-service-operator go run ./cmd/main.go -v=2

install: manifests ## Install CRDs into a cluster
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests ## Uninstall CRDs from a cluster
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: manifests ## Deploy controller in the configured Kubernetes cluster in ~/.kube/config
	cd config/manager && $(KUSTOMIZE) edit set image icr.io/cpopen/common-service-operator=$(QUAY_REGISTRY)/$(OPERATOR_IMAGE_NAME):$(RELEASE_VERSION)
	$(KUSTOMIZE) build config/default | kubectl apply -f -

build-dev-image: cloudpak-theme.jar
	@echo "Building the $(OPERATOR_IMAGE_NAME) docker dev image for $(LOCAL_ARCH)..."
	@docker build -t $(REGISTRY)/$(OPERATOR_IMAGE_NAME)-$(LOCAL_ARCH):dev \
	--build-arg VCS_REF=$(VCS_REF) --build-arg VCS_URL=$(VCS_URL) --build-arg RELEASE_VERSION=$(RELEASE_VERSION) \
	--build-arg GOARCH=$(LOCAL_ARCH) -f Dockerfile .
	@docker push $(REGISTRY)/$(OPERATOR_IMAGE_NAME)-$(LOCAL_ARCH):dev

build-bundle-image: yq
	@cp -f bundle/manifests/ibm-common-service-operator.clusterserviceversion.yaml /tmp/ibm-common-service-operator.clusterserviceversion.yaml
	$(YQ) eval -i 'del(.spec.replaces)' bundle/manifests/ibm-common-service-operator.clusterserviceversion.yaml
	docker build -f bundle.Dockerfile -t $(QUAY_REGISTRY)/$(BUNDLE_IMAGE_NAME):$(VERSION) .
	docker push $(QUAY_REGISTRY)/$(BUNDLE_IMAGE_NAME):$(VERSION)
	@mv /tmp/ibm-common-service-operator.clusterserviceversion.yaml bundle/manifests/ibm-common-service-operator.clusterserviceversion.yaml

run-bundle:
	$(OPERATOR_SDK) run bundle $(QUAY_REGISTRY)/$(BUNDLE_IMAGE_NAME):$(VERSION) --install-mode OwnNamespace

upgrade-bundle:
	$(OPERATOR_SDK) run bundle-upgrade $(QUAY_REGISTRY)/$(BUNDLE_IMAGE_NAME):$(VERSION)

cleanup-bundle:
	$(OPERATOR_SDK) cleanup ibm-common-service-operator

build-catalog-source:
	opm -u docker index add --bundles $(QUAY_REGISTRY)/$(BUNDLE_IMAGE_NAME):$(VERSION) --tag $(QUAY_REGISTRY)/$(OPERATOR_IMAGE_NAME)-catalog:$(VERSION)
	docker push $(QUAY_REGISTRY)/$(OPERATOR_IMAGE_NAME)-catalog:$(VERSION)

update-csv-image: # updates operator image in currently deployed Common Service Operator
	oc patch csv -n ibm-common-services ibm-common-service-operator.v$(RELEASE_VERSION) --type json -p \
		'[{"op": "replace", "path": "/spec/install/spec/deployments/0/spec/template/spec/containers/0/image", "value": "$(QUAY_REGISTRY)/$(OPERATOR_IMAGE_NAME):dev"}]'

build-catalog: build-bundle-image build-catalog-source

deploy-catalog: build-catalog
	./common/scripts/update_catalogsource.sh $(OPERATOR_IMAGE_NAME) $(QUAY_REGISTRY)/$(OPERATOR_IMAGE_NAME)-catalog:$(VERSION)

undeploy-catalog:
	kubectl -n openshift-marketplace delete catalogsource $(OPERATOR_IMAGE_NAME)

test-profile: yq
	./testdata/test_profile.sh $(YQ)

##@ Generate code and manifests

manifests: controller-gen ## Generate manifests e.g. CRD, RBAC etc.
	$(CONTROLLER_GEN) crd rbac:roleName=ibm-common-service-operator webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen ## Generate code e.g. API etc.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

bundle-manifests: clis
	cd config/manager && $(KUSTOMIZE) edit set image icr.io/cpopen/common-service-operator=${IMG}
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle \
	-q --overwrite --version $(RELEASE_VERSION) $(BUNDLE_METADATA_OPTS)
	echo "\n  # OpenShift annotations." >> bundle/metadata/annotations.yaml ;\
	echo "  com.redhat.openshift.versions: $(OPENSHIFT_VERSIONS)" >> bundle/metadata/annotations.yaml ;\
	$(OPERATOR_SDK) bundle validate ./bundle
	$(YQ) eval -i '.metadata.annotations."olm.skipRange" = ">=3.3.0 <${RELEASE_VERSION}"' ${CSV_PATH}
	$(YQ) eval -i '.spec.webhookdefinitions[0].deploymentName = "ibm-common-service-operator" | .spec.webhookdefinitions[1].deploymentName = "ibm-common-service-operator"' ${CSV_PATH}
	$(YQ) eval-all -i '.spec.relatedImages = load("config/manifests/bases/ibm-common-service-operator.clusterserviceversion.yaml").spec.relatedImages' bundle/manifests/ibm-common-service-operator.clusterserviceversion.yaml

generate-all: yq kustomize operator-sdk generate manifests cloudpak-theme-version ## Generate bundle manifests, metadata and package manifests
	$(OPERATOR_SDK) generate kustomize manifests -q
	- make bundle-manifests CHANNELS=v4.14 DEFAULT_CHANNEL=v4.14

##@ Helm Chart Generation

.PHONY: deploy-dryrun
deploy-dryrun: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && \
	$(KUSTOMIZE) edit set image icr.io/cpopen/common-service-operator=${IMG} && \
	$(KUSTOMIZE) edit add patch --path non-olm-env-patch.yaml 
	$(KUSTOMIZE) build config/default --output config/ibm-common-service-operator.yaml
	cd config/manager && $(KUSTOMIZE) edit remove patch --path non-olm-env-patch.yaml

.PHONY: helm
helm: deploy-dryrun kustohelmize
	$(KUSTOHELMIZE) create --from=config/ibm-common-service-operator.yaml helm/ibm-common-service-operator
	helm lint helm/ibm-common-service-operator

KUBERNETES-SPLIT-YAML ?= $(LOCALBIN)/kubernetes-split-yaml
KUSTOHELMIZE ?= $(LOCALBIN)/kustohelmize

.PHONY: kubernetes-split-yaml
kubernetes-split-yaml: $(KUBERNETES-SPLIT-YAML) ## Download kubernetes-split-yaml locally if necessary.
$(KUBERNETES-SPLIT-YAML): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install github.com/mogensen/kubernetes-split-yaml@v0.4.0

.PHONY: kustohelmize
kustohelmize: $(KUSTOHELMIZE) ## Download kustohelmize locally if necessary.
$(KUSTOHELMIZE): $(LOCALBIN) kubernetes-split-yaml
	GOBIN=$(LOCALBIN) go install github.com/yeahdongcn/kustohelmize@v0.4.0

##@ Test

test: ## Run unit test on prow
	@echo good

e2e-test: ## Run e2e test
	@echo "Running e2e tests for the controllers."
	@USE_EXISTING_CLUSTER=true \
	OPERATOR_NAME=ibm-common-service-operator \
	OPERATOR_NAMESPACE=ibm-common-services \
	go test ./controllers/... -coverprofile cover.out

##@ Build

build-operator-image: $(CONFIG_DOCKER_TARGET) cloudpak-theme.jar ## Build the operator image.
	@echo "Building the $(OPERATOR_IMAGE_NAME) docker image for $(LOCAL_ARCH)..."
	@docker build -t $(OPERATOR_IMAGE_NAME)-$(LOCAL_ARCH):$(VERSION) \
	--build-arg VCS_REF=$(VCS_REF) --build-arg VCS_URL=$(VCS_URL) --build-arg RELEASE_VERSION=$(RELEASE_VERSION) \
	--build-arg GOARCH=$(LOCAL_ARCH) -f Dockerfile .

##@ Release

build-push-image: $(CONFIG_DOCKER_TARGET) build-operator-image  ## Build and push the operator images.
	@echo "Pushing the $(OPERATOR_IMAGE_NAME) docker image for $(LOCAL_ARCH)..."
	@docker tag $(OPERATOR_IMAGE_NAME)-$(LOCAL_ARCH):$(VERSION) $(ARTIFACTORYA_REGISTRY)/$(OPERATOR_IMAGE_NAME)-$(LOCAL_ARCH):$(VERSION)
	@docker push $(ARTIFACTORYA_REGISTRY)/$(OPERATOR_IMAGE_NAME)-$(LOCAL_ARCH):$(VERSION)

multiarch-image: $(CONFIG_DOCKER_TARGET) ## Generate multiarch images for operator image.
	@MAX_PULLING_RETRY=20 RETRY_INTERVAL=30 common/scripts/multiarch_image.sh $(ARTIFACTORYA_REGISTRY) $(OPERATOR_IMAGE_NAME) $(VERSION) $(RELEASE_VERSION)

##@ Help
help: ## Display this help
	@echo "Usage:\n  make \033[36m<target>\033[0m"
	@awk 'BEGIN {FS = ":.*##"}; \
		/^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
