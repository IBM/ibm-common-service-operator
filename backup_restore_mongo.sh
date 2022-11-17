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

OC=${3:-oc}
YQ=${3:-yq}
CS_NAMESPACE=$1
TARGET_NAMESPACE=$2
function main() {
    msg "MongoDB Backup and Restore v1.0.0"
    prereq
    prep_backup
    backup
    prep_restore
    restore
}

# verify that all pre-requisite CLI tools exist and parameters set
function prereq() {
    which "${OC}" || error "Missing oc CLI"
    which "${YQ}" || error "Missing yq"
    if [[ -z $CS_NAMESPACE ]]; then
        export CS_NAMESPACE=ibm-common-services
    fi
    if [[ -z $TARGET_NAMESPACE ]]; then
        error "TARGET_NAMESPACE not specified, please specify target namespace parameter and trty again."
    else
        ${OC} create namespace $TARGET_NAMESPACE || info "Target namespace ${TARGET_NAMESPACE} already exists. Moving on..."
    fi
}

function prep_backup() {
    title " Preparing for Mongo backup in namespace $CS_NAMESPACE "
    msg "-----------------------------------------------------------------------"
    
    local pvx=$(${OC} get pv | grep mongodbdir | awk 'FNR==1 {print $1}')
    local storageClassName=$("${OC}" get pv -o yaml ${pvx} | yq '.spec.storageClassName' | awk '{print}')
    
    ${OC} get sc -o yaml ${storageClassName} > sc.yaml
    ${YQ} -i '.metadata.name="backup-sc" | .reclaimPolicy = "Retain"' sc.yaml || error "Error changing the name or retentionPolicy for StorageClass"
    
    info "Creating Storage Class for backup"
    #TODO check if sc already exists in case customer has to run more than once, otherwise will fail
    ${OC} apply -f sc.yaml || error "Error creating StorageClass backup-sc"
    
    info "Creating RBAC for backup"
    cat <<EOF | tee >(oc apply -f -) | cat
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: cs-br
subjects:
- kind: ServiceAccount
  name: default
  namespace: $CS_NAMESPACE
roleRef:
  kind: ClusterRole
  name: cluster-admin
  apiGroup: rbac.authorization.k8s.io
EOF
    success "Backup prep complete"
}

function backup() {
    title " Backing up MongoDB in namespace $CS_NAMESPACE "
    msg "-----------------------------------------------------------------------"

    wget https://raw.githubusercontent.com/IBM/ibm-common-service-operator/scripts/velero/backup/mongoDB/mongodbbackup.yaml
    wget https://raw.githubusercontent.com/IBM/ibm-common-service-operator/scripts/velero/backup/mongoDB/mongo-backup.sh
    chmod +x mongo-backup.sh
    ./mongo-backup.sh true

    info "Verify cs-mongodump PVC exists..."
    local return_value=$("${OC}" get pvc -n $CS_NAMESPACE | grep cs-mongodump || echo failed)
    if [[ $return_value == "failed" ]]; then
        error "Backup PVC cs-mongodump not found"
    else
        return_value="reset"
        info "Backup PVC cs-mongodump found"
        return_value=$("${OC}" get pvc cs-mongodump -n $CS_NAMESPACE -o yaml | yq '.spec.storageClassName' | awk '{print}')
        if [[ "$return_value" != "backup-sc" ]]; then
            error "Backup PVC cs-mongodump not bound to persistent volume provisioned by correct storage class. Provisioned by \"${return_value}\" instead of \"backup-sc\""
        else
            info "Backup PVC cs-mongodump successfully bound to persistent volume provisioned by backup-sc storrage class."
        fi
    fi

    success "MongoDB successfully backed up"
}

function prep_restore() {
    title " Pepare for restore in namespace $TARGET_NAMESPACE "
    msg "-----------------------------------------------------------------------"
    ${OC} get pvc -n ${CS_NAMESPACE} cs-mongodump -o yaml > cs-mongodump-copy.yaml
    local pvx=$(${OC} get pv | grep cs-mongodump | awk '{print $1}')
    export PVX=${pvx}
    ${OC} delete job mongodb-backup -n ${CS_NAMESPACE}
    ${OC} patch pvc -n ${CS_NAMESPACE} cs-mongodump --type=merge -p '{"metadata": {"finalizers":null}}'
    ${OC} delete pvc -n ${CS_NAMESPACE} cs-mongodump
    ${OC} patch pv -n ${CS_NAMESPACE} ${pvx} --type=merge -p '{"spec": {"claimRef":null}}'
    
    #Check if the backup PV has come available yet
    #need to error handle, if a pv/pvc from a previous attempt exists in any ns it will mess this up
    #if cs-mongdump pvc already exists in the target namespace, it will break
    #Not sure if these checks are something to incorporate into the script or include in a troubleshooting section of the doc
    #On a fresh run where you don't have to worry about any existing pv or pvc, it works perfectly
    local pvStatus=$("${OC}" get pv -o yaml ${pvx}| yq '.status.phase' | awk '{print}')
    local retries=6
    echo "PVX: ${pvx} PV status: ${pvStatus}"
    while [ $retries != 0 ]
    do
        if [[ "${pvStatus}" != "Available" ]]; then
            retries=$(( $retries - 1 ))
            info "Persitent Volume ${pvx} not available yet. Retries left: ${retries}. Waiting 30 seconds..."
            sleep 30s
            pvStatus=$("${OC}" get pv -o yaml ${pvx}| yq '.status.phase' | awk '{print}')
            echo "PVX: ${pvx} PV status: ${pvStatus}"
        else
            info "Persitent Volume ${pvx} available. Moving on..."
            break
        fi
    done

    #edit the cs-mongodump-copy.yaml pvc file and apply it in the target namespace
    export TARGET_NAMESPACE=$TARGET_NAMESPACE
    ${YQ} -i '.metadata.namespace=strenv(TARGET_NAMESPACE)' cs-mongodump-copy.yaml
    ${OC} apply -f cs-mongodump-copy.yaml
    
    #Check PV status to make sure it binds to the right PVC
    pvStatus=$("${OC}" get pv -o yaml ${pvx}| yq '.status.phase' | awk '{print}')
    retries=6
    while [ $retries != 0 ]
    do
        if [[ "${pvStatus}" != "Bound" ]]; then
            retries=$(( $retries - 1 ))
            info "Persitent Volume ${pvx} not bound yet. Retries left: ${retries}. Waiting 30 seconds..."
            sleep 30s
            pvStatus=$("${OC}" get pv -o yaml ${pvx}| yq '.status.phase' | awk '{print}')
        else
            info "Persitent Volume ${pvx} bound. Checking PVC..."
            boundPV=$("${OC}" get pvc cs-mongodump -n ${TARGET_NAMESPACE} -o yaml | yq '.spec.volumeName' | awk '{print}')
            if [[ "${boundPV}" != "${pvx}" ]]; then
                error "Error binding cs-mongodump PVC to backup PV ${pvx}. Bound to ${boundPV} instead."
            else
                info "PVC cs-mongodump successfully bound to backup PV ${pvx}"
                break
            fi
        fi
    done

    success "Preparation for Restore completed successfully."
    
}

function restore () {
    title " Restore copy of backup in namespace $TARGET_NAMESPACE "
    msg "-----------------------------------------------------------------------"
    #change csnamespace to reflect the new target namespace
    #restore script is setup to look for CS_NAMESPACE and is used elsewhere
    export CS_NAMESPACE=$TARGET_NAMESPACE
    wget https://raw.githubusercontent.com/IBM/ibm-common-service-operator/scripts/velero/restore/mongoDB/mongodbrestore.yaml
    wget https://raw.githubusercontent.com/IBM/ibm-common-service-operator/scripts/velero/restore/mongoDB/set_access.js
    wget https://raw.githubusercontent.com/IBM/ibm-common-service-operator/scripts/velero/restore/mongoDB/mongo-restore.sh
    chmod +x mongo-restore.sh
    ./mongo-restore.sh

    success "Restore completed successfully in namespace $TARGET_NAMESPACE"

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