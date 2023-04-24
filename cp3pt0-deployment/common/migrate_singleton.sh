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
CONTROL_NS=""
SOURCE_NS="openshift-marketplace"
ENABLE_LICENSING=0
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

. ${BASE_DIR}/utils.sh

function main() {
    parse_arguments "$@"
    pre_req

    if [ "$CONTROL_NS" == "$OPERATOR_NS" ]; then
        # Delete CP2.0 Cert-Manager CR
        ${OC} delete certmanager.operator.ibm.com default --ignore-not-found
        # Delete cert-Manager
        delete_operator "ibm-cert-manager-operator" "$CONTROL_NS"
    else
        # Delgation of CP2 Cert Manager
        ${BASE_DIR}/delegate_cp2_cert_manager.sh --control-namespace $CONTROL_NS "--skip-user-vertify"
    fi
    
    if [[ $ENABLE_LICENSING -eq 1 ]]; then
        if [[ "$CONTROL_NS" == "$OPERATOR_NS" ]]; then
            # Migrate Licensing Services Data
            ${BASE_DIR}/migrate_cp2_licensing.sh --control-namespace $CONTROL_NS "--skip-user-vertify"
        fi
        # Delete IBM Licensing Service instance
        ${OC} delete --ignore-not-found ibmlicensing instance
        # Delete licensing csv/subscriptions
        delete_operator "ibm-licensing-operator" "$CONTROL_NS"
    fi

    success "Migration is completed for Cloud Pak 3.0 Foundational singleton services."
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
        --enable-licensing)
            ENABLE_LICENSING=1
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
    echo "Usage: ${script_name} --operator-namespace <foundational-services-namespace> [OPTIONS]..."
    echo ""
    echo "Migrate Cloud Pak 2.0 Foundational singleton services to in Cloud Pak 3.0 Foundational singleton services"
    echo "The --operator-namespace must be provided."
    echo ""
    echo "Options:"
    echo "   --oc string                                    File path to oc CLI. Default uses oc in your PATH"
    echo "   --yq string                                    File path to yq CLI. Default uses yq in your PATH"
    echo "   --operator-namespace string                    Required. Namespace to migrate Foundational services operator"
    echo "   --enable-licensing                             Set this flag to migrate ibm-licensing-operator"
    echo "   -v, --debug integer                            Verbosity of logs. Default is 0. Set to 1 for debug logs."
    echo "   -h, --help                                     Print usage information"
    echo ""
}

function pre_req() {
    if [ "$OPERATOR_NS" == "" ]; then
        error "Must provide operator namespace"
    fi

    if [ "$CONTROL_NS" == "" ]; then
        CONTROL_NS=$OPERATOR_NS
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