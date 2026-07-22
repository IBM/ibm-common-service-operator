# Build the manager binary
# syntax=docker/dockerfile:1
FROM docker-na-public.artifactory.swg-devops.com/hyc-cloud-private-dockerhub-docker-remote/golang:1.26.5 AS builder
ARG TARGETOS
ARG TARGETARCH
ARG GOARCH

# Private modules hosted on github.ibm.com must not be looked up via the public
# Go checksum database or proxy. Setting these here (in the build stage only)
# means no secret is needed for sum verification.
ENV GONOSUMDB="github.ibm.com/*" \
    GOPRIVATE="github.ibm.com/*" \
    GOFLAGS="-mod=mod"

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
# The netrc secret provides credentials for github.ibm.com (private modules).
# It is mounted read-only at /root/.netrc and is never baked into the image.
# Pass it with: docker buildx build --secret id=netrc,src=$HOME/.netrc
RUN --mount=type=secret,id=netrc,dst=/root/.netrc,mode=0400,required=false \
    go mod download

# Copy the go source
COPY cmd/main.go cmd/main.go
COPY api/ api/
COPY internal/controller/ internal/controller/

# Build
RUN CGO_ENABLED=0 \
  GOOS="${TARGETOS:-linux}" \
  GOARCH="${GOARCH:-${TARGETARCH:-amd64}}" \
  GO111MODULE=on \
  go build -a -o manager cmd/main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
# FROM gcr.io/distroless/static:nonroot
FROM docker-na-public.artifactory.swg-devops.com/hyc-cloud-private-edge-docker-local/build-images/ubi10-minimal:latest
ARG TARGETARCH

ARG VCS_REF
ARG RELEASE_VERSION

LABEL org.label-schema.vendor="IBM" \
  org.label-schema.name="ibm common service operator" \
  org.label-schema.description="Deploy ODLM and IBM Common Services" \
  org.label-schema.vcs-ref=$VCS_REF \
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
