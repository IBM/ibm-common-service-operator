#!/usr/bin/env bash
#
# Copyright 2022 IBM Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

set -o errexit
set -o pipefail
set -o errtrace
set -o nounset

OC=oc
YQ=yq
master_ns=
excluded_raw=
insert_raw=
map_to_cs_ns=
requested_ns=
OPERATOR_NS=""
SERVICES_NS=""
TETHERED_NS=""
CONTROL_NS=""
NEW_MAPPING=""
cm_name="common-service-maps"
cm_maps=

function main () {
    while [ "$#" -gt "0" ]
    do
        case "$1" in
        "-h"|"--help")
            usage
            exit 0
            ;;
        "--original-cs-ns")
            master_ns=$2
            shift
            ;;
        "--excluded-ns")
            excluded_raw=$2
            shift
            ;;
        "--insert-ns")
            insert_raw=$2
            shift
            ;;
        "--control-ns")
            CONTROL_NS=$2
            shift
            ;;
        *)
            error "invalid option -- \`$1\`. Use the -h or --help option for usage info."
            ;;
        esac
        shift
    done
    which "${OC}" || error "Missing oc CLI"
    which "${YQ}" || error "Missing yq"

    if [[ -z $CONTROL_NS ]] &&  [[ -z $master_ns ]]; then
        usage
        error "No parameters entered. Please re-run specifying original and control namespace values. Use -h for help."
    elif [[ -z $CONTROL_NS ]] || [[ -z $master_ns ]]; then
        usage
        error "Required parameters missing. Please re-run specifying original and control namespace values. Use -h for help."
    fi
    #need to get the namespaces for csmaps generation before pausing cs, otherwise namespace-scope cm does not include all namespaces
    cm_maps=$(oc get -n kube-public cm ${cm_name} -o yaml | yq '.data.["common-service-maps.yaml"]')
}

function usage() {
	local script="${0##*/}"

	while read -r ; do echo "${REPLY}" ; done <<-EOF
	Usage: ${script} [OPTION]...
	Isolate and prepare common services for upgrade
	Options:
	Mandatory arguments to long options are mandatory for short options too.
    -h, --help                    display this help and exit
    --original-cs-ns              specify the namespace the original common services installation resides in
    --control-ns                  specify the control namespace value in the common-service-maps configmap
    --excluded-ns                 specify namespaces to be excluded from the common-service-maps configmap. Comma separated no spaces.
    --insert-ns                 specify namespaces to be inserted into the common-service-maps configmap. Comma separated no spaces.
	EOF
}

function create_empty_csmaps() {
    title " Creating empty common-service-maps configmap "
    #what does an empty cs maps look like? Is the datat fiel empty, the embedded yaml empty, or the requested from/map to empty?
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

    success "Empty common-service-maps configmap created in kube-public namespace"

}

function insert_control_ns() {
    return_value=$("${OC}" get configmap -n kube-public -o yaml ${cm_name} | yq '.data' | grep controlNamespace: > /dev/null || echo failed)
    if [[ $return_value == "failed" ]]; then
        #insert control namespace
        ${OC} get configmap -n kube-public -o yaml ${cm_name} > tmp.csmaps.yaml
        yq -i '.data.["common-service-maps.yaml"].controlNamespace = "'${CONTROL_NS}'"' tmp.yaml #only edits existing does not add new field
    fi
    return_value="reset"
}

#the same logic as "gather_csmaps_ns" from isolate script
function compute_ns_list () {
    #read namespaces from ns scope cm as well as exclude/include parameters
    return_value=$(${OC} get -n ${master_ns} cm namespace-scope > /dev/null || echo failed)
    if [[ $return_value == "failed" ]]; then
        error "No namespace-scope configmap in Original CS Namespace ${master_ns}. Verify namespace is correct and IBM common services is installed there."
    else
        namespaces=$(oc get cm namespace-scope -n "${master_ns}" -o json | jq '.data.namespaces')
        #output namespace-scope cm
        echo $(oc get cm namespace-scope -n "${master_ns}" -o yaml)
        namespaces=$(echo $namespaces | tr -d '"')
        IFS=',' read -a nsFromCM <<< "$namespaces"
    fi
    #remove excluded from namespaces
    if [[ $excluded_raw != "" ]]; then
        IFS=',' read -a excluded_ns <<< "$excluded_raw"
        #this is very ugly but very consistent and these lists should not be too long anyway
        for ns in ${nsFromCM[@]}
        do
            skip=0
            for exns in ${excluded_ns[@]}
            do
                if [[ $ns == $exns ]]; then
                    skip=1
                    break
                fi
            done
            if [[ $ns == $master_ns ]]; then
                skip=1
            fi
            if [[ $skip != 1 ]]; then
                if [[ $TETHERED_NS == "" ]]; then
                    TETHERED_NS="$ns"
                else
                    TETHERED_NS="$TETHERED_NS $ns"
                fi
            fi
        done
    else
        echo "excluded empty"
        echo "ns from cm $nsFromCM"
        for ns in ${nsFromCM[@]}
        do
            if [[ $ns != $master_ns ]]; then
                if [[ $TETHERED_NS == "" ]]; then
                    TETHERED_NS="$ns"
                else
                    TETHERED_NS="$TETHERED_NS $ns"
                fi
            fi
        done
    fi
    if [[ $insert_raw != "" ]]; then
        IFS=',' read -a insertNS <<< "$insert_raw"
        for ns in ${insertNS[@]}
        do
            if [[ $TETHERED_NS == "" ]]; then
                TETHERED_NS="$ns"
            else
                TETHERED_NS="$TETHERED_NS $ns"
            fi
        done
    fi
    if [[ $TETHERED_NS == "" ]]; then
        TETHERED_NS=$master_ns
    fi

    OPERATOR_NS=$master_ns
    SERVICES_NS=$master_ns
    info "common-service-maps namespaces: $TETHERED_NS"
}

function read_tenant_from_csmaps() {
    local map_to_ns=$1
    local tenant_ns_list=$(echo "$cm_maps" | yq eval '.namespaceMapping[] | select(.map-to-common-service-namespace == "'${map_to_ns}'").requested-from-namespace' | awk '{ print $2 }')
    echo $tenant_ns_list
}

function update_tenant() {
    map_to_cs_ns=$1
    namespaces=$1
    requested_ns_delimited=""
    for ns in $namespaces
    do
        if [[ $requested_ns_delimited=="" ]]; then
            requested_ns_delimited="\"$ns\""
        else
            requested_ns_delimited="$requested_ns_delimited,\"$ns\""
        fi
    done
    echo "$cm_maps" | yq eval 'with(.namespaceMapping[]; select(.map-to-common-service-namespace == "'${map_to_ns}'").requested-from-namespace = ["'${requested_ns_delimited}'"])'
}

function compare_ns_lists() {
    #compare sorted compute_ns_list and sorted list from common-service-maps
    #if different
        #update tenant entry in common-service-maps
}

#the same logic as "contstruct_mapping" from isolate script
function construct_mapping() {
    NEW_MAPPING='- requested-from-namespace:'

    local unique_ns_list=$(echo $OPERATOR_NS $SERVICES_NS $TETHERED_NS | tr ' ' '\n' | sort | uniq | tr '\n' ' ')

    # Append tenant namespaces to NEW_MAPPING to requested-from-namespace list
    for ns in $unique_ns_list; do
        NEW_MAPPING="$NEW_MAPPING\n  - $ns"
    done

    # Append servicesNamespace to map-to-common-service-namespace
    NEW_MAPPING="$NEW_MAPPING\n  map-to-common-service-namespace: $SERVICES_NS"
}

#the same logic as "mapping_topology" from isolate script
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
        for ns in $OPERATOR_NS $SERVICES_NS $TETHERED_NS; do
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

        NEW_MAPPING=$(echo -e "controlNamespace: $CONTROL_NS\nnamespaceMapping:\n$NEW_MAPPING")
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

function msg() {
    printf '%b\n' "$1"
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

function info() {
    msg "[INFO] ${1}"
}

# --- Run ---

main $*