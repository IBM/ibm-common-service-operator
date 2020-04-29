QUAY_REPO ?= quay.io/opencloudio
IMAGE_NAME ?= common-service-operator
OPERATOR_NAME ?= ibm-common-service-operator
CSV_VERSION ?= 3.4.0
VERSION ?= $(shell git describe --exact-match 2> /dev/null || \
				git describe --match=$(git rev-parse --short=8 HEAD) --always --dirty --abbrev=8)

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
	WATCH_NAMESPACE=${NAMESPACE} go run ./cmd/manager/main.go

code-dev:
	go mod tidy

check: code-dev lint-all

test:
	echo good

build: check
	CGO_ENABLED=0 go build -o build/_output/bin/$(OPERATOR_NAME) cmd/manager/main.go
	@strip build/_output/bin/$(OPERATOR_NAME) || true

build-push-image: build-image push-image

build-image: build
	@echo "Building the $(IMAGE_NAME) docker image for $(LOCAL_ARCH)..."
	@docker build -t $(QUAY_REPO)/$(IMAGE_NAME)-$(LOCAL_ARCH):$(VERSION) -f build/Dockerfile .

push-image: $(CONFIG_DOCKER_TARGET) build-image
	@echo "Pushing the $(IMAGE_NAME) docker image for $(LOCAL_ARCH)..."
	@docker push $(QUAY_REPO)/$(IMAGE_NAME)-$(LOCAL_ARCH):$(VERSION)

generate-csv:
	operator-sdk generate csv --csv-version $(CSV_VERSION) --update-crds

push-csv:
	QUAY_REPO=$(QUAY_REPO) OPERATOR_NAME=$(OPERATOR_NAME) VERSION=$(CSV_VERSION) common/scripts/push-csv.sh

multiarch-image: $(CONFIG_DOCKER_TARGET)
	@MAX_PULLING_RETRY=20 RETRY_INTERVAL=30 common/scripts/multiarch_image.sh $(QUAY_REPO) $(IMAGE_NAME) $(VERSION)
