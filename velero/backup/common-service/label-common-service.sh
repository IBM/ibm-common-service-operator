#!/usr/bin/env bash

# Licensed Materials - Property of IBM
# Copyright IBM Corporation 2023. All Rights Reserved
# US Government Users Restricted Rights -
# Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
#
# This is an internal component, bundled with an official IBM product.
# Please refer to that particular license for additional information.

set -o nounset

# ---------- Command arguments ----------
OC=oc

# Operator and Services namespaces
OPERATOR_NS=""
SERVICES_NS=""
CONTROL_NS=""
CERT_MANAGER_NAMESPACE="ibm-cert-manager"
LICENSING_NAMESPACE="ibm-licensing"
LSR_NAMESPACE="ibm-lsr"

# Catalog sources and namespace
ENABLE_PRIVATE_CATALOG=0
CS_SOURCE_NS="openshift-marketplace"
CM_SOURCE_NS="openshift-marketplace"
LIS_SOURCE_NS="openshift-marketplace"
LSR_SOURCE_NS="openshift-marketplace"
EDB_SOURCE_NS="openshift-marketplace"
CS_SOURCE="opencloud-operators"
CM_SOURCE="ibm-cert-manager-catalog"
LIS_SOURCE="ibm-licensing-catalog"
LSR_SOURCE="ibm-license-service-reporter-bundle-catalog"
EDB_SOURCE="cloud-native-postgresql-catalog"

# default values no change
DEFAULT_SOURCE_NS="openshift-marketplace"

# ---------- Command variables ----------

# script base directory
BASE_DIR=$(cd $(dirname "$0")/$(dirname "$(readlink $0)") && pwd -P)

# ---------- Main functions ----------

. ${BASE_DIR}/../../../cp3pt0-deployment/common/utils.sh

source ${BASE_DIR}/local.properties

function main() {

    pre_req
    label_catalogsource
    label_ns_and_related 
    label_configmap
    label_subscription
    label_cs
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
    if [ "$OPERATOR_NS" == "" ]; then
        error "Must provide operator namespace"
    fi
    if [ "$SERVICES_NS" == "" ]; then
        error "Must provide services namespace, using ibm-common-services for v3 operator"
    fi
}

function label_catalogsource() {

    title "Start to label the catalog sources... "
    # Label the Private CatalogSources in provided namespaces
    if [ $ENABLE_PRIVATE_CATALOG -eq 1 ]; then
        CS_SOURCE_NS=$OPERATOR_NS
        CM_SOURCE_NS=$CERT_MANAGER_NAMESPACE
        LIS_SOURCE_NS=$LICENSING_NAMESPACE
        LSR_SOURCE_NS=$LSR_NAMESPACE
        EDB_SOURCE_NS=$OPERATOR_NS

        ${OC} label catalogsource ibm-operator-catalog foundationservices.cloudpak.ibm.com=catalog -n $DEFAULT_SOURCE_NS --overwrite=true 2>/dev/null 
        ${OC} label catalogsource $CS_SOURCE foundationservices.cloudpak.ibm.com=catalog -n $CS_SOURCE_NS --overwrite=true 2>/dev/null 
        ${OC} label catalogsource $CM_SOURCE foundationservices.cloudpak.ibm.com=catalog -n $CM_SOURCE_NS --overwrite=true 2>/dev/null 
        ${OC} label catalogsource $LIS_SOURCE foundationservices.cloudpak.ibm.com=catalog -n $LIS_SOURCE_NS --overwrite=true 2>/dev/null 
        ${OC} label catalogsource $EDB_SOURCE foundationservices.cloudpak.ibm.com=catalog -n $EDB_SOURCE_NS --overwrite=true 2>/dev/null
    fi

    # Label the CatalogSource with .spec.publisher having "IBM" in openshift-marketplace namespace
    local ibm_catalog_sources=""
    while IFS=' ' read -r -a sources; do
        for source in "${sources[@]}"; do
            if ${OC} get catalogsource "$source" -n "$DEFAULT_SOURCE_NS" -o json | grep -q '"publisher": *"IBM"*'; then
                ibm_catalog_sources+=" $source"
            fi
        done
    done <<< "$(${OC} get catalogsource -n "$DEFAULT_SOURCE_NS" -o jsonpath='{.items[*].metadata.name}')"

    # Remove leading/trailing whitespace
    ibm_catalog_sources=$(echo "${ibm_catalog_sources}" | tr -s ' ' | sed 's/^ *//g' | sed 's/ *$//g')
    for source in $ibm_catalog_sources; do
        ${OC} label catalogsource "$source" foundationservices.cloudpak.ibm.com=catalog -n $DEFAULT_SOURCE_NS --overwrite=true 2>/dev/null
    done
}

function label_ns_and_related() {

    title "Start to label the namespaces, operatorgroups and secrets... "
    namespaces=$(${OC} get configmap namespace-scope -n $OPERATOR_NS -oyaml | awk '/^data:/ {flag=1; next} /^  namespaces:/ {print $2; next} flag && /^  [^ ]+: / {flag=0}')
    # add cert-manager namespace and licensing namespace and lsr namespace into the list with comma separated
    namespaces+=",$CONTROL_NS,$CERT_MANAGER_NAMESPACE,$LICENSING_NAMESPACE,$LSR_NAMESPACE"
    namespaces=$(echo "$namespaces" | tr ',' '\n')

    while IFS= read -r namespace; do
        # Label the namespace
        ${OC} label namespace "$namespace" foundationservices.cloudpak.ibm.com=namespace --overwrite=true 2>/dev/null
        
        # Label the OperatorGroup
        operator_group=$(${OC} get operatorgroup -n "$namespace" -o jsonpath='{.items[*].metadata.name}')
        ${OC} label operatorgroup "$operator_group" foundationservices.cloudpak.ibm.com=operatorgroup -n "$namespace" --overwrite=true 2>/dev/null
        
        # Label the entitlement key
        ${OC} label secret ibm-entitlement-key foundationservices.cloudpak.ibm.com=entitlementkey -n "$namespace" --overwrite=true 2>/dev/null
        
        # Label the OperandRequest
        operand_requests=$(${OC} get operandrequest -n "$namespace" -o custom-columns=NAME:.metadata.name --no-headers)
        # Loop through each OperandRequest name
        while IFS= read -r operand_request; do
            ${OC} label operandrequests $operand_request foundationservices.cloudpak.ibm.com=operand -n "namespace" --overwrite=true 2>/dev/null
        done <<< "$operand_requests"
    done <<< "$namespaces"

    ${OC} label secret pull-secret -n openshift-config foundationservices.cloudpak.ibm.com=pull-secret --overwrite=true

}

function label_configmap() {
    
    title "Start to label the ConfigMaps... "
    ${OC} label configmap common-service-maps foundationservices.cloudpak.ibm.com=configmap -n kube-public --overwrite=true 2>/dev/null
    ${OC} label configmap cs-onprem-tenant-config foundationservices.cloudpak.ibm.com=configmap -n $SERVICES_NS --overwrite=true 2>/dev/null
    ${OC} label configmap platform-auth-idp foundationservices.cloudpak.ibm.com=configmap -n $SERVICES_NS --overwrite=true 2>/dev/null
}

function label_subscription() {

    title "Start to label the Subscriptions... "
    local cs_pm="ibm-common-service-operator"
    local cm_pm="ibm-cert-manager-operator"
    local lis_pm="ibm-licensing-operator-app"
    local lsr_pm="ibm-license-service-reporter-operator"
    
    ${OC} label subscriptions.operators.coreos.com $cs_pm foundationservices.cloudpak.ibm.com=subscription -n $OPERATOR_NS --overwrite=true 2>/dev/null
    ${OC} label subscriptions.operators.coreos.com $cm_pm foundationservices.cloudpak.ibm.com=subscription -n $CERT_MANAGER_NAMESPACE --overwrite=true 2>/dev/null
    ${OC} label subscriptions.operators.coreos.com $lis_pm foundationservices.cloudpak.ibm.com=subscription -n $LICENSING_NAMESPACE --overwrite=true 2>/dev/null
    ${OC} label subscriptions.operators.coreos.com $lsr_pm foundationservices.cloudpak.ibm.com=subscription -n $LSR_NAMESPACE --overwrite=true 2>/dev/null
}

function label_cs(){
    
    title "Start to label the CommonService CR... "
    ${OC} label commonservices common-service foundationservices.cloudpak.ibm.com=commonservice -n $OPERATOR_NS --overwrite=true 2>/dev/null
    ${OC} label customresourcedefinition commonservices.operator.ibm.com foundationservices.cloudpak.ibm.com=crd --overwrite=true 2>/dev/null
    ${OC} label operandconfig common-service foundationservices.cloudpak.ibm.com=operand -n $SERVICES_NS --overwrite=true 2>/dev/null
}

main $*

# ---------------- finish ----------------