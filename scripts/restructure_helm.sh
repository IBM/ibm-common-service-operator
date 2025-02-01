#!/bin/bash

# Copyright 2025 IBM Corporation
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

set -e

# Define source paths
HELM_SRC_DIR="generate-helm/ibm-common-service-operator"
# CONFIG_SRC_DIR="generate-helm/ibm-common-service-operator-generated"

# Define helm destination paths
HELM_DIR="helm"
CLUSTER_SCOPED_DIR="helm-cluster-scoped"

# Ensure directories exist
mkdir -p $HELM_DIR/templates
mkdir -p $CLUSTER_SCOPED_DIR/templates

# Function to merge YAML files with "---" separators
merge_yaml() {
    local pattern=$1
    local output_file=$2
    echo "Merging files matching pattern: $pattern into $output_file"

    # Clear output file
    > $output_file  
    local first_file=true
    for file in $(grep -l "$pattern" $HELM_SRC_DIR/templates/*.yaml); do
        if [ "$first_file" = false ]; then
            echo -e "\n---" >> $output_file
        fi
        cat "$file" >> $output_file
        first_file=false
    done

    # Add service account to rbac.yaml or cluster-rbac.yaml
    if [[ "$pattern" = "kind: Role" ]] || [[ "$pattern" = "kind: ClusterRole" ]]; then
        echo -e "\n---" >> $output_file
        cat $HELM_SRC_DIR/templates/ibm-common-service-operator-sa.yaml >> $output_file
    fi
}

# ----------------- Namespace-scoped resources -----------------

# Move and merge namespace-scoped resources
merge_yaml "kind: Role" "$HELM_DIR/templates/rbac.yaml"
merge_yaml "kind: Deployment" "$HELM_DIR/templates/operator-deployment.yaml"

# ----------------- Cluster-scoped resources -----------------

# Move and merge cluster-scoped resources
merge_yaml "kind: ClusterRole" "$CLUSTER_SCOPED_DIR/templates/cluster-rbac.yaml"
cp $HELM_SRC_DIR/crds/* $CLUSTER_SCOPED_DIR/templates/crds.yaml

# Todo: rest of resources

# Copy Helm values, Chart.yaml and helper.tpl
for dir in $HELM_DIR $CLUSTER_SCOPED_DIR; do cp $HELM_SRC_DIR/{values.yaml,Chart.yaml} "$dir/"; done
for dir in $HELM_DIR $CLUSTER_SCOPED_DIR; do cp $HELM_SRC_DIR/templates/_helpers.tpl "$dir/templates/"; done

# Remove generated ibm-common-service-operator and ibm-common-service-operator-generated directories
rm -rf generate-helm/ibm-common-service-operator
rm -rf generate-helm/ibm-common-service-operator-generated

echo "Helm chart restructuring complete."

