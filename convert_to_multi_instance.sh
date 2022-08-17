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

OC=${1:-oc}
YQ=${1:-yq}

master_ns=
cs_operator_channel=
catalog_source=

function main() {
    "${OC}" get nodes
    #prereq
    #prepare_cluster
    collect_data
    restart_CS_pods
    install_new_CS
}


# verify that all pre-requisite CLI tools exist
function prereq() {
    which "${OC}" || error "Missing oc CLI"
    which "${YQ}" || error "Missing yq"
}

function prepare_cluster() {
    local cm_name="common-service-maps"
    return_value=$("${OC}" get -n kube-public configmap ${cm_name} || echo failed)
    if [[ $return_value == "failed" ]]; then
        error "Missing configmap: ${cm_name}. This must be configured before proceeding"
    fi

    # configmap should have control namespace specified

    # ensure cs-operator is not installed in all namespace mode

    # find all namespaces with cs-operator running
    # each namespace should be in configmap

    # uninstall singleton services
}

function collect_data() {
    title "collecting data"
    msg "-----------------------------------------------------------------------"

    master_ns=$(${OC} get pod --all-namespaces | grep operand-deployment-lifecycle-manager | awk '{print $1}')
    echo MasterNS:${master_ns}
    cs_operator_channel=$(${OC} get sub ibm-common-service-operator -n ${master_ns} -o yaml | yq ".spec.channel") 
    echo channel:${cs_operator_channel}   
    catalog_source=$(${OC} get sub ibm-common-service-operator -n ${master_ns} -o yaml | yq ".spec.source")
    echo catalog_source:${catalog_source}   
    
}

# delete all CS pod and read configmap
function restart_CS_pods() {
    title "restarting ibm-common-service-operator pod"
    msg "-----------------------------------------------------------------------"
    ${OC} get pod --all-namespaces | grep ibm-common-service-operator | while read -r line; do
        local namespace=$(echo $line | awk '{print $1}')
        local cs_pod=$(echo $line | awk '{print $2}')

        msg "deleting pod ${cs_pod} in namespace ${namespace} "
        ${OC} delete pod ${cs_pod} -n ${namespace} || error "Error deleting pod ${cs_pod} in namespace ${namespace}"
    done
    success "All ibm-common-service-operator pod is deleted"
}

# # re-install singleton service
# function re-install_singleton() {
#     title "restarting operand-deployment-lifecycle-manager pod"
#     msg "-----------------------------------------------------------------------"

#     local pod=$(oc get pods -n ${master_ns} | grep operand-deployment-lifecycle-manager | awk '{print $1}')
#     ${OC} delete pod ${pod} -n ${master_ns} 
#     if [[ $? -ne 0 ]]; then
#         error "Error deleting pod ${pod} in namespace ${master_ns}"
#     fi

# }

#  install new instances of CS based on cs mapping configmap
function install_new_CS() {
    title "install new instances of CS based on cs mapping configmap"
    msg "-----------------------------------------------------------------------"

    ${OC} get configmap common-service-maps -n kube-public -o yaml | while read -r line; do
        first_element=$(echo $line | awk '{print $1}')
        
        if [[ "${first_element}" == "-" ]]; then
            namespace=$(echo $line | awk '{print $2}')
            if [[ "${namespace}" != "requested-from-namespace:" ]]; then
                if [[ "${namespace}" != "${master_ns}" ]]; then
                    return_value=$("${OC}" get namespace ${namespace} || echo failed)
                    if [[ $return_value != "failed" ]]; then
                        echo In_CloudpakNS:${namespace}
                        get_sub=$("${OC}" get sub ibm-common-service-operator -n ${namespace} || echo failed)
                        if [[ $get_sub == "failed" ]]; then
                            create_operator_group "${namespace}"
                            install_common_service_operator_sub "${namespace}"
                        fi
                    fi  
                fi
            fi
        fi

        if [[ "${first_element}" == "map-to-common-service-namespace:" ]]; then
            return_value=$("${OC}" get namespace ${namespace} || echo failed)
            if [[ $return_value != "failed" ]]; then
                namespace=$(echo $line | awk '{print $2}')
                echo In_MasterNS:${namespace}
                get_sub=$("${OC}" get sub ibm-common-service-operator -n ${namespace} || echo failed)
                if [[ $get_sub == "failed" ]]; then
                    create_operator_group "${namespace}"
                    install_common_service_operator_sub "${namespace}"
                fi
            fi  
        fi
    done
    
    success "Common Services Operator is converted to multi_instance mode"
}

function create_operator_group() {
    local CS_NAMESPACE=$1

    title "Creating operator group ..."
    msg "-----------------------------------------------------------------------"


    cat <<EOF | tee >(oc apply -f -) | cat
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: common-service
  namespace: ${CS_NAMESPACE}
spec:
  targetNamespaces:
  - ${CS_NAMESPACE}
EOF

    # error handle
}

function install_common_service_operator_sub() {
    local CS_NAMESPACE=$1

    title " Installing IBM Common Service Operator subcription "
    msg "-----------------------------------------------------------------------"

    cat <<EOF | tee >(oc apply -f -) | cat
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: ibm-common-service-operator
  namespace: ${CS_NAMESPACE}
spec:
  channel: ${cs_operator_channel}
  installPlanApproval: Automatic
  name: ibm-common-service-operator
  source: ${catalog_source}
  sourceNamespace: openshift-marketplace
EOF

    # error handle

    info "Waiting for IBM Common Service Operator subscription to become active"
    check_healthy "${CS_NAMESPACE}"

    success "IBM Common Service Operator subscription in namespace ${CS_NAMESPACE} is created"
}

# verify all instances are healthy
function check_healthy() {
    local CS_NAMESPACE=$1

    sleep 10

    retries=20
    sleep_time=15
    total_time_mins=$(( sleep_time * retries / 60))
    info "Waiting for IBM Common Services CR is Succeeded"
    pod=$(oc get pods -n ${CS_NAMESPACE} | grep ibm-common-service-operator | awk '{print $1}')
    
    while true; do
        if [[ ${retries} -eq 0 ]]; then
            error "Timeout after ${total_time_mins} minutes waiting for IBM Common Services is deployed"
        fi

        phase=$(oc get pod ${pod} -o jsonpath='{.status.phase}' -n ${CS_NAMESPACE})

        if [[ "${phase}" != "Running" ]]; then
            retries=$(( retries - 1 ))
            info "RETRYING: Waiting for IBM Common Services CR is Succeeded (${retries} left)"
            sleep ${sleep_time}
        else
            msg "-----------------------------------------------------------------------"    
            success "Common Services is deployed in ${CS_NAMESPACE}"
            break
        fi
    done
}



function check_CSCR() {
    local CS_NAMESPACE=$1

    retries=30
    sleep_time=15
    total_time_mins=$(( sleep_time * retries / 60))
    info "Waiting for IBM Common Services CR is Succeeded"

    while true; do
        if [[ ${retries} -eq 0 ]]; then
            error "Timeout after ${total_time_mins} minutes waiting for IBM Common Services CR is Succeeded"
        fi

        phase=$(oc get commonservice common-service -o jsonpath='{.status.phase}' -n ${CS_NAMESPACE})

        if [[ "${phase}" != "Succeeded" ]]; then
            retries=$(( retries - 1 ))
            info "RETRYING: Waiting for IBM Common Services CR is Succeeded (${retries} left)"
            sleep ${sleep_time}
        else
            msg "-----------------------------------------------------------------------"    
            success "Ready use"
            break
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

# --- Run ---

main $*

