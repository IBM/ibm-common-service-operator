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

set -o errexit
set -o pipefail
set -o errtrace
set -o nounset

OC=oc
YQ=yq
requestedNS=
mapToCSNS=
namespaces=
dc="false"
dj="false"
refresh="false"
cleanup="false"

function main() {
    while [ "$#" -gt "0" ]
    do
        case "$1" in
        "-h"|"--help")
            usage
            exit 0
            ;;
        "--requested-from-ns")
            shift
            requestedNS="$@"
            info "requestedNs= $requestedNS"
            requested="true"
            ;;
        "--map-to-ns")
            mapToCSNS=$2
            shift
            ;;
        "--refresh")
            refresh="true"
            ;;
        "--delete-client")
            dc="true"
            ;;
        "--delete-job")
            dj="true"
            ;;
        "--cleanup")
            cleanup="true"
            ;;
        *)
            if [[ $requested == "false" ]]; then
                error "invalid option -- \`$1\`. Use the -h or --help option for usage info."
            fi
            ;;
        esac
        shift
    done
    namespaces="$mapToCSNS $requestedNS"
    if [[ $refresh == "true" ]]; then
        add_conversion_field
    else if [[ $cleanup == "true" ]]; then
        remove_csns_from_zenservice
    else if [[ $dc == "true" ]]; then
        delete_iam_clients
    else if [[ $dj == "true" ]]; then
        delete_iam_config_jobs
    fi
}

#change this from cleanup and refresh to smaller scale
# remove cs namespace from zenservice in specified namespaces
# delete iam config jobs in specified namespaces
# delete iam clients in specified namespaces
# add conversion=true or remove conversion field to reconcile zenservice

function remove_csns_from_zenservice() {
    title " Removing csNamespace field from zenservices in specified namespaces "
    msg "-----------------------------------------------------------------------"
    for namespace in $namespaces
    do
        return_value=$(${OC} get zenservice -n ${namespace} || echo "fail")
        if [[ $return_value != "fail" ]]; then
            if [[ $return_value != "" ]]; then
                zenServiceCR=$(${OC} get zenservice -n ${namespace} | awk '{if (NR!=1) {print $1}}')
                ${OC} patch zenservice ${zenServiceCR} -n ${namespace} --type json -p '[{ "op": "remove", "path": "/spec/csNamespace" }]' || info "CS Namespace not defined in ${zenServiceCR} in ${namespace}. Moving on..."
            else
                info "No zen service in namespace ${namespace}. Moving on..."
            fi
        else
          info "Zen not installed in ${namespace}. Moving on..."
        fi
        return_value=""
    done
    success "Finished removing csNamespace field from zenservice resources"
}

function delete_iam_config_jobs() {
    title " Deleting IAM config jobs from specified namespaces "
    msg "-----------------------------------------------------------------------"
    for namespace in $namespaces
    do
        return_value=$(${OC} get job -n ${namespace} | grep iam-config-job || echo "fail")
        if [[ $return_value != "fail" ]]; then
            ${OC} delete job iam-config-job -n ${namespace}
        else
            info "iam-config-job not present in namespace ${namespace}. Moving on..."
        fi
    done
    success "IAM config jobs deleted ini specified namespaces"
}

function delete_iam_clients(){
    title " Deleting IAM clients from specified namespaces "
    msg "-----------------------------------------------------------------------"
    for namespace in $namespaces
    do
        return_value=$(${OC} get client -n ${namespace} || echo "fail")
        if [[ $return_value != "fail" ]]; then
            if [[ $return_value != "" ]]; then
                zenClient=$(${OC} get client -n ${namespace} | awk '{if (NR!=1) {print $1}}')
                ${OC} patch client ${zenClient} -n ${namespace} --type=merge -p '{"metadata": {"finalizers":null}}'
                ${OC} delete client ${zenClient} -n ${namespace}
            else
                info "No zen client in ${namespace}. Moving on..."
            fi
        else
            info "Zen not installed in ${namespace}. Moving on..."
        fi
        return_value=""
    done
    success "IAM clients deleted from specified namespaces"
}

function add_conversion_field() {
    title " Reconciling zenservice resources in specified namespaces "
    msg "-----------------------------------------------------------------------"
    for namespace in $namespaces
    do
        return_value=$(${OC} get zenservice -n ${namespace} || echo "fail")
        if [[ $return_value != "fail" ]]; then
            if [[ $return_value != "" ]]; then
                zenServiceCR=$(${OC} get zenservice -n ${namespace} | awk '{if (NR!=1) {print $1}}')
                conversionField=$("${OC}" get zenservice ${zenServiceCR} -n ${namespace} -o yaml | yq '.spec | has("conversion")')
                if [[ $conversionField == "false" ]]; then
                    ${OC} patch zenservice ${zenServiceCR} -n ${namespace} --type='merge' -p '{"spec":{"conversion":"true"}}' || error "Zenservice ${zenServiceCR} in ${namespace} cannot be updated."
                else
                    ${OC} patch zenservice ${zenServiceCR} -n ${namespace} --type json -p '[{ "op": "remove", "path": "/spec/conversion" }]' || error "Zenservice ${zenServiceCR} in ${namespace} cannot be updated."
                fi
                conversionField=""
            else
                info "No zen service in namespace ${namespace}. Moving on..."
            fi
        else
          info "Zen not installed in ${namespace}. Moving on..."
        fi
        return_value=""
    done
    success "Reconcile initiated for zenservice resources in specified namespaces"
}

function cleanupZenService(){
    title " Cleaning up Zen installation "
    msg "-----------------------------------------------------------------------"

    for namespace in $requestedNS
    do
        # remove cs namespace from zen service cr
        return_value=$(${OC} get zenservice -n ${namespace} || echo "fail")
        if [[ $return_value != "fail" ]]; then
            if [[ $return_value != "" ]]; then
                zenServiceCR=$(${OC} get zenservice -n ${namespace} | awk '{if (NR!=1) {print $1}}')
                ${OC} patch zenservice ${zenServiceCR} -n ${namespace} --type json -p '[{ "op": "remove", "path": "/spec/csNamespace" }]' || info "CS Namespace not defined in ${zenServiceCR} in ${namespace}. Moving on..."
            else
                info "No zen service in namespace ${namespace}. Moving on..."
            fi
        else
          info "Zen not installed in ${namespace}. Moving on..."
        fi
        return_value=""

        # delete iam config job
        return_value=$(${OC} get job -n ${namespace} | grep iam-config-job || echo "fail")
        if [[ $return_value != "fail" ]]; then
            ${OC} delete job iam-config-job -n ${namespace}
        else
            info "iam-config-job not present in namespace ${namespace}. Moving on..."
        fi

        # delete zen client
        return_value=$(${OC} get client -n ${namespace} || echo "fail")
        if [[ $return_value != "fail" ]]; then
            if [[ $return_value != "" ]]; then
                zenClient=$(${OC} get client -n ${namespace} | awk '{if (NR!=1) {print $1}}')
                ${OC} patch client ${zenClient} -n ${namespace} --type=merge -p '{"metadata": {"finalizers":null}}'
                ${OC} delete client ${zenClient} -n ${namespace}
            else
                info "No zen client in ${namespace}. Moving on..."
            fi
        else
            info "Zen not installed in ${namespace}. Moving on..."
        fi
        return_value=""
    done
    
    for namespace in $mapToCSNS
    do
        # remove cs namespace from zen service cr
        return_value=$(${OC} get zenservice -n ${namespace} || echo "fail")
        if [[ $return_value != "fail" ]]; then
            if [[ $return_value != "" ]]; then
                zenServiceCR=$(${OC} get zenservice -n ${namespace} | awk '{if (NR!=1) {print $1}}')
                ${OC} patch zenservice ${zenServiceCR} -n ${namespace} --type json -p '[{ "op": "remove", "path": "/spec/csNamespace" }]' || info "CS Namespace not defined in ${zenServiceCR} in ${namespace}. Moving on..."
            else
                info "No zen service in namespace ${namespace}. Moving on..."
            fi
        else
            info "Zen not installed in ${namespace}. Moving on..."
        fi
        return_value=""

        # delete iam config job
        return_value=$(${OC} get job -n ${namespace} | grep iam-config-job || echo "failed")
        if [[ $return_value != "failed" ]]; then
            ${OC} delete job iam-config-job -n ${namespace}
        else
            info "iam-config-job not present in namespace ${namespace}. Moving on..."
        fi

        # delete zen client
        return_value=$(${OC} get client -n ${namespace} || echo "fail")
        if [[ $return_value != "fail" ]]; then
            if [[ $return_value != "" ]]; then
                zenClient=$(${OC} get client -n ${namespace} | awk '{if (NR!=1) {print $1}}')
                ${OC} patch client ${zenClient} -n ${namespace} --type=merge -p '{"metadata": {"finalizers":null}}'
                ${OC} delete client ${zenClient} -n ${namespace}
            else
                info "No zen client in ${namespace}. Moving on..."
            fi
        else
            info "Zen not installed in ${namespace}. Moving on..."
        fi
        return_value=""
    done
    success "Zen instances cleaned up"
}

function refresh_zen(){
    title " Refreshing Zen Services "
    msg "-----------------------------------------------------------------------"
    #make sure IAM is ready before reconciling.
    check_IAM #this will likely need to change in the future depending on how we check iam status
 
    for namespace in $requestedNS
    do
        # remove cs namespace from zen service cr
        return_value=$(${OC} get zenservice -n ${namespace} || echo "fail")
        if [[ $return_value != "fail" ]]; then
            if [[ $return_value != "" ]]; then
                zenServiceCR=$(${OC} get zenservice -n ${namespace} | awk '{if (NR!=1) {print $1}}')
                conversionField=$("${OC}" get zenservice ${zenServiceCR} -n ${namespace} -o yaml | yq '.spec | has("conversion")')
                if [[ $conversionField == "false" ]]; then
                    ${OC} patch zenservice ${zenServiceCR} -n ${namespace} --type='merge' -p '{"spec":{"conversion":"true"}}' || error "Zenservice ${zenServiceCR} in ${namespace} cannot be updated."
                else
                    ${OC} patch zenservice ${zenServiceCR} -n ${namespace} --type json -p '[{ "op": "remove", "path": "/spec/conversion" }]' || error "Zenservice ${zenServiceCR} in ${namespace} cannot be updated."
                fi
                conversionField=""
            else
                info "No zen service in namespace ${namespace}. Moving on..."
            fi
        else
          info "Zen not installed in ${namespace}. Moving on..."
        fi
        return_value=""
    done
    
    for namespace in $mapToCSNS
    do
        # remove cs namespace from zen service cr
        return_value=$(${OC} get zenservice -n ${namespace} || echo "fail")
        if [[ $return_value != "fail" ]]; then
            if [[ $return_value != "" ]]; then
                zenServiceCR=$(${OC} get zenservice -n ${namespace} | awk '{if (NR!=1) {print $1}}')
                conversionField=$(${OC} get zenservice ${zenServiceCR} -n ${namespace} -o yaml | yq '.spec | has("conversion")')
                if [[ $conversionField == "true" ]]; then
                    ${OC} patch zenservice ${zenServiceCR} -n ${namespace} --type='merge' -p '{"spec":{"conversion":"true"}}' || error "Zenservice ${zenServiceCR} in ${namespace} cannot be updated."
                else
                    ${OC} patch zenservice ${zenServiceCR} -n ${namespace} --type json -p '[{ "op": "remove", "path": "/spec/conversion" }]' || error "Zenservice ${zenServiceCR} in ${namespace} cannot be updated."
                fi
                conversionField=""
            else
                info "No zen service in namespace ${namespace}. Moving on..."
            fi
        else
            info "Zen not installed in ${namespace}. Moving on..."
        fi
        return_value=""
    done
    success "Reconcile loop initiated for Zenservice instances"
}

function msg() {
    printf '%b\n' "$1"
}

function success() {
    msg "\33[32m[✔] ${1}\33[0m"
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

# --- Run ---

main $*