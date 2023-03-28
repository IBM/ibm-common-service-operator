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
master_ns=
requestedNS=
mapToCSNS=
controlNs=
cm_name="common-service-maps"
restart="false"
requested="false"
# pause installer
# uninstall singletons
# restart installer

function main () {
    while [ "$#" -gt "0" ]
    do
        case "$1" in
        "-h"|"--help")
            usage
            exit 0
            ;;
        "--original-cs-ns")
            master_ns=$2
            shift
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
        "-r")
            restart="true"
            ;;
        *)
            if [[ $requested == "false" ]]; then
                error "invalid option -- \`$1\`. Use the -h or --help option for usage info."
            fi
            ;;
        esac
        shift
    done
    prereq
    if [[ $restart ==  "false" ]]; then
        ./backup_preload_mongo.sh $master_ns $mapToCSNS #requires cert manager
        pause
        uninstall_singletons
    else
        restart
    fi
}

function usage() {
	local script="${0##*/}"

	while read -r ; do echo "${REPLY}" ; done <<-EOF
	Usage: ${script} [OPTION]...
	Isolate and prepare common services for upgrade
	Options:
	Mandatory arguments to long options are mandatory for short options too.
	  -h, --help                    display this help and exit
	  --original-cs-ns              specify the namespace the original common services installation resides
      --map-to-ns                   specify the namespace value for "map-to-common-service-namespace" in common-service-maps
      -r                            restart common services
      --requested-from-ns           specify the namespace values for "requested-from-namespace" in common-service-maps. Must be last.
	EOF
}

# verify that all pre-requisite CLI tools exist
function prereq() {
    which "${OC}" || error "Missing oc CLI"
    which "${YQ}" || error "Missing yq"
    
    if [[ -z $requestedNS ]] && [[ -z $mapToCSNS ]] &&  [[ -z $master_ns ]]; then
        usage
        error "No parameters entered. Please re-run specifying original, map-to, and requested-from namespace values"
    elif [[ -z $requestedNS ]] || [[ -z $mapToCSNS ]] || [[ -z $master_ns ]]; then
        usage
        error "Required parameters missing. Please re-run specifying original, map-to, and requested-from namespace values"
    fi
    
    namespaces="$requestedNS $mapToCSNS"
    
    return_value=$("${OC}" get -n kube-public configmap ${cm_name} > /dev/null || echo failed)
    if [[ $return_value == "failed" ]]; then
        error "Missing configmap: ${cm_name}. This must be configured before proceeding"
    fi
    return_value="reset"

    # configmap should have control namespace specified
    return_value=$("${OC}" get configmap -n kube-public -o yaml ${cm_name} | yq '.data' | grep controlNamespace: > /dev/null || echo failed)
    if [[ $return_value == "failed" ]]; then
        error "Configmap: ${cm_name} did not specify 'controlNamespace' field. This must be configured before proceeding"
    fi
    return_value="reset"

    controlNs=$("${OC}" get configmap -n kube-public -o yaml ${cm_name} | yq '.data' | grep controlNamespace: | awk '{print $2}')
    return_value=$("${OC}" get ns "${controlNs}" > /dev/null || echo failed)
    if [[ $return_value == "failed" ]]; then
        error "The namespace specified in controlNamespace does not exist. This namespace must be created before proceeding."
    fi
    return_value="reset"

    # LicenseServiceReporter should not be installed because it does not support multi-instance mode
    return_value=$(("${OC}" get crd ibmlicenseservicereporters.operator.ibm.com > /dev/null && echo exists) || echo fail)
    if [[ $return_value == "exists" ]]; then
        return_value=$("${OC}" get ibmlicenseservicereporters -A | wc -l)
        if [[ $return_value -gt 0 ]]; then
            error "LicenseServiceReporter does not support multi-instance mode. Remove before proceeding"
        fi
    fi
    return_value="reset"

    # ensure cs-operator is not installed in all namespace mode
    return_value=$("${OC}" get csv -n openshift-operators | grep ibm-common-service-operator > /dev/null || echo pass)
    if [[ $return_value != "pass" ]]; then
        error "The ibm-common-service-operator must not be installed in AllNamespaces mode"
    fi

    check_cm_ns_exist
}

function pause() {
    
    ${OC} scale deployment -n ${master_ns} ibm-common-service-operator --replicas=0
    ${OC} scale deployment -n ${master_ns} operand-deployment-lifecycle-manager --replicas=0
    ${OC} delete operandregistry -n ${master_ns} --ignore-not-found common-service 
    ${OC} delete operandconfig -n ${master_ns} --ignore-not-found common-service
    
    cleanupCSOperators
    removeNSS
    cleanupZenService
}

function uninstall_singletons() {
    # uninstall singleton services
    "${OC}" delete -n "${master_ns}" --ignore-not-found certmanager default
    "${OC}" delete -n "${master_ns}" --ignore-not-found sub ibm-cert-manager-operator
    csv=$("${OC}" get -n "${master_ns}" csv | (grep ibm-cert-manager-operator || echo "fail") | awk '{print $1}')
    "${OC}" delete -n "${master_ns}" --ignore-not-found csv "${csv}"

    # reason for checking again instead of simply deleting the CR when checking
    # for LSR is to avoid deleting anything until the last possible moment.
    # This makes recovery from simple pre-requisite errors easier.
    return_value=$(("${OC}" get crd ibmlicenseservicereporters.operator.ibm.com > /dev/null && echo exists) || echo fail)
    if [[ $return_value == "exists" ]]; then
        migrate_lic_cms $master_ns
        "${OC}" delete -n "${master_ns}" --ignore-not-found ibmlicensing instance
    fi
    return_value="reset"
    #might need a more robust check for if licensing is installed
    #"${OC}" delete -n "${master_ns}" --ignore-not-found sub ibm-licensing-operator
    csv=$("${OC}" get -n "${master_ns}" csv | (grep ibm-licensing-operator || echo "fail") | awk '{print $1}')
    if [[ $csv != "fail" ]]; then
        "${OC}" delete -n "${master_ns}" --ignore-not-found sub ibm-licensing-operator
        "${OC}" delete -n "${master_ns}" --ignore-not-found csv "${csv}"
    fi

    "${OC}" delete -n "${master_ns}" --ignore-not-found sub ibm-crossplane-operator-app
    "${OC}" delete -n "${master_ns}" --ignore-not-found sub ibm-crossplane-provider-kubernetes-operator-app
    csv=$("${OC}" get -n "${master_ns}" csv | (grep ibm-crossplane-operator || echo "fail") | awk '{print $1}')
    "${OC}" delete -n "${master_ns}" --ignore-not-found csv "${csv}"
    csv=$("${OC}" get -n "${master_ns}" csv | (grep ibm-crossplane-provider-kubernetes-operator || echo "fail") | awk '{print $1}')
    "${OC}" delete -n "${master_ns}" --ignore-not-found csv "${csv}"
}

function restart() {
    info "scaling up ibm-common-service-operator deployment in ${master_ns} namespace"
    ${OC} scale deployment -n ${master_ns} ibm-common-service-operator --replicas=1
    ${OC} scale deployment -n ${master_ns} operand-deployment-lifecycle-manager --replicas=1
    check_CSCR "$master_ns"
    check_CSCR "$mapToCSNS"
}

function check_cm_ns_exist(){
    
    title " Verify all namespaces exist "
    msg "-----------------------------------------------------------------------"

    for ns in $namespaces
    do
        info "Creating namespace $ns"
        ${OC} create namespace $ns || info "$ns already exists, skipping..."
    done
    success "All namespaces in $cm_name exist"
}

function cleanupCSOperators(){
    title "Checking subs of Common Service Operator in Cloudpak Namespaces"
    msg "-----------------------------------------------------------------------"   
    catalog_source=$(${OC} get sub ibm-common-service-operator -n ${master_ns} -o yaml | yq ".spec.source")
    info "catalog_source:${catalog_source}" 
    for namespace in $requestedNS
    do
        # remove cs namespace from zen service cr
        return_value=$(${OC} get sub -n ${namespace} | (grep ibm-common-service-operator || echo "fail"))
        if [[ $return_value != "fail" ]]; then
            local sub=$(${OC} get sub -n ${namespace} | grep ibm-common-service-operator | awk '{print $1}')
            ${OC} get sub ${sub} -n ${namespace} -o yaml > tmp.yaml 
            ${YQ} -i '.spec.source = "'${catalog_source}'"' tmp.yaml || error "Could not replace catalog source for CS operator in namespace ${namespace}"
            ${OC} apply -f tmp.yaml
            info "Common Service Operator Subscription in namespace ${namespace} updated to use catalog source ${catalog_source}"
        else
            info "No Common Service Operator in namespace ${namespace}. Moving on..."
        fi
        return_value=""
    done
    rm tmp.yaml -f
}

#TODO change looping to be more specific? 
#Should this only remove the nss from specified set of namespaces? Or should it be more general?
function removeNSS(){
    
    title " Removing ODLM managed Namespace Scope CRs "
    msg "-----------------------------------------------------------------------"

    failcheck=$(${OC} get nss --all-namespaces | grep nss-managedby-odlm || echo "failed")
    if [[ $failcheck != "failed" ]]; then
        ${OC} get nss --all-namespaces | grep nss-managedby-odlm | while read -r line; do
            local namespace=$(echo $line | awk '{print $1}')
            info "deleting namespace scope nss-managedby-odlm in namespace $namespace"
            ${OC} delete nss nss-managedby-odlm -n ${namespace} || (error "unable to delete namespace scope nss-managedby-odlm in ${namespace}")
        done
    else
        info "Namespace Scope CR \"nss-managedby-odlm\" not present. Moving on..."
    fi
    failcheck=$(${OC} get nss --all-namespaces | grep odlm-scope-managedby-odlm || echo "failed")
    if [[ $failcheck != "failed" ]]; then
        ${OC} get nss --all-namespaces | grep odlm-scope-managedby-odlm | while read -r line; do
            local namespace=$(echo $line | awk '{print $1}')
            info "deleting namespace scope odlm-scope-managedby-odlm in namespace $namespace"
            ${OC} delete nss odlm-scope-managedby-odlm -n ${namespace} || (error "unable to delete namespace scope odlm-scope-managedby-odlm in ${namespace}")
        done
    else
        info "Namespace Scope CR \"odlm-scope-managedby-odlm\" not present. Moving on..."
    fi

    success "Namespace Scope CRs cleaned up"
}

function cleanupZenService(){
    title " Cleaning up Zen installation "
    msg "-----------------------------------------------------------------------"
    for namespace in $namespaces
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
    done
    success "Zen instances cleaned up"
}

function migrate_lic_cms() {
    title "Copying over Licensing Configmaps"
    msg "-----------------------------------------------------------------------"
    local namespace=$1
    POSSIBLE_CONFIGMAPS=("ibm-licensing-config"
"ibm-licensing-annotations"
"ibm-licensing-products"
"ibm-licensing-products-vpc-hour"
"ibm-licensing-cloudpaks"
"ibm-licensing-products-groups"
"ibm-licensing-cloudpaks-groups"
"ibm-licensing-cloudpaks-metrics"
"ibm-licensing-products-metrics"
"ibm-licensing-products-metrics-groups"
"ibm-licensing-cloudpaks-metrics-groups"
"ibm-licensing-services"
)

    for cm in ${POSSIBLE_CONFIGMAPS[@]}
    do
        return_value=$(${OC} get cm -n $namespace --ignore-not-found | (grep $cm || echo "fail") | awk '{print $1}')
        if [[ $return_value != "fail" ]]; then
            if [[ $return_value == $cm ]]; then
                ${OC} get cm -n $namespace $cm -o yaml --ignore-not-found > tmp.yaml
                #edit the file to change the namespace to controlNs
                yq -i '.metadata.namespace = "'${controlNs}'"' tmp.yaml
                ${OC} apply -f tmp.yaml
                info "Licensing configmap $cm copied from $namespace to $controlNs"
            fi
        fi
    done
    rm tmp.yaml -f
    success "Licensing configmaps copied from $namespace to $controlNs"
}

function check_CSCR() {
    local CS_NAMESPACE=$1

    retries=30
    sleep_time=15
    total_time_mins=$(( sleep_time * retries / 60))
    info "Waiting for IBM Common Services CR is Succeeded"
    sleep 10

    while true; do
        if [[ ${retries} -eq 0 ]]; then
            error "Timeout after ${total_time_mins} minutes waiting for IBM Common Services CR is Succeeded"
        fi

        phase=$(oc get commonservice common-service -o jsonpath='{.status.phase}' -n ${CS_NAMESPACE})

        if [[ "${phase}" != "Succeeded" ]]; then
            retries=$(( retries - 1 ))
            info "RETRYING: Waiting for IBM Common Services CR is Succeeded (${retries} left)"
            sleep ${sleep_time}
        else
            msg "-----------------------------------------------------------------------"    
            success "Ready use"
            break
        fi
    done

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