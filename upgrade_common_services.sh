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
# ---------- Command functions ----------
function usage() {
	local script="${0##*/}"

	while read -r ; do echo "${REPLY}" ; done <<-EOF
	Usage: ${script} [OPTION]...
	Upgrade Common Services

	Options:
	Mandatory arguments to long options are mandatory for short options too.
      -h, --help                    display this help and exit
      -csNS                         specify the namespace where common service is installed. By default it is namespace ibm-common-services.
      -cloudpaksNS                  specify the namespace where cloud paks is installed. By default it would be same as csNS.
      -controlNS                    specify the namespace where singleton services are installed. By default it it would be same as csNS.
      -c                            specify the subscription channel where common services switch. By default it is channel v3
	Note:
	If there is no namespace defined through arguments, all Common Service in the cluster will be upgraded.
EOF
}

function main() {
    CS_NAMESPACE=${CS_NAMESPACE:-ibm-common-services}
    DESTINATION_CHANNEL=${DESTINATION_CHANNEL:-v3}
    ALL_NAMESPACE=${ALL_NAMESPACE:-true}
    
    while [ "$#" -gt "0" ]
    do
        case "$1" in
        "-h"|"--help")
            usage
            exit 0
            ;;
        "-csNS")
            CS_NAMESPACE=$2
            ALL_NAMESPACE="false"
            shift
            ;;
        "-cloudpaksNS")
            cloudpaksNS=$2
            ALL_NAMESPACE="false"
            shift
            ;;
        "-controlNS")
            controlNS=$2
            ALL_NAMESPACE="false"
            shift
            ;;
        "-c")
            DESTINATION_CHANNEL=$2
            shift
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
    CONTROL_NAMESPAVE=${CONTROL_NAMESPAVE:-${CS_NAMESPACE}}

    if [[ "${ALL_NAMESPACE}" == "true" ]]; then
        title "Upgrade Commmon Service Operator in all namespaces."
    else
        title "Upgrade Common Service Operator to ${DESTINATION_CHANNEL} channel in ${CS_NAMESPACE} namespace."
    fi
    msg "-----------------------------------------------------------------------"

    check_preqreqs "${CS_NAMESPACE}" "${CLOUDPAKS_NAMESPACE}" "${CONTROL_NAMESPAVE}"
    switch_channel "${CS_NAMESPACE}" "${CLOUDPAKS_NAMESPACE}" "${CONTROL_NAMESPAVE}" "${DESTINATION_CHANNEL}" "${ALL_NAMESPACE}"
    check_switch_complete "${CS_NAMESPACE}" "${CLOUDPAKS_NAMESPACE}" "${CONTROL_NAMESPAVE}" "${DESTINATION_CHANNEL}" "${ALL_NAMESPACE}"
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

function containsElement () {
  local e match="$1"
  shift
  for e; do 
    [[ "$e" == "$match" ]] && return 0; 
  done
  return 1
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
        done < <(oc get sub -n ${csNS} --ignore-not-found | grep ${subName} | awk '{print $1}')
    fi
}

function switch_channel() {
    local csNS=$1
    local cloudpaksNS=$2
    local controlNS=$3
    local channel=$4
    local allNamespace=$5

    STEP=$((STEP + 1 ))

    title "[${STEP}] Switching channel into ${channel}..."
    msg "-----------------------------------------------------------------------"

    # msg "Updating OperandRegistry common-service in namespace ibm-common-services..."
    # msg "-----------------------------------------------------------------------"
    # oc -n ibm-common-services get operandregistry common-service -o yaml | sed 's/stable-v1/v3.20/g' | oc -n ibm-common-services apply -f -

    if [[ "${allNamespace}" == "true" ]]; then
        switch_channel_operator "ibm-common-service-operator" "${csNS}" "${channel}" "${allNamespace}"
    else
        if [[ "$cloudpaksNS" != "$csNS" ]]; then
            switch_channel_operator "ibm-common-service-operator" "${cloudpaksNS}" "${channel}" "${allNamespace}"
        fi
        switch_channel_operator "ibm-common-service-operator" "${csNS}" "${channel}" "${allNamespace}"
    fi

    success "Updated ibm-common-service-operator subscriptions successfully."
    msg ""

    # scale down ibm-common-service-operator deployment
    msg "scaling down ibm-common-service-operator deployment in ${csNS} namespace"
    oc scale deployment -n "${csNS}" "ibm-common-service-operator" --replicas=0
    
    # clean some operators due to historical reason
    delete_operator "operand-deployment-lifecycle-manager-app ibm-namespace-scope-operator-restricted ibm-namespace-scope-operator" "${csNS}"
    delete_operator "ibm-namespace-scope-operator-restricted ibm-namespace-scope-operator ibm-cert-manager-operator" "${controlNS}"


    # switch channel for remaining CS components
    for sub_name in "${CS_LIST_CSNS[@]}"; do
        switch_channel_operator "${sub_name}" "${csNS}" "${channel}" "false"
    done

    for sub_name in "${CS_LIST_CONTROLNS[@]}"; do
        switch_channel_operator "${sub_name}" "${csNS}" "${channel}" "false"
    done


    msg "Creating namespace scope config in namespace ${csNS}..."
    msg "-----------------------------------------------------------------------"
    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: namespace-scope
  namespace: ${csNS}
data:
  namespaces: ${csNS}
EOF
    msg ""
    success "Created namespace scope config in namespace ${csNS}."
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
            msg "Deleting ${sub} in namesapce ${ns}, it will be re-installed after the upgrade is successful ..."
            msg "-----------------------------------------------------------------------"
            csv=$(oc get sub ${sub} -n ${ns} -o=jsonpath='{.status.installedCSV}' --ignore-not-found)
            in_step=1
            msg "[${in_step}] Removing the subscription of ${sub} in namesapce ${ns} ..."
            oc delete sub ${sub} -n ${ns} --ignore-not-found
            in_step=$((in_step + 1))
            msg "[${in_step}] Removing the csv of ${sub} in namesapce ${ns} ..."
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

# function cs_upgrade_list() {
#     case $1 in
#         "operand-deployment-lifecycle-manager-app") eval "$2='1.4.0'";;
#         "ibm-common-service-operator") eval "$2='3.6.0'";;
#         "ibm-cert-manager-operator") eval "$2='3.8.0'";;
#         "ibm-mongodb-operator") eval "$2='1.2.0'";;
#         "ibm-iam-operator") eval "$2='3.8.0'";;
#         "ibm-monitoring-grafana-operator") eval "$2='1.10.0'";;
#         "ibm-healthcheck-operator") eval "$2='3.8.0'";;
#         "ibm-management-ingress-operator") eval "$2='1.4.0'";;
#         "ibm-licensing-operator") eval "$2='1.3.0'";;
#         "ibm-commonui-operator") eval "$2='1.4.0'";;
#         "ibm-ingress-nginx-operator") eval "$2='1.4.0'";;
#         "ibm-auditlogging-operator") eval "$2='3.8.0'";;
#         "ibm-platform-api-operator") eval "$2='3.8.0'";;
#         "ibm-namespace-scope-operator") eval "$2='1.0.0'";;
#         "ibm-zen-operator") eval "$2='1.0.0'";;
#         "ibm-zen-cpp-operator") eval "$2='1.0.0'";;
#         "ibm-crossplane-operator-app") eval "$2='1.0.0'";;
#         "ibm-crossplane-provider-ibm-cloud-operator-app") eval "$2='1.0.0'";;
#         "ibm-crossplane-provider-kubernetes-operator-app") eval "$2='1.0.0'";;
#         *)  eval "$2='0.0.0'";;
#     esac
# }

# --- Run ---

main $*

