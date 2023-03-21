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
ENABLE_LICENSING=0
ENABLE_PRIVATE_CATALOG=0
CHANNEL="v4.0"
SOURCE_NS="openshift-marketplace"
INSTALL_MODE="Automatic"
CERT_MANAGER_SOURCE="ibm-cert-manager-operator-catalog"
LICENSING_SOURCE="ibm-licensing-catalog"

SKIP_INSTALL=false

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
    install_cert_manager
    install_licensing
}

function parse_arguments() {
    # process options
    while [[ "$@" != "" ]]; do
        case "$1" in
        --oc)
            shift
            OC=$1
            ;;
        --enable-licensing)
            ENABLE_LICENSING=1
            ;;
        --enable-private-catalog)
            ENABLE_PRIVATE_CATALOG=1
            ;;
        -c | --channel)
            shift
            CHANNEL=$1
            ;;
        --cert-manager-source)
            shift
            CERT_MANAGER_SOURCE=$1
            ;;
        --licensing-source)
            shift
            LICENSING_SOURCE=$1
            ;;
        --check-cert-manager)
            SKIP_INSTALL=true
            ;;
        --check-licensing)
            CHECK_LICENSING_ONLY=1
            SKIP_INSTALL=true           
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
    echo "Usage: ${script_name} [OPTIONS]..."
    echo ""
    echo "Install Cloud Pak 3 pre-reqs if they do not already exist: ibm-cert-manager-operator and optionally ibm-licensing-operator"
    echo "The ibm-cert-manager-operator will be installed in namespace ibm-cert-manager"
    echo "The ibm-licensing-operator will be installed in namespace ibm-licensing"
    echo ""
    echo "Options:"
    echo "   --oc string                    File path to oc CLI. Default uses oc in your PATH"
    echo "   --enable-licensing             Set this flag to install ibm-licensing-operator"
    echo "   --enable-private-catalog       Set this flag to use namespace scoped CatalogSource. Default is in openshift-marketplace namespace"
    echo "   --cert-manager-source string   CatalogSource name of ibm-cert-manager-operator. This assumes your CatalogSource is already created. Default is ibm-cert-manager-operator-catalog"
    echo "   --licensing-source string      CatalogSource name of ibm-licensing. This assumes your CatalogSource is already created. Default is ibm-licensing-catalog"
    echo "   -c, --channel string           Channel for Subscription(s). Default is v4.0"   
    echo "   -i, --install-mode string      InstallPlan Approval Mode. Default is Automatic. Set to Manual for manual approval mode"
    echo "   -h, --help                     Print usage information"
    echo ""
}

function install_cert_manager() {
    if [ $CHECK_LICENSING_ONLY -eq 1 ]; then
        return
    fi

    title "Installing cert-manager\n"
    is_sub_exist "cert-manager" # this will catch the packagenames of all cert-manager-operators
    if [ $? -eq 0 ]; then
        warning "There is a cert-manager Subscription already\n"
        return 0
    fi

    pods_exist=$(${OC} get pods -A | grep -w cert-manager-webhook)
    if [ $? -eq 0 ]; then
        warning "There is a cert-manager-webhook pod Running, so most likely another cert-manager is already installed\n"
        return 0
    fi

    if [ $ENABLE_PRIVATE_CATALOG -eq 1 ]; then
        SOURCE_NS="ibm-cert-manager"
    fi

    if [ $SKIP_INSTALL = true ]; then
        wait_for_operator "ibm-cert-manager" "ibm-cert-manager-operator" 6 # only wait 1 min by retrying 6 times
        return
    fi

    create_namespace ibm-cert-manager
    create_operator_group "ibm-cert-manager-operator" "ibm-cert-manager" "{}"
    create_subscription "ibm-cert-manager-operator" "ibm-cert-manager" "$CHANNEL" "ibm-cert-manager-operator" "${CERT_MANAGER_SOURCE}" "${SOURCE_NS}" "${INSTALL_MODE}"
    wait_for_operator "ibm-cert-manager" "ibm-cert-manager-operator"
   
}

function install_licensing() {
    if [ $ENABLE_LICENSING -ne 1 ] && [ $CHECK_LICENSING_ONLY -ne 1 ]; then
        return
    fi

    title "Installing licensing\n"
    is_sub_exist "ibm-licensing-operator-app" # this will catch the packagenames of all cert-manager-operators
    if [ $? -eq 0 ]; then
        warning "There is an ibm-licensing-operator Subscription already\n"
        return 0
    fi

    if [ $ENABLE_PRIVATE_CATALOG -eq 1 ]; then
        SOURCE_NS="ibm-licensing"
    fi

    if [ $SKIP_INSTALL = true ]; then
        wait_for_operator "ibm-licensing" "ibm-licensing-operator" 6 # only wait 1 min by retrying 6 times
        return
    fi

    create_namespace ibm-licensing

    target=$(cat <<EOF
        
  targetNamespaces:
    - ibm-licensing
EOF
)
    create_operator_group "ibm-licensing-operator-app" "ibm-licensing" "$target"
    create_subscription "ibm-licensing-operator-app" "ibm-licensing" "$CHANNEL" "ibm-licensing-operator-app" "${LICENSING_SOURCE}" "${SOURCE_NS}" "${INSTALL_MODE}"
    wait_for_operator "ibm-licensing" "ibm-licensing-operator"
}

function pre_req() {
    check_command "${OC}"

    # checking oc command logged in
    user=$(oc whoami 2> /dev/null)
    if [ $? -ne 0 ]; then
        error "You must be logged into the OpenShift Cluster from the oc command line"
    else
        success "oc command logged in as ${user}"
    fi
}

main $*