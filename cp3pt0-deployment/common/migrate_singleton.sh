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
LICENSING_NS=""
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

    # delegate certmanager cr if control namespace exists
    # delete certmanager operator if control namespace does not exist
    # if enable licensing and licensing operator is in operator ns
    #   migrate licensing to LICENSING_NS
    #   delete operator

    parse_arguments "$@"
    pre_req

    if [ ! -z "$CONTROL_NS" ]; then
        # Delegation of CP2 Cert Manager
        ${BASE_DIR}/delegate_cp2_cert_manager.sh --control-namespace $CONTROL_NS "--skip-user-vertify"

        # Delete CP2.0 Cert-Manager CR
        ${OC} delete certmanager.operator.ibm.com default --ignore-not-found --timeout=10s
        if [ $? -ne 0 ]; then
            warning "Failed to delete Cert Manager CR, patching its finalizer to null..."
            ${OC} patch certmanagers.operator.ibm.com default --type="json" -p '[{"op": "remove", "path":"/metadata/finalizers"}]'
        fi
        msg ""
        
        wait_for_no_pod ${CONTROL_NS} "cert-manager-cainjector"
        wait_for_no_pod ${CONTROL_NS} "cert-manager-controller"
        wait_for_no_pod ${CONTROL_NS} "cert-manager-webhook"
    fi

    delete_operator "ibm-cert-manager-operator" "$OPERATOR_NS"
    
    if [[ $ENABLE_LICENSING -eq 1 ]]; then

        is_exists=$("$OC" get deployments ibm-licensing-operator -n "$OPERATOR_NS")
        if [ ! -z "is_exists" ]; then
            # Migrate Licensing Services Data
            ${BASE_DIR}/migrate_cp2_licensing.sh --control-namespace "$OPERATOR_NS" --target-namespace "$LICENSING_NS" "--skip-user-vertify"
            local is_deleted=$(("${OC}" delete -n "${CONTROL_NS}" --ignore-not-found OperandBindInfo ibm-licensing-bindinfo --timeout=10s > /dev/null && echo "success" ) || echo "fail")
            if [[ $is_deleted == "fail" ]]; then
                warning "Failed to delete OperandBindInfo, patching its finalizer to null..."
                ${OC} patch -n "${CONTROL_NS}" OperandBindInfo ibm-licensing-bindinfo --type="json" -p '[{"op": "remove", "path":"/metadata/finalizers"}]'
            fi
        fi
        
        backup_ibmlicensing
        isExists=$("${OC}" get deployments -n "${CONTROL_NS}" --ignore-not-found ibm-licensing-operator)
        if [ ! -z "$isExists" ]; then
            "${OC}" delete  --ignore-not-found ibmlicensing instance
        fi

        # Delete licensing csv/subscriptions
        delete_operator "ibm-licensing-operator" "$OPERATOR_NS"

        # restore licensing configuration so that subsequent License Service install will pick them up
        restore_ibmlicensing
    fi

    success "Migration is completed for Cloud Pak 3.0 Foundational singleton services."
}


function restore_ibmlicensing() {

    # extracts the previously saved IBMLicensing CR from ConfigMap and creates the IBMLicensing CR
    "${OC}" get cm ibmlicensing-instance-bak -n ${CONTROL_NS} -o yaml --ignore-not-found | "${YQ}" .data | sed -e 's/.*ibmlicensing.yaml.*//' | 
    sed -e 's/^  //g' | oc apply -f -

}

function backup_ibmlicensing() {

    instance=`"${OC}" get IBMLicensing instance -o yaml --ignore-not-found | "${YQ}" '
        with(.; del(.metadata.creationTimestamp) |
        del(.metadata.managedFields) |
        del(.metadata.resourceVersion) |
        del(.metadata.uid) |
        del(.status)
        )
    ' | sed -e 's/^/    /g'`
cat << _EOF | oc apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: ibmlicensing-instance-bak
  namespace: ${CONTROL_NS}
data:
  ibmlicensing.yaml: |
${instance}
_EOF

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
        --control-namespace)
            shift
            CONTROL_NS=$1
            ;;
        --licensing-namespace)
            shift
            LICENSING_NS=$1
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
}

main $*