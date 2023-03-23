#!/bin/bash

# Licensed Materials - Property of IBM
# Copyright IBM Corporation 2023. All Rights Reserved
# US Government Users Restricted Rights -
# Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
#
# This is an internal component, bundled with an official IBM product. 
# Please refer to that particular license for additional information. 

# ---------- Info functions ----------

function msg() {
    printf '%b\n' "$1"
}

function info() {
    msg "[INFO] ${1}"
}

function success() {
    msg "\33[32m[✔] ${1}\33[0m"
}

function warning() {
    msg "\33[33m[✗] ${1}\33[0m"
}

function error() {
    msg "\33[31m[✘] ${1}\33[0m"
    exit 1
}

function title() {
    msg "\33[34m# ${1}\33[0m"
}

function debug() {
    msg "\33[33m[DEBUG] ${1}\33[0m"
}


# ---------- Check functions ----------

function check_command() {
    local command=$1

    if [[ -z "$(command -v ${command} 2> /dev/null)" ]]; then
        error "${command} command not available"
    else
        success "${command} command available"
    fi
}

function check_return_code() {
    local rc=$1
    local error_message=$2
    
    if [ "${rc}" -ne 0 ]; then
        error "${error_message}"
    else
        return 0
    fi
}

function restart_job() {
    local namespace=$1
    local job_name=$2

    if [[ ! -z "$(${OC} -n ${namespace} get job ${job_name} --ignore-not-found)" ]]; then
        ${OC} -n ${namespace} patch job ${job_name} --type json -p \
            '[{ "op": "remove", "path": "/spec/selector"}, 
              { "op": "remove", "path": "/spec/template/metadata/labels/controller-uid"}]' \
            -o yaml --dry-run \
            | ${OC} -n ${namespace} replace --force --timeout=20s -f - 2> /dev/null
    else
        error "Job not found: ${job_name}"
    fi
}

function translate_step() {
    local step=$1
    echo "${step}" | tr '[1-9]' '[a-i]'
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


function wait_for_not_condition() {
    local condition=$1
    local retries=$2
    local sleep_time=$3
    local wait_message=$4
    local success_message=$5
    local error_message=$6

    info "${wait_message}"
    while true; do
        result=$(eval "${condition}")

        if [[ ( ${retries} -eq 0 ) && ( ! -z "${result}" ) ]]; then
            error "${error_message}"
        fi
 
        sleep ${sleep_time}
        result=$(eval "${condition}")
        
        if [[ ! -z "${result}" ]]; then
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

function wait_for_configmap() {
    local namespace=$1
    local name=$2
    local condition="${OC} -n ${namespace} get cm --no-headers --ignore-not-found | grep ^${name}"
    local retries=12
    local sleep_time=10
    local total_time_mins=$(( sleep_time * retries / 60))
    local wait_message="Waiting for ConfigMap ${name} in namespace ${namespace} to be made available"
    local success_message="ConfigMap ${name} in namespace ${namespace} is available"
    local error_message="Timeout after ${total_time_mins} minutes waiting for ConfigMap ${name} in namespace ${namespace} to become available"
 
    wait_for_condition "${condition}" ${retries} ${sleep_time} "${wait_message}" "${success_message}" "${error_message}"
}

function wait_for_pod() {
    local namespace=$1
    local name=$2
    local condition="${OC} -n ${namespace} get po --no-headers --ignore-not-found | egrep 'Running|Completed|Succeeded' | grep ^${name}"
    local retries=30
    local sleep_time=30
    local total_time_mins=$(( sleep_time * retries / 60))
    local wait_message="Waiting for pod ${name} in namespace ${namespace} to be running"
    local success_message="Pod ${name} in namespace ${namespace} is running"
    local error_message="Timeout after ${total_time_mins} minutes waiting for pod ${name} in namespace ${namespace} to be running"
 
    wait_for_condition "${condition}" ${retries} ${sleep_time} "${wait_message}" "${success_message}" "${error_message}"
}

function wait_for_no_pod() {
    local namespace=$1
    local name=$2
    local condition="${OC} -n ${namespace} get po --no-headers --ignore-not-found | grep ^${name}"
    local retries=30
    local sleep_time=10
    local total_time_mins=$(( sleep_time * retries / 60))
    local wait_message="Waiting for pod ${name} in namespace ${namespace} to be deleting"
    local success_message="Pod ${name} in namespace ${namespace} is deleted"
    local error_message="Timeout after ${total_time_mins} minutes waiting for pod ${name} in namespace ${namespace} to be deleted"
 
    wait_for_not_condition "${condition}" ${retries} ${sleep_time} "${wait_message}" "${success_message}" "${error_message}"
}

function wait_for_project() {
    local name=$1
    local condition="${OC} get project ${name} --no-headers --ignore-not-found"
    local retries=50
    local sleep_time=10
    local total_time_mins=$(( sleep_time * retries / 60))
    local wait_message="Waiting for project ${name} to be created"
    local success_message="Project ${name} is created"
    local error_message="Timeout after ${total_time_mins} minutes waiting for project ${name} to be created"
 
    wait_for_condition "${condition}" ${retries} ${sleep_time} "${wait_message}" "${success_message}" "${error_message}"
}

function wait_for_operator() {
    local namespace=$1
    local operator_name=$2
    local condition="${OC} -n ${namespace} get csv --no-headers --ignore-not-found | egrep 'Succeeded' | grep ^${operator_name}"
    local retries=50
    local sleep_time=10
    local total_time_mins=$(( sleep_time * retries / 60))
    local wait_message="Waiting for operator ${operator_name} in namespace ${namespace} to be made available"
    local success_message="Operator ${operator_name} in namespace ${namespace} is available"
    local error_message="Timeout after ${total_time_mins} minutes waiting for ${operator_name} in namespace ${namespace} to become available"
 
    wait_for_condition "${condition}" ${retries} ${sleep_time} "${wait_message}" "${success_message}" "${error_message}"
}

function wait_for_service_account() {
    local namespace=$1
    local name=$2
    local condition="${OC} -n ${namespace} get sa ${name} --no-headers --ignore-not-found"
    local retries=20
    local sleep_time=10
    local total_time_mins=$(( sleep_time * retries / 60))
    local wait_message="Waiting for service account ${name} to be created"
    local success_message="Service account ${name} is created"
    local error_message="Timeout after ${total_time_mins} minutes waiting for service account ${name} to be created"
 
    wait_for_condition "${condition}" ${retries} ${sleep_time} "${wait_message}" "${success_message}" "${error_message}"
}

function wait_for_operand_request() {
    local namespace=$1
    local name=$2
    local condition="${OC} -n ${namespace} get operandrequests ${name} --no-headers --ignore-not-found -o jsonpath='{.status.phase}' | grep 'Running'"
    local retries=20
    local sleep_time=10
    local total_time_mins=$(( sleep_time * retries / 60))
    local wait_message="Waiting for operand request ${name} to be running"
    local success_message="Operand request ${name} is running"
    local error_message="Timeout after ${total_time_mins} minutes waiting for operand request ${name} to be running"
 
    wait_for_condition "${condition}" ${retries} ${sleep_time} "${wait_message}" "${success_message}" "${error_message}"
}

function wait_for_nss_patch() {
    local namespace=$1
    local csv_name=$($OC get sub ibm-common-service-operator -n ${namespace} -o jsonpath='{.status.installedCSV}')
    local condition="${OC} -n ${namespace}  get csv ${csv_name} -o jsonpath='{.spec.install.spec.deployments[0].spec.template.spec.containers[0].env[?(@.name==\"WATCH_NAMESPACE\")].valueFrom.configMapKeyRef.name}'| grep 'namespace-scope'"
    local retries=10
    local sleep_time=10
    local total_time_mins=$(( sleep_time * retries / 60))
    local wait_message="Waiting for cs-operator CSV to be patched with NSS configmap"
    local success_message="cs-operator CSV is patched with NSS configmap"
    local error_message="Timeout after ${total_time_mins} minutes waiting for cs-operator CSV to be patched with NSS configmap"
    local pod_name=$($OC get pod -n ${namespace} | grep namespace-scope | awk '{print $1}')

    # wait for nss patch
    info "${wait_message}"
    while true; do
        result=$(eval "${condition}")

        # restart namespace scope operator pod to reconcilie
        if [[ ( ${retries} -eq 0 ) && ( ! -z "${result}" ) ]]; then
            info "Reconciling namespace scope operator"
            echo "deleting pod ${pod_name}"
            $OC delete pod ${pod_name} -n ${namespace}    
            wait_for_condition "${condition}" ${retries} ${sleep_time} "${wait_message}" "${success_message}" "${error_message}"
            break
        fi
 
        
        if [ -z "${result}" ]; then
            info "RETRYING: ${wait_message} (${retries} left)"
            retries=$(( retries - 1 ))
        else
            break
        fi

        sleep ${sleep_time}
    done

    if [[ ! -z "${success_message}" ]]; then
        success "${success_message}"
    fi
    
    # wait for deployment to be ready
    deployment_name=$($OC get deployment -n ${namespace} | grep common-service-operator | awk '{print $1}')
    wait_for_env_var ${namespace} ${deployment_name}
    wait_for_deployment ${namespace} ${deployment_name}
    
}

function wait_for_env_var() {
    local namespace=$1
    local name=$2
    local condition="${OC} -n ${namespace} get deployment ${name} -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name==\"WATCH_NAMESPACE\")].valueFrom.configMapKeyRef.name}'| grep 'namespace-scope'"
    local retries=10
    local sleep_time=30
    local total_time_mins=$(( sleep_time * retries / 60))
    local wait_message="Waiting for OLM to update Deployment ${name} "
    local success_message="Deployment ${name} is updated"
    local error_message="Timeout after ${total_time_mins} minutes waiting for OLM to update Deployment ${name} "

    wait_for_condition "${condition}" ${retries} ${sleep_time} "${wait_message}" "${success_message}" "${error_message}"
}

function wait_for_deployment() {
    local namespace=$1
    local name=$2
    local condition="${OC} -n ${namespace} get deployment ${name} --no-headers --ignore-not-found -o jsonpath='{.status.readyReplicas}' | grep '1'"
    local retries=10
    local sleep_time=30
    local total_time_mins=$(( sleep_time * retries / 60))
    local wait_message="Waiting for Deployment ${name} to be ready"
    local success_message="Deployment ${name} is running"
    local error_message="Timeout after ${total_time_mins} minutes waiting for Deployment ${name} to be running"

    wait_for_condition "${condition}" ${retries} ${sleep_time} "${wait_message}" "${success_message}" "${error_message}"
}

function is_sub_exist() {
    local package_name=$1
    if [ $# -eq 2 ]; then
        local namespace=$2
        local name=$(${OC} get sub -n ${namespace} -o yaml -o jsonpath='{.items[*].spec.name}')
    else
        local name=$(${OC} get sub -A -o yaml -o jsonpath='{.items[*].spec.name}')
    fi
    is_exist=$(echo "$name" | grep -w "$package_name")
}

function check_namespace(){
    local namespace=$1
    if [[ -z "$(${OC} get namespace ${namespace} --ignore-not-found)" ]]; then
        error "Namespace ${namespace} does not exist"
    fi
}

function check_cert_manager(){
    csv_count=`$OC get csv |grep "cert-manager"|wc -l`
    if [[ $csv_count == 0 ]]; then
        error "Missing a cert-manager"
    fi
    if [[ $csv_count > 1 ]]; then
        error "Multiple cert-manager csv found. Only one should be installed per cluster"
    fi
}

function check_licensing(){
    [[ ! $($OC get IBMLicensing) ]] && error "User does not have proper permission to get IBMLicensing"
    instance_count=`$OC get IBMLicensing -o name | wc -l`
    if [[ $instance_count == 0 ]]; then
        error "Missing IBMLicensing"
    fi
    if [[ $instance_count > 1 ]]; then
        error "Multiple IBMLicensing are found. Only one should be installed per cluster"
    fi
}
# ---------- creation functions ----------

function create_namespace() {
    local namespace=$1

    title "Creating namespace ${namespace}\n"
    
    if [[ -z "$(${OC} get namespace ${namespace} --ignore-not-found)" ]]; then
        ${OC} create namespace ${namespace}
        if [[ $? -ne 0 ]]; then
            error "Error creating namespace ${namespace}"
        fi
    else
        info "Namespace ${namespace} already exists. Skip creating"
    fi
}

function create_operator_group() {
    local name=$1
    local ns=$2
    local target=$3
    local og=$(
        cat <<EOF
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: $name
  namespace: $ns
spec: $target
EOF
    )

    info "Checking existing OperatorGroup in $ns:\n"
    existing_og=$(${OC} get operatorgroup -n $ns --no-headers --ignore-not-found | wc -l)
    if [[ ${existing_og} -ne 0 ]]; then
        info "OperatorGroup already exists in $ns. Skip creating"
        return 0
    fi
    echo
    info "Creating following OperatorGroup:\n"
    echo "$og"
    echo "$og" | ${OC} apply -f -
    if [[ $? -ne 0 ]]; then
        error "Failed to create OperatorGroup ${name} in ${ns}"
    fi
}

function create_subscription() {
    local name=$1
    local ns=$2
    local channel=$3
    local package_name=$4
    local source=$5
    local source_ns=$6
    local install_mode=$7
    local sub=$(
        cat <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: $name
  namespace: $ns
spec:
  channel: $channel
  installPlanApproval: $install_mode
  name: $package_name
  source: $source
  sourceNamespace: $source_ns
EOF
    )
    
    echo
    info "Creating following Subscription:\n"
    echo "$sub"
    echo "$sub" | ${OC} apply -f -
    if [[ $? -ne 0 ]]; then
        error "Failed to create subscription ${name} in ${ns}"
    fi
}

# update/create cs cr
function update_cscr() {
    local operator_ns=$1
    local service_ns=$2
    
    # get all the watch_namespaces
    local cp_namespaces=$($OC get configmap namespace-scope -n ${operator_ns} -o jsonpath='{.data.namespaces}')
    local namespaces_array=($(echo $cp_namespaces | tr "," "\n") )
    
    for namespace in "${namespaces_array[@]}"
    do
        echo $namespace
        # update or create cs cr in the tenant namespace
        if [[ "${namespace}" != "${operator_ns}" ]]; then
            get_commonservice=$("${OC}" get commonservice -n ${namespace})
            if [[ "${get_commonservice}" == "" ]]; then
                echo "create in" $namespace
                # copy commonservice from operator namespace
                ${OC} get commonservice common-service -n "${operator_ns}" -o yaml | yq eval '.spec += {"operatorNamespace": "'${operator_ns}'", "servicesNamespace": "'${service_ns}'"}' > common-service.yaml
                yq eval 'select(.kind == "CommonService") | del(.metadata.resourceVersion) | del(.metadata.uid) | .metadata.namespace = "'${namespace}'"' common-service.yaml | ${OC} apply --overwrite=true -f -

            else
                echo "update in" $namespace
                # update commonservice
                cs_name=$(${OC} get commonservice -n ${namespace} --no-headers | awk '{print $1}')
                ${OC} get commonservice ${cs_name} -n "${namespace}" -o yaml | yq eval '.spec += {"operatorNamespace": "'${operator_ns}'", "servicesNamespace": "'${service_ns}'"}' > common-service.yaml
                yq eval 'select(.kind == "CommonService") | del(.metadata.resourceVersion) | del(.metadata.uid) | .metadata.namespace = "'${namespace}'"' common-service.yaml | ${OC} apply --overwrite=true -f -
            fi  
        else
            # update commonservice
            cs_name=$(${OC} get commonservice -n ${namespace} --no-headers | awk '{print $1}')
            ${OC} get commonservice ${cs_name} -n "${namespace}" -o yaml | yq eval '.spec += {"operatorNamespace": "'${operator_ns}'", "servicesNamespace": "'${service_ns}'"}' > common-service.yaml
            yq eval 'select(.kind == "CommonService") | del(.metadata.resourceVersion) | del(.metadata.uid) | .metadata.namespace = "'${namespace}'"' common-service.yaml | ${OC} apply --overwrite=true -f -
        fi
    done

    rm common-service.yaml
}

# ---------- cleanup functions ----------
function cleanup_cp2() {
    cleanup_webhook
    cleanup_secretshare
    cleanup_crossplane
}

# clean up webhook deployment and webhookconfiguration
function cleanup_webhook() {
    cleanup_deployment "ibm-common-service-webhook"

    info "Deleting MutatingWebhookConfiguration..."
    ${OC} delete MutatingWebhookConfiguration ibm-common-service-webhook-configuration --ignore-not-found
    ${OC} delete MutatingWebhookConfiguration ibm-operandrequest-webhook-configuration --ignore-not-found

    info "Deleting MutatingWebhookConfiguration..."
    ${OC} delete ValidatingWebhookConfiguration ibm-cs-ns-mapping-webhook-configuration --ignore-not-found
    
    info "Deleting podpresets..."
    local namespace=$(${OC} get podpresets.operator.ibm.com -A --no-headers | awk '{print $1}')
    ${OC} get podpresets.operator.ibm.com -A --no-headers | awk '{print $2}' | xargs oc delete -n ${namespace} --ignore-not-found podpresets.operator.ibm.com

}

# clean up secretshare deployment and CR
function cleanup_secretshare() {
    cleanup_deployment "secretshare"

    info "Deleting SecretShare..."
    ${OC} get secretshare -A --no-headers | awk '{print $2}' | xargs oc delete --ignore-not-found secretshare 

}

# todo: clean up crossplane sub and CR
function cleanup_crossplane() {
    # delete CR
    info "cleanup crossplane CR"
    ${OC} get configuration.pkg.ibm.crossplane.io -A --no-headers | awk '{print $1}' | xargs oc delete --ignore-not-found configuration.pkg.ibm.crossplane.io
    ${OC} get lock.pkg.ibm.crossplane.io -A --no-headers | awk '{print $1}' | xargs oc delete --ignore-not-found lock.pkg.ibm.crossplane.io
    ${OC} get ProviderConfig -A --no-headers | awk '{print $1}' | xargs oc delete --ignore-not-found ProviderConfig

    sleep 60

    # delete Sub
    info "cleanup crossplane subscription"
    local namespace=$($OC get sub -A --no-headers | grep ibm-crossplane-operator-app | awk '{print $1}')
    ${OC} delete sub ibm-crossplane-provider-kubernetes-operator-app -n ${namespace} --ignore-not-found
    ${OC} delete sub ibm-crossplane-provider-ibm-cloud-operator-app -n ${namespace} --ignore-not-found
    ${OC} delete sub ibm-crossplane-operator-app -n ${namespace} --ignore-not-found
}

function cleanup_OperandBindInfo() {
    local namespace=$1
    ${OC} delete operandbindInfo ibm-commonui-bindinfo -n ${namespace} --ignore-not-found
}

function cleanup_NamespaceScope() {
    local namespace=$1
    ${OC} delete namespacescope odlm-scope-managedby-odlm nss-odlm-scope nss-managedby-odlm -n ${namespace} --ignore-not-found
}

function cleanup_OperandRequest() {
    local namespace=$1
    ${OC} delete operandrequest ibm-commonui-request ibm-mongodb-request -n ${namespace} --ignore-not-found
}

function cleanup_deployment() {
    local name=$1
    local namespace=$($OC get deployment -A | grep ${name} | awk '{print $1}')
    info "Deleting existing Deployment ${name}..."
    ${OC} delete deployment ${name} -n ${namespace} --ignore-not-found

    wait_for_no_pod ${namespace} ${name}
}

