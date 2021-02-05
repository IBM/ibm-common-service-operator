#!/bin/bash
#
# Copyright 2021 IBM Corporation
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

# counter to keep track of installation steps
STEP=0

# script base directory
BASE_DIR=$(dirname "$0")

# ---------- Command functions ----------

function main() {
    title "Moving to CatalogSource ibm-operator-catalog"
    msg ""
    check_preqreqs
    switch_to_ibmcatalog
}

function remove_old_catalog() {
    STEP=$((STEP + 1 ))
    title "[${STEP}] Remove CatalogSource of previous EUS version ..."
    msg "-----------------------------------------------------------------------"
    oc -n openshift-marketplace delete catalogsource opencloud-operators --ignore-not-found
}

function create_ibm_catalog() {
    STEP=$((STEP + 1 ))
    title "[${STEP}] Creating ibm catalog source ..."
    msg "-----------------------------------------------------------------------"

    cat <<EOF | tee >(oc apply -f -) | cat
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: ibm-operator-catalog
  namespace: openshift-marketplace
spec:
  displayName: ibm-operator-catalog 
  publisher: IBM Content
  sourceType: grpc
  image: docker.io/ibmcom/ibm-operator-catalog
  updateStrategy:
    registryPoll:
      interval: 45m
EOF

    if [[ $? -ne 0 ]]; then
        error "Error creating IBM catalog source"
    fi

    wait_for_pod "openshift-marketplace" "ibm-operator-catalog"
}

function check_preqreqs() {
    title "[${STEP}] Checking prerequesites ..."
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

    # remove old catalogsource
    remove_old_catalog

    # create new ibm catalogsource
    create_ibm_catalog
}

function switch_to_ibmcatalog() {
    STEP=$((STEP + 1 ))

    title "[${STEP}] Switch to IBM Operator Catalog Source ..."
    msg "-----------------------------------------------------------------------"

    while read -r ns cssub; do
        msg "Updating subscription ${cssub} in namespace ${ns} ..."
        msg ""
        
        in_step=1
        msg "[${in_step}] Removing the startingCSV ..."
        oc patch sub ${cssub} -n ${ns} --type="json" -p '[{"op": "remove", "path":"/spec/startingCSV"}]' 2> /dev/null

        in_step=$((in_step + 1))
        msg "[${in_step}] Switch Channel from stable-v1 to v3 ..."
        oc patch sub ${cssub} -n ${ns} --type="json" -p '[{"op": "replace", "path":"/spec/channel", "value":"v3"}]' 2> /dev/null

        in_step=$((in_step + 1))
        msg "[${in_step}] Switch CatalogSource from opencloud-operators to ibm-operator-catalog ..."
        oc patch sub ${cssub} -n ${ns} --type="json" -p '[{"op": "replace", "path":"/spec/source", "value":"ibm-operator-catalog"}]' 2> /dev/null
        msg "-----------------------------------------------------------------------"
        msg ""
    done < <(oc get sub --all-namespaces | grep ibm-common-service-operator | awk '{print $1" "$2}')
    
    success "Update all ibm-common-service-operator subscriptions successfully"
}

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

function wait_for_pod() {
    local namespace=$1
    local name=$2
    local condition="oc -n ${namespace} get po --no-headers --ignore-not-found | egrep 'Running|Completed|Succeeded' | grep ^${name}"
    local retries=30
    local sleep_time=10
    local total_time_mins=$(( sleep_time * retries / 60))
    local wait_message="Waiting for pod ${name} in namespace ${namespace} to be running"
    local success_message="Pod ${name} in namespace ${namespace} is running"
    local error_message="Timeout after ${total_time_mins} minutes waiting for pod ${name} in namespace ${namespace} to be running"
 
    wait_for_condition "${condition}" ${retries} ${sleep_time} "${wait_message}" "${success_message}" "${error_message}"
}

function wait_for_condition() {
    local condition=$1
    local retries=$2
    local sleep_time=$3
    local wait_message=$4
    local success_message=$5
    local error_message=$6

    info "${wait_message}"
    while true; do
        result=$(eval "${condition}")

        if [[ ( ${retries} -eq 0 ) && ( -z "${result}" ) ]]; then
            error "${error_message}"
        fi
 
        sleep ${sleep_time}
        result=$(eval "${condition}")
        
        if [[ -z "${result}" ]]; then
            info "RETRYING: ${wait_message} (${retries} left)"
            retries=$(( retries - 1 ))
        else
            break
        fi
    done

    if [[ ! -z "${success_message}" ]]; then
        success "${success_message}"
    fi
}

# --- Run ---

main $*
