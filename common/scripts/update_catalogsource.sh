#!/bin/bash
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

CATALOG_SOURCE_NAME=$1
CATALOG_SOURCE_IMAGE=$2

# checking arguments
if [[ ( -z "${CATALOG_SOURCE_NAME}" || -z "${CATALOG_SOURCE_IMAGE}" ) ]]; then
    echo "[ERROR] Usage: $0 CATALOG_SOURCE_NAME CATALOG_SOURCE_IMAGE"
    exit 1
fi

cat <<EOF | tee >(oc apply -f -) | cat
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: ${CATALOG_SOURCE_NAME}
  namespace: openshift-marketplace
spec:
  displayName: ${CATALOG_SOURCE_NAME}
  publisher: IBM
  sourceType: grpc
  image: ${CATALOG_SOURCE_IMAGE}
  updateStrategy:
    registryPoll:
      interval: 45m
EOF