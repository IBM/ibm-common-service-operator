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
OPERATOR_NS=""
SERVICES_NS=""
TETHERED_NS=""
CONTROL_NS=""
CHANNEL="v4.0"
SOURCE="opencloud-operators"
SOURCE_NS="openshift-marketplace"
INSTALL_MODE="Automatic"
NEW_MAPPING=""
NEW_TENANT=0
DEBUG=0
PREVIEW_MODE=1

# ---------- Command variables ----------

# script base directory
BASE_DIR=$(cd $(dirname "$0")/$(dirname "$(readlink $0)") && pwd -P)

# counter to keep track of installation steps
STEP=0

# ---------- Main functions ----------

. ${BASE_DIR}/common/utils.sh

function main() {
    parse_arguments "$@"
    pre_req
    
    # TODO check Cloud Pak compatibility


    # # Update common-serivce-maps ConfigMap
    # ${BASE_DIR}/common/mapping_tenant.sh --operator-namespace $OPERATOR_NS --services-namespace $SERVICES_NS --tethered-namespaces $TETHERED_NS --control-namespace $CONTROL_NS

    # Delgation of CP2 Cert Manager
    ${BASE_DIR}/common/delegate_cp2_cert_manager.sh --control-namespace $CONTROL_NS

    # Migrate Licensing Services Data
    ${BASE_DIR}/common/migrate_cp2_licensing.sh --control-namespace $CONTROL_NS

    
    # Update CommonService CR with OPERATOR_NS and SERVICES_NS
    configure_cs_kind

    # TODO Propogate CommonService CR
    # Upgrade NSS operator

    # Update ibm-common-service-operator channel
    update_operator_channel ibm-common-service-operator $OPERATOR_NS $SOURCE $SOURCE_NS $INSTALL_MODE
    
    # Update ibm-namespace-scope-operator channel
    update_operator_channel ibm-namespace-scope-operator $OPERATOR_NS $SOURCE $SOURCE_NS $INSTALL_MODE

    # wait for operator upgrade

    # Update NamespaceScope CR common-service

    
    # Clean resources
    # CommonUI OperandBindInfo
    # auditloggings CR(Remove from operandconfig) and csv/subscription
    # OperandRequest: ibm-commonui-request, ibm-mongodb-request
    # Cert-Manager and licensing CR, csv/subscriptions
    
    # Install New CertManager and Licensing
    # Install PostgreSQL
    # Migrate IAM roles
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
        --operator-namespace)
            shift
            OPERATOR_NS=$1
            ;;
        --services-namespace)
            shift
            SERVICES_NS=$1
            ;;
        --tethered-namespaces)
            shift
            TETHERED_NS=$1
            ;;
        --control-namespace)
            shift
            CONTROL_NS=$1
            ;;
        --enable-private-catalog)
            ENABLE_PRIVATE_CATALOG=1
            ;;
        -c | --channel)
            shift
            CHANNEL=$1
            ;;
        -s | --source)
            shift
            SOURCE=$1
            ;;
        -v | --debug)
            shift
            DEBUG=$1
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
    echo "Usage: ${script_name} --operator-namespace <bedrock-namespace> [OPTIONS]..."
    echo ""
    echo "Migrate Cloud Pak 2.0 Foundational services to in Cloud Pak 3.0 Foundational services."
    echo "The --operator-namespace must be provided."
    echo ""
    echo "Options:"
    echo "   --oc string                    File path to oc CLI. Default uses oc in your PATH"
    echo "   --yq string                    File path to yq CLI. Default uses yq in your PATH"
    echo "   --operator-namespace string    Required. Namespace to migrate Foundational services operator"
    echo "   --services-namespace           Namespace to migrate operands of Foundational services, i.e. 'dataplane'. Default is the same as operator-namespace"
    echo "   --tethered-namespaces string   Additional namespaces for this tenant, comma-delimited, e.g. 'ns1,ns2'"
    echo "   --control-namespace string     Namespace to install Cloud Pak 2.0 cluster singleton Foundational services."
    echo "                                  It is required if there are multiple Cloud Pak 2.0 Foundational services instances or it is a co-existence of Cloud Pak 2.0 and Cloud Pak 3.0"
    echo "   --enable-private-catalog       Set this flag to use namespace scoped CatalogSource. Default is in openshift-marketplace namespace"
    echo "   -c, --channel string           Channel for Subscription(s). Default is v4.0"   
    echo "   -i, --install-mode string      InstallPlan Approval Mode. Default is Automatic. Set to Manual for manual approval mode"
    echo "   -s, --source string            CatalogSource name. This assumes your CatalogSource is already created. Default is opencloud-operators"
    echo "   -v, --debug integer            Verbosity of logs. Default is 0. Set to 1 for debug logs."
    echo "   -h, --help                     Print usage information"
    echo ""
}

function pre_req() {
    check_command "${OC}"

    # checking oc command logged in
    user=$(${OC} whoami 2> /dev/null)
    if [ $? -ne 0 ]; then
        error "You must be logged into the OpenShift Cluster from the oc command line"
    else
        success "oc command logged in as ${user}"
    fi

    if [ "$OPERATOR_NS" == "" ]; then
        error "Must provide operator namespace"
    fi

    if [ "$SERVICES_NS" == "" ]; then
        SERVICES_NS=$OPERATOR_NS
    fi

    if [ "$CONTROL_NS" == "" ]; then
        CONTROL_NS=$OPERATOR_NS
    fi

    if [ $ENABLE_PRIVATE_CATALOG -eq 1 ]; then
        SOURCE_NS=$OPERATOR_NS
    fi
    validate_arguments
}

# TODO validate argument
function validate_arguments() {
    validate_control_namespace "$CONTROL_NS"
}

# TODO Parametize values
function configure_cs_kind() {

    ${OC} get commonservice common-service -n "${OPERATOR_NS}" -o yaml | yq eval '.spec += {"operatorNamespace": "'${OPERATOR_NS}'", "servicesNamespace": "'${SERVICES_NS}'"}' > common-service.yaml

    yq eval 'select(.kind == "CommonService") | del(.metadata.resourceVersion) | del(.metadata.uid)' common-service.yaml | ${OC} apply -f -
    if [[ $? -ne 0 ]]; then
        echo "Failed to create CommonService CR in ${OPERATOR_NS}"
    fi
    rm common-service.yaml
}

function debug1() {
    if [ $DEBUG -eq 1 ]; then
       debug "${1}"
    fi
}

main $*