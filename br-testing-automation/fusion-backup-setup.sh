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

#script outline
#validate storage class in use on cluster (no pool if rook ceph, otherwise ODF)
    #create volumesnapshotclass if rook ceph
#download install script form SF repo
#run script to install SF and BR service
#create necesary SF resources
#deploy BR resources
#label existing CS resources

set -o pipefail
set -o errtrace

CATALOG_SOURCE="ibm-operator-catalog-latest"
CAT_SRC_NS="openshift-marketplace"
# OC=
# YQ=
SF_NAMESPACE="ibm-spectrum-fusion-ns"
BR_SERVICE_NAMESPACE="ibm-backup-restore"
# OPERATOR_NAMESPACE=
# SERVICES_NAMESPACE=
# GITHUB_USER=
# GITHUB_TOKEN=
# DOCKER_USER=
# DOCKER_PASS=
STORAGE_CLASS="rook-cephfs"

function main(){
    # parse_arguments
    # prereq
    validate_sc
    install_sf_br
    # create_sf_resources
    # deploy_cs_br_resources
    # label_cs_resources
}

function prereq() {
    #check oc
    #check yq
    # Check yq version
    check_yq

    # Checking oc command logged in
    user=$(${OC} whoami 2> /dev/null)
    if [ $? -ne 0 ]; then
        error "You must be logged into the OpenShift Cluster from the oc command line"
    else
        success "oc command logged in as ${user}"
    fi

    #check docker access
}

function validate_sc(){
    #check sc
    if [[ $STORAGE_CLASS == "" ]]; then
        STORAGE_CLASS=$(${OC} get sc | grep "(default)" | awk '{print $1}') 
    fi
    #if rook ceph, verify no pools
    if [[ $STORAGE_CLASS == "rook-cephfs" ]]; then
        pool_exist=$(oc get sc $STORAGE_CLASS -o jsonpath='{.parameters.pool}')
        if [[ $pool_exist != "" ]]; then
            error "Spectrum Fusion BR will not work with Rook-CephFS if the pool parameter is enabled. See \`oc get sc $STORAGE_CLASS -o jsonpath=\'{.parameters.pool}\'\` for more details."
        else
            info "Pool parameter correctly not specified in rook-cephfs, continuing..."
        fi
    fi
    #if odf, that works

    #ensure volumesnapshotclass created
    vcs_exists=$(${OC} get volumesnapshotclass)
    if [[ $vcs_exists == "" ]]; then
        driver=$(${OC} get sc $STORAGE_CLASS -o jsonpath='{.provisioner}')
        clusterID=$(${OC} get sc $STORAGE_CLASS -o jsonpath='{.parameters.clusterID}')
        snapshotter_secret_name=$(${OC} get sc $STORAGE_CLASS -o jsonpath='{.parameters.csi.storage.k8s.io/provisioner-secret-name}')
        snapshotter_secret_namespace=$(${OC} get sc $STORAGE_CLASS -o jsonpath='{.parameters.csi.storage.k8s.io/provisioner-secret-namespace}')
        cat << EOF | ${OC} apply -f -
apiVersion: snapshot.storage.k8s.io/v1
deletionPolicy: Delete
driver: $driver
kind: VolumeSnapshotClass
metadata:
  name: $STORAGE_CLASS-snapclass
parameters:
  clusterID: $clusterID
  csi.storage.k8s.io/snapshotter-secret-name: $snapshotter_secret_name
  csi.storage.k8s.io/snapshotter-secret-namespace: $snapshotter_secret_namespace
EOF
    fi
}

function install_sf_br(){
    title "Installing Spectrum Fusion and its Backup and Restore service from catalog $CATALOG_SOURCE."
    info "Cloning SF cmd-line-install repo..."
    git clone https://$GITHUB_USER:$GITHUB_TOKEN@github.ibm.com/ProjectAbell/cmd-line-install.git
    
    #TODO verify catalog source pod is actually running
    catalog_image=$(${OC} get catalogsource -o jsonpath='{.spec.image}' $CATALOG_SOURCE -n $CAT_SRC_NS)

    cd cmd-line-install/install/

    info "executing install-isf-br.sh script with catalog image $catalog_image in namespace $SF_NAMESPACE."
    ./install-isf-br.sh $catalog_image -n $SF_NAMESPACE || error "SF install script failed."

    success "Spectrum Fusion and Backup and Restore Service installed."

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