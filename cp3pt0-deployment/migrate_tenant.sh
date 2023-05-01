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
NS_LIST=""
CONTROL_NS=""
CHANNEL="v4.0"
SOURCE="opencloud-operators"
CERT_MANAGER_SOURCE="ibm-cert-manager-catalog"
LICENSING_SOURCE="ibm-licensing-catalog"
SOURCE_NS="openshift-marketplace"
INSTALL_MODE="Automatic"
ENABLE_LICENSING=0
ENABLE_PRIVATE_CATALOG=0
NEW_MAPPING=""
NEW_TENANT=0
DEBUG=0
PREVIEW_MODE=1
LICENSE_ACCEPT=0

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
    
    # Scale down CS, ODLM and delete OperandReigsrty
    # It helps to prevent re-installing licensing and cert-manager services
    scale_down $OPERATOR_NS $SERVICES_NS $CHANNEL $SOURCE

    # Migrate singleton services
    local arguments="--enable-licensing"
    arguments=" -licensingNs $CONTROL_NS"

    if [[ $ENABLE_PRIVATE_CATALOG -eq 1 ]]; then
        arguments+=" --enable-private-catalog"
    fi
    ${BASE_DIR}/setup_singleton.sh "--operator-namespace" "$OPERATOR_NS" "-c" "$CHANNEL" "--cert-manager-source" "$CERT_MANAGER_SOURCE" "--licensing-source" "$LICENSING_SOURCE" "$arguments" "--license-accept"

    # Update CommonService CR with OPERATOR_NS and SERVICES_NS
    # Propogate CommonService CR to every namespace in the tenant
    update_cscr "$OPERATOR_NS" "$SERVICES_NS" "$NS_LIST"
    accept_license "commonservice" "$OPERATOR_NS"  "common-service"

    # Update ibm-common-service-operator channel
    for ns in ${NS_LIST//,/ }; do
        if [ $ENABLE_PRIVATE_CATALOG -eq 0 ]; then
            update_operator ibm-common-service-operator $ns $CHANNEL $SOURCE $SOURCE_NS $INSTALL_MODE
        else
            update_operator ibm-common-service-operator $ns $CHANNEL $SOURCE $ns $INSTALL_MODE
        fi
    done

    # Wait for CS operator upgrade
    wait_for_operator_upgrade $OPERATOR_NS ibm-common-service-operator $CHANNEL $INSTALL_MODE
    # Scale up CS
    scale_up $OPERATOR_NS $SERVICES_NS ibm-common-service-operator ibm-common-service-operator

    # Wait for ODLM upgrade
    wait_for_operator_upgrade $OPERATOR_NS ibm-odlm $CHANNEL $INSTALL_MODE
    # Scale up ODLM
    scale_up $OPERATOR_NS $SERVICES_NS ibm-odlm operand-deployment-lifecycle-manager

    # Clean resources
    cleanup_cp2 "$OPERATOR_NS" "$CONTROL_NS" "$NS_LIST"

    # Update ibm-namespace-scope-operator channel
    is_sub_exist ibm-namespace-scope-operator-restricted $OPERATOR_NS
    if [ $? -eq 0 ]; then
        warning "There is a ibm-namespace-scope-operator-restricted Subscription\n"
        delete_operator ibm-namespace-scope-operator-restricted $OPERATOR_NS
        create_subscription ibm-namespace-scope-operator $OPERATOR_NS $CHANNEL ibm-namespace-scope-operator $SOURCE $SOURCE_NS $INSTALL_MODE
    else
        update_operator ibm-namespace-scope-operator $OPERATOR_NS $CHANNEL $SOURCE $SOURCE_NS $INSTALL_MODE
    fi

    wait_for_operator_upgrade "$OPERATOR_NS" "ibm-namespace-scope-operator" "$CHANNEL" $INSTALL_MODE
    accept_license "namespacescope" "$OPERATOR_NS" "common-service"
    # Authroize NSS operator
    for ns in ${NS_LIST//,/ }; do
        if [ "$ns" != "$OPERATOR_NS" ]; then
            ${BASE_DIR}/common/authorize-namespace.sh $ns -to $OPERATOR_NS
        fi
    done

    # Update NamespaceScope CR common-service
    update_nss_kind "$OPERATOR_NS" "$NS_LIST"

    success "Preparation is completed for upgrading Cloud Pak 3.0"
    info "Please update OperandRequest to upgrade foundational core services"
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
        --enable-licensing)
            ENABLE_LICENSING=1
            ;;
        --enable-private-catalog)
            ENABLE_PRIVATE_CATALOG=1
            ;;
        --cert-manager-source)
            shift
            CERT_MANAGER_SOURCE=$1
            ;;
        --licensing-source)
            shift
            LICENSING_SOURCE=$1
            ;;
        --license-accept)
            LICENSE_ACCEPT=1
            ;;
        -c | --channel)
            shift
            CHANNEL=$1
            ;;
        -i | --install-mode)
            shift
            INSTALL_MODE=$1
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
    echo "Usage: ${script_name} --license-accept --operator-namespace <foundational-services-namespace> [OPTIONS]..."
    echo ""
    echo "Migrate Cloud Pak 2.0 Foundational services to in Cloud Pak 3.0 Foundational services"
    echo "The --license-accept and --operator-namespace <operator-namespace> must be provided."
    echo ""
    echo "Options:"
    echo "   --oc string                    File path to oc CLI. Default uses oc in your PATH"
    echo "   --yq string                    File path to yq CLI. Default uses yq in your PATH"
    echo "   --operator-namespace string    Required. Namespace to migrate Foundational services operator"
    echo "   --services-namespace           Namespace to migrate operands of Foundational services, i.e. 'dataplane'. Default is the same as operator-namespace"
    echo "   --cert-manager-source string   CatalogSource name of ibm-cert-manager-operator. This assumes your CatalogSource is already created. Default is ibm-cert-manager-catalog"
    echo "   --licensing-source string      CatalogSource name of ibm-licensing. This assumes your CatalogSource is already created. Default is ibm-licensing-catalog"
    echo "   --enable-licensing             Set this flag to migrate ibm-licensing-operator"
    echo "   --enable-private-catalog       Set this flag to use namespace scoped CatalogSource. Default is in openshift-marketplace namespace"
    echo "   --license-accept               Set this flag to accept the license agreement."
    echo "   -c, --channel string           Channel for Subscription(s). Default is v4.0"   
    echo "   -i, --install-mode string      InstallPlan Approval Mode. Default is Automatic. Set to Manual for manual approval mode"
    echo "   -s, --source string            CatalogSource name. This assumes your CatalogSource is already created. Default is opencloud-operators"
    echo "   -v, --debug integer            Verbosity of logs. Default is 0. Set to 1 for debug logs."
    echo "   -h, --help                     Print usage information"
    echo ""
}

function pre_req() {
    check_command "${OC}"
    check_command "${YQ}"

    # checking oc command logged in
    user=$(${OC} whoami 2> /dev/null)
    if [ $? -ne 0 ]; then
        error "You must be logged into the OpenShift Cluster from the oc command line"
    else
        success "oc command logged in as ${user}"
    fi

    if [ $LICENSE_ACCEPT -ne 1 ]; then
        error "License not accepted. Rerun script with --license-accept flag set. See https://ibm.biz/integration-licenses for more details"
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

    # Check INSTALL_MODE
    if [[ "$INSTALL_MODE" != "Automatic" && "$INSTALL_MODE" != "Manual" ]]; then
        error "Invalid INSTALL_MODE: $INSTALL_MODE, allowed values are 'Automatic' or 'Manual'"
    fi
    
    # Check if channel is semantic vx.y
    if [[ $CHANNEL =~ ^v[0-9]+\.[0-9]+$ ]]; then
        # Check if channel is equal or greater than v4.0
        if [[ $CHANNEL == v[4-9].* || $CHANNEL == v[4-9] ]]; then  
            success "Channel is valid"
        else
            error "Channel is less than v4.0"
        fi
    else
        error "Channel is not semantic vx.y"
    fi

    NS_LIST=$(${OC} get configmap namespace-scope -n ${OPERATOR_NS} -o jsonpath='{.data.namespaces}')
    if [[ -z "$NS_LIST" ]]; then
        error "Failed to get tenant scope from ConfigMap namespace-scope in namespace ${OPERATOR_NS}"
    fi
    
    get_and_validate_arguments
}

# TODO validate argument
function get_and_validate_arguments() {
    get_control_namespace
}

function debug1() {
    if [ $DEBUG -eq 1 ]; then
       debug "${1}"
    fi
}

main $*