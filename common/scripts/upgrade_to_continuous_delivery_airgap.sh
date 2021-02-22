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
    title "Upgrade Common Service Operator to continous delivery channel."
    msg "-----------------------------------------------------------------------"

    check_preqreqs
    switch_to_continous_delivery
}

function check_preqreqs() {
    title "[${STEP}] Checking prerequesites ..."
    msg "-----------------------------------------------------------------------"

    # checking oc command
    if [[ -z "$(command -v oc 2> /dev/null)" ]]; then
        error "OpenShift Command Line tool oc is not available"
    else
        success "OpenShift Command Line tool oc is available."
    fi

    # checking oc command logged in
    user=$(oc whoami 2> /dev/null)
    if [ $? -ne 0 ]; then
        error "You must be logged into the OpenShift Cluster from the oc command line."
    else
        success "oc command logged in as ${user}"
    fi
}

function switch_to_continous_delivery() {
    STEP=$((STEP + 1 ))

    title "[${STEP}] Switching to Continous Delivery Version (switching into v3 channel)..."
    msg "-----------------------------------------------------------------------"

    
    while read -r ns cssub; do

        msg "Updating subscription ${cssub} in namespace ${ns}..."
        msg "-----------------------------------------------------------------------"
        
        in_step=1
        msg "[${in_step}] Removing the startingCSV ..."
        oc patch sub ${cssub} -n ${ns} --type="json" -p '[{"op": "remove", "path":"/spec/startingCSV"}]' 2> /dev/null

        in_step=$((in_step + 1))
        msg "[${in_step}] Switching channel from stable-v1 to v3 ..."
        oc patch sub ${cssub} -n ${ns} --type="json" -p '[{"op": "replace", "path":"/spec/channel", "value":"v3"}]' 2> /dev/null

        msg ""
    done < <(oc get sub --all-namespaces | grep ibm-common-service-operator | awk '{print $1" "$2}')

    success "Updated all ibm-common-service-operator subscriptions successfully."
    msg ""

    while read -r sub; do

        msg "Updating subscription ${sub} in namespace ibm-common-services..."
        msg "-----------------------------------------------------------------------"
        
        in_step=1
        msg "[${in_step}] Removing the startingCSV ..."
        oc patch sub ${sub} -n ibm-common-services --type="json" -p '[{"op": "remove", "path":"/spec/startingCSV"}]' 2> /dev/null

        in_step=$((in_step + 1))
        msg "[${in_step}] Switching channel from stable-v1 to v3 ..."
        oc patch sub ${sub} -n ibm-common-services --type="json" -p '[{"op": "replace", "path":"/spec/channel", "value":"v3"}]' 2> /dev/null

        msg ""
    done < <(oc get sub -n ibm-common-services | awk '{print $1}')

    success "Updated all operator subscriptions in namesapce ibm-common-services successfully."

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

# --- Run ---

main $*
