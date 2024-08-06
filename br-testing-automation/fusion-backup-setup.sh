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

# CATALOG_SOURCE="ibm-operator-catalog-latest"
# CAT_SRC_NS="openshift-marketplace"
# # OC=
# # YQ=
# SF_NAMESPACE="ibm-spectrum-fusion-ns"
# BR_SERVICE_NAMESPACE="ibm-backup-restore"
# # OPERATOR_NS=
# # SERVICES_NS=
# # GITHUB_USER=
# # GITHUB_TOKEN=
# # DOCKER_USER=
# # DOCKER_PASS=
# STORAGE_CLASS="rook-cephfs"


BASE_DIR=$(cd $(dirname "$0")/$(dirname "$(readlink $0)") && pwd -P)
source ${BASE_DIR}/env.properties

function main(){
    # parse_arguments
    prereq
    echo $BASE_DIR
    validate_sc
    install_sf_br
    create_sf_resources
    deploy_cs_br_resources
    label_cs_resources
    success "Backup cluster prepped for BR."
}

function prereq() {
    #check oc
    #check yq
    #check skopeo
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
    cd $BASE_DIR

    success "Spectrum Fusion and Backup and Restore Service installed."

}

function create_sf_resources(){
    title "Creating Spectrum Fusion BR resources in namespace $SF_NAMESPACE."

    if [ -d "templates" ]; then
        rm -rf templates
    fi

    mkdir templates
    info "Copying template files..."
    cp ../velero/spectrum-fusion/application.yaml ./templates/application.yaml
    cp ../velero/spectrum-fusion/backup_storage_location_secret.yaml ./templates/backup_storage_location_secret.yaml
    cp ../velero/spectrum-fusion/backup_storage_location.yaml ./templates/backup_storage_location.yaml
    cp ../velero/spectrum-fusion/policy_assignment.yaml ./templates/policy_assignment.yaml
    cp ../velero/spectrum-fusion/policy.yaml ./templates/policy.yaml
    cp ../velero/spectrum-fusion/recipes/4.7-example-recipe-multi-ns.yaml ./templates/multi-ns-recipe.yaml
    
    info "Editing backup storage location resources..."
    #backup storage secret
    sed -i -E "s/<location name>/$BACKUP_STORAGE_LOCATION_NAME/" ./templates/backup_storage_location_secret.yaml
    sed -i -E "s/<spectrum fusion ns>/$SF_NAMESPACE/" ./templates/backup_storage_location_secret.yaml
    encoded_access_key=$(echo $STORAGE_SECRET_ACCESS_KEY | base64)
    sed -i -E "s/<base 64 encoded secret access key>/$encoded_access_key/" ./templates/backup_storage_location_secret.yaml
    encoded_access_key_id=$(echo $STORAGE_SECRET_ACCESS_KEY_ID | base64)
    sed -i -E "s/<base 64 encoded access key id>/$encoded_access_key_id/" ./templates/backup_storage_location_secret.yaml
    
    #backup storage location
    sed -i -E "s/<location name>/$BACKUP_STORAGE_LOCATION_NAME/" ./templates/backup_storage_location.yaml
    sed -i -E "s/<spectrum fusion ns>/$SF_NAMESPACE/" ./templates/backup_storage_location.yaml
    sed -i -E "s/<bucket name>/$STORAGE_BUCKET_NAME/" ./templates/backup_storage_location.yaml
    #s3 url is breaking the sed command somehow, must be something in how the url is entered, maybe need to escape characters in the env.properties file
    #escaping the : and // after https worked
    sed -i -E "s/<s3 url>/$S3_URL/" ./templates/backup_storage_location.yaml
    ${OC} apply -f ./templates/backup_storage_location_secret.yaml -f ./templates/backup_storage_location.yaml || error "Unable to create backup storage location resources to namespace $SF_NAMESPACE."
    
    change_ns="false"
    if [[ $SF_NAMESPACE != "ibm-spectrum-fusion-ns" ]]; then
        change_ns="true"
    fi
    
    #application
    info "Editing application resource..."
    sed -i -E "s/<operator namespace>/$OPERATOR_NS/" ./templates/application.yaml
    sed -i -E "s/<service namespace>/$SERVICES_NS/" ./templates/application.yaml
    sed -i -E "s/<tenant namespace 1>/$TETHERED_NAMESPACE1/" ./templates/application.yaml
    sed -i -E "s/<tenant namespace 2>/$TETHERED_NAMESPACE2/" ./templates/application.yaml
    sed -i -E "s/<cert manager namespace>/$CERT_MANAGER_NAMESPACE/" ./templates/application.yaml
    sed -i -E "s/<licensing namespace>/$LICENSING_NAMESPACE/" ./templates/application.yaml
    sed -i -E "s/<lsr namespace>/$LSR_NAMESPACE/" ./templates/application.yaml
    if [[ $change_ns == "true" ]]; then
        ${YQ} -i '.metadata.namesace = "'${SF_NAMESPACE}'"' ./templates/application.yaml || error "Could not update namespace value in application.yaml."
    fi
    ${OC} apply -f ./templates/application.yaml || error "Unable to create application in namespace $SF_NAMESPACE."

    #backup policy
    info "Editing backup policy resource..."
    sed -i -E "s/<storage_location>/$BACKUP_STORAGE_LOCATION_NAME/" ./templates/policy.yaml
    ${OC} apply -f ./templates/policy.yaml -n $SF_NAMESPACE || error "Unable to create policy in namespace $SF_NAMESPACE."

    #recipe
    info "Editing recipe resource..."
    sed -i -E "s/<operator namespace>/$OPERATOR_NS/" ./templates/multi-ns-recipe.yaml
    sed -i -E "s/<service namespace>/$SERVICES_NS/" ./templates/multi-ns-recipe.yaml
    sed -i -E "s/<cert manager namespace>/$CERT_MANAGER_NAMESPACE/" ./templates/multi-ns-recipe.yaml
    sed -i -E "s/<licensing namespace>/$LICENSING_NAMESPACE/" ./templates/multi-ns-recipe.yaml
    sed -i -E "s/<lsr namespace>/$LSR_NAMESPACE/" ./templates/multi-ns-recipe.yaml
    sed -i -E "s/<zenservice name>/$ZENSERVICE_NAME/" ./templates/multi-ns-recipe.yaml
    tethered_namespaces="$TETHERED_NAMESPACE1,$TETHERED_NAMESPACE2"
    sed -i -E "s/<comma delimited (no spaces) list of Cloud Pak workload namespaces that use this foundational services instance>/$tethered_namespaces/" ./templates/multi-ns-recipe.yaml
    sed -i -E "s/<foundational services version number in use i.e. 4.0, 4.1, 4.2, etc>/$CPFS_VERSION/" ./templates/multi-ns-recipe.yaml
    size=$(${OC} get commonservice common-service -n $OPERATOR_NS -o jsonpath='{.spec.size}')
    sed -i -E "s/<.spec.size value from commonservice cr>/$size/" ./templates/multi-ns-recipe.yaml
    sed -i -E "s/<install mode, either Manual or Automatic>/Automatic/" ./templates/multi-ns-recipe.yaml
    sed -i -E "s/<catalog source name>/$CATALOG_SOURCE/" ./templates/multi-ns-recipe.yaml
    sed -i -E "s/<catalog source namespace>/$CAT_SRC_NS/" ./templates/multi-ns-recipe.yaml

    if [[ $change_ns == "true" ]]; then
        ${YQ} -i '.metadata.namesace = "'${SF_NAMESPACE}'"' ./templates/multi-ns-recipe.yaml || error "Could not update namespace value in multi-ns-recipe.yaml."
    fi
    ${OC} apply -f ./templates/multi-ns-recipe.yaml || error "Unable to create recipe in namespace $SF_NAMESPACE."

    #policyassignment
    info "Editing policy assignment resource..."
    if [[ $change_ns == "true" ]]; then
        ${YQ} -i '.metadata.namesace = "'${SF_NAMESPACE}'"' ./templates/policy_assignment.yaml || error "Could not update namespace value in policy_assignment.yaml."
    fi
    ${OC} apply -f ./templates/policy_assignment.yaml || error "Unable to create policy assignment in namespace $SF_NAMESPACE."

    success "Spectrum Fusion BR resources created in namespace $SF_NAMESPACE."

}

function deploy_cs_br_resources() {
    title "Deploying necessary BR resources for persistent CPFS components."
    cd ../velero/schedule
    tethered_namespaces="$TETHERED_NAMESPACE1,$TETHERED_NAMESPACE2"
    ./deploy_cs_br_resources --services-ns $SERVICES_NS --operator-ns $OPERATOR_NS --lsr-ns $LSR_NAMESPACE --im --zen --tethered-ns $tethered_namespaces --util --storage-class $STORAGE_CLASS || error "Script deploy-br-resources.sh failed to deploy BR resources."
    cd $BASE_DIR
    success "BR resources for persistent CPFS components deployed."
}

function label_cs_resources() {
    title "Labeling CS resources."

    info "Labeling cert manager resources..."
    ./../velero/backup/cert-manager/label-cert-manager.sh || error "Unable to complete labeling of cert manager resources."

    info "Labeling licensing resources..."
    ./../velero/backup/licensing/label-licensing-configmaps.sh $LICENSING_NAMESPACE || error "Unable to complete labeling of licensing resources."

    mv ../velero/backup/common-service/env.properties ../velero/backup/common-service/og-env.properties
    cp env.properties ../velero/backup/common-service/env.properties
    info "Labeling remaining CPFS resources..."
    ./../velero/backup/common-service/label-common-service.sh || error "Unable to complete labeling of CPFS resources."

    success "CPFS resources labeled."
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