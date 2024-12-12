#!/usr/bin/env bash

# Licensed Materials - Property of IBM
# Copyright IBM Corporation 2023. All Rights Reserved
# US Government Users Restricted Rights -
# Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
#
# This is an internal component, bundled with an official IBM product.
# Please refer to that particular license for additional information.

set -o errtrace
set -o nounset

# ---------- Command arguments ----------
OC=oc
LSR_NAMESPACE="ibm-lsr"

# Catalog sources and namespace
ENABLE_PRIVATE_CATALOG=0
LSR_SOURCE="ibm-license-service-reporter-catalog"
LSR_SOURCE_NS="openshift-marketplace"

# ---------- Command variables ----------

# script base directory
BASE_DIR=$(cd $(dirname "$0")/$(dirname "$(readlink $0)") && pwd -P)

# ---------- Main functions ----------
function main() {
    parse_arguments "$@"
    pre_req
    label_catalogsource
    label_ns_and_related 
    label_subscription
    label_lsr
    success "Successfully labeled all the resources"
}

function print_usage(){ #TODO update usage definition
    script_name=`basename ${0}`
    echo "Usage: ${script_name} [OPTIONS]"
    echo ""
    echo "Label License Service Reporter resources to prepare for Backup."
    echo "License Service Reporter namespace is always required."
    echo ""
    echo "Options:"
    echo "   --oc string                    Optional. File path to oc CLI. Default uses oc in your PATH. Can also be set in env.properties."
    echo "   --lsr-ns                       Optional. Specifying will enable labeling of the license service reporter operator. Permissions may need to be updated to include the namespace."
    echo "   --enable-private-catalog       Optional. Specifying will look for catalog sources in the operator namespace. If enabled, will look for license service reporter in its respective namespaces."
    echo "   --lsr-catalog                  Optional. Specifying will look for the license service reporter catalog source name."
    echo "   --lsr-catalog-ns               Optional. Specifying will look for the license service reporter catalog source namespace."
    echo "   -h, --help                     Print usage information"
    echo ""
    
}

function parse_arguments() {
    script_name=`basename ${0}`
    echo "All arguments passed into the ${script_name}: $@"
    echo ""

    # process options
    while [[ "$@" != "" ]]; do
        case "$1" in
        --oc)
            shift
            OC=$1
            ;;
        --lsr-ns )
            shift
            LSR_NAMESPACE=$1
            ;;
        --enable-private-catalog)
            ENABLE_PRIVATE_CATALOG=1
            ;;
        --lsr-catalog)
            shift
            LSR_SOURCE=$1
            ;;
        --lsr-catalog-ns)
            shift
            LSR_SOURCE_NS=$1
            ;;
        -h | --help)
            print_usage
            exit 1
            ;;
        *)
            echo "Entered option $1 not supported. Run ./${script_name} -h for script usage info."
            ;;
        esac
        shift
    done
    echo ""
}

function pre_req(){

    title "Start to validate the parameters passed into script... "
    # Checking oc command logged in
    user=$($OC whoami 2> /dev/null)
    if [ $? -ne 0 ]; then
        error "You must be logged into the OpenShift Cluster from the oc command line"
    else
        success "oc command logged in as ${user}"
    fi
}

function label_catalogsource() {

    title "Start to label the License Service Reporter catalog sources... "
    # Label the Private CatalogSources in provided namespaces
    if [ $ENABLE_PRIVATE_CATALOG -eq 1 ]; then
        LSR_SOURCE_NS=$LSR_NAMESPACE
    fi
    ${OC} label catalogsource "$LSR_SOURCE" foundationservices.cloudpak.ibm.com=lsr -n "$CSR_SOURCE_NS" --overwrite=true 2>/dev/null
    echo ""
}

function label_ns_and_related() {

    title "Start to label the namespaces, operatorgroups... "

    # Label the cert manager namespace
    ${OC} label namespace "$LSR_NAMESPACE" foundationservices.cloudpak.ibm.com=lsr --overwrite=true 2>/dev/null
    
    # Label the cert manager OperatorGroup
    operator_group=$(${OC} get operatorgroup -n "$LSR_NAMESPACE" -o jsonpath='{.items[*].metadata.name}')
    ${OC} label operatorgroup "$operator_group" foundationservices.cloudpak.ibm.com=lsr -n "$LSR_NAMESPACE" --overwrite=true 2>/dev/null
    
    echo ""
}


function label_subscription() {

    title "Start to label the Subscriptions... "
    local lsr_pm="ibm-license-service-reporter-operator"
    ${OC} label subscriptions.operators.coreos.com $lsr_pm foundationservices.cloudpak.ibm.com=lsr -n $LSR_NAMESPACE --overwrite=true 2>/dev/null
    echo ""
}

function label_lsr() {
    
    title "Start to label the License Service Reporter... "
    ${OC} label customresourcedefinition ibmlicenseservicereporters.operator.ibm.com foundationservices.cloudpak.ibm.com=lsr --overwrite=true 2>/dev/null

    info "Start to label the LSR instances"
    lsr_instances=$(${OC} get ibmlicenseservicereporters.operator.ibm.com -n $LSR_NAMESPACE -o jsonpath='{.items[*].metadata.name}')
    while IFS= read -r lsr_instance; do
        ${OC} label ibmlicenseservicereporters.operator.ibm.com $lsr_instance foundationservices.cloudpak.ibm.com=lsr -n $LSR_NAMESPACE --overwrite=true 2>/dev/null
        
        # Label the secrets with OIDC configured
        client_secret_name=$(${OC} get ibmlicenseservicereporters.operator.ibm.com $lsr_instance -n $LSR_NAMESPACE -o yaml | awk -F '--client-secret-name=' '{print $2}' | tr -d '"' | tr -d '\n')
        ${OC} label secret $client_secret_name foundationservices.cloudpak.ibm.com=lsr -n $LSR_NAMESPACE --overwrite=true 2>/dev/null

        provider_ca_secret_name=$(${OC} get ibmlicenseservicereporters.operator.ibm.com $lsr_instance -n $LSR_NAMESPACE -o yaml | awk -F '--provider-ca-secret-name=' '{print $2}' | tr -d '"' | tr -d '\n')
        ${OC} label secret $provider_ca_secret_name foundationservices.cloudpak.ibm.com=lsr -n $LSR_NAMESPACE --overwrite=true 2>/dev/null
    done <<< "$lsr_instances"

    info "Start to label the necessary secrets"
    secrets=$(${OC} get secrets -n $LSR_NAMESPACE | grep ibm-license-service-reporter-token | cut -d ' ' -f1)
    for secret in ${secrets[@]}; do
        ${OC} label secret $secret foundationservices.cloudpak.ibm.com=lsr -n $LSR_NAMESPACE --overwrite=true 2>/dev/null
    done    
    secrets=$(${OC} get secrets -n $LSR_NAMESPACE | grep ibm-license-service-reporter-credential | cut -d ' ' -f1)
    for secret in ${secrets[@]}; do
        ${OC} label secret $secret foundationservices.cloudpak.ibm.com=lsr -n $LSR_NAMESPACE --overwrite=true 2>/dev/null
    done

    echo ""
}

# ---------- Info functions ----------#

function msg() {
    printf '%b\n' "$1"
}

function success() {
    msg "\33[32m[✔] ${1}\33[0m"
}

function error() {
    msg "\33[31m[✘] ${1}\33[0m"
    exit 1
}

function title() {
    msg "\33[34m# ${1}\33[0m"
}

function info() {
    msg "[INFO] ${1}"
}

function warning() {
    msg "\33[33m[✗] ${1}\33[0m"
}

main $*

# ---------------- finish ----------------
