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
}

function parse_arguments() {
    # process options
    while [[ "$@" != "" ]]; do
        case "$1" in
        --oc)
            shift
            OC=$1
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
    echo "Remove controlNamespace and all the remaining resources in control namespace"
    echo "The ibm-cert-manager-operator will be installed in namespace ibm-cert-manager"
    echo "The ibm-licensing-operator will be installed in namespace ibm-licensing"
    echo ""
    echo "Options:"
    echo "   --oc string                                    File path to oc CLI. Default uses oc in your PATH"
    echo "   -h, --help                                     Print usage information"
    echo ""
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

    # checking if there is any CS operator is still in v3.x.x
    info "[Step 1] Checking ibm-common-service-operator channel ..."
    cs_namespace=$(${OC} -n kube-public get cm common-service-maps -o jsonpath='{.data.common-service-maps\.yaml}' | grep 'map-to-common-service-namespace' | awk '{print $2}')
    for ns in $cs_namespace
    do
        csv=$(${OC} get subscription.operators.coreos.com -l operators.coreos.com/ibm-common-service-operator.${ns}='' -n ${ns} -o yaml -o jsonpath='{.items[*].status.installedCSV}')
        # csv=$(${OC} get subscription.operators.coreos.com -l operators.coreos.com/cert-manager.openshift-operators='' -n openshift-operators -o yaml -o jsonpath='{.items[*].status.installedCSV}')
        if [[ "${csv}" != "null" ]] && [[ "${csv}" != "" ]]; then
            info "found ibm-common-service-operator, checking the channel"
            channel=$(echo ${csv} | cut -d "." -f 2 | awk '{print $1}')
            if [[ "${channel}" == "v3" ]]; then
                error "Found ibm-common-service-operator in v3.x version, user need to remove it before running this script"
            fi
        fi
    done
    success "Not found any any ibm-common-service-operator in v3.x version"

    # checking licensing instance
    info "[Step 2] Checking IBMLicensing instance..."


    # checking cert manager 
    info "[Step 3] Checking if there is an available cert-manager in the cluster..."

}

main $*
