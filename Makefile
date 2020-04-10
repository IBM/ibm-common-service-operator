QUAY_REPO ?= quay.io/opencloudio
IMAGE_NAME ?= common-service-operator
OPERATOR_NAME ?= ibm-common-service-operator
CSV_VERSION ?= 0.0.1
VERSION ?= $(shell git describe --exact-match 2> /dev/null || \
				git describe --match=$(git rev-parse --short=8 HEAD) --always --dirty --abbrev=8)

QUAY_USERNAME ?=
QUAY_PASSWORD ?=

BUILD_LOCALLY ?= 1

include common/Makefile.common.mk

code-dev:
	go mod tidy

check: code-dev lint-all

test:
	echo good

build: check
	CGO_ENABLED=0 go build -o build/_output/bin/$(OPERATOR_NAME) cmd/manager/main.go
	@strip build/_output/bin/$(OPERATOR_NAME) || true

image: build
	docker build -t $(QUAY_REPO)/$(IMAGE_NAME):$(VERSION) -f build/Dockerfile .

push-image:
	docker push $(QUAY_REPO)/$(IMAGE_NAME):$(VERSION)

generate-csv:
	operator-sdk generate csv --csv-version $(CSV_VERSION) --update-crds

push-csv:
	QUAY_REPO=$(QUAY_REPO) OPERATOR_NAME=$(OPERATOR_NAME) VERSION=$(CSV_VERSION) common/scripts/push-csv.sh

ifeq ($(BUILD_LOCALLY),0)
    export CONFIG_DOCKER_TARGET = config-docker
endif

multiarch-image: $(CONFIG_DOCKER_TARGET)
	@MAX_PULLING_RETRY=20 RETRY_INTERVAL=30 common/scripts/multiarch_image.sh $(QUAY_REPO) $(IMAGE_NAME) $(VERSION)
