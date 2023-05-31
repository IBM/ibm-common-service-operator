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

# ---------- Command arguments ----------

OC=oc
YQ=yq
MASTER_NS=
EXCLUDED_NS=""
ADDITIONAL_NS=""
CONTROL_NS=""
CS_MAPPING_YAML=""
CM_NAME="common-service-maps"
CERT_MANAGER_MIGRATED="false"
DEBUG=0
BACKUP_LICENSING="false"

# ---------- Command variables ----------

# script base directory
BASE_DIR=$(cd $(dirname "$0")/$(dirname "$(readlink $0)") && pwd -P)

#log file
LOG_FILE="isolate_log_$(date +'%Y%m%d%H%M%S').log"

# ---------- Main functions ----------

. ${BASE_DIR}/cp3pt0-deployment/common/utils.sh

function main() {
    while [ "$#" -gt "0" ]
    do
        case "$1" in
        "-h"|"--help")
            usage
            exit 0
            ;;
        "--original-cs-ns")
            MASTER_NS=$2
            shift
            ;;
        "--excluded-ns")
            EXCLUDED_NS=$2
            shift
            ;;
        "--insert-ns")
            ADDITIONAL_NS=$2
            shift
            ;;
        "--control-ns")
            CONTROL_NS=$2
            shift
            ;;
        "-v"|"--debug")
            shift
            DEBUG=$1
            ;;
        *)
            error "invalid option -- \`$1\`. Use the -h or --help option for usage info."
            ;;
        esac
        shift
    done

    save_log "cp3pt0-deployment/logs" "isolate_log" "$DEBUG"
    trap cleanup_log EXIT

    which "${OC}" || error "Missing oc CLI"
    which "${YQ}" || error "Missing yq"

    if [[ -z $CONTROL_NS ]] &&  [[ -z $MASTER_NS ]]; then
        usage
        error "No parameters entered. Please re-run specifying original and control namespace values. Use -h for help."
    elif [[ -z $CONTROL_NS ]] || [[ -z $MASTER_NS ]]; then
        usage
        error "Required parameters missing. Please re-run specifying original and control namespace values. Use -h for help."
    fi
    #need to get the namespaces for csmaps generation before pausing cs, otherwise namespace-scope cm does not include all namespaces
    prereq
    local ns_list=$(gather_csmaps_ns)
    pause
    cleanup_webhook
    create_empty_csmaps
    insert_control_ns
    update_tenant "${MASTER_NS}" "${ns_list}"
    check_cm_ns_exist "$ns_list $CONTROL_NS" # debating on turning this off by default since this technically falls outside the scope of isolate
    removeNSS
    uninstall_singletons
    isolate_odlm "ibm-odlm" $MASTER_NS
    if [[ $BACKUP_LICENSING == "true" ]]; then
        restore_ibmlicensing
    else
        info "Licensing not marked for backup, skipping."
    fi
    restart
    if [[ $CERT_MANAGER_MIGRATED == "true" ]]; then
        wait_for_certmanager "$CONTROL_NS" "${ns_list}"
    else
        info "Cert Manager not migrated, skipping wait."
    fi
    wait_for_nss_update "${ns_list}"
    success "Isolation complete"
}

function usage() {
	local script="${0##*/}"

	while read -r ; do echo "${REPLY}" ; done <<-EOF
Usage: ${script} [OPTION]...
Isolate and prepare Cloud Pak 2.0 Foundational Services for upgrade

Examples:

# isolate the existing insance scope in ibm-common-serivces namespace and re-deploy cluster sigleton services in cs-control namespace
isolated.sh --original-cs-ns ibm-common-services --control-ns cs-control

# remove cloudpak-1 and cloudpak-2 namespace from the existing instance scope in ibm-common-services
isolated.sh --original-cs-ns ibm-common-services --control-ns cs-control --excluded-ns cloudpak-1,cloudpak-2

# add cloudpak-1 and cloudpak-2 namespace into the existing instance scope in ibm-common-services
isolated.sh --original-cs-ns ibm-common-services --control-ns cs-control --insert-ns cloudpak-1,cloudpak-2

Options:
Mandatory arguments to long options are mandatory for short options too.
    -h, --help                    display this help and exit
    --original-cs-ns              specify the namespace the original common services installation resides in
    --control-ns                  specify the control namespace value in the common-service-maps configmap
    --excluded-ns                 specify namespaces to be excluded from the instance scope in original-cs-ns. Comma separated no spaces.
    --insert-ns                   specify namespaces to be inserted into the instance scope in original-cs-ns. Comma separated no spaces.
    -v, --debug integer           Verbosity of logs. Default is 0. Set to 1 for debug logs.
	EOF
}

function prereq() {
    # Check the value of DEBUG
    if [[ "$DEBUG" != "1" && "$DEBUG" != "0" ]]; then
        error "Invalid value for DEBUG. Expected 0 or 1."
    fi

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

    local isExists=$("${OC}" get deploy --ignore-not-found -n ${MASTER_NS} operand-deployment-lifecycle-manager)
    if [ -z "$isExists" ]; then
        error "Missing operand-deployment-lifecycle-manager deployment (ODLM) in namespace $MASTER_NS"
    fi

    local cs_version=$("${OC}" get csv -n ${MASTER_NS} | grep common-service-operator | grep 3.2 || echo fail)
    if [[ $cs_version == "fail" ]]; then
        cs_LTSR_version=$("${OC}" get csv -n ${MASTER_NS} | grep common-service-operator | grep 3.19 || echo fail)
        if [[ $cs_LTSR_version != "fail" ]]; then
            version=$(${OC} get csv -n ${MASTER_NS} | grep common-service-operator | awk '{print $7}')
            IFS='.' read -a z_version <<< "$version"
            if [[ $((${z_version[2]})) -lt 9 ]]; then 
                error "Foundational Services installation does not meet the minimum version requirement. Upgrade to either 3.20+ or 3.19.9+"
            fi
        else
            error "Foundational Services installation does not meet the minimum version requirement. Upgrade to either 3.20+ or 3.19.9+"
        fi
    fi
}

# update_cs_maps Updates the common-service-maps with the given yaml. Note that
# the given yaml should have the right indentation/padding, minimum 2 spaces per
# line. If there are multiple lines in the yaml, ensure that each line has
# correct indentation.
function update_cs_maps() {
    local yaml=$1

    local object="$(
        cat <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: "$CM_NAME"
  namespace: kube-public
data:
  common-service-maps.yaml: |
${yaml}
EOF
)"
    echo "$object" | oc apply -f -
}

# create_empty_csmaps Creates a new common-service-maps configmap and inserts
# an empty common-service-maps.yaml field.
#
# If the common-service-maps already exists, then will error
function create_empty_csmaps() {
    title " Creating empty common-service-maps configmap "
    local isExists=$("${OC}" get configmap --ignore-not-found -n kube-public "$CM_NAME")
    if [ ! -z "$isExists" ]; then
        info "The $CM_NAME already exists, skipping"
        return
    fi
    update_cs_maps ""
    success "Empty common-service-maps configmap created in kube-public namespace"
}

# insert_control_ns Insert the controlNamespace field into the configmap if it
# does not exist
function insert_control_ns() {
    local current_yaml=$("${OC}" get -n kube-public cm "$CM_NAME" -o yaml | "${YQ}" '.data.["common-service-maps.yaml"]')

    current=$(echo "$current_yaml" | "${YQ}" '.controlNamespace')
    if [[ "$current" != "$CONTROL_NS" && "$current" != "" && "$current" != "null" ]]; then
        error "The controlNamespace field in common-service-maps is already set to: $current, and cannot be changed"
    fi

    local updated_yaml=$(echo "$current_yaml" | "${YQ}" '.controlNamespace = "'$CONTROL_NS'"')
    local padded_yaml=$(echo "$updated_yaml" | awk '$0="    "$0')
    update_cs_maps "$padded_yaml"
}

# read_tenant_from_csmaps Gets the list in requested-from-namespace for a given
# map_to_cs_ns and prints it out. If map_to_cs_ns does not exist, then output is
# empty
function read_tenant_from_csmaps() {
    local map_to_cs_ns=$1
    local current_yaml=$("${OC}" get -n kube-public cm "$CM_NAME" -o yaml | "${YQ}" '.data.["common-service-maps.yaml"]')
    local tenant_ns_list=$(echo "$current_yaml" | "${YQ}" eval '.namespaceMapping[] | select(.map-to-common-service-namespace == "'${map_to_cs_ns}'").requested-from-namespace' | awk '{ print $2 }')
    echo "$tenant_ns_list"
}

# update_tenant Updates an entire tenant in common-service-maps. The tenant is
# identified by map_to_cs_ns, and will be updated with the given list of
# namespaces which must be space delimited.
#
# If tenant does not exist, then it will be added.
# The map_to_cs_ns will always be added to the requested-from-namespace list.
# Before the common-service-maps is updated, the requested-from-namespace list
# will be made unique, so that there are no duplicates
function update_tenant() {
    local map_to_cs_ns=$1
    shift
    local namespaces=$@

    local current_yaml=$("${OC}" get -n kube-public cm "$CM_NAME" -o yaml | "${YQ}" '.data.["common-service-maps.yaml"]')
    local updated_yaml="$current_yaml"

    local isExists=$(echo "$current_yaml" | "${YQ}" '.namespaceMapping[] | select(.map-to-common-service-namespace == "'$map_to_cs_ns'")')
    if [ -z "$isExists" ]; then
        info "The provided map-to-common-service-namespace: $map_to_cs_ns, does not exist in common-service-maps"
        info "Adding new map-to-common-service-namespace"
        updated_yaml=$(echo "$current_yaml" | "${YQ}" eval 'with(.namespaceMapping; . += [{"map-to-common-service-namespace": "'$map_to_cs_ns'"}])')
    fi

    local tmp="\"$map_to_cs_ns\""
    debug1 "map $map_to_cs_ns namespace $namespaces tmp $tmp"
    for ns in $namespaces; do
        debug1 "ns $ns mapto: $map_to_cs_ns"
        if [[ $ns != $map_to_cs_ns ]]; then
            tmp="$tmp,\"$ns\""
        fi
    done
    local ns_delimited="${tmp}"
    debug1 "ns_delimited: $ns_delimited"

    updated_yaml=$(echo "$updated_yaml" | "${YQ}" eval 'with(.namespaceMapping[]; select(.map-to-common-service-namespace == "'$map_to_cs_ns'").requested-from-namespace = ['$ns_delimited'])')
    updated_yaml=$(echo "$updated_yaml" | "${YQ}" eval 'with(.namespaceMapping[]; select(.map-to-common-service-namespace == "'$map_to_cs_ns'").requested-from-namespace |= unique)')
    local padded_yaml=$(echo "$updated_yaml" | awk '$0="    "$0')
    update_cs_maps "$padded_yaml"
}

# gather_csmaps_ns Reads in all the namespaces from namespace-scope configmap
# and namesapces from arguments, to output a unique sorted list of namespaces
# with excluded namespaces removed
function gather_csmaps_ns() {
    local ns_scope=$("${OC}" get cm -n "$MASTER_NS" namespace-scope -o yaml | yq '.data.namespaces')

    # excluding namespaces is implemented via duplicate removal with uniq -u,
    # so need to make unique the combined lists of namespaces first to avoid
    # accidental removals of namespace which should be included
    local tenant_scope="${ns_scope},${MASTER_NS},${ADDITIONAL_NS}"
    tenant_scope=$(echo "${tenant_scope//,/$'\n'}" | sort -u)

    # adding excluded namespaces to the list allows uniq -u to remove duplicates
    tenant_scope="${tenant_scope},${EXCLUDED_NS},${EXCLUDED_NS}"
    tenant_scope=$(echo "${tenant_scope//,/$'\n'}" | sort | uniq -u)
    echo "$tenant_scope"
}

function pause() {
    title "Pausing Common Services in namespace $MASTER_NS"
    msg "-----------------------------------------------------------------------"
    ${OC} scale deployment -n ${MASTER_NS} ibm-common-service-operator --replicas=0
    ${OC} scale deployment -n ${MASTER_NS} operand-deployment-lifecycle-manager --replicas=0
    ${OC} delete operandregistry -n ${MASTER_NS} --ignore-not-found common-service 
    ${OC} delete operandconfig -n ${MASTER_NS} --ignore-not-found common-service
    
    success "Common Services successfully isolated in namespace ${MASTER_NS}"
}

# uninstall_singletons Deletes resources related to singletons so that when
# cs-operator and ODLM are restarted, these resources will be re-created in the
# controlNamespace.
#
# Everything here can be deleted without backing up because they will eventually
# be re-created, except for the licensing configmaps. These configmaps will only
# be deleted after successful migration. The configmaps should be deleted
# to avoid overwriting any licensing data if isolate script is run multiple
# times.
function uninstall_singletons() {
    title "Uninstalling Singleton Operators"
    msg "-----------------------------------------------------------------------"

    local isExists=$("${OC}" get deployments -n "${MASTER_NS}" --ignore-not-found ibm-cert-manager-operator)
    if [ ! -z "$isExists" ]; then
        "${OC}" delete --ignore-not-found certmanagers.operator.ibm.com default
        CERT_MANAGER_MIGRATED="true"
        debug1 "Cert Manager marked for migration."
    fi
    "${OC}" delete -n "${MASTER_NS}" --ignore-not-found sub ibm-cert-manager-operator
    local csv=$("${OC}" get -n "${MASTER_NS}" csv | (grep ibm-cert-manager-operator || echo "fail") | awk '{print $1}')
    "${OC}" delete -n "${MASTER_NS}" --ignore-not-found csv "${csv}"

    migrate_lic_cms $MASTER_NS

    licensing_exists=$(${OC} get IBMLicensing || echo "fail")
    if [[ $licensing_exists != "fail" ]]; then
        backup_ibmlicensing
        BACKUP_LICENSING="true"
    else
        info "No ibmlicensing resources on cluster, skipping backup."
    fi
    isExists=$("${OC}" get deployments -n "${MASTER_NS}" --ignore-not-found ibm-licensing-operator)
    if [ ! -z "$isExists" ]; then
        "${OC}" delete -n "${MASTER_NS}" --ignore-not-found ibmlicensing instance
    fi

    #might need a more robust check for if licensing is installed
    #"${OC}" delete -n "${MASTER_NS}" --ignore-not-found sub ibm-licensing-operator
    csv=$("${OC}" get -n "${MASTER_NS}" csv | (grep ibm-licensing-operator || echo "fail") | awk '{print $1}')
    if [[ $csv != "fail" ]]; then
        "${OC}" delete -n "${MASTER_NS}" --ignore-not-found sub ibm-licensing-operator
        "${OC}" delete -n "${MASTER_NS}" --ignore-not-found csv "${csv}"
        local is_deleted=$(("${OC}" delete -n "${MASTER_NS}" --ignore-not-found OperandBindInfo ibm-licensing-bindinfo --timeout=10s > /dev/null && echo "success" ) || echo "fail")
        if [[ $is_deleted == "fail" ]]; then
            warning "Failed to delete OperandBindInfo, patching its finalizer to null..."
            ${OC} patch -n "${MASTER_NS}" OperandBindInfo ibm-licensing-bindinfo --type="json" -p '[{"op": "remove", "path":"/metadata/finalizers"}]'
        fi
    fi
    "${OC}" delete -n "${MASTER_NS}" --ignore-not-found sub ibm-crossplane-operator-app
    "${OC}" delete -n "${MASTER_NS}" --ignore-not-found sub ibm-crossplane-provider-kubernetes-operator-app
    csv=$("${OC}" get -n "${MASTER_NS}" csv | (grep ibm-crossplane-operator || echo "fail") | awk '{print $1}')
    "${OC}" delete -n "${MASTER_NS}" --ignore-not-found csv "${csv}"
    csv=$("${OC}" get -n "${MASTER_NS}" csv | (grep ibm-crossplane-provider-kubernetes-operator || echo "fail") | awk '{print $1}')
    "${OC}" delete -n "${MASTER_NS}" --ignore-not-found csv "${csv}"
    
    cleanup_deployment "secretshare" "$MASTER_NS"
    
    success "Singletons successfully uninstalled"
}

function restart() {
    title "Scaling up ibm-common-service-operator deployment in ${MASTER_NS} namespace"
    msg "-----------------------------------------------------------------------"
    ${OC} scale deployment -n ${MASTER_NS} ibm-common-service-operator --replicas=1
    ${OC} scale deployment -n ${MASTER_NS} operand-deployment-lifecycle-manager --replicas=1
    check_CSCR "$MASTER_NS"
    success "Common Service Operator restarted."
}

function check_cm_ns_exist() {
    title " Verify all namespaces exist "
    msg "-----------------------------------------------------------------------"
    local namespaces=$1
    for ns in $namespaces
    do
        info "Creating namespace $ns"
        ${OC} create namespace $ns || info "$ns already exists, skipping..."
    done
    success "All namespaces in $CM_NAME exist"
}

#TODO change looping to be more specific? 
#Should this only remove the nss from specified set of namespaces? Or should it be more general?
function removeNSS(){

    title " Removing ODLM managed Namespace Scope CRs "
    msg "-----------------------------------------------------------------------"

    info "deleting namespace scope nss-managedby-odlm in namespace ${MASTER_NS}"
    ${OC} delete nss nss-managedby-odlm -n ${MASTER_NS} --ignore-not-found || (error "unable to delete namespace scope nss-managedby-odlm in ${MASTER_NS}")

    info "deleting namespace scope odlm-scope-managedby-odlm in namespace ${MASTER_NS}"
    ${OC} delete nss odlm-scope-managedby-odlm -n ${MASTER_NS} --ignore-not-found || (error "unable to delete namespace scope odlm-scope-managedby-odlm in ${MASTER_NS}")
    
    info "deleting namespace scope nss-odlm-scope in namespace ${MASTER_NS}"
    ${OC} delete nss nss-odlm-scope -n ${MASTER_NS} --ignore-not-found || (error "unable to delete namespace scope nss-odlm-scope in ${MASTER_NS}")
    
    info "deleting namespace scope common-service in namespace ${MASTER_NS}"
    ${OC} delete nss common-service -n ${MASTER_NS} --ignore-not-found || (error "unable to delete namespace scope common-service in ${MASTER_NS}")

    success "Namespace Scope CRs cleaned up"
}

function migrate_lic_cms() {
    title "Copying over Licensing Configmaps"
    msg "-----------------------------------------------------------------------"
    local namespace=$1
    local possible_cms=("ibm-licensing-config"
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

    local cm_list=$("${OC}" get cm -n $namespace "${possible_cms[@]}" -o yaml --ignore-not-found)
    if [ -z "$cm_list" ]; then
        info "No licensing configmaps to migrate"
        return
    fi

    local cleaned_cm_list=$(export_k8s_list_yaml "$cm_list")
    echo "$cleaned_cm_list" | "${OC}" apply -n "$CONTROL_NS" -f -
    success "Licensing configmaps copied from $namespace to $CONTROL_NS"
    "${OC}" delete cm --ignore-not-found -n "${namespace}" "${possible_cms[@]}"
}

function backup_ibmlicensing() {

    instance=`"${OC}" get IBMLicensing instance -o yaml --ignore-not-found | "${YQ}" '
        with(.; del(.metadata.creationTimestamp) |
        del(.metadata.managedFields) |
        del(.metadata.resourceVersion) |
        del(.metadata.uid) |
        del(.status)
        )
    ' | sed -e 's/^/    /g'`
cat << _EOF | oc apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: ibmlicensing-instance-bak
  namespace: ${CONTROL_NS}
data:
  ibmlicensing.yaml: |
${instance}
_EOF

}

function restore_ibmlicensing() {

    # extracts the previously saved IBMLicensing CR from ConfigMap and creates the IBMLicensing CR
    "${OC}" get cm ibmlicensing-instance-bak -n ${CONTROL_NS} -o yaml --ignore-not-found | "${YQ}" .data | sed -e 's/.*ibmlicensing.yaml.*//' | 
    sed -e 's/^  //g' | oc apply -f -

}

# export_k8s_list_yaml Takes a k8s list in YAML form,
# e.g. oc get configmap -o yaml, and cleans up the cluster/namespace metadata,
# and prints out a YAML that can be applied into any namespace
function export_k8s_list_yaml() {
    local yaml=$1
    echo "$yaml" | "${YQ}" '
        with(.items[].metadata;
            del(.creationTimestamp) |
            del(.managedFields) |
            del(.resourceVersion) |
            del(.uid) |
            del(.namespace)
        )
    '
}

function check_CSCR() {
    local ns=$1

    local retries=30
    local sleep_time=15
    local total_time_mins=$(( sleep_time * retries / 60))
    info "Waiting for IBM Common Services CR is Succeeded"
    sleep 10

    while true; do
        if [[ ${retries} -eq 0 ]]; then
            error "Timeout after ${total_time_mins} minutes waiting for IBM Common Services CR is Succeeded"
        fi

        local phase=$(oc get commonservice common-service -o jsonpath='{.status.phase}' -n ${ns})

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
    local package_name=$1
    local ns=$2
    # get subscription of ODLM based on namespace 
    local sub_name=$(${OC} get subscription.operators.coreos.com -n ${ns} -l operators.coreos.com/${package_name}.${ns}='' --no-headers | awk '{print $1}')
    if [ -z "$sub_name" ]; then
        warning "Not found subscription ${package_name} in ${ns}"
        return 0
    fi
    #merge patch overwrites the entire array if you update any values so we need to get any other value specified and make sure it is unchanged
    #loop through all of the values specified in spec.config.env
    env_range=$(${OC} get subscription.operators.coreos.com ${sub_name} -n ${ns} -o yaml | yq '.spec.config.env[].name')
    patch_string=""
    count=0
    for name in $env_range
    do
        #If isolated mode, set value to true. Otherwise keep name value pairs unchanged.
        if [[ $name == "ISOLATED_MODE" ]]; then
            env_value="true"
        else
            env_value=$(${OC} get subscription.operators.coreos.com ${sub_name} -n ${ns} -o yaml | yq '.spec.config.env['"${count}"'].value')
        fi
        #Add name value pair in json format to the patch string
        if [[ $patch_string == "" ]]; then
            patch_string="{\"name\": \"$name\", \"value\": \"$env_value\"}"
        else
            patch_string="$patch_string, {\"name\": \"$name\", \"value\": \"$env_value\"}"
        fi
        count=$((count + 1))
    done
    debug1 "patch string $patch_string"
    #use the patch string to apply the isolate mode patch
    ${OC} patch subscription.operators.coreos.com ${sub_name} -n ${ns} --type=merge -p '{"spec": {"config": {"env": ['"${patch_string}"']}}}'
    if [[ $? -ne 0 ]]; then
        error "Failed to update subscription ${package_name} in ${ns}"
    fi

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

function cleanup_deployment() {
    local name=$1
    local namespace=$2
    info "Deleting existing Deployment ${name} in namespace ${namespace}..."
    ${OC} delete deployment ${name} -n ${namespace} --ignore-not-found
}

function cleanup_webhook() {
    podpreset_exist="true"
    podpreset_exist=$(${OC} get podpresets.operator.ibm.com -n $MASTER_NS --no-headers || echo "false")
    if [[ $podpreset_exist != "false" ]] && [[ $podpreset_exist != "" ]]; then
        info "Deleting podpresets in namespace $MASTER_NS..."
	${OC} get podpresets.operator.ibm.com -n $MASTER_NS --no-headers --ignore-not-found | awk '{print $1}' | xargs ${OC} delete -n $MASTER_NS --ignore-not-found podpresets.operator.ibm.com
        msg ""
    fi

    cleanup_deployment "ibm-common-service-webhook" $MASTER_NS

    info "Deleting MutatingWebhookConfiguration..."
    ${OC} delete MutatingWebhookConfiguration ibm-common-service-webhook-configuration --ignore-not-found
    ${OC} delete MutatingWebhookConfiguration ibm-operandrequest-webhook-configuration --ignore-not-found
    msg ""

    info "Deleting ValidatingWebhookConfiguration..."
    ${OC} delete ValidatingWebhookConfiguration ibm-cs-ns-mapping-webhook-configuration --ignore-not-found

}

function check_if_certmanager_deployed() {
    local namespaces=$@
    info "Checking for cert manager deployed in scope."
    local deployed="false"
    for ns in $namespaces
    do
        opreqs=$(${OC} get opreq -n $ns --no-headers | awk '{print $1}' | tr '\n' ' ')
        for opreq in $opreqs
        do
            local return_value=$(${OC} get opreq $opreq -n $ns -o yaml | ${YQ} '.spec.requests[]' | grep "name: ibm-cert-manager-operator" || echo "fail")
            if [[ $return_value != "fail" ]]; then
                deployed="true"
                info "Cert manager requested in scope, moving on..."
                break
            fi
        done
    done

    if [[ $deployed == "false" ]]; then
        info "Cert manager not requested in scope, deploying..."
        cat <<EOF > tmp-opreq.yaml
apiVersion: operator.ibm.com/v1alpha1
kind: OperandRequest
metadata:
  labels:
    app.kubernetes.io/instance: operand-deployment-lifecycle-manager
    app.kubernetes.io/managed-by: operand-deployment-lifecycle-manager
    app.kubernetes.io/name: odlm
  name: ibm-cert-manager-operator
  namespace: $MASTER_NS
spec:
  requests:
    - operands:
        - name: ibm-cert-manager-operator
      registry: common-service
      registryNamespace: $MASTER_NS
EOF

    oc apply -f tmp-opreq.yaml
    rm -f tmp-opreq.yaml
    fi

}

function wait_for_certmanager() {
    local namespace=$1
    shift
    local namespaces=$@
    title " Wait for Cert Manager pods to come ready in namespace $namespace "
    msg "-----------------------------------------------------------------------"
    
    check_if_certmanager_deployed "${namespaces}"

    #check cert manager operator pod
    local name="ibm-cert-manager-operator"
    local condition="${OC} -n ${namespace} get deploy --no-headers --ignore-not-found | egrep '1/1' | grep ^${name} || true"
    local retries=20
    local sleep_time=15
    local total_time_mins=$(( sleep_time * retries / 60))
    local wait_message="Waiting for deployment ${name} in namespace ${namespace} to be running ..."
    local success_message="Deployment ${name} in namespace ${namespace} is running."
    local error_message="Timeout after ${total_time_mins} minutes waiting for deployment ${name} in namespace ${namespace} to be running."
    wait_for_condition "${condition}" ${retries} ${sleep_time} "${wait_message}" "${success_message}" "${error_message}"

    #check webhook pod runnning
    name="cert-manager-webhook"
    condition="${OC} get pod -A --no-headers --ignore-not-found | egrep '1/1' | grep ${name} || true"
    wait_message="Waiting for pod ${name} to be running ..."
    success_message="Pod ${name} is running."
    error_message="Timeout after ${total_time_mins} minutes waiting for pod ${name} to be running."
    wait_for_condition "${condition}" ${retries} ${sleep_time} "${wait_message}" "${success_message}" "${error_message}"

    #check no duplicate webhook pod
    webhook_deployments=$(${OC} get deploy -A --no-headers --ignore-not-found | grep ${name} -c)
    if [[ $webhook_deployments != "1" ]]; then
        error "More than one cert-manager-webhook deployment exists on the cluster."
    fi
    webhook_ns=$(${OC} get deploy -A | grep cert-manager-webhook | awk '{print $1}')
    success "Cert Manager ready in namespace $namespace. Cert Manager operands deployed in $webhook_ns"
}

function wait_for_nss_update() {
    local expected_ns_list=${1//$'\n'/ }
    wait_for_nss_exist
    
    local actual_ns_list=$(${OC} get cm namespace-scope -n ${MASTER_NS} -o yaml | ${YQ} '.data.namespaces')
    actual_ns_list=$(echo "${actual_ns_list//,/ }" | xargs -n1 | sort | xargs)
    expected_ns_list=$(echo "${expected_ns_list}" | xargs -n1 | sort | xargs)
    debug1 "expected ns list: $expected_ns_list"
    debug1 "actual ns list: $actual_ns_list"
    if [[ "${expected_ns_list}" == "${actual_ns_list}" ]]; then
        success "Namespaces in namespace-scope configmap match expected output."
    else
        error "Namespaces in namespace-scope configmap do not match expected output."
    fi
}

function wait_for_nss_exist() {
    local condition="${OC} get cm namespace-scope -n ${MASTER_NS} || true"
    local retries=10
    local sleep_time=15
    local total_time_mins=$(( sleep_time * retries / 60))
    local wait_message="Waiting for configmap namespace-scope in namespace ${MASTER_NS} to be created ..."
    local success_message="Namespace-scope configmap created in ${MASTER_NS}."
    local error_message="Timeout after ${total_time_mins} minutes waiting for namespace-scope configmap to be created."
    wait_for_condition "${condition}" ${retries} ${sleep_time} "${wait_message}" "${success_message}" "${error_message}"
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