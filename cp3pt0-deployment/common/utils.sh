#!/usr/bin/env bash

# Licensed Materials - Property of IBM
# Copyright IBM Corporation 2023. All Rights Reserved
# US Government Users Restricted Rights -
# Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
#
# This is an internal component, bundled with an official IBM product. 
# Please refer to that particular license for additional information. 

# ---------- Info functions ----------#

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

# ---------- Check functions start ----------#

function check_command() {
    local command=$1

    if [[ -z "$(command -v ${command} 2> /dev/null)" ]]; then
        error "${command} command not available"
    else
        success "${command} command available"
    fi
}

function check_version() {
    local command=$1
    local version_cmd=$2
    local variant=$3
    local version=$4

    result=$(${command} ${version_cmd})
    echo "$result" | grep -q "${variant}" && echo "$result" | grep -Eq "${version}"
    if [[ $? -ne 0 ]]; then
        error "${command} command is not supported"
    else
        success "${command} command is supported"
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
        success "${success_message}\n"
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
    local retries=12
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
    local csv_name=$($OC get subscription.operators.coreos.com ibm-common-service-operator -n ${namespace} -o jsonpath='{.status.installedCSV}')
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
        if [[ ( ${retries} -eq 0 ) && ( -z "${result}" ) ]]; then
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
        success "${success_message}\n"
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
    local install_mode=$4
    local condition="${OC} get subscription.operators.coreos.com -l operators.coreos.com/${package_name}.${namespace}='' -n ${namespace} -o yaml -o jsonpath='{.items[*].status.installedCSV}' | grep -w $channel"

    local retries=10
    local sleep_time=30
    local total_time_mins=$(( sleep_time * retries / 60))
    local wait_message="Waiting for operator ${package_name} to be upgraded"
    local success_message="Operator ${package_name} is upgraded to latest version in channel ${channel}"
    local error_message="Timeout after ${total_time_mins} minutes waiting for operator ${package_name} to be upgraded"

    if [[ "${install_mode}" == "Manual" ]]; then
        wait_message="Waiting for operator ${package_name} to be upgraded \nPlease manually approve installPlan to make upgrade proceeding..."
        error_message="Timeout after ${total_time_mins} minutes waiting for operator ${package_name} to be upgraded \nInstallPlan is not manually approved yet"
    fi

    wait_for_condition "${condition}" ${retries} ${sleep_time} "${wait_message}" "${success_message}" "${error_message}"
}

function wait_for_cs_webhook() {
    local namespace=$1
    local name=$2
    local condition="${OC} -n ${namespace} get service --no-headers | (grep ${name})"
    local retries=20
    local sleep_time=10
    local total_time_mins=$(( sleep_time * retries / 60))
    local wait_message="Waiting for CS webhook service to be ready"
    local success_message="CS Webhook Service ${name} is ready"
    local error_message="Timeout after ${total_time_mins} minutes waiting for common service webhook service to be ready"

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
    local service_name=$1    
    local namespace=$2
    title " Checking whether Cert Manager exist..." 
    if [[ $PREVIEW_MODE -eq 1 ]]; then
        info "Preview mode is on, skip checking whether Cert Manager exist\n"
        return 0       
    fi
    csv_count=`$OC get csv -n "$namespace" | grep "$service_name" | wc -l`
    if [[ $csv_count == 1 ]]; then
        success "Found only one Cert Manager exists in namespace "$namespace"\n"
    elif [[ $csv_count == 0 ]]; then
        error "Missing a Cert Manager\n"
    elif [[ $csv_count > 1 ]]; then
        error "Multiple Cert Manager csv found. Only one should be installed per cluster\n"
    fi
}

function check_licensing(){
    title " Checking IBMLicensing..."
    if [[ $PREVIEW_MODE -eq 1 ]]; then
        info "Preview mode is on, skip checking IBMLicensing\n"
        return 0
    fi
    [[ ! $($OC get IBMLicensing) ]] && error "User does not have proper permission to get IBMLicensing or IBMLicensing is not installed"
    instance_count=`$OC get IBMLicensing -o name | wc -l`
    if [[ $instance_count == 1 ]]; then
        success "Found only one IBMLicensing\n"
    elif [[ $instance_count == 0 ]]; then
        error "Missing IBMLicensing\n"
    elif [[ $instance_count > 1 ]]; then
        error "Multiple IBMLicensing are found. Only one should be installed per cluster\n"
    fi
}
# ---------- Check functions end ----------#

# ---------- creation functions start ----------#

function create_namespace() {
    local namespace=$1
    title "Checking whether Namespace $namespace exist..."
    if [[ -z "$(${OC} get namespace ${namespace} --ignore-not-found)" ]]; then
        info "Creating namespace ${namespace}"
        ${OC} create namespace ${namespace}
        if [[ $? -ne 0 ]]; then
            error "Error creating namespace ${namespace}"
        fi
        if [[ $PREVIEW_MODE -eq 0 ]]; then
            wait_for_project ${namespace}
        fi
    else
        success "Namespace ${namespace} already exists. Skip creating\n"
    fi
}

function create_operator_group() {
    local name=$1
    local ns=$2
    local target=$3
    cat <<EOF > ${PREVIEW_DIR}/operatorgroup.yaml
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: $name
  namespace: $ns
spec: $target
EOF

    title "Checking whether OperatorGroup in $ns exist..."
    existing_og=$(${OC} get operatorgroup -n $ns --no-headers --ignore-not-found | wc -l)
    if [[ ${existing_og} -ne 0 ]]; then
        success "OperatorGroup already exists in $ns. Skip creating\n"
        return 0
    fi
    info "Creating following OperatorGroup:\n"
    cat ${PREVIEW_DIR}/operatorgroup.yaml
    echo ""
    cat "${PREVIEW_DIR}/operatorgroup.yaml" | ${OC} apply -f -
    if [[ $? -ne 0 ]]; then
        error "Failed to create OperatorGroup ${name} in ${ns}\n"
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
    cat <<EOF > ${PREVIEW_DIR}/${name}-subscription.yaml
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

    info "Creating following Subscription:\n"
    cat ${PREVIEW_DIR}/${name}-subscription.yaml
    echo ""
    cat ${PREVIEW_DIR}/${name}-subscription.yaml | ${OC} apply -f -
    if [[ $? -ne 0 ]]; then
        error "Failed to create subscription ${name} in ${ns}\n"
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
        ${YQ} eval 'select(.kind == "CommonService") | del(.metadata.resourceVersion) | del(.metadata.uid) | .metadata.namespace = "'${namespace}'"' common-service.yaml | ${OC} apply --overwrite=true -f -
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
# ---------- creation functions end----------#

# ---------- cleanup functions start----------#

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
    #check if crossplane operator is installed or not
    local is_exist=$($OC get subscription.operators.coreos.com -A --no-headers | (grep ibm-crossplane || echo "fail") | awk '{print $1}')
    if [[ $is_exist != "fail" ]]; then
        # delete CR
        info "cleanup crossplane CR"
        ${OC} get configuration.pkg.ibm.crossplane.io -A --no-headers | awk '{print $1}' | xargs ${OC} delete --ignore-not-found configuration.pkg.ibm.crossplane.io
        ${OC} get lock.pkg.ibm.crossplane.io -A --no-headers | awk '{print $1}' | xargs ${OC} delete --ignore-not-found lock.pkg.ibm.crossplane.io
        ${OC} get ProviderConfig -A --no-headers | awk '{print $1}' | xargs ${OC} delete --ignore-not-found ProviderConfig

        sleep 30

        # delete Sub
        info "cleanup crossplane Subscription and ClusterServiceVersion"
        local namespace=$($OC get subscription.operators.coreos.com -A --no-headers | (grep ibm-crossplane-operator-app || echo "fail") | awk '{print $1}')
        if [[ $namespace != "fail" ]]; then
            delete_operator "ibm-crossplane-provider-kubernetes-operator-app" "$namespace"
            delete_operator "ibm-crossplane-provider-ibm-cloud-operator-app" "$namespace"
            delete_operator "ibm-crossplane-operator-app" "$namespace"
        fi
    else
        info "crossplane operator not exist, skip clean crossplane"
    fi
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
# ---------- cleanup functions end ----------#

function get_control_namespace() {
    # Define the ConfigMap name and namespace
    local config_map_name="common-service-maps"

    # Get the ConfigMap data
    config_map_data=$(${OC} get configmap "${config_map_name}" -n kube-public -o yaml | ${YQ} '.data[]')

    # Check if the ConfigMap exists
    if [[ -z "${config_map_data}" ]]; then
        warning "Not found common-serivce-maps ConfigMap in kube-public namespace. It is a single shared Common Service instance upgrade"
    else
        # Get the controlNamespace value
        control_namespace=$(echo "${config_map_data}" | ${YQ} -r '.controlNamespace')

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
        return 1
    elif [[ $major1 -lt $major2 ]]; then
        info "$1 is less than $2"
        return 2
    elif [[ $minor1 -gt $minor2 ]]; then
        info "$1 is greater than $2"
        return 1
    elif [[ $minor1 -lt $minor2 ]]; then
        info "$1 is less than $2"
        return 2
    elif [[ $patch1 -gt $patch2 ]]; then
        info "$1 is greater than $2"
        return 1
    elif [[ $patch1 -lt $patch2 ]]; then
        info "$1 is less than $2"
        return 2
    else
        info "$1 is equal to $2"
        return 0
    fi
}

function compare_catalogsource(){
    # Compare the catalogsource
    if [[ $1 == $2 ]]; then
        info "catalogsource $1 is the same as $2"
        return 0
    else
        info "catalogsource $1 is different from $2"
        return 1
    fi
}

function update_operator() {
    local package_name=$1
    local ns=$2
    local channel=$3
    local source=$4
    local source_ns=$5
    local install_mode=$6
    local retries=5 # Number of retries
    local delay=5 # Delay between retries in seconds
    
    local sub_name=$(${OC} get subscription.operators.coreos.com -n ${ns} -l operators.coreos.com/${package_name}.${ns}='' --no-headers | awk '{print $1}')
    if [ -z "$sub_name" ]; then
        warning "Not found subscription ${package_name} in ${ns}"
        return 0
    fi

    title "Updating ${sub_name} in namesapce ${ns}..."
    while [ $retries -gt 0 ]; do
        # Retrieve the latest version of the subscription
        ${OC} get subscription.operators.coreos.com ${sub_name} -n ${ns} -o yaml > sub.yaml
    
        existing_channel=$(${YQ} eval '.spec.channel' sub.yaml)
        existing_catalogsource=$(${YQ} eval '.spec.source' sub.yaml)

        compare_semantic_version $existing_channel $channel
        return_channel_value=$?

        compare_catalogsource $existing_catalogsource $source
        return_catsrc_value=$?

        if [[ $return_channel_value -eq 1 ]]; then
            error "Failed to update channel subscription ${package_name} in ${ns}"
        elif [[ $return_channel_value -eq 2 || $return_catsrc_value -eq 1 ]]; then
            info "$package_name is ready for updating the subscription."      
        elif [[ $return_channel_value -eq 0 && $return_catsrc_value -eq 0 ]]; then
            info "$package_name has already updated channel $existing_channel and catalogsource $existing_catalogsource in the subscription."
        fi

        # Update the subscription with the desired changes
        ${YQ} -i eval 'select(.kind == "Subscription") | .spec += {"channel": "'${channel}'"}' sub.yaml
        ${YQ} -i eval 'select(.kind == "Subscription") | .spec += {"source": "'${source}'"}' sub.yaml
        ${YQ} -i eval 'select(.kind == "Subscription") | .spec += {"sourceNamespace": "'${source_ns}'"}' sub.yaml
        ${YQ} -i eval 'select(.kind == "Subscription") | .spec += {"installPlanApproval": "'${install_mode}'"}' sub.yaml

        # Apply the patch
        ${OC} apply -f sub.yaml
    
        # Check if the patch was successful
        if [[ $? -eq 0 ]]; then
            success "Successfully patched subscription ${package_name} in ${ns}"
            rm sub.yaml
            return 0
        else
            warning "Failed to patch subscription ${package_name} in ${ns}. Retrying in ${delay} seconds..."
            sleep ${delay}
            retries=$((retries-1))
        fi
    done

    error "Maximum retries reached. Failed to patch subscription ${sub_name} in ${ns}"
    rm sub.yaml
    return 1
}

function delete_operator() {
    subs=$1
    ns=$2
    for sub in ${subs}; do
        title "Deleting ${sub} in namesapce ${ns}..."
        csv=$(${OC} get subscription.operators.coreos.com ${sub} -n ${ns} -o=jsonpath='{.status.installedCSV}' --ignore-not-found)
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

function scale_deployment_csv() {
    local ns=$1
    local csv=$2
    local replicas=$3
    ${OC} patch csv ${csv} -n ${ns} --type='json' -p='[{"op": "replace", "path": "/spec/install/spec/deployments/0/spec/replicas", "value": '$((replicas))'}]'
}

function check_deployment(){
    local ns=$1
    local deployment=$2
    local replicas=$3
    local retries=5 
    local count=0

    while [ $count -lt $retries ]; do
        local current_replicas=$(${OC} get deployment ${deployment} -n ${ns} --ignore-not-found -o jsonpath='{.spec.replicas}')

        if [[ -z "$current_replicas" ]]; then
            current_replicas=0
        fi
            
        if [ "$current_replicas" -eq "$replicas" ]; then
            success "Replicas count is as expected: $current_replicas"
            return 0
        else
            warning "Replica count is not as expected: $current_replicas (expected: $replicas)"
            count=$((count+1))
            sleep 5
        fi
    done

    msg "Failed to reach expected replica count after $retries attempts, scaling deployment..."
    return 1
}

function scale_deployment() {
    local ns=$1
    local deployment=$2
    ${OC} scale deployment ${deployment} -n ${ns} --replicas=$3
}

function scale_down() {
    local operator_ns=$1
    local services_ns=$2
    local channel=$3
    local source=$4
    local cs_sub=$(${OC} get subscription.operators.coreos.com -n ${operator_ns} -l operators.coreos.com/ibm-common-service-operator.${operator_ns}='' --no-headers | awk '{print $1}')
    local cs_CSV=$(${OC} get subscription.operators.coreos.com ${cs_sub} -n ${operator_ns} --ignore-not-found -o jsonpath={.status.installedCSV})
    local odlm_sub=$(${OC} get subscription.operators.coreos.com -n ${operator_ns} -l operators.coreos.com/ibm-odlm.${operator_ns}='' --no-headers | awk '{print $1}')
    local odlm_CSV=$(${OC} get subscription.operators.coreos.com ${odlm_sub} -n ${operator_ns} --ignore-not-found -o jsonpath={.status.installedCSV})
    
    ${OC} get subscription.operators.coreos.com ${cs_sub} -n ${operator_ns} -o yaml > sub.yaml

    existing_channel=$(${YQ} eval '.spec.channel' sub.yaml)
    existing_catalogsource=$(${YQ} eval '.spec.source' sub.yaml)
    compare_semantic_version $existing_channel $channel
    return_channel_value=$?

    compare_catalogsource $existing_catalogsource $source
    return_catsrc_value=$?

    if [[ $return_channel_value -eq 1 ]]; then 
        error "Must provide correct channel. The channel $CHANNEl is less than $existing_channel found in subscription ibm-common-service-operator in $operator_ns"
    elif [[ $return_channel_value -eq 2 || $return_catsrc_value -eq 1 ]]; then
        info "$cs_sub is ready for scaling down."
    elif [[ $return_channel_value -eq 0 && $return_catsrc_value -eq 0 ]]; then
        info "$cs_sub already has updated channel $existing_channel and catalogsource $existing_catalogsource in the subscription."
    fi

    # Scale down CS
    msg "Patching CSV ${cs_sub} to scale down deployment in ${operator_ns} namespace to 0..."
    if [[ ! -z "$cs_CSV" ]]; then
        scale_deployment_csv $operator_ns $cs_CSV 0
    fi
    check_deployment $operator_ns ibm-common-service-operator 0
    if [[ $? -ne 0 ]]; then
        msg "Scaling down ibm-common-service-operator deployment in ${operator_ns} namespace to 0..."
        scale_deployment $operator_ns ibm-common-service-operator 0
    fi
    
    # Scale down ODLM
    msg "Patching CSV to scale down operand-deployment-lifecycle-manager deployment in ${operator_ns} namespace to 0..."
    if [[ ! -z "$odlm_CSV" ]]; then
        scale_deployment_csv $operator_ns $odlm_CSV 0
    fi
    check_deployment $operator_ns operand-deployment-lifecycle-manager 0
    if [[ $? -ne 0 ]]; then
        msg "Scaling down operand-deployment-lifecycle-manager deployment in ${operator_ns} namespace to 0..."
        scale_deployment $operator_ns operand-deployment-lifecycle-manager 0
    fi
    
    # delete OperandRegistry
    msg "Deleting OperandRegistry common-service in ${services_ns} namespace..."
    ${OC} delete operandregistry common-service -n ${services_ns} --ignore-not-found
    # delete validatingwebhookconfiguration
    msg "Deleting ValidatingWebhookConfiguration ibm-common-service-validating-webhook-${operator_ns} in ${operator_ns} namespace..."
    ${OC} delete ValidatingWebhookConfiguration ibm-common-service-validating-webhook-${operator_ns} --ignore-not-found
    rm sub.yaml 
}

function wait_for_operand_registry() {
    local namespace=$1
    local name=$2
    local condition="${OC} -n ${namespace} get operandregistry ${name} --no-headers --ignore-not-found"
    local retries=20
    local sleep_time=10
    local total_time_mins=$(( sleep_time * retries / 60))
    local wait_message="Waiting for OperandRegistry ${name} to be present"
    local success_message="OperandRegistry ${name} is present"
    local error_message="Timeout after ${total_time_mins} minutes waiting for operand registry ${name} to be present"
 
    wait_for_condition "${condition}" ${retries} ${sleep_time} "${wait_message}" "${success_message}" "${error_message}"
}

function scale_up() {
    local operator_ns=$1
    local services_ns=$2
    local package_name=$3
    local deployment=$4
    local sub=$(${OC} get subscription.operators.coreos.com -n ${operator_ns} -l operators.coreos.com/${package_name}.${operator_ns}='' --no-headers | awk '{print $1}')
    local csv=$(${OC} get subscription.operators.coreos.com ${sub} -n ${operator_ns} --ignore-not-found -o jsonpath={.status.installedCSV})

    if [[ "$deployment" == "operand-deployment-lifecycle-manager" ]]; then
        wait_for_operand_registry ${services_ns} common-service
    fi
    msg "Patching CSV ${csv} to scale up deployment in ${operator_ns} namespace back to 1..."
    scale_deployment_csv $operator_ns $csv 1
    check_deployment $operator_ns $deployment 1
    if [[ $? -ne 0 ]]; then
        msg "Scaling up ${deployment} deployment in ${operator_ns} namespace back to 1..."
        scale_deployment $operator_ns $deployment 1
    fi
}

function accept_license() {
    local kind=$1
    local namespace=$2
    local cr_name=$3
    title "Accepting license for $kind $cr_name in namespace $namespace..."
    if [[ $PREVIEW_MODE -eq 1 ]]; then
        info "Preview mode is on, skip patching license acceptance\n"
        return 0       
    fi
    ${OC} patch "$kind" "$cr_name" -n "$namespace" --type='merge' -p '{"spec":{"license":{"accept":true}}}' || export fail="true"
    if [[ $fail == "true" ]]; then
        warning "Failed to update license acceptance for $kind CR $cr_name\n"
    else
        success "License accepted for $kind $cr_name\n"
    fi
}


function fetch_sub_from_package() {
    local package=$1
    local ns=$2

    ${OC} get sub -n "$ns" -o jsonpath="{.items[?(@.spec.name=='$package')].metadata.name}"
}

function fetch_csv_from_sub() {
    local sub=$1
    local ns=$2

    ${OC} get csv -n "$ns" | grep "$sub" | cut -d ' ' -f1
}

function remove_all_finalizers() {
    local ns=$1

    apiGroups=$(${OC} api-resources --namespaced -o name)
    delete_operand_finalizer "${apiGroups}" "${ns}"

}

function delete_operand_finalizer() {
    local crds=$1
    local ns=$2
    for crd in ${crds}; do
        if [ "${crd}" != "packagemanifests.packages.operators.coreos.com" ] && [ "${crd}" != "events" ] && [ "${crd}" != "events.events.k8s.io" ]; then
            crs=$(${OC} get ${crd} --no-headers --ignore-not-found -n ${ns} 2>/dev/null | awk '{print $1}')
            for cr in ${crs}; do
                msg "Removing the finalizers for resource: ${crd}/${cr}"
                ${OC} patch ${crd} ${cr} -n ${ns} --type="json" -p '[{"op": "remove", "path":"/metadata/finalizers"}]' 2>/dev/null
            done
        fi
    done
}

function save_log(){
    local LOG_DIR="$BASE_DIR/$1"
    LOG_FILE="$LOG_DIR/$2_$(date +'%Y%m%d%H%M%S').log"
    local debug=$3

    if [ $debug -eq 1 ]; then
        if [[ ! -d $LOG_DIR ]]; then
            mkdir -p "$LOG_DIR"
        fi

        # Create a named pipe
        PIPE=$(mktemp -u)
        mkfifo "$PIPE"

        # Tee the output to both the log file and the terminal
        tee "$LOG_FILE" < "$PIPE" &

        # Redirect stdout and stderr to the named pipe
        exec > "$PIPE" 2>&1

        # Remove the named pipe
        rm "$PIPE"
    fi
}

function cleanup_log() {
    # Check if the log file already exists
    if [[ -e $LOG_FILE ]]; then
        # Remove ANSI escape sequences from log file
        sed -E 's/\x1B\[[0-9;]+[A-Za-z]//g' "$LOG_FILE" > "$LOG_FILE.tmp" && mv "$LOG_FILE.tmp" "$LOG_FILE"
    fi
}

function debug1() {
    if [ $DEBUG -eq 1 ]; then
       debug "${1}"
    fi
}

# check if version of CS supports delegation for ibm-cert-manager-operator
# >= v3.19.9 if in v3 channel
# or >= v3.21.0 in any other channel
function is_supports_delegation() {
    local version=$1
    major=$(echo "$version" | cut -d '.' -f1 | cut -d 'v' -f2)
    minor=$(echo "$version" | cut -d '.' -f2)
    patch=$(echo "$version" | cut -d '.' -f3)

    if [ -z "$version" ]; then
        info "No ibm-common-service-operator found on the cluster, skipping delegation check"
        return 0
    fi

    if [ "$major" -gt 3 ]; then
        info "Major version is greater than 3, skipping delegation check"
        return 0
    fi

    if [ "$major" -lt 3 ]; then
        return 1
    fi

    if [ "$minor" -lt 19 ]; then
        return 1
    fi

    # only LTSR starting from 3.19.9 supported delegation
    if [ "$minor" -eq 19 ]; then
        if [ "$patch" -lt 9 ]; then
            return 1
        fi
    fi

    echo "Version: $version supports cert-manager delegation"
}

