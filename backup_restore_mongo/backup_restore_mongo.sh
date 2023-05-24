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

# set -o errexit
set -o pipefail
set -o errtrace
set -o nounset

OC=oc
YQ=yq
ORIGINAL_NAMESPACE=
TARGET_NAMESPACE=
backup="false"
restore="false"
cleanup="false"

function main() {
    while [ "$#" -gt "0" ]
    do
        case "$1" in
        "-h"|"--help")
            usage
            exit 0
            ;;
        "--bns")
            ORIGINAL_NAMESPACE=$2
            shift
            ;;
        "--rns")
            TARGET_NAMESPACE=$2
            shift
            ;;
        "-b")
            backup="true"
            ;;
        "-r")
            restore="true"
            ;;
        "-c")
            cleanup="true"
            ;;
        *)
            error "invalid option -- \`$1\`. Use the -h or --help option for usage info."
            ;;
        esac
        shift
    done

    msg "MongoDB Backup and Restore v1.0.0"
    prereq
    
    if [[ $backup == "true" ]]; then
        prep_backup
        backup
    fi
    if [[ $restore == "true" ]]; then
        prep_restore
        restore
        check_ldap_secret
        refresh_auth_idp
    fi
    if [[ $cleanup == "true" ]]; then
        cleanup
    fi
}

function usage() {
	local script="${0##*/}"

	while read -r ; do echo "${REPLY}" ; done <<-EOF
	Usage: ${script} [OPTION]...
	Uninstall common services
	Options:
	Mandatory arguments to long options are mandatory for short options too.
	  -h, --help                    display this help and exit
      --bns                          specify the namespace to backup/where the backup exists
      --rns                          specify the namespace where data is to be restored
      -b                            run the backup process
      -r                            run the restore process
      -c                            cleanup resources used or created by this script
	EOF
}

# verify that all pre-requisite CLI tools exist and parameters set
function prereq() {
    which "${OC}" || error "Missing oc CLI"
    which "${YQ}" || error "Missing yq"

    if [[ -z $ORIGINAL_NAMESPACE ]] && [[ -z $TARGET_NAMESPACE ]]; then
        error "Neither backup nor restore namespaces were set. Use -h or --help to see script usage options"
    elif [[ -z $ORIGINAL_NAMESPACE ]] && [[ $cleanup == "false" ]]; then
        if [[ $backup == "true" || $restore == "true" ]]; then
            error "Backup namespace not specified. Please specify backup namespace with --bns. Use -h or --help for script usage"
        fi
    fi
    
    if [[ $backup == "false" ]] && [[ $restore == "false" ]] && [[ $cleanup == "false" ]]; then
        error "Neither backup nor restore processes were triggered. Use -h or --help to see script usage options"
    fi

    success "Prerequisites present."
}

function prep_backup() {
    title " Preparing for Mongo backup in namespace $ORIGINAL_NAMESPACE "
    msg "-----------------------------------------------------------------------"
    
    #check if files are already present on machine before trying to download (airgap)
    #TODO add clarifying messages and check response code to make more transparent
    #backup files
    info "Checking for necessary backup files..."
    if [[ -f "mongodbbackup.yaml" ]]; then
        info "mongodbbackup.yaml already present"
    else
        info "mongodbbackup.yaml not found, downloading from https://raw.githubusercontent.com/IBM/ibm-common-service-operator/scripts/backup_restore_mongo/mongodbbackup.yaml"
        wget -O mongodbbackup.yaml https://raw.githubusercontent.com/IBM/ibm-common-service-operator/scripts/backup_restore_mongo/mongodbbackup.yaml || error "Failed to download mongodbbackup.yaml"
    fi

    if [[ -f "mongo-backup.sh" ]]; then
        info "mongo-backup.sh already present"
    else
        info "mongodbbackup.yaml not found, downloading from https://raw.githubusercontent.com/IBM/ibm-common-service-operator/scripts/backup_restore_mongo/mongo-backup.sh"
        wget -O mongo-backup.sh https://raw.githubusercontent.com/IBM/ibm-common-service-operator/scripts/backup_restore_mongo/mongo-backup.sh
    fi

    success "Backup prep complete"
}

function backup() {
    title " Backing up MongoDB in namespace $ORIGINAL_NAMESPACE "
    msg "-----------------------------------------------------------------------"
    export CS_NAMESPACE=$ORIGINAL_NAMESPACE
    export ibm_mongodb_image=$(${OC} get pod icp-mongodb-0 -n $ORIGINAL_NAMESPACE -o=jsonpath='{range .spec.containers[0]}{.image}{end}')
    local pvx=$(${OC} get pv | grep mongodbdir | awk 'FNR==1 {print $1}')
    local storageClassName=$("${OC}" get pv -o yaml ${pvx} | yq '.spec.storageClassName' | awk '{print}')
    chmod +x mongo-backup.sh
    ./mongo-backup.sh "$storageClassName"

    local jobPod=$(${OC} get pods -n $ORIGINAL_NAMESPACE | grep mongodb-backup | awk '{ print $1 }')
    local fileName="backup_from_${ORIGINAL_NAMESPACE}_for_${TARGET_NAMESPACE}.log"
    ${OC} logs $jobPod -n $ORIGINAL_NAMESPACE > $fileName
    info "Backup logs can be found in $fileName. Job pod will be cleaned up."

    info "Verify cs-mongodump PVC exists..."
    local return_value=$("${OC}" get pvc -n $ORIGINAL_NAMESPACE | grep cs-mongodump || echo failed)
    if [[ $return_value == "failed" ]]; then
        error "Backup PVC cs-mongodump not found"
    else
        return_value="reset"
        info "Backup PVC cs-mongodump found"
        
        VOL=$(${OC} get pvc cs-mongodump -n $ORIGINAL_NAMESPACE  -o=jsonpath='{.spec.volumeName}')
        ${OC} patch pv $VOL -p '{"spec": { "persistentVolumeReclaimPolicy" : "Retain" }}'
        
        return_value=$(${OC} get pvc cs-mongodump -n $ORIGINAL_NAMESPACE -o yaml | yq '.spec.storageClassName' | awk '{print}')
        if [[ "$return_value" != "$storageClassName" ]]; then
            error "Backup PVC cs-mongodump not bound to persistent volume provisioned by correct storage class. Provisioned by \"${return_value}\" instead of \"$storageClassName\""
            #TODO probably need to handle this situation as the script may not be able to handle it as is
            #should be an edge case though as script is designed to attach to specific pv
        else
            info "Backup PVC cs-mongodump successfully bound to persistent volume provisioned by $storageClassName storage class."
        fi
    fi

    success "MongoDB successfully backed up"
}

function prep_restore() {
    title " Pepare for restore in namespace $TARGET_NAMESPACE "
    msg "-----------------------------------------------------------------------"
    
    #Restore files
    info "Checking for necessary restore files..."
    if [[ -f "mongodbrestore.yaml" ]]; then
        info "mongodbrestore.yaml already present"
    else
        info "mongodbrestore.yaml not found, downloading from https://raw.githubusercontent.com/IBM/ibm-common-service-operator/scripts/backup_restore_mongo/mongodbrestore.yaml"
        wget -O mongodbrestore.yaml https://raw.githubusercontent.com/IBM/ibm-common-service-operator/scripts/backup_restore_mongo/mongodbrestore.yaml || error "Failed to download mongodbrestore.yaml"
    fi

    if [[ -f "set_access.js" ]]; then
        info "set_access.js already present"
    else
        info "set_access.js not found, downloading from https://raw.githubusercontent.com/IBM/ibm-common-service-operator/scripts/backup_restore_mongo/set_access.js"
        wget -O set_access.js https://raw.githubusercontent.com/IBM/ibm-common-service-operator/scripts/backup_restore_mongo/set_access.js || error "Failed to download set_access.js"
    fi

    if [[ -f "mongo-restore.sh" ]]; then
        info "mongo-restore.sh already present"
    else
        info "mongo-restore.sh not found, downloading from https://raw.githubusercontent.com/IBM/ibm-common-service-operator/scripts/backup_restore_mongo/mongo-restore.sh"
        wget -O mongo-restore.sh https://raw.githubusercontent.com/IBM/ibm-common-service-operator/scripts/backup_restore_mongo/mongo-restore.sh || error "Failed to download mongo-restore.sh"
    fi
    
    ${OC} get pvc -n ${ORIGINAL_NAMESPACE} cs-mongodump -o yaml > cs-mongodump-copy.yaml
    local pvx=$(${OC} get pv | grep cs-mongodump | awk '{print $1}')
    export PVX=${pvx}
    ${OC} delete job mongodb-backup -n ${ORIGINAL_NAMESPACE}
    ${OC} delete pvc cs-mongodump -n ${ORIGINAL_NAMESPACE} --ignore-not-found --timeout=10s
    if [ $? -ne 0 ]; then
        info "Failed to delete pvc cs-mongodump, patching its finalizer to null..."
        ${OC} patch pvc cs-mongodump -n ${ORIGINAL_NAMESPACE} --type="json" -p '[{"op": "remove", "path":"/metadata/finalizers"}]'
    fi
    ${OC} patch pv -n ${ORIGINAL_NAMESPACE} ${pvx} --type=merge -p '{"spec": {"claimRef":null}}'
    
    #Check if the backup PV has come available yet
    #need to error handle, if a pv/pvc from a previous attempt exists in any ns it will mess this up
    #if cs-mongdump pvc already exists in the target namespace, it will break
    #Not sure if these checks are something to incorporate into the script or include in a troubleshooting section of the doc
    #On a fresh run where you don't have to worry about any existing pv or pvc, it works perfectly
    #New cleanup function running before and after completion should solve this problem
    local pvStatus=$("${OC}" get pv -o yaml ${pvx}| yq '.status.phase' | awk '{print}')
    local retries=6
    echo "PVX: ${pvx} PV status: ${pvStatus}"
    while [ $retries != 0 ]
    do
        if [[ "${pvStatus}" != "Available" ]]; then
            retries=$(( $retries - 1 ))
            info "Persistent Volume ${pvx} not available yet. Retries left: ${retries}. Waiting 30 seconds..."
            sleep 30s
            pvStatus=$("${OC}" get pv -o yaml ${pvx}| yq '.status.phase' | awk '{print}')
            echo "PVX: ${pvx} PV status: ${pvStatus}"
        else
            info "Persistent Volume ${pvx} available. Moving on..."
            break
        fi
    done

    #edit the cs-mongodump-copy.yaml pvc file and apply it in the target namespace
    export TARGET_NAMESPACE=$TARGET_NAMESPACE
    ${YQ} -i '.metadata.namespace=strenv(TARGET_NAMESPACE)' cs-mongodump-copy.yaml
    ${OC} apply -f cs-mongodump-copy.yaml
    
    #Check PV status to make sure it binds to the right PVC
    #If more than one pv provisioned by the sc created in this script exists, this part will break as it lists all of the pvs provisioned by backup-sc as $PVX
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
    #export csnamespace to reflect the new target namespace
    #restore script is setup to look for CS_NAMESPACE and is used in other backup/restore processes unrelated to this script
    export CS_NAMESPACE=$TARGET_NAMESPACE
    export ibm_mongodb_image=$(${OC} get pod icp-mongodb-0 -n $ORIGINAL_NAMESPACE -o=jsonpath='{range .spec.containers[0]}{.image}{end}')

    chmod +x mongo-restore.sh
    ./mongo-restore.sh "$ORIGINAL_NAMESPACE"

    local jobPod=$(${OC} get pods -n $TARGET_NAMESPACE | grep mongodb-restore | awk '{ print $1 }')
    local fileName="restore_to_${TARGET_NAMESPACE}_from_${ORIGINAL_NAMESPACE}.log"
    ${OC} logs $jobPod -n $TARGET_NAMESPACE > $fileName
    info "Restore logs can be found in $fileName. Job pod will be cleaned up."

    success "Restore completed successfully in namespace $TARGET_NAMESPACE"

}

function cleanup(){
    title " Cleaning up resources created during backup restore process "
    msg "-----------------------------------------------------------------------"
    
    if [[ $ORIGINAL_NAMESPACE != "" ]]; then
        info "Deleting resources used in backup process from namespace $ORIGINAL_NAMESPACE"
        
        #clean up backup resources
        local return_value=$("${OC}" get pvc -n $ORIGINAL_NAMESPACE | grep cs-mongodump || echo failed)
        if [[ $return_value != "failed" ]]; then
        #delete backup items in original namespace
            ${OC} delete job mongodb-backup -n ${ORIGINAL_NAMESPACE} || info "Backup job already deleted. Moving on..."
            ${OC} delete pvc cs-mongodump -n $ORIGINAL_NAMESPACE --ignore-not-found --timeout=10s
            if [ $? -ne 0 ]; then
                info "Failed to delete pvc cs-mongodump, patching its finalizer to null..."
                ${OC} patch pvc cs-mongodump -n $ORIGINAL_NAMESPACE --type="json" -p '[{"op": "remove", "path":"/metadata/finalizers"}]'
            fi
        else
            info "Resources used in backup already cleaned up. Moving on..."
        fi

        local rbac=$(${OC} get clusterrolebinding cs-br -n $ORIGINAL_NAMESPACE || echo failed)
        if [[ $rbac != "failed" ]]; then
            info "Deleting RBAC from backup restore process"
            ${OC} delete clusterrolebinding cs-br -n $ORIGINAL_NAMESPACE
        fi

        local scExist=$(${OC} get sc backup-sc -n $ORIGINAL_NAMESPACE || echo failed)
        if [[ $scExist != "failed" ]]; then
            info "Deleting storage class used in backup restore process"
            ${OC} delete sc backup-sc
        fi
    fi

    if [[ $TARGET_NAMESPACE != "" ]]; then
        info "Deleting resources used in restore process from namespace $TARGET_NAMESPACE"
        #clean up restore resources
        local return_value=$("${OC}" get pvc -n $TARGET_NAMESPACE | grep cs-mongodump || echo failed)
        if [[ $return_value != "failed" ]]; then
        #delete retore items in target namespace
            local boundPV=$(${OC} get pvc cs-mongodump -n $TARGET_NAMESPACE -o yaml | yq '.spec.volumeName' | awk '{print}')
            ${OC} delete job mongodb-restore -n ${TARGET_NAMESPACE} || info "Restore job already deleted. Moving on..."
            ${OC} delete pvc cs-mongodump -n $TARGET_NAMESPACE --ignore-not-found --timeout=10s
            if [ $? -ne 0 ]; then
                info "Failed to delete pvc cs-mongodump, patching its finalizer to null..."
                ${OC} patch pvc cs-mongodump -n $TARGET_NAMESPACE --type="json" -p '[{"op": "remove", "path":"/metadata/finalizers"}]'
            fi
            ${OC} patch pv $boundPV --type=merge -p '{"metadata": {"finalizers":null}}'
            ${OC} delete pv $boundPV
        else
            info "Resources used in restore already cleaned up. Moving on..."
        fi
    fi

    success "Cleanup complete."

}

function refresh_auth_idp(){
    title " Restarting auth-idp pod in namespace $TARGET_NAMESPACE "
    msg "-----------------------------------------------------------------------"
    local auth_pod=$(${OC} get pods -n $TARGET_NAMESPACE | grep auth-idp | awk '{print $1}')
    ${OC} delete pod $auth_pod -n $TARGET_NAMESPACE || warning "Pod $auth_pod could not be deleted, try deleting manually"
    success "Pod $auth_pod deleted. Please allow a few minutes for it to restart."
}

function check_ldap_secret() {
    exists=$(${OC} get secret -n $TARGET_NAMESPACE | (grep platform-auth-ldaps-ca-cert || echo fail))
    if [[ $exists != "fail" ]]; then
        certificate=$(${OC} get secret -n $TARGET_NAMESPACE platform-auth-ldaps-ca-cert -o yaml | yq '.data.certificate' )
        og_certificate=$(${OC} get secret -n $ORIGINAL_NAMESPACE platform-auth-ldaps-ca-cert -o yaml | yq '.data.certificate' )
        if [[ $certificate == "" ]] || [[ $certificate != $og_certificate ]]; then
            ${OC} patch secret -n $TARGET_NAMESPACE platform-auth-ldaps-ca-cert --type=merge -p '{"data": {"certificate":'$og_certificate'}}'
            info "Secret platform-auth-ldaps-ca-cert in $TARGET_NAMESPACE patched to match secret in $ORIGINAL_NAMESPACE"
        else
            info "Secret platform-auth-ldaps-ca-cert already populated. Moving on..."
        fi
    fi
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