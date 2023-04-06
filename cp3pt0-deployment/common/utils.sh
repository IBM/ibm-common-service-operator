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
    local needReplicas=$(${OC} -n ${namespace} get deployment ${name} --no-headers --ignore-not-found -o jsonpath='{.spec.replicas}' | awk '{print $1}')
    local readyReplicas="${OC} -n ${namespace} get deployment ${name} --no-headers --ignore-not-found -o jsonpath='{.status.readyReplicas}' | grep '${needReplicas}'"
    local replicas="${OC} -n ${namespace} get deployment ${name} --no-headers --ignore-not-found -o jsonpath='{.status.replicas}' | grep '${needReplicas}'"
    local condition="(${readyReplicas} && ${replicas})"
    local retries=10
    local sleep_time=30
    local total_time_mins=$(( sleep_time * retries / 60))
    local wait_message="Waiting for Deployment ${name} to be ready"
    local success_message="Deployment ${name} is running"
    local error_message="Timeout after ${total_time_mins} minutes waiting for Deployment ${name} to be running"

    wait_for_condition "${condition}" ${retries} ${sleep_time} "${wait_message}" "${success_message}" "${error_message}"
}

function wait_for_operator_upgrade() {
    local namespace=$1
    local package_name=$2
    local channel=$3
    local sub_name=$(${OC} get subscription.operators.coreos.com -n ${namespace} -l operators.coreos.com/${package_name}.${namespace}='' --no-headers | awk '{print $1}')
    local condition="${OC} get subscription.operators.coreos.com ${sub_name} -n ${namespace} -o jsonpath='{.status.installedCSV}' | grep -w $channel"

    local retries=10
    local sleep_time=30
    local total_time_mins=$(( sleep_time * retries / 60))
    local wait_message="Waiting for operator ${package_name} to be upgraded"
    local success_message="Operator ${package_name} is upgraded to latest version in channel ${channel}"
    local error_message="Timeout after ${total_time_mins} minutes waiting for operator ${package_name} to be upgraded"

    wait_for_condition "${condition}" ${retries} ${sleep_time} "${wait_message}" "${success_message}" "${error_message}"
}

function is_sub_exist() {
    local package_name=$1
    if [ $# -eq 2 ]; then
        local namespace=$2
        local name=$(${OC} get subscription.operators.coreos.com -n ${namespace} -o yaml -o jsonpath='{.items[*].spec.name}')
    else
        local name=$(${OC} get subscription.operators.coreos.com -A -o yaml -o jsonpath='{.items[*].spec.name}')
    fi
    is_exist=$(echo "$name" | grep -w "$package_name")
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
    
    if [[ -z "$(${OC} get namespace ${namespace} --ignore-not-found)" ]]; then
        title "Creating namespace ${namespace}\n"
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
    local nss_list=$3

    for namespace in ${nss_list//,/ }
    do
        # update or create default CS CR in every namespace
        result=$("${OC}" get commonservice common-service -n ${namespace} --ignore-not-found)
        if [[ -z "${result}" ]]; then
            info "Creating CommonService CR common-service in $namespace"
            # copy commonservice from operator namespace
            ${OC} get commonservice common-service -n "${operator_ns}" -o yaml | ${YQ} eval '.spec += {"operatorNamespace": "'${operator_ns}'", "servicesNamespace": "'${service_ns}'"}' > common-service.yaml
        else
            info "Configuring CommonService CR common-service in $namespace"
            ${OC} get commonservice common-service -n "${namespace}" -o yaml | ${YQ} eval '.spec += {"operatorNamespace": "'${operator_ns}'", "servicesNamespace": "'${service_ns}'"}' > common-service.yaml
            
        fi  
        yq eval 'select(.kind == "CommonService") | del(.metadata.resourceVersion) | del(.metadata.uid) | .metadata.namespace = "'${namespace}'"' common-service.yaml | ${OC} apply --overwrite=true -f -
        if [[ $? -ne 0 ]]; then
            error "Failed to apply CommonService CR in ${namespace}"
        fi
    done

    rm common-service.yaml
}

# Update nss cr
function update_nss_kind() {
    local operator_ns=$1
    local nss_list=$2
    for n in ${nss_list//,/ }
    do
        local members=$members$(cat <<EOF

    - $n
EOF
    )
    done

    local object=$(
        cat <<EOF
apiVersion: operator.ibm.com/v1
kind: NamespaceScope
metadata:
  name: common-service
  namespace: $operator_ns
spec:
  csvInjector:
    enable: true
  namespaceMembers: $members
  restartLabels:
    intent: projected
EOF
    )
    
    echo
    info "Updating the NamespaceScope object"
    echo "$object" | ${OC} apply -f -
    if [[ $? -ne 0 ]]; then
        error "Failed to create NSS CR in ${OPERATOR_NS}"
    fi
}

# ---------- cleanup functions ----------
function cleanup_cp2() {
    local operator_ns=$1
    local control_ns=$2
    local nss_list=$3
    local enable_multi_instance=0

    if [[ "$operator_ns" != "$control_ns" ]]; then
        enable_multi_instance=1
    fi

    if [[ enable_multi_instance -eq 0 ]]; then
        cleanup_webhook $control_ns $nss_list
        cleanup_secretshare $control_ns $nss_list
        cleanup_crossplane $control_ns $nss_list
    fi
    

    cleanup_OperandBindInfo $operator_ns
    cleanup_NamespaceScope $operator_ns
}

# clean up webhook deployment and webhookconfiguration
function cleanup_webhook() {
    local control_ns=$1
    local nss_list=$2
    for ns in ${nss_list//,/ }
    do
        info "Deleting podpresets in namespace {$ns}..."
        ${OC} get podpresets.operator.ibm.com -n $ns --no-headers | awk '{print $1}' | xargs ${OC} delete -n $ns --ignore-not-found podpresets.operator.ibm.com
    done
    msg ""

    cleanup_deployment "ibm-common-service-webhook" $control_ns

    info "Deleting MutatingWebhookConfiguration..."
    ${OC} delete MutatingWebhookConfiguration ibm-common-service-webhook-configuration --ignore-not-found
    ${OC} delete MutatingWebhookConfiguration ibm-operandrequest-webhook-configuration --ignore-not-found
    msg ""

    info "Deleting MutatingWebhookConfiguration..."
    ${OC} delete ValidatingWebhookConfiguration ibm-cs-ns-mapping-webhook-configuration --ignore-not-found

}

# TODO: clean up secretshare deployment and CR in service_ns
function cleanup_secretshare() {
    local control_ns=$1
    local nss_list=$2

    for ns in ${nss_list//,/ }
    do
        info "Deleting SecretShare in namespace $ns..."
        ${OC} get secretshare -n $ns --no-headers | awk '{print $1}' | xargs ${OC} delete -n $ns --ignore-not-found secretshare
    done
    msg ""

    cleanup_deployment "secretshare" "$control_ns"

}

# TODO: clean up crossplane sub and CR in operator_ns and service_ns
function cleanup_crossplane() {
    # delete CR
    info "cleanup crossplane CR"
    ${OC} get configuration.pkg.ibm.crossplane.io -A --no-headers | awk '{print $1}' | xargs ${OC} delete --ignore-not-found configuration.pkg.ibm.crossplane.io
    ${OC} get lock.pkg.ibm.crossplane.io -A --no-headers | awk '{print $1}' | xargs ${OC} delete --ignore-not-found lock.pkg.ibm.crossplane.io
    ${OC} get ProviderConfig -A --no-headers | awk '{print $1}' | xargs ${OC} delete --ignore-not-found ProviderConfig

    sleep 60

    # delete Sub
    info "cleanup crossplane subscription"
    local namespace=$($OC get subscription.operators.coreos.com -A --no-headers | grep ibm-crossplane-operator-app | awk '{print $1}')
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
    local namespace=$2
    info "Deleting existing Deployment ${name} in namespace ${namespace}..."
    ${OC} delete deployment ${name} -n ${namespace} --ignore-not-found

    wait_for_no_pod ${namespace} ${name}
}

function get_control_namespace() {
    # Define the ConfigMap name and namespace
    local config_map_name="common-service-maps"

    # Get the ConfigMap data
    config_map_data=$(${OC} get configmap "${config_map_name}" -n kube-public -o jsonpath='{.data.common-service-maps\.yaml}')

    # Check if the ConfigMap exists
    if [[ -z "${config_map_data}" ]]; then
        warning "Not found common-serivce-maps ConfigMap in kube-public namespace. It is a single shared Common Service instance upgrade"
    else
        # Get the controlNamespace value
        control_namespace=$(echo "${config_map_data}" | yq -r '.controlNamespace')

        # Check if the controlNamespace key exists
        if [[ "${control_namespace}" == "null" ]] || [[ "${control_namespace}" == "" ]]; then
            warning "No controlNamespace is found from common-serivce-maps ConfigMap in kube-public namespace. It is a single shared Common Service instance upgrade"
        else
            CONTROL_NS=$control_namespace
        fi
    fi
}

function compare_semantic_version() {
    # Extract major, minor, and patch versions from the arguments
    regex='^v([0-9]+)\.?([0-9]*)\.?([0-9]*)?$'
    if [[ $1 =~ $regex ]]; then
        major1=${BASH_REMATCH[1]}
        minor1=${BASH_REMATCH[2]:-0}
        patch1=${BASH_REMATCH[3]:-0}

        if [[  $2 =~ $regex ]]; then
            major2=${BASH_REMATCH[1]}
            minor2=${BASH_REMATCH[2]:-0}
            patch2=${BASH_REMATCH[3]:-0}
        else
            error "Invalid version format: $2"
        fi
    else
        error "Invalid version format: $1"
    fi
    
    # If the versions have different number of components, add the missing parts
    if [[ -z "$minor1" && -z "$minor2" ]]; then
        minor1=0
        minor2=0
        patch1=0
        patch2=0
    elif [[ -z "$minor1" ]]; then
        minor1=0
        patch1=0
    elif [[ -z "$minor2" ]]; then
        minor2=0
        patch2=0
    fi

    # Compare the versions
    if [[ $major1 -gt $major2 ]]; then
        info "$1 is greater than $2"
        return 0
    elif [[ $major1 -lt $major2 ]]; then
        info "$1 is less than $2"
        return 2
    elif [[ $minor1 -gt $minor2 ]]; then
        info "$1 is greater than $2"
        return 0
    elif [[ $minor1 -lt $minor2 ]]; then
        info "$1 is less than $2"
        return 2
    elif [[ $patch1 -gt $patch2 ]]; then
        info "$1 is greater than $2"
        return 0
    elif [[ $patch1 -lt $patch2 ]]; then
        info "$1 is less than $2"
        return 2
    else
        info "$1 is equal to $2"
        return 3
    fi
}

function update_operator_channel() {
    local package_name=$1
    local ns=$2
    local channel=$3
    local source=$4
    local source_ns=$5
    local install_mode=$6
    
    local sub_name=$(${OC} get subscription.operators.coreos.com -n ${ns} -l operators.coreos.com/${package_name}.${ns}='' --no-headers | awk '{print $1}')
    ${OC} get subscription.operators.coreos.com ${sub_name} -n ${ns} -o yaml > sub.yaml
    
    existing_channel=$(yq eval '.spec.channel' sub.yaml)
    compare_semantic_version $existing_channel $channel
    return_value=$?
    
    if [[ $return_value -eq 3 ]]; then
        info "$package_name already has channel $existing_channel in the subscription."
        return 0
    elif [[ $return_value -ne 2 ]]; then
        error "Failed to update channel subscription ${package_name} in ${ns}"
    fi

    yq -i eval 'select(.kind == "Subscription") | .spec += {"channel": "'${channel}'"}' sub.yaml
    yq -i eval 'select(.kind == "Subscription") | .spec += {"source": "'${source}'"}' sub.yaml
    yq -i eval 'select(.kind == "Subscription") | .spec += {"sourceNamespace": "'${source_ns}'"}' sub.yaml
    yq -i eval 'select(.kind == "Subscription") | .spec += {"installPlanApproval": "'${install_mode}'"}' sub.yaml

    ${OC} apply -f sub.yaml
    if [[ $? -ne 0 ]]; then
        error "Failed to update subscription ${package_name} in ${ns}"
    fi
    rm sub.yaml
}

function delete_operator() {
    subs=$1
    ns=$2
    for sub in ${subs}; do
        title "Deleting ${sub} in namesapce ${ns}..."
        csv=$(${OC} get sub ${sub} -n ${ns} -o=jsonpath='{.status.installedCSV}' --ignore-not-found)
        in_step=1
        msg "[${in_step}] Removing the subscription of ${sub} in namesapce ${ns} ..."
        ${OC} delete sub ${sub} -n ${ns} --ignore-not-found
        in_step=$((in_step + 1))
        msg "[${in_step}] Removing the csv of ${sub} in namesapce ${ns} ..."
        [[ "X${csv}" != "X" ]] && ${OC} delete csv ${csv}  -n ${ns} --ignore-not-found
        msg ""

        success "Remove $sub successfully."
        msg ""
    done
}