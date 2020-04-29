FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/ibm-common-service-operator \
    RESOURCES_DIR=/resources
    USER_UID=1001 \
    USER_NAME=ibm-common-service-operator

# install operator binary
COPY build/_output/bin/ibm-common-service-operator ${OPERATOR}

COPY build/resources ${RESOURCES_DIR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
