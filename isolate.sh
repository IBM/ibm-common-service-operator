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
requested_ns=
excluded_ns=
excluded_raw=""
insert_raw=""
map_to_cs_ns=
OPERATOR_NS=""
SERVICES_NS=""
TETHERED_NS=""
CONTROL_NS=""
NEW_MAPPING=""
cm_name="common-service-maps"
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
        "--excluded-ns")
            excluded_raw=$2
            shift
            ;;
        "--insert-ns")
            insert_raw=$2
            shift
            ;;
        "--control-ns")
            CONTROL_NS=$2
            shift
            ;;
        *)
            error "invalid option -- \`$1\`. Use the -h or --help option for usage info."
            ;;
        esac
        shift
    done
    which "${OC}" || error "Missing oc CLI"
    which "${YQ}" || error "Missing yq"

    if [[ -z $CONTROL_NS ]] &&  [[ -z $master_ns ]]; then
        usage
        error "No parameters entered. Please re-run specifying original and control namespace values. Use -h for help."
    elif [[ -z $CONTROL_NS ]] || [[ -z $master_ns ]]; then
        usage
        error "Required parameters missing. Please re-run specifying original and control namespace values. Use -h for help."
    fi
    #need to get the namespaces for csmaps generation before pausing cs, otherwise namespace-scope cm does not include all namespaces
    abort_check
    gather_csmaps_ns
    pause
    mapping_topology
    prereq
    uninstall_singletons
    isolate_odlm "ibm-odlm" $master_ns
    restart
}

function usage() {
	local script="${0##*/}"

	while read -r ; do echo "${REPLY}" ; done <<-EOF
	Usage: ${script} [OPTION]...
	Isolate and prepare common services for upgrade
	Options:
	Mandatory arguments to long options are mandatory for short options too.
    -h, --help                    display this help and exit
    --original-cs-ns              specify the namespace the original common services installation resides in
    --control-ns                  specify the control namespace value in the common-service-maps configmap
    --excluded-ns                 specify namespaces to be excluded from the common-service-maps configmap. Comma separated no spaces.
    --insert-ns                 specify namespaces to be inserted into the common-service-maps configmap. Comma separated no spaces.
	EOF
}

function abort_check() {
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
    
    local csmaps_exists=$("${OC}" get configmap -n kube-public -o yaml ${cm_name} || echo fail)
    if [[ $csmaps_exists != "fail" ]]; then
        local control_ns_exists=$("${OC}" get configmap -n kube-public -o yaml ${cm_name} | yq '.data' | grep controlNamespace: || echo fail)
        if [[ $control_ns_exists != "fail" ]]; then
            error "Configmap common-service-maps already exists in kube-pubic namespace. Isolate.sh exiting, it is recommended to make futher changes to the configmap manually."
        fi
        
        local map_to_cs_ns_check=$("${OC}" get configmap -n kube-public -o yaml ${cm_name} | yq '.data[]' | yq '.namespaceMapping[].map-to-common-service-namespace' | awk '{print}')
        if [[ $map_to_cs_ns_check != $master_ns ]]; then
            error "Existing value for map-to-common-service-namespace in common-service-maps configmap not equal to argument passed as --original-cs-ns. Exiting..."
        fi
    fi
    

    local cs_version=$("${OC}" get csv -n ${master_ns} | grep common-service-operator | grep 3.2 || echo fail)
    if [[ $cs_version == "fail" ]]; then
        cs_LTSR_version=$("${OC}" get csv -n ${master_ns} | grep common-service-operator | grep 3.19 || echo fail)
        if [[ $cs_LTSR_version != "fail" ]]; then
            version=$(${OC} get csv -n ${master_ns} | grep common-service-operator | awk '{print $7}')
            IFS='.' read -a z_version <<< "$version"
            if [[ $((${z_version[2]})) -lt 9 ]]; then 
                error "Foundational Services installation does not meet the minimum version requirement. Upgrade to either 3.20+ or 3.19.9+"
            fi
        else
            error "Foundational Services installation does not meet the minimum version requirement. Upgrade to either 3.20+ or 3.19.9+"
        fi
    fi
}

function gather_csmaps_ns() {
    #read list of namespaces from nss common-service in original namespace
    return_value=$(${OC} get -n ${master_ns} cm namespace-scope > /dev/null || echo failed)
    if [[ $return_value == "failed" ]]; then
        error "No namespace-scope configmap in Original CS Namespace ${master_ns}. Verify namespace is correct and IBM common services is installed there."
    else
        namespaces=$(oc get cm namespace-scope -n "${master_ns}" -o json | jq '.data.namespaces')
        #output namespace-scope cm
        echo $(oc get cm namespace-scope -n "${master_ns}" -o yaml)
        namespaces=$(echo $namespaces | tr -d '"')
        IFS=',' read -a nsFromCM <<< "$namespaces"
    fi
    #remove excluded from namespaces
    if [[ $excluded_raw != "" ]]; then
        IFS=',' read -a excluded_ns <<< "$excluded_raw"
        #this is very ugly but very consistent and these lists should not be too long anyway
        for ns in ${nsFromCM[@]}
        do
            skip=0
            for exns in ${excluded_ns[@]}
            do
                if [[ $ns == $exns ]]; then
                    skip=1
                    break
                fi
            done
            if [[ $ns == $master_ns ]]; then
                skip=1
            fi
            if [[ $skip != 1 ]]; then
                if [[ $TETHERED_NS != $master_ns ]]; then
                    if [[ $TETHERED_NS == "" ]]; then
                        TETHERED_NS="$ns"
                    else
                        TETHERED_NS="$TETHERED_NS $ns"
                    fi
                fi
            fi
        done
    else
        echo "excluded empty"
        echo "ns from cm $nsFromCM"
        for ns in ${nsFromCM[@]}
        do
            if [[ $ns != $master_ns ]]; then
                if [[ $TETHERED_NS == "" ]]; then
                    TETHERED_NS="$ns"
                else
                    TETHERED_NS="$TETHERED_NS $ns"
                fi
            fi
        done
    fi
    if [[ $insert_raw != "" ]]; then
        IFS=',' read -a insertNS <<< "$insert_raw"
        for ns in ${insertNS[@]}
        do
            if [[ $TETHERED_NS == "" ]]; then
                TETHERED_NS="$ns"
            else
                TETHERED_NS="$TETHERED_NS $ns"
            fi
        done
    fi
    if [[ $TETHERED_NS == "" ]]; then
        TETHERED_NS=$master_ns
    fi

    OPERATOR_NS=$master_ns
    SERVICES_NS=$master_ns
    requested_ns=$TETHERED_NS
    info "common-service-maps namespaces: $requested_ns"
}

function construct_mapping() {
    NEW_MAPPING='- requested-from-namespace:'

    local unique_ns_list=$(echo $OPERATOR_NS $SERVICES_NS $TETHERED_NS | tr ' ' '\n' | sort | uniq | tr '\n' ' ')

    for ns in $unique_ns_list; do
        NEW_MAPPING="$NEW_MAPPING\n  - $ns"
    done

    # Append servicesNamespace to map-to-common-service-namespace
    NEW_MAPPING="$NEW_MAPPING\n  map-to-common-service-namespace: $SERVICES_NS"
}

function mapping_topology() {
    construct_mapping

    info "Creating common-service-maps ConfigMap in kube-public namespace"

    NEW_MAPPING=$(echo -e "controlNamespace: $CONTROL_NS\nnamespaceMapping:\n$NEW_MAPPING")
    local object=$(
        cat <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
    name: common-service-maps
    namespace: kube-public
data:
    common-service-maps.yaml: |
$(echo "$NEW_MAPPING" | awk '{print "        "$0}')
EOF
)
    echo "$object" | ${OC} apply -f -
}

# verify that all pre-requisite CLI tools exist
function prereq() {

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

    #this command gets all of the ns listed in requested from namesapce fields
    requested_ns=$("${OC}" get configmap -n kube-public -o yaml ${cm_name} | yq '.data[]' | yq '.namespaceMapping[].requested-from-namespace' | awk '{print $2}')
    #this command gets all of the ns listed in map-to-common-service-namespace
    map_to_cs_ns=$("${OC}" get configmap -n kube-public -o yaml ${cm_name} | yq '.data[]' | yq '.namespaceMapping[].map-to-common-service-namespace' | awk '{print}')

    check_cm_ns_exist
}

function pause() {
    title "Pausing Common Services in namespace $master_ns"
    msg "-----------------------------------------------------------------------"
    ${OC} scale deployment -n ${master_ns} ibm-common-service-operator --replicas=0
    ${OC} scale deployment -n ${master_ns} operand-deployment-lifecycle-manager --replicas=0
    ${OC} delete operandregistry -n ${master_ns} --ignore-not-found common-service 
    ${OC} delete operandconfig -n ${master_ns} --ignore-not-found common-service
    
    cleanupCSOperators # only updates cs operators in requested_ns list passed in as parameter to script
    removeNSS
    success "Common Services successfully isolated in namespace ${master_ns}"
}

function uninstall_singletons() {
    title "Uninstalling Singleton Operators"
    msg "-----------------------------------------------------------------------"
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
    success "Singletons successfully uninstalled"
}

function restart() {
    title "Scaling up ibm-common-service-operator deployment in ${master_ns} namespace"
    msg "-----------------------------------------------------------------------"
    ${OC} scale deployment -n ${master_ns} ibm-common-service-operator --replicas=1
    ${OC} scale deployment -n ${master_ns} operand-deployment-lifecycle-manager --replicas=1
    check_CSCR "$master_ns"
    if [[ $master_ns != $map_to_cs_ns ]]; then
        check_CSCR "$map_to_cs_ns"
    fi
    success "Common Service Operator restarted."
}

function check_cm_ns_exist(){
    title " Verify all namespaces exist "
    msg "-----------------------------------------------------------------------"
    local namespaces="$requested_ns $map_to_cs_ns $CONTROL_NS"
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
    for namespace in $requested_ns #may need to rethink this variable, maybe Tetheredns?
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
    rm -f tmp.yaml
}

#TODO change looping to be more specific? 
#Should this only remove the nss from specified set of namespaces? Or should it be more general?
function removeNSS(){

    title " Removing ODLM managed Namespace Scope CRs "
    msg "-----------------------------------------------------------------------"

    failcheck=$(${OC} get nss -n ${master_ns} | grep nss-managedby-odlm || echo "failed")
    if [[ $failcheck != "failed" ]]; then
        info "deleting namespace scope nss-managedby-odlm in namespace ${master_ns}"
        ${OC} delete nss nss-managedby-odlm -n ${master_ns} || (error "unable to delete namespace scope nss-managedby-odlm in ${master_ns}")
    else
        info "Namespace Scope CR \"nss-managedby-odlm\" not present. Moving on..."
    fi

    failcheck=$(${OC} get nss -n ${master_ns} | grep odlm-scope-managedby-odlm || echo "failed")
    if [[ $failcheck != "failed" ]]; then
        info "deleting namespace scope odlm-scope-managedby-odlm in namespace ${master_ns}"
        ${OC} delete nss odlm-scope-managedby-odlm -n ${master_ns} || (error "unable to delete namespace scope odlm-scope-managedby-odlm in ${master_ns}")
    else
        info "Namespace Scope CR \"odlm-scope-managedby-odlm\" not present. Moving on..."
    fi

    failcheck=$(${OC} get nss -n ${master_ns} | grep nss-odlm-scope || echo "failed")
    if [[ $failcheck != "failed" ]]; then
        info "deleting namespace scope nss-odlm-scope in namespace ${master_ns}"
        ${OC} delete nss nss-odlm-scope -n ${master_ns} || (error "unable to delete namespace scope nss-odlm-scope in ${master_ns}")
    else
        info "Namespace Scope CR \"nss-odlm-scope\" not present. Moving on..."
    fi

    failcheck=$(${OC} get nss -n ${master_ns} | grep common-service || echo "failed")
    if [[ $failcheck != "failed" ]]; then
        info "deleting namespace scope common-service in namespace ${master_ns}"
        ${OC} delete nss common-service -n ${master_ns} || (error "unable to delete namespace scope common-service in ${master_ns}")
    else
        info "Namespace Scope CR \"common-service\" not present. Moving on..."
    fi

    success "Namespace Scope CRs cleaned up"
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
                #edit the file to change the namespace to CONTROL_NS
                yq -i '.metadata.namespace = "'${CONTROL_NS}'"' tmp.yaml
                ${OC} apply -f tmp.yaml
                info "Licensing configmap $cm copied from $namespace to $CONTROL_NS"
            fi
        fi
    done
    rm -f tmp.yaml 
    success "Licensing configmaps copied from $namespace to $CONTROL_NS"
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

function isolate_odlm() {
    package_name=$1
    ns=$2
    # get subscription of ODLM based on namespace 
    sub_name=$(${OC} get subscription.operators.coreos.com -n ${ns} -l operators.coreos.com/${package_name}.${ns}='' --no-headers | awk '{print $1}')
    if [ -z "$sub_name" ]; then
        warning "Not found subscription ${package_name} in ${ns}"
        return 0
    fi
    ${OC} get subscription.operators.coreos.com ${sub_name} -n ${ns} -o yaml > sub.yaml

    # set ISOLATED_MODE to true
    yq e '.spec.config.env |= (map(select(.name == "ISOLATED_MODE").value |= "true") + [{"name": "ISOLATED_MODE", "value": "true"}] | unique_by(.name))' sub.yaml -i

    # apply updated subscription back to cluster
    ${OC} apply -f sub.yaml
    if [[ $? -ne 0 ]]; then
        error "Failed to update subscription ${package_name} in ${ns}"
    fi
    rm sub.yaml

    check_odlm_env "${ns}" 
}

function check_odlm_env() {
    local namespace=$1
    local name="operand-deployment-lifecycle-manager"
    local condition="${OC} -n ${namespace} get deployment ${name} -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name==\"ISOLATED_MODE\")].value}'| grep "true" || true"
    local retries=10
    local sleep_time=12
    local total_time_mins=$(( sleep_time * retries / 60))
    local wait_message="Waiting for OLM to update Deployment ${name} "
    local success_message="Deployment ${name} is updated"
    local error_message="Timeout after ${total_time_mins} minutes waiting for OLM to update Deployment ${name} "

    wait_for_condition "${condition}" ${retries} ${sleep_time} "${wait_message}" "${success_message}" "${error_message}"
}

function wait_for_condition() {
    local condition=$1
    local retries=$2
    local sleep_time=$3
    local wait_message=$4
    local success_message=$5
    local error_message=$6

    info "${wait_message}"
    while true; do
        result=$(eval "${condition}")

        if [[ ( ${retries} -eq 0 ) && ( -z "${result}" ) ]]; then
            error "${error_message}"
        fi

        sleep ${sleep_time}
        result=$(eval "${condition}")

        if [[ -z "${result}" ]]; then
            info "RETRYING: ${wait_message} (${retries} left)"
            retries=$(( retries - 1 ))
        else
            break
        fi
    done

    if [[ ! -z "${success_message}" ]]; then
        success "${success_message}"
    fi
}

function msg() {
    printf '%b\n' "$1"
}

function success() {
    msg "\33[32m[✔] ${1}\33[0m"
}

function warning() {
    msg "\33[33m[✗] ${1}\33[0m"
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

