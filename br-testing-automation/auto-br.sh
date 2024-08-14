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
TARGET_CLUSTER=""
TARGET_CLUSTER_TYPE=""
BACKUP_NAME=""
RESTORE_NAME=""
APPLICATION="cs-application"
BACKUP_POLICY="cs-backup-policy"
BACKUP_STORAGE_LOCATION_NAME="luztest"
ROUTE=""
SF_NAMESPACE="ibm-spectrum-fusion-ns"
#hub oc token
#hub server
#spoke oc token
#spoke server
OC="oc"
YQ="yq"

BASE_DIR=$(cd $(dirname "$0")/$(dirname "$(readlink $0)") && pwd -P)
. ../cp3pt0-deployment/common/utils.sh
source ${BASE_DIR}/env.properties

#parse arguments
#prereq

#trigger backup
#wait for backup to complete
#verify backup (?)

#check restore cluster
#verify data not present/cleanup (?)
#trigger restore
#wait for restore to complete (will take a while)
#verify restore (?)

function main(){
    parse_arguments "$@"
    prereq
    ROUTE=$(${OC} get route -n $SF_NAMESPACE --no-headers | awk '{print $2}')
    if [[ $BACKUP == "true" ]]; then
        create_br "backup" $TARGET_CLUSTER_TYPE
        wait_for_br "backup" $BACKUP_NAME 30
        success "Backup $BACKUP_NAME of cluster $TARGET_CLUSTER completed. See results in Fusion UI here: https://$ROUTE/backupAndRestore/jobs/backups/$BACKUP_NAME"
    fi
    if [[ $RESTORE == "true" ]]; then
        create_br "restore" $TARGET_CLUSTER_TYPE
        wait_for_br "restore" $RESTORE_NAME 90
        success "Restore $RESTORE_NAME to cluster $TARGET_CLUSTER completed. See results in Fusion UI here: https://$ROUTE/backupAndRestore/jobs/restores/$RESTORE_NAME"
    fi

}

function print_usage(){
    script_name=`basename ${0}`
    echo "Usage: ${script_name} [OPTIONS]"
    echo ""
    echo "Set up either a hub cluster or a spoke cluster to use Spectrum Fusion Backup and Restore."
    echo "One of --hub-setup or --spoke-setup is required."
    echo "This script assumes the following:"
    echo "    * An existing CPFS instance on the hub cluster with IM, Zen, Licensing, Cert Manager, and License Service Reporter present"
    echo "    * Filled in required variables in the accompanying env.properties file"
    echo ""
    echo "Options:"
    echo "   --oc string                    Optional. File path to oc CLI. Default uses oc in your PATH. Can also be set in env.properties."
    echo "   --yq string                    Optional. File path to yq CLI. Default uses yq in your PATH. Can also be set in env.properties."
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
        --restore-name)
            shift
            RESTORE_NAME=$1
            ;;
        --sf-namespace)
            shift
            SF_NAMESPACE=$1
            ;;
        --target-cluster)
            shift
            TARGET_CLUSTER=$1
            ;;
        --cluster-type)
            shift
            TARGET_CLUSTER_TYPE=$1
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
    #check that oc yq and skopeo are available
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

    #check docker access (so far not necessary)

    #check variables are present
}

function create_br() {
    brtype=$1
    clustertype=$2
    title "Creating Spectrum Fusion $brtype resource for $clustertype cluster."

    if [ -d "templates" ]; then
        rm -rf templates
    fi

    mkdir templates
    if [[ $brtype == "backup" ]]; then
        info "Copying template files..."
        cp ../velero/spectrum-fusion/templates/sf-backup.yaml ./templates/sf-backup.yaml
        
        info "Editing backup yaml..."
        sed -i -E "s/<backup storage location name>/$BACKUP_STORAGE_LOCATION_NAME/" ./templates/sf-backup.yaml
        sed -i -E "s/<application name>/$APPLICATION/" ./templates/sf-backup.yaml
        sed -i -E "s/<backup policy name>/$BACKUP_POLICY/" ./templates/sf-backup.yaml
        sed -i -E "s/<backup name>/$BACKUP_NAME/" ./templates/sf-backup.yaml
        if [[ $clustertype == "spoke" ]]; then
            sed -i -E "s/<Cluster CR name only for backups on spoke cluster>/$TARGET_CLUSTER/" ./templates/sf-backup.yaml
        else
            ${YQ} -i 'del(.spec.appCluster)' ./templates/sf-backup.yaml || error "Could not remove appCluster field from backup yaml."
        fi
        if [[ $SF_NAMESPACE != "ibm-spectrum-fusion-ns" ]]; then
            ${YQ} -i '.metadata.namesace = "'${SF_NAMESPACE}'"' ./templates/sf-backup.yaml || error "Could not update namespace value to $SF_NAMESPACE in backup yaml."
        fi

        ${OC} apply -f ./templates/sf-backup.yaml || error "Failed to apply backup yaml."
        success "Backup successfully applied on hub server $HUB_SERVER to backup target cluster $TARGET_CLUSTER"
        
    fi
    
    if [[ $brtype == "restore" ]]; then
        cp ../velero/spectrum-fusion/templates/sf-restore.yaml ./templates/sf-restore.yaml

        info "Editing restore yaml..."
        sed -i -E "s/<backup storage location name>/$BACKUP_STORAGE_LOCATION_NAME/" ./templates/sf-restore.yaml
        sed -i -E "s/<application name>/$APPLICATION/" ./templates/sf-restore.yaml
        sed -i -E "s/<backup policy name>/$BACKUP_POLICY/" ./templates/sf-restore.yaml
        sed -i -E "s/<restore name>/$RESTORE_NAME/" ./templates/sf-restore.yaml

        if [[ $clustertype == "spoke" ]]; then
            sed -i -E "s/<Cluster CR name only for restores to spoke cluster>/$TARGET_CLUSTER/" ./templates/sf-restore.yaml
        else
            ${YQ} -i 'del(.spec.targetCluster)' ./templates/sf-restore.yaml || error "Could not remove targetCluster field from restore yaml."
        fi
        if [[ $SF_NAMESPACE != "ibm-spectrum-fusion-ns" ]]; then
            ${YQ} -i '.metadata.namesace = "'${SF_NAMESPACE}'"' ./templates/sf-restore.yaml || error "Could not update namespace value to $SF_NAMESPACE in restore yaml."
        fi
    fi
}

function wait_for_br(){
    type=$1
    resource_name=$2
    retries=$3
    time=30
    title "Waiting for $type $resource_name to complete..."
    status=$(${OC} get $type $resource_name -n $SF_NAMESPACE -o jsonpath='{.status.phase}')
    echo "$status && ${status} && ${OC} get $type $resource_name -n $SF_NAMESPACE -o jsonpath='{.status.phase}'"
    info "$type $resource_name can be further tracked in the UI here: https://$ROUTE/backupAndRestore/jobs/${type}s/$resource_name"
    while [[ $status != "Completed" ]] && [[ $retries > 0 ]]; do
        status=$(eval $status)
        info "Waiting on $type $resource_name to complete. Current status: $status"
        if [[ $((retries%10)) == 0 ]]; then
            info "Current sequence status:"
            ${OC} get $type $resource_name -n $SF_NAMESPACE -o yaml | ${YQ} '.status.summary.sequence'
        fi
        checkFail=$(echo $status | grep "Failed")
        if [[ $checkFail != "" ]] || [[ $status == "Redundant" ]]; then
            error "$type failed with error: $status. \nFor more info, see job in the UI (https://$ROUTE/backupAndRestore/jobs/${type}s/$resource_name) or use \"oc get $type $resource_name -n $SF_NAMESPACE -o yaml | yq '.status'\"."
        fi
        sleep $time
        retries=$((retries-1))
    done

    if [[ $status == "Completed" ]]; then
        success "$type $resource_name completed successfully for $TARGET_CLUSTER."
        info "For more info, see job in the UI (https://$ROUTE/backupAndRestore/jobs/${type}s/$resource_name) or use \"oc get $type $resource_name -n $SF_NAMESPACE -o yaml | yq '.status'\"."
    elif [[ $status != "Completed" ]] && [[ $retries == 0 ]]; then
        error "Timed out waiting for $type $resource_name for $TARGET_CLUSTER. \nFor more info, see job in the UI (https://$ROUTE/backupAndRestore/jobs/${type}s/$resource_name) or use \"oc get $type $resource_name -n $SF_NAMESPACE -o yaml | yq '.status'\"."
    fi
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