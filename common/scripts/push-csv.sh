#!/bin/bash
#
# Copyright 2020 IBM Corporation
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

set -e

# QUAY_REPO ?= quay.io/opencloudio
# OPERATOR_NAME ?= ibm-common-service-operator
# VERSION ?= 0.0.1

DELETE_EXIST=${DELETE_EXIST:-no}
QUAY_ORG=${QUAY_REPO##*/}
BUNDLE_DIR=deploy/olm-catalog/${OPERATOR_NAME}

[[ "X$QUAY_USERNAME" == "X" ]] && read -rp "Enter username quay.io: " QUAY_USERNAME
[[ "X$QUAY_PASSWORD" == "X" ]] && read -rsp "Enter password quay.io: " QUAY_PASSWORD && echo

# Fetch authentication token used to push to Quay.io
AUTH_TOKEN=$(curl -sH "Content-Type: application/json" -XPOST https://quay.io/cnr/api/v1/users/login -d '
{
    "user": {
        "username": "'"${QUAY_USERNAME}"'",
        "password": "'"${QUAY_PASSWORD}"'"
    }
}' | awk -F'"' '{print $4}')

function cleanup() {
    rm -f bundle.tar.gz
}
trap cleanup EXIT

tar czf bundle.tar.gz "${BUNDLE_DIR}"

if [[ "${OSTYPE}" == "darwin"* ]]; then
  BLOB=$(base64 -b0 < bundle.tar.gz)
else
  BLOB=$(base64 -w0 < bundle.tar.gz)
fi

# Push application to repository
function push_csv() {
  echo "Push package ${OPERATOR_NAME} into namespace ${QUAY_ORG}"
  curl -H "Content-Type: application/json" \
      -H "Authorization: ${AUTH_TOKEN}" \
      -XPOST https://quay.io/cnr/api/v1/packages/"${QUAY_ORG}"/"${OPERATOR_NAME}" -d '
  {
      "blob": "'"${BLOB}"'",
      "release": "'"${VERSION}"'",
      "media_type": "helm"
  }'
}

# Delete application release in repository
function delete_csv() {
  echo "Delete release ${VERSION} of package ${OPERATOR_NAME} from namespace ${QUAY_ORG}"
  curl -H "Content-Type: application/json" \
      -H "Authorization: ${AUTH_TOKEN}" \
      -XDELETE https://quay.io/cnr/api/v1/packages/"${QUAY_ORG}"/"${OPERATOR_NAME}"/"${VERSION}"/helm
}

#-------------------------------------- Main --------------------------------------#
delete_csv
push_csv
