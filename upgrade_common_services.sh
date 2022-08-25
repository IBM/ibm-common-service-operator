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
# set -x
STEP=0

# script base directory
BASE_DIR=$(dirname "$0")

# ---------- Command functions ----------
function usage() {
	local script="${0##*/}"

	while read -r ; do echo "${REPLY}" ; done <<-EOF
	Usage: ${script} [OPTION]...
	Upgrade Common Services

	Options:
	Mandatory arguments to long options are mandatory for short options too.
      -h, --help                    display this help and exit
      -a                            Upgrade all Common Service instances in the cluster. By default it only uprades the common service in ibm-common-services namespace
      -csNS                         specify the namespace where common service is installed. By default it is namespace ibm-common-services.
      -cloudpaksNS                  specify the namespace where cloud paks is installed. By default it would be same as csNS.
      -controlNS                    specify the namespace where singleton services are installed. By default it it would be same as csNS.
      -c                            specify the subscription channel where common services switch. By default it is channel v3
      -sub                          specify the subscription name if it is not ibm-common-service-operator
EOF
}

function main() {
    CS_NAMESPACE=${CS_NAMESPACE:-ibm-common-services}
    DESTINATION_CHANNEL=${DESTINATION_CHANNEL:-v3}
    ALL_NAMESPACE=${ALL_NAMESPACE:-false}
    
    while [ "$#" -gt "0" ]
    do
        case "$1" in
        "-h"|"--help")
            usage
            exit 0
            ;;
        "-csNS")
            CS_NAMESPACE=$2
            shift
            ;;
        "-cloudpaksNS")
            CLOUDPAKS_NAMESPACE=$2
            shift
            ;;
        "-controlNS")
            CONTROL_NAMESPACE=$2
            shift
            ;;
        "-sub")
            subName=$2
            shift
            ;;
        "-c")
            DESTINATION_CHANNEL=$2
            shift
            ;;
        "-a")
            ALL_NAMESPACE="true"
            ;;
        *)
            warning "invalid option -- \`$1\`"
            usage
            exit 1
            ;;
        esac
        shift
    done

    CLOUDPAKS_NAMESPACE=${CLOUDPAKS_NAMESPACE:-${CS_NAMESPACE}}
    CONTROL_NAMESPACE=${CONTROL_NAMESPACE:-${CS_NAMESPACE}}
    subName=${subName:-"ibm-common-service-operator"}

    if [[ "${ALL_NAMESPACE}" == "true" ]]; then
        title "Upgrade Commmon Service Operator in all namespaces."
    else
        title "Upgrade Common Service Operator to ${DESTINATION_CHANNEL} channel in ${CS_NAMESPACE} namespace."
    fi
    msg "-----------------------------------------------------------------------"

    check_preqreqs "${CS_NAMESPACE}" "${CLOUDPAKS_NAMESPACE}" "${CONTROL_NAMESPACE}"
    switch_channel "${subName}" "${CS_NAMESPACE}" "${CLOUDPAKS_NAMESPACE}" "${CONTROL_NAMESPACE}" "${DESTINATION_CHANNEL}" "${ALL_NAMESPACE}"
}


function check_preqreqs() {
    local csNS=$1
    local cloudpaksNS=$2
    local controlNS=$3
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
    if [[ -z "$(oc get namespace ${csNS})" ]]; then
        error "Namespace ${csNS} for Common Service Operator is not found."
    fi

    # checking namespace if it is specified
    if [[ -z "$(oc get namespace ${cloudpaksNS})" ]]; then
        error "Namespace ${cloudpaksNS} for Cloud Paks is not found."
    fi

    # checking namespace if it is specified
    if [[ -z "$(oc get namespace ${controlNS})" ]]; then
        error "Namespace ${controlNS} for singleton services is not found."
    fi

}

function switch_channel_operator() {
    local subName=$1
    local namespace=$2
    local channel=$3
    local allNamespace=$4

    if [[ "${allNamespace}" == "true" ]]; then
        while read -r ns cssub; do
            msg "Updating subscription ${cssub} in namespace ${ns}..."
            msg "-----------------------------------------------------------------------"
            
            in_step=1
            msg "[${in_step}] Removing the startingCSV ..."
            oc patch sub ${cssub} -n ${ns} --type="json" -p '[{"op": "remove", "path":"/spec/startingCSV"}]' 2> /dev/null

            in_step=$((in_step + 1))
            msg "[${in_step}] Switching channel to ${channel} ..."
            
            cat <<EOF | oc patch sub ${cssub} -n ${ns} --type="json" -p '[{"op": "replace", "path":"/spec/channel", "value":"'"${channel}"'"}]' | 2> /dev/null
EOF

            msg ""
        done < <(oc get sub --all-namespaces --ignore-not-found | grep ${subName} | awk '{print $1" "$2}')
    else
        while read -r cssub; do
            msg "Updating subscription ${cssub} in namespace ${namespace}..."
            msg "-----------------------------------------------------------------------"
            
            in_step=1
            msg "[${in_step}] Removing the startingCSV ..."
            oc patch sub ${cssub} -n ${namespace} --type="json" -p '[{"op": "remove", "path":"/spec/startingCSV"}]' 2> /dev/null

            in_step=$((in_step + 1))
            msg "[${in_step}] Switching channel to ${channel} ..."
            
            cat <<EOF | oc patch sub ${cssub} -n ${namespace} --type="json" -p '[{"op": "replace", "path":"/spec/channel", "value":"'"${channel}"'"}]' | 2> /dev/null
EOF

            msg ""
        done < <(oc get sub -n ${namespace} --ignore-not-found | grep ${subName} | awk '{print $1}')
    fi
}

function compare_channel() {
    local subName=$1
    local namespace=$2
    local channel=$3
    local cur_channel=$4
    
    # remove first char "v"
    channel="${channel:1}"
    cur_channel="${cur_channel:1}"

    msg "Comparing channels in ${namespace} namespace ..."
    msg "-----------------------------------------------------------------------"

    # compare channel before channel switching
    IFS='.' read -ra current_channel <<< "${cur_channel}"
    IFS='.' read -ra upgrade_channel <<< "${channel}"

    # fill empty fields in current base version with zeros
    for ((i=${#current_channel[@]}; i<${#upgrade_channel[@]}; i++)); do
        current_channel[i]=0
    done

    for index in ${!current_channel[@]}; do

        # fill empty fields in upgrade version with zeros
        if [[ -z ${upgrade_channel[index]} ]]; then
            upgrade_channel[index]=0
        fi

        if [[ ${current_channel[index]} -gt ${upgrade_channel[index]} ]]; then
            error "Upgrade channel v${channel} is lower than current channel v${cur_channel}, abort the upgrade procedure."
        elif [[ ${current_channel[index]} -lt ${upgrade_channel[index]} ]]; then
            success "Upgrade channel v${channel} is greater than current channel v${cur_channel}, ready for channel switch"
            return 0
        fi
    done
    success "Upgrade channel v${channel} is equal to current channel v${cur_channel}, ready for channel switch."
}


function switch_channel() {
    local subName=$1
    local csNS=$2
    local cloudpaksNS=$3
    local controlNS=$4
    local channel=$5
    local allNamespace=$6

    STEP=$((STEP + 1 ))

    title "[${STEP}] Compareing given upgrade channel version ${channel} with current one ..."
    msg "-----------------------------------------------------------------------"

    # msg "Updating OperandRegistry common-service in namespace ibm-common-services..."
    # msg "-----------------------------------------------------------------------"
    # oc -n ibm-common-services get operandregistry common-service -o yaml | sed 's/stable-v1/v3.20/g' | oc -n ibm-common-services apply -f -

    if [[ "${allNamespace}" == "true" ]]; then
        while read -r ns cur_channel; do
            compare_channel "${subname}" "${ns}" "${channel}" "${cur_channel}"
        done < <(oc get sub --all-namespaces --ignore-not-found | grep ${subName}  | awk '{print $1" "$5}')
        if [[ $? == 0 ]]; then
            STEP=$((STEP + 1 ))
            title "[${STEP}] Switching channel into ${channel}..."
            msg "-----------------------------------------------------------------------"
            switch_channel_operator "${subName}" "${csNS}" "${channel}" "${allNamespace}"
        fi   
    else
        if [[ "$cloudpaksNS" != "$csNS" ]]; then
            
            while read -r cur_channel; do
                compare_channel "${subName}" "${cloudpaksNS}" "${channel}" "${cur_channel}"
            done < <(oc get sub -n ${cloudpaksNS} --ignore-not-found | grep ${subName}  | awk '{print $4}')

            if [[ $? == 0 ]]; then
                STEP=$((STEP + 1 ))
                title "[${STEP}] Switching channel into ${channel}..."
                msg "-----------------------------------------------------------------------"
                switch_channel_operator "${subName}" "${cloudpaksNS}" "${channel}" "${allNamespace}"
            fi
        fi
        
        while read -r cur_channel; do
            compare_channel "${subname}" "${csNS}" "${channel}" "${cur_channel}"
        done < <(oc get sub -n ${csNS} --ignore-not-found | grep ${subName}  | awk '{print $4}')
        
        if [[ $? == 0 ]]; then
            STEP=$((STEP + 1 ))
            title "[${STEP}] Switching channel into ${channel}..."
            msg "-----------------------------------------------------------------------"
            switch_channel_operator "${subName}" "${csNS}" "${channel}" "${allNamespace}"
        fi
    fi

    success "Updated ${subName} subscriptions successfully."
    msg ""

    # scale down ODLM to prevent reconciliation
    msg "scaling down operand-deployment-lifecycle-manager deployment in ${csNS} namespace"
    oc scale deployment -n "${csNS}" "operand-deployment-lifecycle-manager" --replicas=0

    msg "Updating OperandRegistry common-service in ${csNS} namespace..."
    oc -n ${csNS} get operandregistry common-service -o yaml | sed 's/ibm-zen-operator/dummy-ibm-zen-operator/g' | oc -n ${csNS} apply -f -

    switch_channel_operator "ibm-zen-operator" "${csNS}" "${channel}" "false"

    info "Please wait a moment for ${subName} to upgrade all foundational services."
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

function warning() {
  msg "\33[33m[✗] ${1}\33[0m"
}

# --- Run ---

main $*
