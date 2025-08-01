# Build the manager binary
FROM docker-na-public.artifactory.swg-devops.com/hyc-cloud-private-dockerhub-docker-remote/golang:1.23.2 AS builder
ARG GOARCH

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/main.go cmd/main.go
COPY api/ api/
COPY internal/controller/ internal/controller/

# Build
RUN CGO_ENABLED=0 GOOS=linux GO111MODULE=on go build -a -o manager cmd/main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
# FROM gcr.io/distroless/static:nonroot
FROM docker-na-public.artifactory.swg-devops.com/hyc-cloud-private-edge-docker-local/build-images/ubi9-minimal:latest

ARG VCS_REF
ARG VCS_URL
ARG RELEASE_VERSION

LABEL org.label-schema.vendor="IBM" \
  org.label-schema.name="ibm common service operator" \
  org.label-schema.description="Deploy ODLM and IBM Common Services" \
  org.label-schema.vcs-ref=$VCS_REF \
  org.label-schema.vcs-url=$VCS_URL \
  org.label-schema.license="Licensed Materials - Property of IBM" \
  org.label-schema.schema-version="1.0" \
  name="common-service-operator" \
  maintainer="IBM" \
  vendor="IBM" \
  version=$RELEASE_VERSION \
  release=$RELEASE_VERSION \
  description="Deploy ODLM and IBM Common Services" \
  summary="Deploy ODLM and IBM Common Services"

WORKDIR /
COPY --from=builder /workspace/manager .
COPY hack/keycloak-themes/cloudpak-theme.jar /hack/keycloak-themes/cloudpak-theme.jar

# copy licenses
RUN mkdir /licenses
COPY LICENSE /licenses

# USER nonroot:nonroot
USER 1001

ENTRYPOINT ["/manager"]
