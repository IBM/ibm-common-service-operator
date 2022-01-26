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
# secretshare
CURRENT_SECRETSHARE_CSV=$4
NEW_SECRETSHARE_CSV=$5
# webhook
CURRENT_WEBHOOK_CSV=$6
NEW_WEBHOOK_CSV=$7

# Update bundle/manifests/ibm-common-service-operator.clusterserviceversion.yaml
sed -i "s/$CURRENT_DEV_CSV/$NEW_DEV_CSV/g" bundle/manifests/ibm-common-service-operator.clusterserviceversion.yaml
sed -i "s/$PREVIOUS_DEV_CSV/$CURRENT_DEV_CSV/g" bundle/manifests/ibm-common-service-operator.clusterserviceversion.yaml
sed -i "s/$CURRENT_SECRETSHARE_CSV/$NEW_SECRETSHARE_CSV/" bundle/manifests/ibm-common-service-operator.clusterserviceversion.yaml
sed -i "s/$CURRENT_WEBHOOK_CSV/$NEW_WEBHOOK_CSV/" bundle/manifests/ibm-common-service-operator.clusterserviceversion.yaml
echo "Updated the bundle/manifests/ibm-common-service-operator.clusterserviceversion.yaml"

# Update config/manifests/bases/ibm-common-service-operator.clusterserviceversion.yaml
sed -i "s/$CURRENT_DEV_CSV/$NEW_DEV_CSV/g" config/manifests/bases/ibm-common-service-operator.clusterserviceversion.yaml
sed -i "s/$CURRENT_SECRETSHARE_CSV/$NEW_SECRETSHARE_CSV/" config/manifests/bases/ibm-common-service-operator.clusterserviceversion.yaml
sed -i "s/$CURRENT_WEBHOOK_CSV/$NEW_WEBHOOK_CSV/" config/manifests/bases/ibm-common-service-operator.clusterserviceversion.yaml
echo "Updated the config/manifests/bases/ibm-common-service-operator.clusterserviceversion.yaml"

# Update cs operator version only
sed -i "s/$CURRENT_DEV_CSV/$NEW_DEV_CSV/g" version/version.go
echo "Updated the version.go"
sed -i "s/$CURRENT_DEV_CSV/$NEW_DEV_CSV/" common/scripts/multiarch_image.sh
echo "Updated the multiarch_image.sh"
sed -i "s/$CURRENT_DEV_CSV/$NEW_DEV_CSV/" README.md
echo "Updated the README.md"
sed -i "s/$CURRENT_DEV_CSV/$NEW_DEV_CSV/" controllers/constant/secretshare.go
echo "Updated the controllers/constant/secretshare.go"
sed -i "s/$CURRENT_DEV_CSV/$NEW_DEV_CSV/" controllers/constant/webhook.go
echo "Updated the controllers/constant/webhook.go"

# update cs operator & webhook & secretshare version
sed -i "s/$CURRENT_DEV_CSV/$NEW_DEV_CSV/" testdata/deploy/deploy.yaml
sed -i "s/$CURRENT_SECRETSHARE_CSV/$NEW_SECRETSHARE_CSV/" testdata/deploy/deploy.yaml
sed -i "s/$CURRENT_WEBHOOK_CSV/$NEW_WEBHOOK_CSV/" testdata/deploy/deploy.yaml
echo "Updated the testdata/deploy/deploy.yaml"

sed -i "s/$CURRENT_SECRETSHARE_CSV/$NEW_SECRETSHARE_CSV/" config/manager/manager.yaml
sed -i "s/$CURRENT_WEBHOOK_CSV/$NEW_WEBHOOK_CSV/" config/manager/manager.yaml
echo "Updated the testdata/deploy/deploy.yaml"
