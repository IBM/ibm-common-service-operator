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


CS_LIST_CSNS=("operand-deployment-lifecycle-manager-app"
        "ibm-common-service-operator"
        "ibm-cert-manager-operator"
        "ibm-mongodb-operator"
        "ibm-iam-operator"
        "ibm-monitoring-grafana-operator"
        "ibm-healthcheck-operator"
        "ibm-management-ingress-operator"
        "ibm-licensing-operator"
        "ibm-commonui-operator"
        "ibm-ingress-nginx-operator"
        "ibm-auditlogging-operator"
        "ibm-platform-api-operator"
        "ibm-namespace-scope-operator"
        "ibm-namespace-scope-operator-restricted"
        "ibm-zen-operator"
        "ibm-zen-cpp-operator"
        "ibm-crossplane-operator-app"
        "ibm-crossplane-provider-ibm-cloud-operator-app"
        "ibm-crossplane-provider-kubernetes-operator-app")

CS_LIST_CONTROLNS=(
        "ibm-cert-manager-operator"
        "ibm-namespace-scope-operator"
        "ibm-namespace-scope-operator-restricted"
        "ibm-crossplane-operator-app"
        "ibm-crossplane-provider-ibm-cloud-operator-app"
        "ibm-crossplane-provider-kubernetes-operator-app")

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
    check_switch_complete "${CS_NAMESPACE}" "${CLOUDPAKS_NAMESPACE}" "${CONTROL_NAMESPAVE}" "${DESTINATION_CHANNEL}" "${ALL_NAMESPACE}"

}


function check_preqreqs() {
    local csNS=$1
    local cloudpaksNS=$2
    local controlNS=$3

    msg ""
    title "[${STEP}] Checking prerequisites..."
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
            
            in_step=1
            msg "[${in_step}] Removing the startingCSV..."
            oc patch sub ${cssub} -n ${ns} --type="json" -p '[{"op": "remove", "path":"/spec/startingCSV"}]' 2> /dev/null

            in_step=$((in_step + 1))
            msg "[${in_step}] Upgrading channel to ${channel}..."
            
            cat <<EOF | oc patch sub ${cssub} -n ${ns} --type="json" -p '[{"op": "replace", "path":"/spec/channel", "value":"'"${channel}"'"}]' | 2> /dev/null
EOF

            msg ""
        done < <(oc get sub --all-namespaces --ignore-not-found | grep ${subName} | awk '{print $1" "$2}')
    else
        while read -r cssub; do
            msg "Updating subscription ${cssub} in namespace ${namespace}..."
            
            in_step=1
            msg "[${in_step}] Removing the startingCSV..."
            oc patch sub ${cssub} -n ${namespace} --type="json" -p '[{"op": "remove", "path":"/spec/startingCSV"}]' 2> /dev/null

            in_step=$((in_step + 1))
            msg "[${in_step}] Upgrading channel to ${channel}..."
            
            cat <<EOF | oc patch sub ${cssub} -n ${namespace} --type="json" -p '[{"op": "replace", "path":"/spec/channel", "value":"'"${channel}"'"}]' | 2> /dev/null
EOF

            msg ""
        done < <(oc get sub -n ${namespace} --ignore-not-found | grep ${subName} | awk '{print $1}')
    fi
}

# This function checks the current version of the installed bedrock instance automatically
# so that the user doesn't have to do so manually. If the current version on-cluster 
# is already higher than the desired one the user indicates, the script will indicate this
# to the user and abort the script since bedrock is already above the desired version. 
function compare_channel() {
    local subName=$1
    local namespace=$2
    local channel=$3
    local cur_channel=$4
    
    # remove all chars before "v"
    trimmed_channel="$(echo $channel | awk -Fv '{print $NF}')"
    trimmed_cur_channel="$(echo $cur_channel | awk -Fv '{print $NF}')"

    msg "Comparing channels in ${namespace} namespace..."

    # compare channel before channel switching
    IFS='.' read -ra current_channel <<< "${trimmed_cur_channel}"
    IFS='.' read -ra upgrade_channel <<< "${trimmed_channel}"

    # fill empty fields in current channel version with zeros
    for ((i=${#current_channel[@]}; i<${#upgrade_channel[@]}; i++)); do
        current_channel[i]=0
    done

    for index in ${!current_channel[@]}; do

        # fill empty fields in upgrade channel version with zeros
        if [[ -z ${upgrade_channel[index]} ]]; then
            upgrade_channel[index]=0
        fi

        if [[ ${current_channel[index]} -gt ${upgrade_channel[index]} ]]; then
            error "Upgrade channel ${channel} is lower than current channel ${cur_channel}, abort the upgrade procedure."
        elif [[ ${current_channel[index]} -lt ${upgrade_channel[index]} ]]; then
            success "Upgrade channel ${channel} is greater than current channel ${cur_channel}, ready for channel switch"
            return 0
        fi
    done
    success "Upgrade channel ${channel} is equal to current channel ${cur_channel}, ready for channel switch."
}

function switch_channel() {
    local subName=$1
    local csNS=$2
    local cloudpaksNS=$3
    local controlNS=$4
    local channel=$5
    local allNamespace=$6

    STEP=$((STEP + 1 ))
    msg ""
    title "[${STEP}] Comparing given upgrade channel version ${channel} with current one..."
    msg "-----------------------------------------------------------------------"

    if [[ "${allNamespace}" == "true" ]]; then
        while read -r ns cur_channel; do
            compare_channel "${subname}" "${ns}" "${channel}" "${cur_channel}"
        done < <(oc get sub --all-namespaces --ignore-not-found | grep ${subName}  | awk '{print $1" "$5}')
        if [[ $? == 0 ]]; then
            STEP=$((STEP + 1 ))
            msg ""
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
                msg ""
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
            msg ""
            title "[${STEP}] Switching channel into ${channel}..."
            msg "-----------------------------------------------------------------------"
            switch_channel_operator "${subName}" "${csNS}" "${channel}" "${allNamespace}"
        fi
    fi

    success "Updated ${subName} subscriptions successfully."

    STEP=$((STEP + 1 ))
    msg ""
    title "[${STEP}] Checking ibm-common-service-operator deployment in ${csNS} namespace..."
    msg "-----------------------------------------------------------------------"

    # remove all chars before "v"
    trimmed_channel="$(echo $channel | awk -Fv '{print $NF}')"
    sub_channel=$(oc get sub ${subName} -n ${csNS} -o jsonpath='{.spec.channel}')
    msg "existing channel version in cs operator subscription is ${sub_channel}"
    trimmed_cur_channel="$(echo $sub_channel | awk -Fv '{print $NF}')"

    # get ibm-common-service-operator replicas number
    cs_replica=$(oc get deployment ibm-common-service-operator -n ${csNS} -o jsonpath='{.spec.replicas}')
    msg "existing number of replicas in cs operator is ${cs_replica}"
    msg ""

    if [[ $cs_replica == "0" ]]; then
        if [[ "$trimmed_cur_channel" == "$trimmed_channel" ]]; then
            # scale up ibm-common-service-operator deployment back to 1
            msg "scaling up ibm-common-service-operator deployment in ${csNS} namespace to 1"
            oc scale deployment -n "${csNS}" "ibm-common-service-operator" --replicas=1
        fi
    elif [[ $cs_replica == "1" ]]; then
        IFS='.' read -ra upgrade_version <<< "${trimmed_channel}"
        IFS='.' read -ra current_version <<< "${trimmed_cur_channel}"

        # fill empty fields in current version with zeros
        for ((i=${#current_version[@]}; i<${#upgrade_version[@]}; i++)); do
            current_version[i]=0
        done

        for index in ${!current_version[@]}; do
            # fill empty fields in upgrade channel version with zeros
            if [[ -z ${upgrade_version[index]} ]]; then
                upgrade_version[index]=0
            fi
            if [[ ${current_version[index]} -lt ${upgrade_version[index]} ]]; then
                # scale down ibm-common-service-operator deployment to avoid ODLM re-installation
                msg "scaling down ibm-common-service-operator deployment in ${csNS} namespace to 0"
                oc scale deployment -n "${csNS}" "ibm-common-service-operator" --replicas=0

                STEP=$((STEP + 1 ))
                msg ""
                title "[${STEP}] Deleting ODLM to avoid reverting the channel changes for other operators."
                msg "-----------------------------------------------------------------------"
                delete_operator "operand-deployment-lifecycle-manager-app" "${csNS}"
                return 0
            fi
        done
    fi
    
    # switch channel for remaining CS components
    for sub_name in "${CS_LIST_CSNS[@]}"; do
        switch_channel_operator "${sub_name}" "${csNS}" "${channel}" "false"
    done

    for sub_name in "${CS_LIST_CONTROLNS[@]}"; do
        switch_channel_operator "${sub_name}" "${csNS}" "${channel}" "false"
    done

}

function check_switch_complete() {
    local csNS=$1
    local cloudpaksNS=$2
    local controlNS=$3
    local destChannel=$4
    local allNamespace=$5

    STEP=$((STEP + 1 ))
    
    title "[${STEP}] Checking whether the channel switch is completed..."
    msg "-----------------------------------------------------------------------"

    for sub_name in "${CS_LIST_CSNS[@]}"; do
        channel=$(oc get sub ${sub_name} -n ${csNS} -o jsonpath='{.spec.channel}' --ignore-not-found)
        if [[ "X${channel}" != "X" ]] && [[ "$channel" != "${destChannel}" ]]; then
            error "the channel of subscription ${sub_name} in namespace ${csNS} is not ${destChannel}, please try to re-run the script"
        fi
    done

    for sub_name in "${CS_LIST_CONTROLNS[@]}"; do
        channel=$(oc get sub ${sub_name} -n ${controlNS} -o jsonpath='{.spec.channel}' --ignore-not-found)
        if [[ "X${channel}" != "X" ]] && [[ "$channel" != "${destChannel}" ]]; then
            error "the channel of subscription ${sub_name} in namespace ${controlNS} is not ${destChannel}, please try to re-run the script"
        fi
    done

    success "Updated all Common Service components' subscriptions successfully."
}

function delete_operator() {
    subs=$1
    ns=$2
    for sub in ${subs}; do
        exist=$(oc get sub ${sub} -n ${ns} --ignore-not-found)
        if [[ "X${exist}" != "X" ]]; then
            msg "Deleting ${sub} in namespace ${ns}, it will be re-installed after the upgrade is successful..."
            csv=$(oc get sub ${sub} -n ${ns} -o=jsonpath='{.status.installedCSV}' --ignore-not-found)
            in_step=1
            msg "[${in_step}] Removing the subscription of ${sub} in namespace ${ns}..."
            oc delete sub ${sub} -n ${ns} --ignore-not-found
            in_step=$((in_step + 1))
            msg "[${in_step}] Removing the csv of ${sub} in namespace ${ns}..."
            [[ "X${csv}" != "X" ]] && oc delete csv ${csv}  -n ${ns} --ignore-not-found
            msg ""

            success "Remove $sub successfully."
            msg ""
        fi
    done
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
