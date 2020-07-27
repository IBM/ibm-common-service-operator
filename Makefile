IMAGE_REPO ?= quay.io/danielxlee
IMAGE_NAME ?= common-service-operator
BUNDLE_IMAGE_NAME ?= common-service-operator-bundle
BUNDLE_IMAGE_VERSION ?= latest
BUNDLE_MANIFESTS_PATH ?= manifests
INDEX_IMAGE_NAME ?= common-service-catalog
INDEX_IMAGE_VERSION ?= latest
INDEX_IMAGE_VERSION_HIS ?= 3.4.4
CHANNELS ?= dev
DEFAULT_CHANNEL ?= dev
OPERATOR_NAME ?= ibm-common-service-operator
CSV_VERSION ?= 3.4.5
VERSION ?= $(shell git describe --exact-match 2> /dev/null || \
				git describe --match=$(git rev-parse --short=8 HEAD) --always --dirty --abbrev=8)

VCS_URL ?= https://github.com/IBM/ibm-common-service-operator
VCS_REF ?= $(shell git rev-parse HEAD)

OPERATOR_SDK ?= $(shell command -v operator-sdk)

# The namespce that operator will be deployed in
NAMESPACE=ibm-common-services

QUAY_USERNAME ?=
QUAY_PASSWORD ?=

BUILD_LOCALLY ?= 1

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

ifeq ($(BUILD_LOCALLY),0)
    export CONFIG_DOCKER_TARGET = config-docker
endif

include common/Makefile.common.mk

install: ## Install all resources (CR/CRD's, RBAC and Operator)
	@echo ....... Set environment variables ......
	- export WATCH_NAMESPACE=${NAMESPACE}
	@echo ....... Creating namespace .......
	- kubectl create namespace ${NAMESPACE}
	@echo ....... Applying CRDs .......
	- kubectl apply -f deploy/crds/operator.ibm.com_commonservices_crd.yaml
	@echo ....... Applying RBAC .......
	- kubectl apply -f deploy/service_account.yaml -n ${NAMESPACE}
	- kubectl apply -f deploy/role.yaml -n ${NAMESPACE}
	- kubectl apply -f deploy/role_binding.yaml -n ${NAMESPACE}
	@echo ....... Applying Operator .......
	- kubectl apply -f deploy/operator.yaml -n ${NAMESPACE}
	@echo ....... Creating the Instances .......
	- kubectl apply -f deploy/crds/operator.ibm.com_v3_commonservice_cr.yaml -n ${NAMESPACE}
uninstall: ## Uninstall all that all performed in the $ make install
	@echo ....... Uninstalling .......
	@echo ....... Deleting the Instances .......
	- kubectl delete -f deploy/crds/operator.ibm.com_v3_commonservice_cr.yaml -n ${NAMESPACE} --ignore-not-found
	@echo ....... Deleting Operator .......
	- kubectl delete -f deploy/operator.yaml -n ${NAMESPACE} --ignore-not-found
	@echo ....... Deleting CRDs .......
	- kubectl delete -f deploy/crds/operator.ibm.com_commonservices_crd.yaml --ignore-not-found
	@echo ....... Deleting RBAC .......
	- kubectl delete -f deploy/role_binding.yaml -n ${NAMESPACE} --ignore-not-found
	- kubectl delete -f deploy/service_account.yaml -n ${NAMESPACE} --ignore-not-found
	- kubectl delete -f deploy/role.yaml -n ${NAMESPACE} --ignore-not-found
	@echo ....... Deleting namespace ${NAMESPACE}.......
	- kubectl delete namespace ${NAMESPACE} --ignore-not-found

run: ## Run against the configured Kubernetes cluster in ~/.kube/config
	@echo ....... Start Operator locally with go run ......
	WATCH_NAMESPACE= go run ./cmd/manager/main.go -v=2 --zap-encoder=console

code-dev:
	go mod tidy

check: code-dev lint-all
	./common/scripts/lint-csv.sh

test:
	echo good

build: check
	CGO_ENABLED=0 go build -o build/_output/bin/$(OPERATOR_NAME) cmd/manager/main.go
	@strip build/_output/bin/$(OPERATOR_NAME) || true

build-push-image: build-image push-image

build-image: build
	@echo "Building the $(IMAGE_NAME) docker image for $(LOCAL_ARCH)..."
	@docker build -t $(IMAGE_REPO)/$(IMAGE_NAME)-$(LOCAL_ARCH):$(VERSION) --build-arg VCS_REF=$(VCS_REF) --build-arg VCS_URL=$(VCS_URL) -f build/Dockerfile .

push-image: $(CONFIG_DOCKER_TARGET) build-image
	@echo "Pushing the $(IMAGE_NAME) docker image for $(LOCAL_ARCH)..."
	@docker push $(IMAGE_REPO)/$(IMAGE_NAME)-$(LOCAL_ARCH):$(VERSION)

build-bundle-image: ## Create operator bundle image
	@echo "Bulding the operator bundle image"
	- $(OPERATOR_SDK) bundle create $(IMAGE_REPO)/$(BUNDLE_IMAGE_NAME)-$(LOCAL_ARCH):$(BUNDLE_IMAGE_VERSION) \
	--directory deploy/olm-catalog/ibm-common-service-operator/$(BUNDLE_MANIFESTS_PATH) \
	--package ibm-common-service-operator-app \
	--channels $(CHANNELS) \
	--default-channel $(DEFAULT_CHANNEL) \
	--overwrite

push-bundle-image: build-bundle-image ## Push operator bundle image
	@echo "Pushing the $(BUNDLE_IMAGE_NAME) docker image for $(LOCAL_ARCH)..."
	@docker push $(IMAGE_REPO)/$(BUNDLE_IMAGE_NAME)-$(LOCAL_ARCH):$(BUNDLE_IMAGE_VERSION)

# Build latest index image
build-index-image: push-bundle-image
	- opm index add --permissive -c docker \
	--bundles $(IMAGE_REPO)/$(BUNDLE_IMAGE_NAME)-$(LOCAL_ARCH):$(BUNDLE_IMAGE_VERSION) \
	--from-index $(IMAGE_REPO)/$(INDEX_IMAGE_NAME)-$(LOCAL_ARCH):$(INDEX_IMAGE_VERSION_HIS) \
	--tag $(IMAGE_REPO)/$(INDEX_IMAGE_NAME)-$(LOCAL_ARCH):$(INDEX_IMAGE_VERSION)

# Push the latest index image
push-index-image: build-index-image ## Push operator index image
	@echo "Pushing the $(INDEX_IMAGE_NAME) docker image for $(LOCAL_ARCH)..."
	@docker push $(IMAGE_REPO)/$(INDEX_IMAGE_NAME)-$(LOCAL_ARCH):$(INDEX_IMAGE_VERSION)

generate-csv:
	$(OPERATOR_SDK) generate csv --csv-version $(CSV_VERSION)

push-csv:
	IMAGE_REPO=$(IMAGE_REPO) OPERATOR_NAME=$(OPERATOR_NAME) VERSION=$(CSV_VERSION) common/scripts/push-csv.sh

multiarch-image: $(CONFIG_DOCKER_TARGET)
	@MAX_PULLING_RETRY=20 RETRY_INTERVAL=30 common/scripts/multiarch_image.sh $(IMAGE_REPO) $(IMAGE_NAME) $(VERSION)
