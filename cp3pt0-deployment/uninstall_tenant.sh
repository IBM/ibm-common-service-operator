#!/usr/bin/env bash

# Licensed Materials - Property of IBM
# Copyright IBM Corporation 2023. All Rights Reserved
# US Government Users Restricted Rights -
# Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
#
# This is an internal component, bundled with an official IBM product. 
# Please refer to that particular license for additional information. 

# Base on https://github.ibm.com/IBMPrivateCloud/cs-dev-tools/blob/master/install/cp3pt0-install/uninstall_tenant.sh
# ---------- Command arguments ----------

OC=oc
TENANT_NAMESPACES=""
FORCE_DELETE=0

# ---------- Command variables ----------

# script base directory
BASE_DIR=$(cd $(dirname "$0")/$(dirname "$(readlink $0)") && pwd -P)

# ---------- Main functions ----------

. ${BASE_DIR}/common/utils.sh

function main() {
    parse_arguments "$@"
    pre_req
    set_tenant_namespaces
    uninstall_odlm
    uninstall_cs_operator
    uninstall_nss
    delete_rbac_resource
    delete_webhook
    delete_unavailable_apiservice
    delete_tenant_ns
}

function parse_arguments() {
    # process options
    while [[ "$@" != "" ]]; do
        case "$1" in
        --oc)
            shift
            OC=$1
            ;;
        --operator-namespace)
            shift
            OPERATOR_NS=$1
            ;;
        -f)
            shift
            FORCE_DELETE=1
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
    echo "Set up an advanced topology tenant for Cloud Pak 3.0 Foundational services."
    echo "The --operator-namespace must be provided."
    echo ""
    echo "Options:"
    echo "   --oc string                    File path to oc CLI. Default uses oc in your PATH"
    echo "   --operator-namespace string    Required. Namespace to uninstall Foundational services operators and the whole tenant."
    echo "   -f                             Enable force delete. It will take much more time if you add this label, we suggest run this script without -f label first"
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

    if [ $FORCE_DELETE -eq 1 ]; then
        warning "It will take much more time"
    fi
}

function set_tenant_namespaces() {
    TENANT_NAMESPACES=$(${OC} get -n "$OPERATOR_NS" configmap namespace-scope -o jsonpath='{.data.namespaces}')
    if [ "$TENANT_NAMESPACES" == "" ]; then
        TENANT_NAMESPACES=$OPERATOR_NS
    fi
    info "Tenant namespaces are: $TENANT_NAMESPACES"

    # TODO: have a fallback to populate TENANT_NAMESPACES, so that script
    # can be run multiple times, i.e. handle case where NSS configmap has been
    # deleted, but script hits error before namespace cleanup
    # if [ "$TENANT_NAMESPACES" == "" ]; then
    #     error "Failed to get tenant namespaces"
    # fi
}

function uninstall_odlm() {
    title "Uninstalling OperandRequests and ODLM"

    local grep_args=""
    for ns in ${TENANT_NAMESPACES//,/ }; do
        local opreq=$(${OC} get -n "$ns" operandrequests --no-headers | cut -d ' ' -f1)
        if [ "$opreq" != "" ]; then
            ${OC} delete -n "$ns" operandrequests ${opreq//$'\n'/ }
        fi
        grep_args="${grep_args}-e $ns "
    done

    if [ "$grep_args" == "" ]; then
        grep_args='no-operand-requests'
    fi

    local namespace=$1
    local name=$2
    local condition="${OC} get operandrequests -A --no-headers | cut -d ' ' -f1 | grep -w ${grep_args} || echo Success"
    local retries=20
    local sleep_time=10
    local total_time_mins=$(( sleep_time * retries / 60))
    local wait_message="Waiting for all OperandRequests in tenant namespaces to be deleted"
    local success_message="All tenant OperandRequests deleted"
    local error_message="Timeout after ${total_time_mins} minutes waiting for tenant OperandRequests to be deleted"

    echo "$condition"

    # ideally ODLM will ensure OperandRequests are cleaned up neatly
    wait_for_condition "${condition}" ${retries} ${sleep_time} "${wait_message}" "${success_message}" "${error_message}"

    local sub=$(fetch_sub_from_package ibm-odlm $OPERATOR_NS)
    if [ "$sub" != "" ]; then
        ${OC} delete --ignore-not-found -n "$OPERATOR_NS" sub "$sub"
    fi

    local csv=$(fetch_csv_from_sub operand-deployment-lifecycle-manager "$OPERATOR_NS")
    if [ "$csv" != "" ]; then
        ${OC} delete --ignore-not-found -n "$OPERATOR_NS" csv "$csv"
    fi
}

function uninstall_cs_operator() {
    title "Uninstalling ibm-common-service-operator in tenant namespaces"

    for ns in ${TENANT_NAMESPACES//,/ }; do
        local sub=$(fetch_sub_from_package ibm-common-service-operator $ns)
        if [ "$sub" != "" ]; then
            ${OC} delete --ignore-not-found -n "$ns" sub "$sub"
        fi

        local csv=$(fetch_csv_from_sub "$sub" "$ns")
        if [ "$csv" != "" ]; then
            ${OC} delete --ignore-not-found -n "$ns" csv "$csv"
        fi
    done
}

function uninstall_nss() {
    title "Uninstall ibm-namespace-scope-operator"

    ${OC} delete --ignore-not-found nss -n "$OPERATOR_NS" common-service

    for ns in ${TENANT_NAMESPACES//,/ }; do
        ${OC} delete --ignore-not-found rolebinding "nss-managed-role-from-$OPERATOR_NS"
        ${OC} delete --ignore-not-found role "nss-managed-role-from-$OPERATOR_NS"
    done

    sub=$(fetch_sub_from_package ibm-namespace-scope-operator "$OPERATOR_NS")
    if [ "$sub" != "" ]; then
        ${OC} delete --ignore-not-found -n "$OPERATOR_NS" sub "$sub"
    fi

    csv=$(fetch_csv_from_sub "$sub" "$OPERATOR_NS")
    if [ "$csv" != "" ]; then
        ${OC} delete --ignore-not-found -n "$OPERATOR_NS" csv "$csv"
    fi
}

function delete_webhook() {
    title "Deleting webhookconfigurations in ${TENANT_NAMESPACES}"
    for ns in ${TENANT_NAMESPACES//,/ }; do
        ${OC} delete ValidatingWebhookConfiguration ibm-common-service-validating-webhook-${ns} --ignore-not-found
        ${OC} delete MutatingWebhookConfiguration ibm-common-service-webhook-configuration ibm-operandrequest-webhook-configuration namespace-admission-config ibm-common-service-validating-webhook-${ns} --ignore-not-found
    done
}

function delete_rbac_resource() {
    info "delete rbac resource"
    for ns in ${TENANT_NAMESPACES//,/ }; do
        ${OC} delete ClusterRoleBinding ibm-common-service-webhook secretshare-${ns} $(${OC} get ClusterRoleBinding | grep nginx-ingress-clusterrole | awk '{print $1}') --ignore-not-found
        ${OC} delete ClusterRole ibm-common-service-webhook secretshare nginx-ingress-clusterrole --ignore-not-found
        ${OC} delete scc nginx-ingress-scc --ignore-not-found
    done
}

function delete_unavailable_apiservice() {
    info "delete unavailable apiservice"
    rc=0
    apis=$(${OC} get apiservice | grep False | awk '{print $1}')
    if [ "X${apis}" != "X" ]; then
        warning "Found some unavailable apiservices, deleting ..."
        for api in ${apis}; do
        msg "${OC} delete apiservice ${api}"
        ${OC} delete apiservice ${api}
        if [[ "$?" != "0" ]]; then
            error "Delete apiservcie ${api} failed"
            rc=$((rc + 1))
            continue
        fi
        done
    fi
    return $rc
}

function cleanup_dedicate_cr() {
    for ns in ${TENANT_NAMESPACES//,/ }; do
        cleanup_webhook $ns $TENANT_NAMESPACES
        cleanup_secretshare $ns $TENANT_NAMESPACES
        cleanup_crossplane $ns
    done
}

function delete_tenant_ns() {
    title "Deleting tenant namespaces"
    for ns in ${TENANT_NAMESPACES//,/ }; do
        if [ $FORCE_DELETE -eq 1 ]; then
            title "Force delete remaining resources"
            remove_all_finalizers $ns
        fi
        ${OC} patch namespace ${ns} --type=merge -p '{"spec": {"finalizers":null}}'
        ${OC} delete --ignore-not-found ns "$ns"
    done
    success "Common Services uninstall finished and successfull." 
}



main $*