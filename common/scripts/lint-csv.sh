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

STATUS=0
ARCH=$(uname -m)
VERSION=${CSV_VERSION:-3.4.1}

if [[ $ARCH == "x86_64" ]]; then
    curl -L -o /tmp/jq https://github.com/stedolan/jq/releases/download/jq-1.6/jq-linux64
    curl -L -o /tmp/yq https://github.com/mikefarah/yq/releases/download/3.3.0/yq_linux_amd64
    chmod +x /tmp/jq /tmp/yq
else
    exit 0
fi

CSV_PATH=deploy/olm-catalog/ibm-common-service-operator/${VERSION}/ibm-common-service-operator.v${VERSION}.clusterserviceversion.yaml

# Lint alm-examples
echo "Lint alm-examples"
/tmp/yq r $CSV_PATH metadata.annotations.alm-examples | /tmp/jq . >/dev/null || STATUS=1

# Lint yamls, only CS Operator needs this part
for section in csNamespace csOperandConfig csOperandRegistry odlmSubscription; do
    echo "Lint $section"
    /tmp/yq r $CSV_PATH metadata.annotations.$section | /tmp/yq r - >/dev/null || STATUS=1
done

sections=$(/tmp/yq r $CSV_PATH metadata.annotations.extraResources)
for section in ${sections//,/ }; do
    echo "Lint $section"
    /tmp/yq r $CSV_PATH metadata.annotations.$section | /tmp/yq r - >/dev/null || STATUS=1
done

exit $STATUS
