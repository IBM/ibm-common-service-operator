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

OUTPUT_FILE="env-oadp.properties"
WRITE="false"

BASE_DIR=$(cd $(dirname "$0")/$(dirname "$(readlink $0)") && pwd -P)
. ../cp3pt0-deployment/common/utils.sh

function main() {
    parse_arguments "$@"
    prereq
    if [[ $RESTORE == "true" ]]; then
        restore_cpfs
    fi
}

function print_usage(){
    script_name=`basename ${0}`
    echo "Usage: ${script_name} [OPTIONS]"
    echo ""
    echo "Automate running OADP/Velero Backup or Restore."
    echo "One of --backup or --restore is required."
    echo "This script assumes the following:"
    echo "    * At least a Fusion Hub cluster setup with Fusion Backup and Restore Service and CPFS installed."
    echo "    * If 'spoke' selected for --cluster-type, Fusion Backup and Restore Agent Service installed and matching Storageclass to Hub cluster."
    echo "    * Fusion setup was completed with the fusion-backup-setup.sh script"
    echo ""
    echo "Options:"
    echo "   --oc string                    Optional. File path to oc CLI. Default uses oc in your PATH. Can also be set in env.properties."
    echo "   --yq string                    Optional. File path to yq CLI. Default uses yq in your PATH. Can also be set in env.properties."
    echo "   --backup                       Optional. Enable backup mode, it will trigger a backup job."
    echo "   --backup-name                  Necessary. Name of backup. A unique name is required when --backup is enabled. An existing name is required when --restore is enabled"
    echo "   --restore                      Optional. Enable restore mode, it will trigger a restore job."
    echo "   --env-file                     Optional. Enter env var file to populate necessary parameters. Default file name is env-oadp.properties."
    echo "   --write-env-file               Optional. Write set of env variables to specified output file. If --env-file not specified, defaults to env-oadp.properties. File must already exist."
    echo "   -h, --help                     Print usage information"
    echo ""
}

function parse_arguments() {
    script_name=`basename ${0}`
    echo "All arguments passed into the ${script_name}: $@"
    echo ""

    # process options
    while [[ "$@" != "" ]]; do
        case "$1" in
        --oc)
            shift
            OC=$1
            ;;
        --yq)
            shift
            YQ=$1
            ;;
        --backup)
            BACKUP="true"
            ;;
        --backup-name)
            shift
            BACKUP_NAME=$1
            ;;
        --restore)
            RESTORE="true"
            ;;
        --env-file)
            shift
            OUTPUT_FILE=$1
            source ${BASE_DIR}/$OUTPUT_FILE
            ;;

        --write-env-file)
            WRITE="true"
            ;;
        -h | --help)
            print_usage
            exit 1
            ;;
        *)
            echo "Entered option $1 not supported. Run ./${script_name} -h for script usage info."
            ;;
        esac
        shift
    done
    echo ""
}

function prereq() {
    #check that oc and yq are available
    check_command "${OC}"
    check_command "${YQ}"
    # Check yq version
    check_yq

    # Checking oc command logged in
    user=$(${OC} whoami 2> /dev/null)
    if [ $? -ne 0 ]; then
        error "You must be logged into the OpenShift Cluster from the oc command line"
    else
        success "oc command logged in as ${user}"
    fi

    #check variables are present
    # check backup/restore name
    if [[ $BACKUP == "true" ]]; then
        if [[ $BACKUP_NAME == "" ]]; then
            error "Backup name is necessary if Backup is enabled"
        fi
    elif [[ $RESTORE == "true" ]]; then
        if [[ $BACKUP_NAME == "" ]]; then
            error "An existing backup's name must be specified with --backup-name if Restore is enabled."
        fi
        #TODO add checks for namespace values
        if [[ $OPERATOR_NS == "" ]]; then
            error "OPERATOR_NS value not set. Make sure it is either set in the env-oadp.properties file or as an env variable."
        elif [[ $SERVICES_NS == "" ]]; then
            warning "SERVICES_NS value not set. Setting value equal to OPERATOR_NS value $OPERATOR_NS."
            SERVICES_NS=$OPERATOR_NS
        fi
        
        # check if any singleton is enabled individually and then check namespace values for each one
        #if any singleton enabled, trigger singleton enabled var
        if [[ $ENABLE_CERT_MANAGER == "true" ]]; then
            RESTORE_SINGLETONS="true"
            if [[ $CERT_MANAGER_NAMESPACE == "" ]]; then
                warning "Cert manager namespace not specified, setting to default ibm-cert-manager"
                CERT_MANAGER_NAMESPACE="ibm-cert-manager"
            fi
        fi
        if [[ $ENABLE_LICENSING == "true" ]]; then
            RESTORE_SINGLETONS="true"
            if [[ $LICENSING_NAMESPACE == "" ]]; then
                warning "Licensing namespace not specified, setting to default ibm-licensing"
                LICENSING_NAMESPACE="ibm-licensing"
            fi
        fi
        if [[ $ENABLE_LSR == "true" ]]; then
            RESTORE_SINGLETONS="true"
            if [[ $LSR_NAMESPACE == "" ]]; then
                warning "License service reporter namespace not specified, setting to default ibm-ls-reporter"
                LSR_NAMESPACE="ibm-licensing"
            fi
        fi

        #if zen enabled, verify zen namespace and zenservice name are both present
        if [[ $ZEN_ENABLED == "true" ]]; then
            if [[ $ZEN_NAMESPACE == "" ]]; then
                error "ZEN_NAMESPACE value not set. Make sure it is either set in the env-oadp.properties file or as an env variable."
            fi
            if [[ $ZENSERVICE_NAME == "" ]]; then
                error "ZENSERVICE_NAME value not set. Make sure it is either set in the env-oadp.properties file or as an env variable."
            fi
        fi

        check_cluster_credentials
    else
        error "Neither Backup nor Restore options were specified."
    fi
    
    #OADP setup checks
    # TODO check if OADP br service is installed on cluster
    # TODO check OADP related variables
    # if [[ $BACKUP_STORAGE_LOCATION_NAME == "" ]]; then
    #     error "Backup Storage Location name not specified in env-oadp.properties."
    # fi
    #also need secret key and id and any other values necessary for creating dataprotectionapplication
    
    #write env variables to output file
    if [[ $WRITE == "true" ]]; then
        write_specific_env_vars_to_file $OUTPUT_FILE "OC YQ OPERATOR_NS SERVICES_NS TETHERED_NS BACKUP RESTORE SETUP OADP_INSTALL OADP_RESOURCE_CREATION OADP_NS BACKUP_STORAGE_LOCATION_NAMESTORAGE_BUCKET_NAME S3_URL STORAGE_SECRET_ACCESS_KEY STORAGE_SECRET_ACCESS_KEY_ID IM_ENABLED ZEN_ENABLED NSS_ENABLED UMS_ENABLED CERT_MANAGER_NAMESPACE LICENSING_NAMESPACE LSR_NAMESPACE CPFS_VERSION ZENSERVICE_NAME ZEN_NAMESPACE ENABLE_CERT_MANAGER ENABLE_LICENSING ENABLE_LSR ENABLE_PRIVATE_CATALOG ENABLE_DEFAULT_CS ADDITIONAL_SOURCES CONTROL_NS BACKUP_CLU_SERVER BACKUP_CLU_TOKEN RESTORE_CLU_SERVER RESTORE_CLU_TOKEN TARGET_CLUSTER_TYPE BACKUP_NAME"
    fi
}


#cluster credentials validation function (will be necessary for setup as well)
#need some kind of validation for restoring to different cluster
# if restore to or setup for different cluster enabled, need login creds for both clusters
# test by logging in to the other cluster and then logging back into the base cluster
function check_cluster_credentials() {
    if [[ $TARGET_CLUSTER_TYPE == "" ]]; then
        error "TARGET_CLUSTER_TYPE value not set. Make sure it is either set in the env-oadp.properties file or as an env variable."
    else
        if [[ $TARGET_CLUSTER_TYPE == "diff" ]]; then
            if [[ $BACKUP_CLU_SERVER == "" ]] || [[ $BACKUP_CLU_TOKEN == "" ]] || [[ $RESTORE_CLU_SERVER == "" ]] || [[ $RESTORE_CLU_TOKEN == "" ]]; then
                error "If interacting with a different cluster (either restore or setup), all of BACKUP_CLU_SERVER, BACKUP_CLU_TOKEN, RESTORE_CLU_SERVER, and RESTORE_CLU_TOKEN must be defined either in the env-oadp.properties file or as environment variables." 
            else
                info "Different cluster selected. Validating login credentials work..."
                ${OC} login --token=$RESTORE_CLU_TOKEN--server=$RESTORE_CLU_SERVER --insecure-skip-tls-verify=true
                info "Logging back into home cluster..."
                ${OC} login --token=$BACKUP_CLU_TOKEN--server=$BACKUP_CLU_SERVER --insecure-skip-tls-verify=true
            fi
        fi
        success "Backup and Restore cluster login credentials verified."
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
    sed -i -E "s/__BACKUP_NAME__/$BACKUP_NAME/" ${BASE_DIR}/templates/restore/restore-lsr.yaml
    sed -i -E "s/__BACKUP_NAME__/$BACKUP_NAME/" ${BASE_DIR}/templates/restore/restore-lsr-data.yaml
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
    info "Cleanup existing pull secret..."
    ${OC} delete secret pull-secret -n openshift-config --ignore-not-found
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
        #we restore licensing before subs because the configmaps need to be there before licensing starts up
        if [[ $ENABLE_LICENSING == "true" ]]; then
            info "Restoring licensing configmaps..."
            ${OC} apply -f ${BASE_DIR}/templates/restore/restore-licensing.yaml
            wait_for_restore restore-licensing
        fi
        # same principle for lsr here as for licensing above
        if [[ $ENABLE_LSR == "true" ]]; then
            info "Restoring License Service Reporter instance..."
            ${OC} apply -f ${BASE_DIR}/templates/restore/restore-lsr.yaml
            wait_for_restore restore-lsr
        fi
        #this step restores the cert manager and licensing subs
        info "Restoring Singleton subscriptions..."
        ${OC} apply -f ${BASE_DIR}/templates/restore/restore-singleton-subscriptions.yaml
        wait_for_restore restore-singleton-subscription

        if [[ $ENABLE_LSR == "true" ]]; then
            info "Restoring License Service Reporter data..."
            ${OC} apply -f ${BASE_DIR}/templates/restore/restore-lsr-data.yaml
            wait_for_restore restore-lsr-data
        fi
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
    
    if [[ $IM_ENABLED == "true" ]]; then
        restore_im
    fi
    if [[ $ZEN_ENABLED == "true" ]]; then
        restore_zen
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
    elif [[ $restore == "restore-lsr-data" ]]; then
        retries=60
        sleep_time=15
    fi
    while [[ $status != "Completed" ]] && [[ $retries -gt 0 ]]; do
        info "Wait for restore $restore to complete. Try again in $sleep_time seconds."
        sleep $sleep_time
        status=$(${OC} get restores.velero.io $restore -n $OADP_NS -o jsonpath='{.status.phase}')
        retries=$((retries-1))
        if [[ $status == "Failed" ]] || [[ $status == "PartiallyFailed" ]]; then
            if [[ $restore == "restore-zen5-data" ]]; then
                apply_zen_workaround $ZEN_NAMESPACE $ZENSERVICE_NAME
                status="Completed"
            elif [[ $restore == "restore-cs-db-data" ]]; then
                apply_im_workaround $SERVICES_NS
                status="Completed"
            elif [[ $restore == "restore-lsr-data" ]]; then
                apply_lsr_workaround $LSR_NAMESPACE
                status="Completed"
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

function restore_im() {
    info "Restoring IM Data..."
    wait_for_im example-authentication $SERVICES_NS
    ${OC} apply -f ${BASE_DIR}/templates/restore/restore-cs-db.yaml
    wait_for_restore restore-cs-db-data
    success "IM data restored successfully."
}

function wait_for_im() {
    info "Sleep for 3 minutes for IM operator to create authentication cr"
    sleep 180
    local auth_cr=$1
    local namespace=$2
    local condition="${OC} get authentications.operator.ibm.com ${auth_cr} -n ${namespace} -o jsonpath='{.status.service.status}' | egrep Ready"
    local retries=30
    local sleep_time=30
    local total_time_mins=$(( sleep_time * retries / 60))
    local wait_message="Waiting on IM Service to be online. Checking status of authentication CR ${auth_CR} in namespace ${namespace}."
    local success_message="IM service ready in namespace ${namespace}."
    local error_message="Timeout after ${total_time_mins} minutes waiting for IM service in namespace ${namespace} to become available"
    wait_for_condition "${condition}" ${retries} ${sleep_time} "${wait_message}" "${success_message}" "${error_message}"
}

function restore_zen() {
    info "Restoring zenservice..."
    ${OC} apply -f ${BASE_DIR}/templates/restore/restore-zen.yaml
    wait_for_restore restore-zen
    wait_for_zenservice
    info "Restoring zen data..."
    ${OC} apply -f ${BASE_DIR}/templates/restore/restore-zen5-data.yaml
    wait_for_restore restore-zen5-data
    success "Zen data restored successfully"
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

function apply_lsr_workaround() {
    local namespace=$1
    info "License Service Reporter data restored failed, attempting workaround..."
    ${OC} delete deployment lsr-backup -n $namespace
    sed -i -E "s/<lsr namespace>/$namespace/" ${BASE_DIR}/templates/restore/lsr/lsr-data-restore-job.yaml
    info "Creating workaround job lsr-restore-job"
    ${OC} apply -f ${BASE_DIR}/templates/restore/lsr/lsr-data-restore-job.yaml
    wait_for_job_complete lsr-restore-job $namespace
}

function wait_for_job_complete() {
  local job_name=$1
  local namespace=$2
  local condition="${OC} get pod -n $namespace --no-headers --ignore-not-found | grep ${job_name} | grep 'Completed' || true"
  local retries=50
  local sleep_time=30
  local total_time_mins=$(( sleep_time * retries / 60))
  local wait_message="Waiting for job pod $job_name to complete"
  local success_message="Job $job_name completed in namespace $namespace"
  local error_message="Timeout after ${total_time_mins} minutes waiting for job $job_name"
  wait_for_condition "${condition}" ${retries} ${sleep_time} "${wait_message}" "${success_message}" "${error_message}"
  info "For more details on ${job_name}, check its pod logs."
}

function wait_for_cert_manager() {
    local cm_namespace=$1
    local name="cert-manager-webhook"
    local test_namespace=$2
    local needReplicas=$(${OC} get deployment ${name} -n ${cm_namespace} --no-headers --ignore-not-found -o jsonpath='{.spec.replicas}' | awk '{print $1}')
    local readyReplicas="${OC} get deployment ${name} -n ${cm_namespace} --no-headers --ignore-not-found -o jsonpath='{.status.readyReplicas}' | grep '${needReplicas}'"
    local replicas="${OC} get deployment ${name} -n ${cm_namespace} --no-headers --ignore-not-found -o jsonpath='{.status.replicas}' | grep '${needReplicas}'"
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

function write_specific_env_vars_to_file() {
    local output_file="$1"
    local vars=$2
    
    info "Writing specific environment variables to '$output_file'..."
    
    if ! touch "$output_file" 2>/dev/null; then
        error "Error: Cannot write to file '$output_file'"
    fi
    
    # Write each specified variable
    local count=0
    for var_name in $vars; do
        local var_value=$(printenv "$var_name")
        echo "$var_name=${var_value}" >> "$output_file"
        ((count++))
    done
    
    success "Successfully wrote $count environment variables to '$output_file'"
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
