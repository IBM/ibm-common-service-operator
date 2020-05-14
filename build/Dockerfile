FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
ARG VCS_REF
ARG VCS_URL

LABEL org.label-schema.vendor="IBM" \
    org.label-schema.name="ibm common service operator" \
    org.label-schema.description="Deploy ODLM and related instances" \
    org.label-schema.vcs-ref=$VCS_REF \
    org.label-schema.vcs-url=$VCS_URL \
    org.label-schema.license="Licensed Materials - Property of IBM" \
    org.label-schema.schema-version="1.0" \
    name="common-service-operator" \
    vendor="IBM" \
    description="Deploy ODLM and related instances" \
    summary="Deploy ODLM and related instances"

ENV OPERATOR=/usr/local/bin/ibm-common-service-operator \
    RESOURCES_DIR=/resources \
    USER_UID=1001 \
    USER_NAME=ibm-common-service-operator

# install operator binary
COPY build/_output/bin/ibm-common-service-operator ${OPERATOR}

COPY build/resources ${RESOURCES_DIR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
