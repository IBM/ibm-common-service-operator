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
. ${BASE_DIR}/utils.sh

function main() {
    title "Moving to CatalogSource ibm-operator-catalog"
    msg ""
    check_preqreqs
    switch_to_ibmcatalog
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

    # checking ibm catalogsource is existing
    catalog=$(oc -n openshift-marketplace get pod | grep ibm-operator-catalog 2> /dev/null)
    if [$? -ne 0]; then
        error "You must deploy CatalofSource ${catalog} first"
    else
        success "CatalogSource ${catalog} is deployed"
    fi
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
# --- Run ---

main $*