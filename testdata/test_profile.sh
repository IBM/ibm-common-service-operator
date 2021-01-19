#!/usr/bin/env bash
#
# Copyright 2021 IBM Corporation
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

set -o errexit
set -o nounset
set -o errtrace
set -o pipefail

YQ=$1
BASE_DIR=testdata

test_profile() {
    local size_file=$1

    # keys to remove from controllers/size files before comparison
    # necessary because size files in controllers/size missing many keys that
    # OperandConfig can have
    local REMOVE_KEYS=".IBMLicenseServiceReporter?, .operandBindInfo?, .operandRequest?, \
        .datasource?, .mustgatherConfig?, .mustgatherService?, .navconfiguration?, \
        .switcheritem?, .certificate?, .clusterIssuer?, .issuer?"

    # get expected size
    # first sed gets the variable in size file
    # second sed strips first and last line to remove "`"
    # jq commands are to sort and delete unnecessary keys for proper comparison
    sed -n '/`/,/`/p' $size_file | sed '1d;$d' |  $YQ eval -j '.' - \
        | jq -S '. |= sort_by (.name)' | jq "del(.. | $REMOVE_KEYS)" \
        > $BASE_DIR/expected.yaml

    # get actual size
    oc -n ibm-common-services get operandconfig common-service -o json \
        | jq '.spec.services' - | jq -S '. |= sort_by (.name)' \
        | jq "del(.. | $REMOVE_KEYS)" \
        > $BASE_DIR/actual.yaml

    # if files are different, show the difference
    diff --brief $BASE_DIR/expected.yaml $BASE_DIR/actual.yaml
}

NAMESPACE=test-profile-ns
oc create ns $NAMESPACE

cleanup() {
    oc delete ns $NAMESPACE
}

trap 'cleanup' EXIT

test_profile controllers/size/small_amd64.go

oc -n $NAMESPACE apply -f testdata/sizing/medium_size.yaml
sleep 15
test_profile controllers/size/medium_amd64.go

oc -n $NAMESPACE apply -f testdata/sizing/large_size.yaml
sleep 15
test_profile controllers/size/large_amd64.go
