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
    
    if [[ $# -eq 0 ]]; then
        CS_NAMESPACES=""
        msg "Upgrade Commmon Service Operator in all namespaces."
    else
        CS_NAMESPACES=("$@")
    fi

    check_preqreqs "${CS_NAMESPACES[@]}"
    switch_to_continous_delivery "${CS_NAMESPACES[@]}"
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

    # checking namespace if it is specified
    local namespaces=("$@")

    if [[ "$namespaces" != "" ]]; then
        for ns in "${namespaces[@]}"
        do
            if [[ -z "$(oc get namespace ${ns} --ignore-not-found)" ]]; then
            error "Namespace ${ns} for Common Service Operator is not found."
            fi
        done
    fi

    wait_for_pod "openshift-marketplace" "opencloud-operators"
    wait_for_eus
}

function wait_for_eus() {
    STEP=$((STEP + 1 ))

    title "[${STEP}] Upgrading Common Service to latest EUS version..."
    msg "-----------------------------------------------------------------------"
    # declare -A csmaps=( ["operand-deployment-lifecycle-manager-app"]="1.4.2"
    #                     ["ibm-common-service-operator.v3.7.1"]="3.6.3"
    #                     ["ibm-cert-manager-operator"]="3.8.1"
    #                     ["ibm-mongodb-operator"]="1.2.1"
    #                     ["ibm-iam-operator"]="3.8.2"
    #                     ["ibm-monitoring-exporters-operator"]="1.9.5"
    #                     ["ibm-monitoring-prometheus-operator-ext"]="1.9.5"
    #                     ["ibm-monitoring-grafana-operator"]="1.10.4"
    #                     ["ibm-healthcheck-operator"]="3.8.1"
    #                     ["ibm-management-ingress-operator"]="1.4.3"
    #                     ["ibm-licensing-operator"]="1.3.1"
    #                     ["ibm-metering-operator"]="3.8.1"
    #                     ["ibm-commonui-operator"]="1.4.1"
    #                     ["ibm-elastic-stack-operator"]="3.2.5"
    #                     ["ibm-ingress-nginx-operator"]="1.4.1"
    #                     ["ibm-auditlogging-operator"]="3.8.2"
    #                     ["ibm-platform-api-operator"]="3.8.2"
    #                     ["ibm-namespace-scope-operator"]="1.0.1"
    #                     )
    declare -A csmaps=( ["operand-deployment-lifecycle-manager"]="1.4.0"
                        ["ibm-common-service-operator"]="3.6.0"
                        ["ibm-cert-manager-operator"]="3.8.0"
                        ["ibm-mongodb-operator"]="1.2.0"
                        ["ibm-iam-operator"]="3.8.0"
                        ["ibm-monitoring-exporters-operator"]="1.9.0"
                        ["ibm-monitoring-prometheus-operator-ext"]="1.9.0"
                        ["ibm-monitoring-grafana-operator"]="1.10.0"
                        ["ibm-healthcheck-operator"]="3.8.0"
                        ["ibm-management-ingress-operator"]="1.4.0"
                        ["ibm-licensing-operator"]="1.3.0"
                        ["ibm-metering-operator"]="3.8.0"
                        ["ibm-commonui-operator"]="1.4.0"
                        ["ibm-elastic-stack-operator"]="3.2.0"
                        ["ibm-ingress-nginx-operator"]="1.4.0"
                        ["ibm-auditlogging-operator"]="3.8.0"
                        ["ibm-platform-api-operator"]="3.8.0"
                        ["ibm-namespace-scope-operator"]="1.0.0"
                        )

    while true; do
        succeed=true
        while read -r csv version phase; do
            if [[ ${csv} == "NAME" ]]; then
                continue
            fi
            csv=${csv%.v*}

            if [[ "${phase}" == "Succeeded" ]]; then
                # compare version with EUS
                if [[ -v csmaps["${csv}"] ]]; then
                    IFS='.' read -ra cur_version <<< "${version}"
                    IFS='.' read -ra eus_version <<< "${csmaps["${csv}"]}"
                    for index in ${!cur_version[@]}; do
                        if [[ ${cur_version[index]} > ${eus_version[index]} ]]; then
                            break
                        elif [[ ${cur_version[index]} == ${eus_version[index]} ]]; then
                            continue
                        else
                            succeed=false
                            break
                        fi
                    done
                    # current operator has smaller version than eus version
                    if [[ "$succeed" != "true" ]]; then
                        msg "${csv} v${version} is not EUS version yet." 
                        break
                    fi
                    msg "${csv} v${version} is ${phase}."
                fi
            else
                # current operator did not install successfully
                succeed=false
                msg "${csv} v${version} is ${phase}."   
                break
            fi
        done < <(oc get csv -n ibm-common-services -o=custom-columns=NAME:.metadata.name,Version:.spec.version,PHASE:.status.phase | awk '{print $1" "$2" "$3}')

        if [[ "$succeed" == "true" ]]; then
            break
        fi
        info "Waiting Common Service upgrading to latest EUS version..."
        sleep 60
    done
    success "Common Service has successfully upgraded to latest EUS version."
}

function switch_to_continous_delivery() {
    STEP=$((STEP + 1 ))

    title "[${STEP}] Switching to Continous Delivery Version (switching into v3 channel)..."
    msg "-----------------------------------------------------------------------"

    local namespaces=("$@")
    
    msg ${get_sub}
    while read -r ns cssub; do
        if [[ "$namespaces" != "" ]] && [[ ! " ${namespaces[@]} " =~ " ${ns} " ]]; then
            continue
        fi

        msg "Updating subscription ${cssub} in namespace ${ns} ..."
        msg ""
        
        in_step=1
        msg "[${in_step}] Removing the startingCSV ..."
        oc patch sub ${cssub} -n ${ns} --type="json" -p '[{"op": "remove", "path":"/spec/startingCSV"}]' 2> /dev/null

        in_step=$((in_step + 1))
        msg "[${in_step}] Switching channel from stable-v1 to v3 ..."
        oc patch sub ${cssub} -n ${ns} --type="json" -p '[{"op": "replace", "path":"/spec/channel", "value":"v3"}]' 2> /dev/null

        msg "-----------------------------------------------------------------------"
        msg ""
    done < <(oc get sub --all-namespaces | grep ibm-common-service-operator | awk '{print $1" "$2}')
    
    success "Updated all ibm-common-service-operator subscriptions successfully."
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
    local wait_message="Waiting for pod ${name} in namespace ${namespace} to be running ..."
    local success_message="Pod ${name} in namespace ${namespace} is running."
    local error_message="Timeout after ${total_time_mins} minutes waiting for pod ${name} in namespace ${namespace} to be running."
 
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
