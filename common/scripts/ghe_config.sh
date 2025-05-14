#!/bin/bash
#
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
#

KUBECTL=$(which kubectl)
GIT_USERNAME=$(${KUBECTL} -n default get secret helm-repo-cred -o jsonpath='{.data.username}' | base64 --decode)
GIT_TOKEN=$(${KUBECTL} -n default get secret helm-repo-cred -o jsonpath='{.data.password}' | base64 --decode)

URL_ENCODED_USERNAME=$(echo $GIT_USERNAME | jq -Rr @uri)

# support other container tools, e.g. podman
GIT=$(which git)

# login the docker registry
${GIT} clone "https://$URL_ENCODED_USERNAME:$GIT_TOKEN@github.ibm.com/IBMPrivateCloud/helm-charts-reduction.git"

${GIT} config --global user.email "operator@operator.com"
${GIT} config --global user.name "ibm-common-service-operator"

echo "clone repo"
ls

echo "check reduction folder 1"
ls helm-charts-reduction

cp -r helm-cluster-scoped/* helm-charts-reduction/source-charts/ibm-common-service-operator-cluster-scoped
cp -r helm/* helm-charts-reduction/source-charts/ibm-common-service-operator

cd helm-charts-reduction
${GIT} checkout staging
echo "check reduction folder 2"
ls
${GIT} status
${GIT} add .
${GIT} commit -s -m "updated helm files for ibm-common-service-operator"
${GIT} push
