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

# This script needs to inputs
# The CSV version that is currently in dev

# cs operator
CURRENT_DEV_CSV=$1
NEW_DEV_CSV=$2
PREVIOUS_DEV_CSV=$3
# cs operator channel
CURRENT_CHANNEL=$4
NEW_CHANNEL=$5

if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    # Linux OS
    # Update bundle/manifests/ibm-common-service-operator.clusterserviceversion.yaml
    sed -i "s/$CURRENT_DEV_CSV/$NEW_DEV_CSV/g" bundle/manifests/ibm-common-service-operator.clusterserviceversion.yaml
    sed -i "s/$PREVIOUS_DEV_CSV/$CURRENT_DEV_CSV/g" bundle/manifests/ibm-common-service-operator.clusterserviceversion.yaml
    sed -i "/operatorChannel/s/$CURRENT_CHANNEL/$NEW_CHANNEL/g" bundle/manifests/ibm-common-service-operator.clusterserviceversion.yaml
    echo "Updated the bundle/manifests/ibm-common-service-operator.clusterserviceversion.yaml"

    # Update config/manifests/bases/ibm-common-service-operator.clusterserviceversion.yaml
    sed -i "s/$CURRENT_DEV_CSV/$NEW_DEV_CSV/g" config/manifests/bases/ibm-common-service-operator.clusterserviceversion.yaml
    sed -i "/operatorChannel/s/$CURRENT_CHANNEL/$NEW_CHANNEL/g" config/manifests/bases/ibm-common-service-operator.clusterserviceversion.yaml
    echo "Updated the config/manifests/bases/ibm-common-service-operator.clusterserviceversion.yaml"

    # Update cs operator version only
    sed -i "s/$CURRENT_DEV_CSV/$NEW_DEV_CSV/g" version/version.go
    echo "Updated the version.go"
    sed -i "s/$CURRENT_DEV_CSV/$NEW_DEV_CSV/g" common/scripts/multiarch_image.sh
    echo "Updated the multiarch_image.sh"
    sed -i "s/$CURRENT_DEV_CSV/$NEW_DEV_CSV/g" README.md
    echo "Updated the README.md"

    # update cs operator & channel in deply.yaml
    sed -i "s/$CURRENT_DEV_CSV/$NEW_DEV_CSV/g" testdata/deploy/deploy.yaml
    sed -i "/operatorChannel/s/$CURRENT_CHANNEL/$NEW_CHANNEL/g" testdata/deploy/deploy.yaml
    echo "Updated the testdata/deploy/deploy.yaml"

    # update cs operator channel in Makefile & bundle.Dockerfile & annotations.yaml
    sed -i "s/$CURRENT_CHANNEL/$NEW_CHANNEL/g" Makefile
    echo "Updated the Makefile"

    sed -i "s/$CURRENT_CHANNEL/$NEW_CHANNEL/g" bundle.Dockerfile
    echo "Updated the bundle.Dockerfile"

    sed -i "s/$CURRENT_CHANNEL/$NEW_CHANNEL/g" bundle/metadata/annotations.yaml
    echo "Updated the bundle/metadata/annotations.yaml"

elif [[ "$OSTYPE" == "darwin"* ]]; then
    # Mac OSX
    # Update bundle/manifests/ibm-common-service-operator.clusterserviceversion.yaml
    sed -i "" "s/$CURRENT_DEV_CSV/$NEW_DEV_CSV/g" bundle/manifests/ibm-common-service-operator.clusterserviceversion.yaml
    sed -i "" "s/$PREVIOUS_DEV_CSV/$CURRENT_DEV_CSV/g" bundle/manifests/ibm-common-service-operator.clusterserviceversion.yaml
    sed -i "" "/operatorChannel/s/$CURRENT_CHANNEL/$NEW_CHANNEL/g" bundle/manifests/ibm-common-service-operator.clusterserviceversion.yaml
    echo "Updated the bundle/manifests/ibm-common-service-operator.clusterserviceversion.yaml"

    # Update config/manifests/bases/ibm-common-service-operator.clusterserviceversion.yaml
    sed -i "" "s/$CURRENT_DEV_CSV/$NEW_DEV_CSV/g" config/manifests/bases/ibm-common-service-operator.clusterserviceversion.yaml
    sed -i "" "/operatorChannel/s/$CURRENT_CHANNEL/$NEW_CHANNEL/g" config/manifests/bases/ibm-common-service-operator.clusterserviceversion.yaml
    echo "Updated the config/manifests/bases/ibm-common-service-operator.clusterserviceversion.yaml"

    # Update cs operator version only
    sed -i "" "s/$CURRENT_DEV_CSV/$NEW_DEV_CSV/g" version/version.go
    echo "Updated the version.go"
    sed -i "" "s/$CURRENT_DEV_CSV/$NEW_DEV_CSV/g" common/scripts/multiarch_image.sh
    echo "Updated the multiarch_image.sh"
    sed -i "" "s/$CURRENT_DEV_CSV/$NEW_DEV_CSV/g" README.md
    echo "Updated the README.md"

    # update cs operator & channel in deply.yaml
    sed -i "" "s/$CURRENT_DEV_CSV/$NEW_DEV_CSV/g" testdata/deploy/deploy.yaml
    sed -i "" "/operatorChannel/s/$CURRENT_CHANNEL/$NEW_CHANNEL/g" testdata/deploy/deploy.yaml
    echo "Updated the testdata/deploy/deploy.yaml"

    # update cs operator channel in Makefile & bundle.Dockerfile & annotations.yaml
    sed -i "" "s/$CURRENT_CHANNEL/$NEW_CHANNEL/g" Makefile
    echo "Updated the Makefile"

    sed -i "" "s/$CURRENT_CHANNEL/$NEW_CHANNEL/g" bundle.Dockerfile
    echo "Updated the bundle.Dockerfile"

    sed -i "" "s/$CURRENT_CHANNEL/$NEW_CHANNEL/g" bundle/metadata/annotations.yaml
    echo "Updated the bundle/metadata/annotations.yaml"

else
    echo "Not support on other operating systems"
fi