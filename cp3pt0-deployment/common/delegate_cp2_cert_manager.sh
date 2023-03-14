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
CONTROL_NS=""
DEBUG=0
PREVIEW_MODE=1

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
    deactivate_cp2_cert_manager
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
    echo "Usage: ${script_name} --control-namespace <cert-manager-namespace> [OPTIONS]..."
    echo ""
    echo "De-activate Certificate Management for IBM Cloud Pak 2.0 Cert Manager."
    echo "The --operator-namespace must be provided."
    echo ""
    echo "Options:"
    echo "   --oc string                    File path to oc CLI. Default uses oc in your PATH"
    echo "   --control-namespace string     Required. Namespace to de-activate Cloud Pak 2.0 Cert Manager services."
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

    if [ "$CONTROL_NS" == "" ]; then
        error "Must provide control namespace"
    fi
}

function deactivate_cp2_cert_manager() {
    title "De-activating IBM Cloud Pak 2.0 Cert Manager in ${CONTROL_NS}...\n"

    info "Configuring Common Services Cert Manager.."
    ${OC} patch configmap ibm-cpp-config -n ${CONTROL_NS} --type='json' -p='[{"op": "add", "path": "/data/deployCSCertManagerOperands", "value": "false"}]' 
    if [ $? -ne 0 ]; then
        error "Failed to patch ibm-cpp-config ConfigMap in ${CONTROL_NS}"
        return 0
    fi

    info "Deleting existing Cert Manager CR..."
    ${OC} delete certmanager.operator.ibm.com default --ignore-not-found

    info "Restarting IBM Cloud Pak 2.0 Cert Manager to provide cert-rotation only..."
    oc delete pod -l name=ibm-cert-manager-operator -n ${CONTROL_NS} --ignore-not-found

    wait_for_no_pod ${CONTROL_NS} "cert-manager-cainjector"
    wait_for_no_pod ${CONTROL_NS} "cert-manager-controller"
    wait_for_no_pod ${CONTROL_NS} "cert-manager-webhook"

    wait_for_pod ${CONTROL_NS} "ibm-cert-manager-operator"

}

function debug1() {
    if [ $DEBUG -eq 1 ]; then
       debug "${1}"
    fi
}

main $*
