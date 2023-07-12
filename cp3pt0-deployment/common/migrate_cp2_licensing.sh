#!/usr/bin/env bash

# Licensed Materials - Property of IBM
# Copyright IBM Corporation 2023. All Rights Reserved
# US Government Users Restricted Rights -
# Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
#
# This is an internal component, bundled with an official IBM product. 
# Please refer to that particular license for additional information. 

# ---------- Command arguments ----------

OC=oc
YQ=yq
CONTROL_NS=""
TARGET_NS=ibm-licensing
DEBUG=0
PREVIEW_MODE=0
SKIP_USER_VERIFY=0

# ---------- Command variables ----------

# script base directory
BASE_DIR=$(cd $(dirname "$0")/$(dirname "$(readlink $0)") && pwd -P)

# counter to keep track of installation steps
STEP=0

# ---------- Main functions ----------

. ${BASE_DIR}/utils.sh

function main() {
    parse_arguments "$@"
    pre_req
    create_namespace $TARGET_NS
    migrate_lic_cms
    # TODO: restore ibm-license-service Secrets
}

function parse_arguments() {
    # process options
    while [[ "$@" != "" ]]; do
        case "$1" in
        --oc)
            shift
            OC=$1
            ;;
        --yq)
            shift
            YQ=$1
            ;;
        --control-namespace)
            shift
            CONTROL_NS=$1
            ;;
        --target-namespace)
            shift
            TARGET_NS=$1
            ;;
        -v | --debug)
            shift
            DEBUG=$1
            ;;
        --skip-user-vertify)
            SKIP_USER_VERIFY=1
            ;;
        -h | --help)
            print_usage
            exit 1
            ;;
        *) 
            echo "wildcard"
            ;;
        esac
        shift
    done
}

function print_usage() {
    script_name=`basename ${0}`
    echo "Usage: ${script_name} --control-namespace <licensing-services-namespace> [OPTIONS]..."
    echo ""
    echo "Migrate Licensing Data for IBM Cloud Pak 2.0 Licensing Service."
    echo "The --control-namespace must be provided."
    echo ""
    echo "Options:"
    echo "   --oc string                    File path to oc CLI. Default uses oc in your PATH"
    echo "   --yq string                    File path to yq CLI. Default uses yq in your PATH"
    echo "   --control-namespace string     Required. Source Namespace where Cloud Pak 2.0 Licensing Data is located."
    echo "   --target-namespace string      Target Namespace where Cloud Pak 3.0 Licensing Operator is located. Default is ibm-licensing"
    echo "   -v, --debug integer            Verbosity of logs. Default is 0. Set to 1 for debug logs."
    echo "   --skip-user-vertify string     Skip checking user logged into oc command"
    echo "   -h, --help                     Print usage information"
    echo ""
}

function pre_req() {
    if [[ $SKIP_USER_VERIFY -eq 0 ]]; then
        check_command "${OC}"

        # checking oc command logged in
        user=$(${OC} whoami 2> /dev/null)
        if [ $? -ne 0 ]; then
            error "You must be logged into the OpenShift Cluster from the oc command line"
        else
            success "oc command logged in as ${user}"
        fi
    fi

    if [ "$CONTROL_NS" == "" ]; then
        error "Must provide control namespace"
    fi

}

function migrate_lic_cms() {
    
    title "Migrating IBM License Service data from ${CONTROL_NS} into ${TARGET_NS} namespace"

    POSSIBLE_CONFIGMAPS=("ibm-licensing-config"
"ibm-licensing-annotations"
"ibm-licensing-products"
"ibm-licensing-products-vpc-hour"
"ibm-licensing-cloudpaks"
"ibm-licensing-products-groups"
"ibm-licensing-cloudpaks-groups"
"ibm-licensing-cloudpaks-metrics"
"ibm-licensing-products-metrics"
"ibm-licensing-products-metrics-groups"
"ibm-licensing-cloudpaks-metrics-groups"
"ibm-licensing-services"
)

    for configmap in ${POSSIBLE_CONFIGMAPS[@]}
    do
        ${OC} get configmap "${configmap}" -n "${CONTROL_NS}" > /dev/null 2>&1
        if [ $? -eq 0 ]
        then
            info "Copying Licensing Services ConfigMap $cm from $CONTROL_NS to $TARGET_NS"
            ${OC} get configmap "${configmap}" -n "${CONTROL_NS}" -o yaml | ${YQ} -e '.metadata.namespace = "'${TARGET_NS}'"' > ${configmap}.yaml
            ${YQ} eval 'select(.kind == "ConfigMap") | del(.metadata.resourceVersion) | del(.metadata.uid)' ${configmap}.yaml | ${OC} apply -f -

            if [[ $? -eq 0 ]]; then
                info "Licensing Services ConfigMap $configmap is copied from $CONTROL_NS to $TARGET_NS"
                # delete the original
                ${OC} delete cm -n $CONTROL_NS $configmap --ignore-not-found
            else
                error "Failed to move Licensing Services ConfigMap $configmap to $TARGET_NS"
            fi

            rm ${configmap}.yaml
            msg ""
        fi
    done
    success "Licensing Service ConfigMaps are migrated from $CONTROL_NS to $TARGET_NS"
}

main $*
