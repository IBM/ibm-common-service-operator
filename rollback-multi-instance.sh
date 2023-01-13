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
MASTER_NS=$2
CONTROL_NS=$1
OC=${3:-oc}
YQ=${3:-yq}


function main() {
    which "${OC}" || error "Missing oc CLI"
    which "${YQ}" || error "Missing yq"
    if [[ -z $CONTROL_NS ]]; then
        error "Control namespace not specified, please specify control namespace parameter and try again."
    fi
    if [[ -z $MASTER_NS ]]; then
        error "Master common services namespace not specified, please specify common services namespace parameter and try again."
    fi
    rollback
}

function rollback() {
    info "Reverting multi-instance environment to shared instance environment."

    #checking if control namespace removed from common-service-maps
    local cm_name="common-service-maps"
    return_value=$("${OC}" get configmap -n kube-public -o yaml ${cm_name} | yq '.data' | grep controlNamespace: > /dev/null || echo passed)
    if [[ $return_value != "passed" ]]; then
        error "Configmap: ${cm_name} still has controlNamespace field. This must be removed before proceeding with rollback."
    fi
    return_value="reset"
    #TODO uninstall added instances
    
    info "Converting back to shared instance in ${MASTER_NS} namespace."
    # scale down
    ${OC} scale deployment -n ${MASTER_NS} ibm-common-service-operator --replicas=0
    ${OC} scale deployment -n ${MASTER_NS} operand-deployment-lifecycle-manager --replicas=0
    
    #delete operand config and operand registry
    ${OC} delete operandregistry -n ${MASTER_NS} --ignore-not-found common-service 
    ${OC} delete operandconfig -n ${MASTER_NS} --ignore-not-found common-service
    
    # uninstall singleton services
    "${OC}" delete -n "${CONTROL_NS}" --ignore-not-found certmanager default
    "${OC}" delete -n "${CONTROL_NS}" --ignore-not-found sub ibm-cert-manager-operator
    csv=$("${OC}" get -n "${CONTROL_NS}" csv | (grep ibm-cert-manager-operator || echo "fail") | awk '{print $1}')
    "${OC}" delete -n "${CONTROL_NS}" --ignore-not-found csv "${csv}"
    "${OC}" delete -n "${CONTROL_NS}" --ignore-not-found deploy cert-manager-cainjector cert-manager-controller cert-manager-webhook ibm-cert-manager-operator

    return_value=$(("${OC}" get crd ibmlicenseservicereporters.operator.ibm.com > /dev/null && echo exists) || echo fail)
    if [[ $return_value == "exists" ]]; then
        "${OC}" delete -n "${CONTROL_NS}" --ignore-not-found ibmlicensing instance
    fi
    return_value="reset"
    "${OC}" delete -n "${CONTROL_NS}" --ignore-not-found sub ibm-licensing-operator
    csv=$("${OC}" get -n "${CONTROL_NS}" csv | (grep ibm-licensing-operator || echo "fail") | awk '{print $1}')
    "${OC}" delete -n "${CONTROL_NS}" --ignore-not-found csv "${csv}"
    "${OC}" delete -n "${CONTROL_NS}" --ignore-not-found deploy ibm-licensing-operator ibm-licensing-service-instance
    "${OC}" patch -n "${CONTROL_NS}" operandbindinfo ibm-licensing-bindinfo --type=merge -p '{"metadata": {"finalizers":null}}' || info "Licensing OperandBindInfo not found in ${CONTROL_NS}. Moving on..."
    "${OC}" delete --ignore-not-found -n "${CONTROL_NS}" operandbindinfo ibm-licensing-bindinfo
    
    #not sure if there is more to uninstalling crossplane once it is up and running
    "${OC}" delete -n "${CONTROL_NS}" --ignore-not-found sub ibm-crossplane-operator-app
    "${OC}" delete -n "${CONTROL_NS}" --ignore-not-found sub ibm-crossplane-provider-kubernetes-operator-app
    csv=$("${OC}" get -n "${CONTROL_NS}" csv | (grep ibm-crossplane-operator || echo "fail") | awk '{print $1}')
    "${OC}" delete -n "${CONTROL_NS}" --ignore-not-found csv "${csv}"
    csv=$("${OC}" get -n "${CONTROL_NS}" csv | (grep ibm-crossplane-provider-kubernetes-operator || echo "fail") | awk '{print $1}')
    "${OC}" delete -n "${CONTROL_NS}" --ignore-not-found csv "${csv}"

    csv=$("${OC}" get -n "${CONTROL_NS}" csv | (grep ibm-namespace-scope-operator || echo "fail") | awk '{print $1}')
    "${OC}" delete -n "${CONTROL_NS}" --ignore-not-found csv "${csv}"
    "${OC}" patch namespacescope common-service -n "${CONTROL_NS}" --type=merge -p '{"metadata": {"finalizers":null}}' || info "Namespacescope resource not found in ${CONTROL_NS}. Moving on..."
    "${OC}" delete namespacescope common-service -n "${CONTROL_NS}" --ignore-not-found
    "${OC}" delete -n "${CONTROL_NS}" --ignore-not-found deploy ibm-namespace-scope-operator

    #delete misc items in control namespace
    "${OC}" delete deploy -n "${CONTROL_NS}" --ignore-not-found secretshare ibm-common-service-webhook
    webhookPod=$("${OC}" get pods -n ${MASTER_NS} | grep ibm-common-service-webhook | awk '{print $1}')
    ${OC} delete pod ${webhookPod} -n ${MASTER_NS} --ignore-not-found

    # scale back up
    scale_up_pod

    #verify singleton's are installed in master ns
    retries=10
    sleep_time=15
    total_time_mins=$(( sleep_time * retries / 60))
    info "Waiting for singleton services to deploy to ${MASTER_NS}..."
    sleep 10
    
    while true; do
        if [[ ${retries} -eq 0 ]]; then
            error "Timeout after ${total_time_mins} minutes waiting for IBM Common Services is deployed"
        fi

        certPodCheck=$("${OC}" get pods -n "${MASTER_NS}" | (grep ibm-cert-manager || echo "fail") | awk '{print $1}')
        licPodCheck=$("${OC}" get pods -n "${MASTER_NS}" | (grep ibm-licensing-service || echo "fail") | awk '{print $1}')
        if [ $certPodCheck != "fail" ] || [ $licPodCheck != "fail" ]; then
            info "Singleton services successfully re-deployed in ${MASTER_NS}"
            break
        else
            certPodCheck=$("${OC}" get pods -n "${CONTROL_NS}" | (grep ibm-cert-manager || echo "fail") | awk '{print $1}')
            licPodCheck=$("${OC}" get pods -n "${CONTROL_NS}" | (grep ibm-licensing-service || echo "fail") | awk '{print $1}')
            if [ $certPodCheck != "fail" ] || [ $licPodCheck != "fail" ]; then
                error "Singleton services re-deployed into control namespace. Verify that the common-services-map configmap in kube-public namespace has had the \"controlNamespace\" field removed and run again."
            fi
        fi
    done

    success "Cluster successfully rolled back. Namespace ${CONTROL_NS} can be safely deleted."

}

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

function scale_up_pod() {
    msg "scaling back ibm-common-service-operator deployment in ${MASTER_NS} namespace"
    ${OC} scale deployment -n ${MASTER_NS} ibm-common-service-operator --replicas=1
    ${OC} scale deployment -n ${MASTER_NS} operand-deployment-lifecycle-manager --replicas=1
    check_healthy "${MASTER_NS}"
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