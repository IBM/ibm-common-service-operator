#!/usr/bin/env bash
#
# Copyright 2024 IBM Corporation
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

set -o pipefail
set -o errtrace

BACKUP="false"
RESTORE="false"
OC="oc"
YQ="yq"

BASE_DIR=$(cd $(dirname "$0")/$(dirname "$(readlink $0)") && pwd -P)
. ../cp3pt0-deployment/common/utils.sh
source ${BASE_DIR}/env-oadp.properties

function main() {
    if [[ $RESTORE == "true" ]]; then
        restore_cpfs
    fi
}

function restore_cpfs(){
    title "Start CPFS restore."
    if [ -d "templates" ]; then
        rm -rf templates
    fi
    mkdir templates
    info "Copying template files..."
    cp -r ../velero/restore ${BASE_DIR}/templates/

    sed -i -E "s/__BACKUP_NAME__/$BACKUP_NAME/" ${BASE_DIR}/templates/restore/restore-namespace.yaml
    sed -i -E "s/__BACKUP_NAME__/$BACKUP_NAME/" ${BASE_DIR}/templates/restore/restore-entitlementkey.yaml
    sed -i -E "s/__BACKUP_NAME__/$BACKUP_NAME/" ${BASE_DIR}/templates/restore/restore-configmap.yaml
    sed -i -E "s/__BACKUP_NAME__/$BACKUP_NAME/" ${BASE_DIR}/templates/restore/restore-crd.yaml
    sed -i -E "s/__BACKUP_NAME__/$BACKUP_NAME/" ${BASE_DIR}/templates/restore/restore-commonservice.yaml
    sed -i -E "s/__BACKUP_NAME__/$BACKUP_NAME/" ${BASE_DIR}/templates/restore/restore-cert-manager.yaml
    sed -i -E "s/__BACKUP_NAME__/$BACKUP_NAME/" ${BASE_DIR}/templates/restore/no-olm/restore-cluster-scope.yaml
    sed -i -E "s/__BACKUP_NAME__/$BACKUP_NAME/" ${BASE_DIR}/templates/restore/no-olm/restore-namespace-scope.yaml
    sed -i -E "s/__BACKUP_NAME__/$BACKUP_NAME/" ${BASE_DIR}/templates/restore/no-olm/restore-installer-ns-charts.yaml
    sed -i -E "s/__BACKUP_NAME__/$BACKUP_NAME/" ${BASE_DIR}/templates/restore/no-olm/restore-im-ns-charts.yaml
    sed -i -E "s/__BACKUP_NAME__/$BACKUP_NAME/" ${BASE_DIR}/templates/restore/no-olm/restore-zen-ns-chart.yaml
    sed -i -E "s/__BACKUP_NAME__/$BACKUP_NAME/" ${BASE_DIR}/templates/restore/no-olm/restore-ibm-cm-chart.yaml
    sed -i -E "s/__BACKUP_NAME__/$BACKUP_NAME/" ${BASE_DIR}/templates/restore/restore-operands.yaml
    sed -i -E "s/__BACKUP_NAME__/$BACKUP_NAME/" ${BASE_DIR}/templates/restore/restore-cs-db.yaml
    sed -i -E "s/__BACKUP_NAME__/$BACKUP_NAME/" ${BASE_DIR}/templates/restore/restore-zen5-data.yaml
    sed -i -E "s/__BACKUP_NAME__/$BACKUP_NAME/" ${BASE_DIR}/templates/restore/restore-licensing.yaml
    sed -i -E "s/__BACKUP_NAME__/$BACKUP_NAME/" ${BASE_DIR}/templates/restore/restore-nss.yaml
    sed -i -E "s/__BACKUP_NAME__/$BACKUP_NAME/" ${BASE_DIR}/templates/restore/restore-operatorgroup.yaml
    sed -i -E "s/__BACKUP_NAME__/$BACKUP_NAME/" ${BASE_DIR}/templates/restore/restore-pull-secret.yaml
    sed -i -E "s/__BACKUP_NAME__/$BACKUP_NAME/" ${BASE_DIR}/templates/restore/restore-singleton-subscriptions.yaml
    sed -i -E "s/__BACKUP_NAME__/$BACKUP_NAME/" ${BASE_DIR}/templates/restore/restore-catalog.yaml
    sed -i -E "s/__BACKUP_NAME__/$BACKUP_NAME/" ${BASE_DIR}/templates/restore/restore-subscriptions.yaml
    sed -i -E "s/__BACKUP_NAME__/$BACKUP_NAME/" ${BASE_DIR}/templates/restore/restore-zen.yaml
    sed -i -E "s/__BACKUP_NAME__/$BACKUP_NAME/" ${BASE_DIR}/templates/restore/restore-ums.yaml
    
    custom_columns_str="-o custom-columns=NAME:.metadata.name,STATUS:.status.phase,ITEMS_RESTORED:.status.progress.itemsRestored,TOTAL_ITEMS:.status.progress.totalItems,BACKUP:.spec.backupName,WARN:.status.warnings,ERR:.status.errors"
    info "Begin restore process..."
    #Initial restore objects, rarely fail, could theoretically be applied at once   
    info "Restoring namespaces, pull secret and entitlement keys..."
    ${OC} apply -f ${BASE_DIR}/templates/restore/restore-namespace.yaml -f ${BASE_DIR}/templates/restore/restore-pull-secret.yaml -f ${BASE_DIR}/templates/restore/restore-entitlementkey.yaml
    ${OC} get restores.velero.io -n $OADP_NS $custom_columns_str
    wait_for_restore restore-namespace
    wait_for_restore restore-pull-secret
    wait_for_restore restore-entitlementkey
    
    ${OC} get restores.velero.io -n $OADP_NS $custom_columns_str
    info "Restoring catalog sources..."
    ${OC} apply -f ${BASE_DIR}/templates/restore/restore-catalog.yaml
    wait_for_restore restore-catalog
    info "Restore operator groups, CRDs, and configmaps..."
    ${OC} apply -f ${BASE_DIR}/templates/restore/restore-operatorgroup.yaml -f ${BASE_DIR}/templates/restore/restore-crd.yaml -f ${BASE_DIR}/templates/restore/restore-configmap.yaml
    ${OC} get restores.velero.io -n $OADP_NS $custom_columns_str
    wait_for_restore restore-operatorgroup
    wait_for_restore restore-crd
    wait_for_restore restore-configmap
    
    #Singleton subscriptions (Cert manager, licensing, LSR)
    if [[ $RESTORE_SINGLETONS == "true" ]]; then
        info "Restoring Singleton subscriptions..."
        ${OC} apply -f ${BASE_DIR}/templates/restore/restore-singleton-subscriptions.yaml
        wait_for_restore restore-singleton-subscription
    fi
    ${OC} get restores.velero.io -n $OADP_NS $custom_columns_str
    
    wait_for_cert_manager $CERT_MANAGER_NAMESPACE $SERVICES_NS
    info "Restoring cert manager resources (secrets, certificates, issuers, etc.)..."
    ${OC} apply -f ${BASE_DIR}/templates/restore/restore-cert-manager.yaml
    wait_for_restore restore-cert-manager
    
    #Restore the common service CR and the tenant scope via nss
    info "Restoring common service CR..."
    ${OC} apply -f ${BASE_DIR}/templates/restore/restore-commonservice.yaml
    wait_for_restore restore-commonservice
    if [[ $NSS_ENABLED == "true" ]]; then 
        info "Restoring Namespace Scope resources..."
        ${OC} apply -f ${BASE_DIR}/templates/restore/restore-nss.yaml
        wait_for_restore restore-nss
        ${OC} get restores.velero.io -n $OADP_NS $custom_columns_str
        validate_nss $OPERATOR_NS
    fi

    #restore common service subscription and odlm operator
    info "Restore CS and ODLM Operators..."
    ${OC} apply -f ${BASE_DIR}/templates/restore/restore-subscriptions.yaml
    wait_for_restore restore-subscription
    validate_cs_odlm $OPERATOR_NS
    if [[ $UMS_ENABLED == "true" ]]; then
        info "Restoring UMS resources..."
        ${OC} apply -f ${BASE_DIR}/templates/restore/restore-ums.yaml
        wait_for_restore restore-ums
    fi
    ${OC} get restores.velero.io -n $OADP_NS $custom_columns_str
    info "Restoring operands..."
    ${OC} apply -f ${BASE_DIR}/templates/restore/restore-operands.yaml
    wait_for_restore restore-operands
    #TODO put into its own function to be called here (and so it can be called independently and multiple times for multiple tenants)
    if [[ $IM_ENABLED == "true" ]]; then
        info "Restoring IM Data..."
        wait_for_im example-authentication $SERVICES_NS
        ${OC} apply -f ${BASE_DIR}/templates/restore/restore-cs-db-data.yaml
        wait_for_restore restore-cs-db-data
    fi
    #TODO put into its own function to be called here (and so it can be called independently and multiple times for multiple tenants)
    if [[ $ZEN_ENABLED == "true" ]]; then
        info "Restoring zenservice..."
        ${OC} apply -f ${BASE_DIR}/templates/restore/restore-zen.yaml
        wait_for_restore restore-zen
        wait_for_zenservice
        info "Restoring zen data..."
        ${OC} apply -f ${BASE_DIR}/templates/restore/restore-zen5-data.yaml
        wait_for_restore restore-zen5-data
    fi
    success "CPFS Restore completed."
}

function wait_for_restore() {
    restore=$1
    status=$(${OC} get restores.velero.io $restore -n $OADP_NS -o jsonpath='{.status.phase}')
    retries=30
    sleep_time=20
    if [[ $restore == "restore-zen5-data" ]]; then
        retries=120
        sleep_time=30
    fi
    while [[ $status != "Completed" ]] && [[ $retries -gt 0 ]]; do
        info "Wait for restore $restore to complete. Try again in $sleep_time seconds."
        sleep $sleep_time
        status=$(${OC} get restores.velero.io $restore -n $OADP_NS -o jsonpath='{.status.phase}')
        retries=$((retries-1))
        if [[ $status == "Failed" ]] || [[ $status == "PartiallyFailed" ]]; then
            if [[ $restore == "restore-zen5-data" ]]; then
                #TODO write apply_zen_workaround
                apply_zen_workaround $ZEN_NAMESPACE $ZENSERVICE_NAME
            elif [[ $restore == "restore-cs-db-data" ]]; then
                #TODO write apply_im_workaround
                apply_im_workaround $SERVICES_NS
            else
                ${OC} get restores.velero.io $restore -n $OADP_NS $custom_columns_str
                error "Restore $restore failed with status: $status. For more details, run \"velero restore describe --details $restore\"."
            fi
        fi
    done
    if [[ $status == "Completed" ]]; then
        info "Restore $restore completed successfully. For more details, run \"velero restore describe --details $restore\"."
    else
        error "Timed out waiting for restore $restore to complete successfully. For more details, run \"velero restore describe --details $restore\"."
    fi
}

function validate_nss() {
    local namespace=$1
    wait_for_csv "$namespace" "ibm-namespace-scope-operator"
    wait_for_operator "$namespace" "ibm-namespace-scope-operator"
}

function validate_cs_odlm() {
    local namespace=$1
    wait_for_csv "$namespace" "ibm-common-service-operator"
    wait_for_operator "$namespace" "ibm-common-service-operator"
    wait_for_csv "$namespace" "ibm-odlm"
    wait_for_operator "$namespace" "operand-deployment-lifecycle-manager"
    wait_for_cscr_status "$namespace" "common-service"
}

function wait_for_im() {
    local auth_cr=$1
    local namespace=$2
    local condition="${OC} get authentications.operator.ibm.com ${auth_cr} -n ${namespace} -o jsonpath='{.status.service.status}' | egrep Ready"
    local retries=30
    local sleep_time=30
    local total_time_mins=$(( sleep_time * retries / 60))
    local wait_message="Waiting on IM Service to be online. Checking status of authentication CR ${auth_CR} in namespace ${namespace}."
    local success_message="IM service ready in namespace ${namespace}."
    local error_message "Timeout after ${total_time_mins} minutes waiting for IM service in namespace ${namespace} to become available"
    wait_for_condition "${condition}" ${retries} ${sleep_time} "${wait_message}" "${success_message}" "${error_message}"
}

function wait_for_zenservice {
    info "Waiting for zenservice $ZENSERVICE_NAME to complete in namespace $ZEN_NAMESPACE."
    zenservice_exists=$(oc get zenservice $ZENSERVICE_NAME -n $ZEN_NAMESPACE --no-headers || echo fail)
    if [[ $zenservice_exists != "fail" ]]; then
        completed=$(oc get zenservice $ZENSERVICE_NAME -n $ZEN_NAMESPACE -o jsonpath='{.status.progress}')
        retry_count=60
        sleep_time=60
        while [[ $completed != "100%" ]] && [[ $retry_count > 0 ]]
        do
            info "Wait for zenservice $ZENSERVICE_NAME to complete. Completion % is $completed. Try again in 60s."
            sleep $sleep_time
            completed=$(oc get zenservice $ZENSERVICE_NAME -n $ZEN_NAMESPACE -o jsonpath='{.status.progress}')
            retry_count=$((retry_count-1))
        done

        if [[ $retry_count == 0 ]] && [[ $completed != "100%" ]]; then
            error "Timed out waiting for zenservice $ZENSERVICE_NAME."
        else
            info "Zenservice $ZENSERVICE_NAME ready."
        fi
    else
        error "Zenservice $ZENSERVICE_NAME not present."
    fi
}

function apply_im_workaround() {
    local namespace=$1
    info "IM data restored failed, attempting workaround..."
    ${OC} delete deployment cs-db-backup -n $namespace
    sed -i -E "s/<cs-db namespace>/$namespace/" ${BASE_DIR}/templates/restore/common-service-db/cs-db-restore-job.yaml
    info "Creating workaround job cs-db-restore-job"
    ${OC} apply -f ${BASE_DIR}/templates/restore/common-service-db/cs-db-restore-job.yaml
    wait_for_job_complete cs-db-restore-job $namespace

}

function apply_zen_workaround() {
    local namespace=$1
    local zenservice=$2
    info "Zen data restored failed, attempting workaround..."
    ${OC} delete deployment zen5-backup -n $namespace
    sed -i -E "s/<zenservice namespace>/$namespace/" ${BASE_DIR}/templates/restore/zen/zen5-restore-job.yaml
    sed -i -E "s/<zenservice name>/$zenservice/" ${BASE_DIR}/templates/restore/zen/zen5-restore-job.yaml
    info "Creating workaround job zen5-restore-job"
    ${OC} apply -f ${BASE_DIR}/templates/restore/zen/zen5-restore-job.yaml
    wait_for_job_complete zen5-restore-job $namespace
}

function wait_for_job_complete() {
  local job_name=$1
  local namespace=$2
  local condition="${OC} get pod -n $namespace --no-headers --ignore-not-found | grep ${job_name} | grep 'Completed' || true"
  local retries=15
  local sleep_time=15
  local total_time_mins=$(( sleep_time * retries / 60))
  local wait_message="Waiting for job pod $job_name to complete"
  local success_message="Job $job_name completed in namespace $namespace"
  local error_message="Timeout after ${total_time_mins} minutes waiting for pod $pod "
  wait_for_condition "${condition}" ${retries} ${sleep_time} "${wait_message}" "${success_message}" "${error_message}"
  dumplogs $job_name
  info "Deleting job $job_name"
  ${OC} delete job $job_name -n $namespace
}

function wait_for_cert_manager() {
    local cm_namespace=$1
    local name="cert-manager-webhook"
    local test_namespace=$2
    local needReplicas=$(${OC} -n ${namespace} get deployment ${name} --no-headers --ignore-not-found -o jsonpath='{.spec.replicas}' | awk '{print $1}')
    local readyReplicas="${OC} -n ${cm_namespace} get deployment ${name} --no-headers --ignore-not-found -o jsonpath='{.status.readyReplicas}' | grep '${needReplicas}'"
    local replicas="${OC} -n ${namespace} get deployment ${name} --no-headers --ignore-not-found -o jsonpath='{.status.replicas}' | grep '${needReplicas}'"
    local condition="(${readyReplicas} && ${replicas})"
    local retries=20
    local sleep_time=30
    local total_time_mins=$(( sleep_time * retries / 60))
    local wait_message="Waiting for cert manager webhook pod to come ready"
    local success_message="Cert Manager operator in namespace $cm_namespace ready."
    local error_message="Timeout after ${total_time_mins} minutes waiting for cert manager webhook pod."
    wait_for_condition "${condition}" ${retries} ${sleep_time} "${wait_message}" "${success_message}" "${error_message}"
    #from utils.sh, checks if cert manager exists and then runs smoke test
    check_cert_manager cert-manager $test_namespace
}


function check_yq() {
  yq_version=$("${YQ}" --version | awk '{print $NF}' | sed 's/^v//')
  yq_minimum_version=4.18.1

  if [ "$(printf '%s\n' "$yq_minimum_version" "$yq_version" | sort -V | head -n1)" != "$yq_minimum_version" ]; then 
    error "yq version $yq_version must be at least $yq_minimum_version or higher.\nInstructions for installing/upgrading yq are available here: https://github.com/marketplace/actions/yq-portable-yaml-processor"
  fi
}

function msg() {
    printf '%b\n' "$1"
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

function info() {
    msg "[INFO] ${1}"
}

main $*