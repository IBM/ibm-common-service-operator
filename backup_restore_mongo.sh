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
#TARGET_NAMESPACE=$2
function main() {
    msg "MongoDB Backup and Restore v1.0.0"
    prereq
    prep_backup
    backup
}

# verify that all pre-requisite CLI tools exist and parameters set
function prereq() {
    which "${OC}" || error "Missing oc CLI"
    which "${YQ}" || error "Missing yq"
    if [[ -z $CS_NAMESPACE ]]; then
        export CS_NAMESPACE=ibm-common-services
    fi
    # if [[ -z $TARGET_NAMESPACE ]]; then
    #     error "TARGET_NAMESPACE not specified, please specify target namespace parameter and trty again."
    # else
    #     ${OC} create namespace $ns || info "Target namespace $ns already exists. Moving on..."
    # fi
}

function prep_backup() {
    title " Preparing for Mongo backup "
    msg "-----------------------------------------------------------------------"
    pvx=$(${OC} get pv | grep mongodbdir | awk 'FNR==1 {print $1}')
    storageClassName=$("${OC}" get pv -o yaml ${pvx} | yq '.spec.storageClassName' | awk '{print}')
    
    ${OC} get sc -o yaml ${storageClassName} > sc.yaml
    ${YQ} -i '.metadata.name="backup-sc" | .reclaimPolicy = "Retain"' sc.yaml || error "Error changing the name or retentionPolicy for StorageClass"
    
    info "Creating Storage Class for backup"
    #TODO check if sc already exists in case customer has to run more than once
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
    title " Backing up MongoDB "
    msg "-----------------------------------------------------------------------"

    wget https://raw.githubusercontent.com/IBM/ibm-common-service-operator/scripts/velero/backup/mongodb/mongodbbackup.yaml
    wget https://raw.githubusercontent.com/IBM/ibm-common-service-operator/scripts/velero/backup/mongodb/mongo-backup.sh
    chmod +x mongo-backup.sh
    ./mongo-backup.sh

    info "Verify cs-mongodump PVC exists..."
    local return_value=$("${OC}" get pvc -n $CS_NAMESPACE | grep cs-mongodump || echo failed)
    if [[ $return_value == "failed" ]]; then
        error "Backup PVC cs-mongodump not found"
    else
        return_value="reset"
        info "Backup PVC cs-mongodump found"
        return_value=$("${OC}" get pvc cs-mongodump -n $CS_NAMESPACE -o yaml | yq '.spec.storageClassName' | awk '{print}')
        if [[ return_value != "backup-sc" ]]; then
            error "Backup PVC cs-mongodump not bound to persistent volume provisioned by correct storage class. Provisioned by \"${return_value}\" instead of \"backup-sc\""
        else
            info "Backup PVC cs-mongodump successfully bound to persistent volume provisioned by backup-sc storrage class."
        fi
    fi

    success "MongoDB successfully backed up"
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