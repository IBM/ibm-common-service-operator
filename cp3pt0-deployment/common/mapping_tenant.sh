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
TETHERED_NS=""
CONTROL_NS=""
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
    mapping_topology
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
        --tethered-namespaces)
            shift
            TETHERED_NS=$1
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
    echo "Usage: ${script_name} --operator-namespace <bedrock-namespace> [OPTIONS]..."
    echo ""
    echo "Set up common-service-maps ConfigMap in kube-public namespace for an advanced topology tenant in Cloud Pak 3.0 Foundational services."
    echo "The --operator-namespace must be provided."
    echo ""
    echo "Options:"
    echo "   --oc string                    File path to oc CLI. Default uses oc in your PATH"
    echo "   --yq string                    File path to yq CLI. Default uses yq in your PATH"
    echo "   --operator-namespace string    Required. Namespace to install Foundational services operator"
    echo "   --services-namespace           Namespace to install operands of Foundational services, i.e. 'dataplane'. Default is the same as operator-namespace"
    echo "   --tethered-namespaces string   Additional namespaces for this tenant, comma-delimited, e.g. 'ns1,ns2'"
    echo "   --control-namespace string     Namespace to install Cloud Pak 2.0 cluster singleton Foundational services."
    echo "                                  It is required if there are multiple Cloud Pak 2.0 Foundational services instances or it is a co-existence of Cloud Pak 2.0 and Cloud Pak 3.0"
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

    if [ "$OPERATOR_NS" == "" ]; then
        error "Must provide operator namespace"
    fi

    if [ "$SERVICES_NS" == "" ]; then
        SERVICES_NS=$OPERATOR_NS
    fi
}

function construct_mapping() {
    NEW_MAPPING='- requested-from-namespace:'

    unique_ns_list=()
    # Loop over each tenant namespace and add each unique namespace value to the 'unique' array
    for ns in $OPERATOR_NS $SERVICES_NS ${TETHERED_NS//,/ }; do
        if [[ ! " ${uniqueNsList[@]} " =~ " ${ns} " ]]; then
            unique_ns_list+=("$ns")
        fi
    done

    # Append tenant namespaces to NEW_MAPPING to requested-from-namespace list
    for ns in "${unique_ns_list[@]}"; do
        NEW_MAPPING="$NEW_MAPPING\n  - $ns"
    done

    # Append servicesNamespace to map-to-common-service-namespace
    NEW_MAPPING="$NEW_MAPPING\n  map-to-common-service-namespace: $SERVICES_NS"
}

function mapping_topology() {
    construct_mapping

    # Check if ConfigMap exists in the cluster
    if ${OC} get configmap common-service-maps -n kube-public > /dev/null 2>&1; then
        # ConfigMap exists, retrieve its current data
        local current_mapping=$(${OC} get configmap common-service-maps -n kube-public -o jsonpath='{.data.common-service-maps\.yaml}')

        # Remove the defaultCsNs key-value mapping if it exists
        current_mapping=$(echo "$current_mapping" | awk '/defaultCsNs:/ {next} {print}')

        # Check if servicesNamespace already exists in the map-to-common-service-namespace
        # extract the mapped namespaces from the ConfigMap
        map_to_ns=$(echo "$current_mapping" | yq -r '.namespaceMapping[].map-to-common-service-namespace')
        echo "map_to_ns"
        echo "$map_to_ns"
        if grep -Fxq $SERVICES_NS <<< "$map_to_ns"; then
            info "map-to-common-service-namespace $SERVICES_NS already exists in the namespaceMapping array. Skipping updating common-service-maps ConfigMap"
            return 0
        fi
        
        # Check if each tenant namespace already exists in the requested-from-namespace array
        # extract the requested namespaces from the ConfigMap
        requested_ns=$(echo "$current_mapping" | yq -r '.namespaceMapping[].requested-from-namespace[]')
        echo "requested_ns"
        echo "$requested_ns"
        # loop over each namespace in the list and check if it exists in the ConfigMap
        for ns in $OPERATOR_NS $SERVICES_NS ${TETHERED_NS//,/ }; do
            if grep -Fxq $ns <<< "$requested_ns"; then
                info "requested-from-namespace $ns already exists in the namespaceMapping array. Skipping updating common-service-maps ConfigMap"
                return 0
            fi
        done

        current_control_ns=$(echo "$current_mapping" | awk '/controlNamespace:/ {print $2}')

        # If controlNamespace is not set, assign the value of CONTROL_NS to it
        if [ -z "$current_control_ns" ]; then
            if [ -z "$CONTROL_NS" ]; then
                error "MUST provide control namespace, controlNamespace is not set in common-service-maps ConfigMap"
            else
                info "controlNamespace not set in common-service-maps ConfigMap, setting to $CONTROL_NS"
                current_mapping="controlNamespace: ${CONTROL_NS}\n$current_mapping"
            fi
        else
            # Otherwise, if controlNamespace is set but different from CONTROL_NS, raise an error and abort the script
            if [[ ! -z "$CONTROL_NS" && "$current_control_ns" != "$CONTROL_NS" ]]; then
                error "controlNamespace is set to $current_control_ns but the script receives is $CONTROL_NS for --control-namespace"
            fi
        fi

        # Update ConfigMap data
        info "Updating common-service-maps ConfigMap in kube-public namespace"
        NEW_MAPPING=$(echo -e "$current_mapping\n$NEW_MAPPING")

        local object=$(
            cat <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
    name: common-service-maps
    namespace: kube-public
data:
    common-service-maps.yaml: |
$(echo "$NEW_MAPPING" | awk '{print "        "$0}')
EOF
)

        echo "$object" | ${OC} apply -f -
    else
        # ConfigMap does not exist, create it
        info "Creating common-service-maps ConfigMap in kube-public namespace"

        NEW_MAPPING=$(echo -e "namespaceMapping:\n$NEW_MAPPING")
        local object=$(
            cat <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
    name: common-service-maps
    namespace: kube-public
data:
    common-service-maps.yaml: |
$(echo "$NEW_MAPPING" | awk '{print "        "$0}')
EOF
)
        echo "$object" | ${OC} apply -f -
    fi
}

function debug1() {
    if [ $DEBUG -eq 1 ]; then
       debug "${1}"
    fi
}

main $*
