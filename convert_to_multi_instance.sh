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
    prereq
    collect_data
    prepare_cluster
    scale_up_pod
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
    return_value=$("${OC}" get -n kube-public configmap ${cm_name} > /dev/null || echo failed)
    if [[ $return_value == "failed" ]]; then
        error "Missing configmap: ${cm_name}. This must be configured before proceeding"
    fi
    return_value="reset"

    # configmap should have control namespace specified
    return_value=$("${OC}" get configmap -n kube-public -o yaml ${cm_name} | yq '.data' | grep controlNamespace: > /dev/null || echo failed)
    if [[ $return_value == "failed" ]]; then
        error "Configmap: ${cm_name} did not specify 'controlNamespace' field. This must be configured before proceeding"
    fi
    return_value="reset"

    controlNs=$("${OC}" get configmap -n kube-public -o yaml ${cm_name} | yq '.data' | grep controlNamespace: | awk '{print $2}')
    return_value=$("${OC}" get ns "${controlNs}" > /dev/null || echo failed)
    if [[ $return_value == "failed" ]]; then
        error "The namespace specified in controlNamespace does not exist"
    fi
    return_value="reset"

    # LicenseServiceReporter should not be installed because it does not support multi-instance mode
    return_value=$(("${OC}" get crd ibmlicenseservicereporters.operator.ibm.com > /dev/null && echo exists) || echo fail)
    if [[ $return_value == "exists" ]]; then
        return_value=$("${OC}" get ibmlicenseservicereporters -A | wc -l)
        if [[ $return_value -gt 0 ]]; then
            error "LicenseServiceReporter does not support multi-instance mode. Remove before proceeding"
        fi
    fi
    return_value="reset"

    # ensure cs-operator is not installed in all namespace mode
    return_value=$("${OC}" get csv -n openshift-operators | grep ibm-common-service-operator > /dev/null || echo pass)
    if [[ $return_value != "pass" ]]; then
        error "The ibm-common-service-operator must not be installed in AllNamespaces mode"
    fi

    # TODO for more advanced checking
    # find all namespaces with cs-operator running
    # each namespace should be in configmap
    # all namespaces in configmap should exist
    check_cm_ns_exist $cm_name

    ${OC} scale deployment -n ${master_ns} ibm-common-service-operator --replicas=0
    ${OC} scale deployment -n ${master_ns} operand-deployment-lifecycle-manager --replicas=0
    ${OC} delete operandregistry -n ${master_ns} --ignore-not-found common-service 
    ${OC} delete operandconfig -n ${master_ns} --ignore-not-found common-service

    # remove existing namespace scope CRs
    removeNSS

    # uninstall singleton services
    "${OC}" delete -n "${master_ns}" --ignore-not-found certmanager default
    "${OC}" delete -n "${master_ns}" --ignore-not-found sub ibm-cert-manager-operator
    csv=$("${OC}" get -n "${master_ns}" csv | (grep ibm-cert-manager-operator || echo "fail") | awk '{print $1}')
    "${OC}" delete -n "${master_ns}" --ignore-not-found csv "${csv}"

    # reason for checking again instead of simply deleting the CR when checking
    # for LSR is to avoid deleting anything until the last possible moment.
    # This makes recovery from simple pre-requisite errors easier.
    return_value=$(("${OC}" get crd ibmlicenseservicereporters.operator.ibm.com > /dev/null && echo exists) || echo fail)
    if [[ $return_value == "exists" ]]; then
        "${OC}" delete -n "${master_ns}" --ignore-not-found ibmlicensing instance
    fi
    return_value="reset"
    "${OC}" delete -n "${master_ns}" --ignore-not-found sub ibm-licensing-operator
    csv=$("${OC}" get -n "${master_ns}" csv | (grep ibm-licensing-operator || echo "fail") | awk '{print $1}')
    "${OC}" delete -n "${master_ns}" --ignore-not-found csv "${csv}"

    "${OC}" delete -n "${master_ns}" --ignore-not-found sub ibm-crossplane-operator-app
    "${OC}" delete -n "${master_ns}" --ignore-not-found sub ibm-crossplane-provider-kubernetes-operator-app
    csv=$("${OC}" get -n "${master_ns}" csv | (grep ibm-crossplane-operator || echo "fail") | awk '{print $1}')
    "${OC}" delete -n "${master_ns}" --ignore-not-found csv "${csv}"
    csv=$("${OC}" get -n "${master_ns}" csv | (grep ibm-crossplane-provider-kubernetes-operator || echo "fail") | awk '{print $1}')
    "${OC}" delete -n "${master_ns}" --ignore-not-found csv "${csv}"
}

# scale back cs pod 
function scale_up_pod() {
    msg "scaling back ibm-common-service-operator deployment in ${master_ns} namespace"
    ${OC} scale deployment -n ${master_ns} ibm-common-service-operator --replicas=1
    ${OC} scale deployment -n ${master_ns} operand-deployment-lifecycle-manager --replicas=1
    check_healthy "${master_ns}"
}

function collect_data() {
    title "collecting data"
    msg "-----------------------------------------------------------------------"

    master_ns=$(${OC} get deployment --all-namespaces | grep operand-deployment-lifecycle-manager | awk '{print $1}')
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
                        get_sub=$("${OC}" get sub ibm-common-service-operator -n ${namespace} > /dev/null || echo failed)
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
                get_sub=$("${OC}" get sub ibm-common-service-operator -n ${namespace} > /dev/null || echo failed)
                if [[ $get_sub == "failed" ]]; then
                    create_operator_group "${namespace}"
                    install_common_service_operator_sub "${namespace}"
                    check_CSCR "${namespace}"
                fi
            fi  
        fi
    done
    
    success "Common Services Operator is converted to multi_instance mode"
}

function create_operator_group() {
    local cs_namespace=$1

    title "Checking if OperatorGroup exists in ${cs_namespace}"
    msg "-----------------------------------------------------------------------"

    exists=$("${OC}" get operatorgroups -n "${cs_namespace}" | wc -l)
    if [[ "$exists" -ne 0 ]]; then
        msg "Already an OperatorGroup in ${cs_namespace}, skip creating OperatorGroup"
    else
        title "Creating operator group ..."
        msg "-----------------------------------------------------------------------"


        cat <<EOF | tee >("${OC}" apply -f -) | cat
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: common-service
  namespace: ${cs_namespace}
spec:
  targetNamespaces:
  - ${cs_namespace}
EOF

    fi
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
    sleep 10
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
    sleep 10

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


# check that all namespaces in common-service-maps cm exist. 
# Create them if not already present 
# Does not create cs-control namespace
function check_cm_ns_exist(){
    local cm_name=$1

    #this command gets all of the ns listed in requested from namesapce fields
    requestedNS=$("${OC}" get configmap -n kube-public -o yaml ${cm_name} | yq '.data[]' | yq '.namespaceMapping[].requested-from-namespace' | awk '{print $2}')

    #this command gets all of the ns listed in map-to-common-service-namespace
    mapToCSNS=$("${OC}" get configmap -n kube-public -o yaml ${cm_name} | yq '.data[]' | yq '.namespaceMapping[].map-to-common-service-namespace' | awk '{print}')

    for ns in $requestedNS
    do
        echo "creating $ns"
        ${OC} create namespace $ns || info "$ns already exists, skipping..."
    done
    for ns in $mapToCSNS
    do
        echo "creating $ns"
        ${OC} create namespace $ns || info "$ns already exists, skipping..."
    done
    echo "all namespaces in $cm_name exist"
}

function removeNSS(){
    ${OC} get nss --all-namespaces | grep nss-managedby-odlm | while read -r line; do
        local namespace=$(echo $line | awk '{print $1}')
        info "deleting namespace scope nss-managedby-odlm in namespace $namespace"
        ${OC} delete nss nss-managedby-odlm -n ${namespace} || error "unable to delete namespace scope nss-managedby-odlm in ${namespace}"
    done
    ${OC} get nss --all-namespaces | grep odlm-scope-managedby-odlm | while read -r line; do
        local namespace=$(echo $line | awk '{print $1}')
        info "deleting namespace scope odlm-scope-managedby-odlm in namespace $namespace"
        ${OC} delete nss odlm-scope-managedby-odlm -n ${namespace} || error "unable to delete namespace scope odlm-scope-managedby-odlm in ${namespace}"
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

