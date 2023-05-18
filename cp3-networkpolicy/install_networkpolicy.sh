#!/bin/bash

# Licensed Materials - Property of IBM
# Copyright IBM Corporation 2023. All Rights Reserved
# US Government Users Restricted Rights -
# Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
#
# This is an internal component, bundled with an official IBM product. 
# Please refer to that particular license for additional information. 


# ---------- Command variables ----------

# script base directory
BASE_DIR=$(cd $(dirname "$0")/$(dirname "$(readlink $0)") && pwd -P)

# common-services namespace
CS_NAMESPACE=

# operators namespace
OPERATORS_NAMESPACE=

# is uninstall flag?
UNINSTALL=

IFS='
'


# counter to keep track of installation steps
STEP=0

# ---------- Main functions ----------

function msg() {
    printf '%b\n' "$1"
}

function info() {
    msg "[INFO] ${1}"
}

function success() {
    msg "\33[32m[✔] ${1}\33[0m"
}

function warning() {
    msg "\33[33m[✗] ${1}\33[0m"
}

function error() {
    msg "\33[31m[✘] ${1}\33[0m"
    exit 1
}

function title() {
    msg "\33[34m# ${1}\33[0m"
}

function translate_step() {
    local step=$1
    echo "${step}" | tr '[1-9]' '[a-i]'
}

function main() {
    parse_arguments "$@"
    install_policies
}

function install_policies() {
    check_prereqs # TODO: uncomment me

    if [[ -z ${UNINSTALL} ]]; then
        install_networkpolicy
    fi
    if [[ ${UNINSTALL} == "true" ]]; then
        delete_networkpolicy
    fi

    msg "-----------------------------------------------------------------------"
    success "IBM Common Services NetworkPolicies installation or removal completed at $(date) ."
    exit 0
}

function print_usage() {
    script_name=`basename ${0}`
    echo "Usage: ${script_name} [OPTIONS]..."
    echo ""
    echo "Install IBM Common Services NetworkPolicies"
    echo ""
    echo "Options:"
    echo "   -n, --namespace string       IBM Common Services operand namespace. Default is same namespace for both operators and services"
    echo "   -o, --operators-namespace string   Operators namespace. Default is same namespace as IBM Common Services"
    echo "   -u, --uninstall              Uninstall IBM Common Services Network Policies"
    echo "   -e, --egress                 Deploy egress NetworkPolicies"
    echo "   -h, --help                   Print usage information"
    echo ""
}

function parse_arguments() {
    # process options
    while [[ "$1" != "" ]]; do
        case "$1" in
        -n | --namespace)
            shift
            CS_NAMESPACE=$1
            ;;
        -o | --operators-namespace)
            shift
            OPERATORS_NAMESPACE=$1
            ;;
        -u | --uninstall)
            shift
            UNINSTALL=true
            ;;
        -e | --egress)
            shift
            EGRESS=true
            ;;
        -h | --help)
            print_usage
            exit 1
            ;;
        *) 
            ;;
        esac
        shift
    done
}

# ---------- Supporting functions ----------

function check_prereqs() {
    title "[$(translate_step ${STEP})] Checking prerequisites ..."
    msg "-----------------------------------------------------------------------"

    # checking oc command
    if [[ -z "$(command -v oc 2> /dev/null)" ]]; then
        error "oc command not available"
    else
        success "oc command available"
    fi

    # checking oc command logged in
    user=$(oc whoami 2> /dev/null)
    if [ $? -ne 0 ]; then
        error "You must be logged into the OpenShift Cluster from the oc command line"
    else
        success "oc command logged in as ${user}"
    fi

    # checking for CS_NAMESPACE
    if [[ -z "${CS_NAMESPACE}" ]]; then
        CS_NAMESPACE=$(oc project -q)
    fi

    # check existence of CS_NAMESPACE
    cs_namespace_exists=$(oc get project "${CS_NAMESPACE}" 2> /dev/null)
    if [ $? -ne 0 ]; then
        info "Creating IBM Common Services namespace: ${CS_NAMESPACE}"
        oc create namespace "${CS_NAMESPACE}"
    fi

    # checking for ibm-common-service-operator in CS_NAMESPACE
    if [[ -z "$(oc -n ${OPERATORS_NAMESPACE} get csv --ignore-not-found | grep 'ibm-common-service-operator')" ]]; then
        info "IBM Common Services are not installed in namespace ${OPERATORS_NAMESPACE}"
    else
        success "IBM Common Services found in namespace ${OPERATORS_NAMESPACE}"
    fi

    # if OPERATORS_NAMESPACE is not specified, use CS_NAMESPACE
    if [[ -z "${OPERATORS_NAMESPACE}" ]]; then
        OPERATORS_NAMESPACE=${CS_NAMESPACE}
    fi

    # check existence of OPERATORS_NAMESPACE
    operators_namespace_exists=$(oc get project "${OPERATORS_NAMESPACE}" 2> /dev/null)
    if [ $? -ne 0 ]; then
        info "Creating operators namespace: ${OPERATORS_NAMESPACE}"
        oc create namespace "${OPERATORS_NAMESPACE}"
    fi

}

function install_networkpolicy() {
    title "[$(translate_step ${STEP})] Installing IBM Common Services Network Policies ..."
    msg "-----------------------------------------------------------------------"

    info "Using IBM Common Services namespace: ${CS_NAMESPACE}"
    info "Using operators namespace: ${OPERATORS_NAMESPACE}"

    if [[ ${EGRESS} == "true" ]]; then
        BASE_DIR="${BASE_DIR}/egress"
    else
        BASE_DIR="${BASE_DIR}/ingress"
    fi
    
    for policyfile in `ls -1 ${BASE_DIR}/services/*.yaml`; do
        info "Installing `basename ${policyfile}` ..."
        cat ${policyfile} | sed -e "s/csNamespace/${CS_NAMESPACE}/g" | sed -e "s/opNamespace/${OPERATORS_NAMESPACE}/g" | oc apply -f -
    done

    for policyfile in `ls -1 ${BASE_DIR}/operators/*.yaml`; do
        info "Installing `basename ${policyfile}` ..."
        cat ${policyfile} | sed -e "s/csNamespace/${OPERATORS_NAMESPACE}/g" | oc apply -f -
    done

}

function delete_networkpolicy() {
    title "[$(translate_step ${STEP})] Removing IBM Common Services Network Policies ..."
    msg "-----------------------------------------------------------------------"

    oc delete networkpolicies -n ${CS_NAMESPACE} --selector=component=cpfs3
    oc delete networkpolicies -n ${OPERATORS_NAMESPACE} --selector=component=cpfs3

}

# --- Run ---

main $*
