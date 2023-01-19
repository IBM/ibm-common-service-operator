#!/bin/bash
#
# Copyright 2023 IBM Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

function usage() {
	local script="${0##*/}"

	while read -r ; do echo "${REPLY}" ; done <<-EOF
	Usage: ${script} [OPTION]...
	Uninstall common services
	Options:
	Mandatory arguments to long options are mandatory for short options too.
	  -h, --help                    display this help and exit
	  -n                            specify the namespace where common service is installed
      -cpn                          specify the cloud pak namespace
	  -f                            force delete specified or default ibm-common-services namespace, skip normal uninstall steps
	EOF
}

function msg() {
  printf '\n%b\n' "$1"
}

function wait_msg() {
  printf '%s\r' "${1}"
}

function success() {
  msg "\33[32m[✔] ${1}\33[0m"
}

function warning() {
  msg "\33[33m[✗] ${1}\33[0m"
}

function error() {
  msg "\33[31m[✘] ${1}\33[0m"
}

function title() {
  msg "\33[1m# [$step] ${1}\33[0m"
  step=$((step + 1))
}

# Sometime delete namespace stuck due to some reousces remaining, use this method to get these
# remaining resources to force delete them.
function get_remaining_resources_from_namespace() {
  local namespace=$1
  local remaining=
  if ${KUBECTL} get namespace ${namespace} &>/dev/null; then
    message=$(${KUBECTL} get namespace ${namespace} -o=jsonpath='{.status.conditions[?(@.type=="NamespaceContentRemaining")].message}' | awk -F': ' '{print $2}')
    [[ "X$message" == "X" ]] && return 0
    remaining=$(echo $message | awk '{len=split($0, a, ", ");for(i=1;i<=len;i++)print a[i]" "}' | while read res; do
      [[ "$res" =~ "pod" ]] && continue
      echo ${res} | awk '{print $1}'
    done)
  fi
  echo $remaining
}
# Get remaining resource with kinds
#this is where the problem is, it keeps adding every word of every message in this function to the new remaining variable. I have no idea why that is
#this update function only ever adds items, it does not remove them
#the script works, cs is uninstalled only from the namespaces specified, but each delete function goess through the entire "wait for deleted" and times out even though the item(s) is(are) deleted in seconds
function update_remaining_resources() { 
  local remaining=$2
  local namespace=$1
  local new_remaining=""
  [[ "X$3" != "X" ]] && ns="-n $3" #no idea what this line does
  for kind in ${remaining}; do
    kindExist=$(${KUBECTL} get ${kind} -n ${namespace} --ignore-not-found || echo "fail")
    if [[ "$kindExist" != "fail" ]]; then
      if [[ $new_remaining == "" ]]; then
        new_remaining="${kind}"
      else
        new_remaining="${kind} ${new_remaining}"
      fi
    else
      new_remaining="${new_remaining//$kind/}"
    fi
    
    # if [[ "X$(${KUBECTL} get ${kind} -n ${namespace} --ignore-not-found)" != "X" ]]; then
    #   new_remaining="${new_remaining} ${kind}"
    # fi
  done
  msg "${new_remaining}"
}

#instead of updating the list with specific resources that need to be deleted, why don't we pass in the resource type that needs to be deleted and count how many are left in a given namespace?
#in cases with muliple resource types, we can use a sum total
function wait_for_deleted() {
  local remaining=${2}
  local namespace=${1}
#   msg "namespace: $namespace"
#   msg "remaining: $remaining"
  retries=${3:-10}
  interval=${4:-30}
  index=0
  while true; do
    remaining=$(update_remaining_resources "$namespace" "$remaining")
    remaining=${remaining//[$'\t\r\n']}
    msg "remaining in wait for delete: $remaining"
    if [[ "X$remaining" != "X" ]]; then
      if [[ ${index} -eq ${retries} ]]; then
        error "Timeout delete resources: $remaining"
        return 1
      fi
      sleep $interval
      ((index++))
      wait_msg "DELETE - Waiting: resource "${remaining}" delete complete [$(($retries - $index)) retries left]"
    else
      break
    fi
  done
}
function wait_for_namespace_deleted() {
  local namespace=$1
  retries=30
  interval=5
  index=0
  while true; do
    nsExist=$(${KUBECTL} get namespace ${namespace} || echo "fail")
    if [[ $nsExist != "fail" ]]; then
      if [[ ${index} -eq ${retries} ]]; then
        error "Timeout delete namespace: $namespace"
        return 1
      fi
      sleep $interval
      ((index++))
      wait_msg "DELETE - Waiting: namespace "${namespace}" delete complete [$(($retries - $index)) retries left]"
    else
      break
    fi
  done
  return 0
}
function delete_operator() {
  local subs=$2 
  local namespace=$1 
  for sub in ${subs}; do
    csv=$(${KUBECTL} get sub ${sub} -n ${namespace} -o=jsonpath='{.status.installedCSV}' --ignore-not-found)
    if [[ "X${csv}" != "X" ]]; then
      msg "Delete operator ${sub} from namespace ${namespace}"
      ${KUBECTL} delete csv ${csv} -n ${namespace} --ignore-not-found
      ${KUBECTL} delete sub ${sub} -n ${namespace} --ignore-not-found
    fi
  done
}
function delete_operand() { 
  local crds=$2  
  local namespace=$1 
  for crd in ${crds}; do
    if ${KUBECTL} api-resources | grep $crd &>/dev/null; then
      crs=$(${KUBECTL} get ${crd} --no-headers --ignore-not-found -n ${namespace} 2>/dev/null | awk '{print $1}')
      if [[ "X${crs}" != "X" ]]; then
        msg "Deleting ${crd} from namespace ${namespace}"
        ${KUBECTL} delete ${crd} --all -n ${namespace} --ignore-not-found &
      fi
    fi
  done
}
function delete_operand_finalizer() {
  local crds=$2 
  local ns=$1 
  for crd in ${crds}; do
    crs=$(${KUBECTL} get ${crd} --no-headers --ignore-not-found -n ${ns} 2>/dev/null | awk '{print $1}')
    for cr in ${crs}; do
      msg "Removing the finalizers for resource: ${crd}/${cr}"
      ${KUBECTL} patch ${crd} ${cr} -n ${ns} --type="json" -p '[{"op": "remove", "path":"/metadata/finalizers"}]' 2>/dev/null
    done
  done
}

function cleanup_cluster() {
  title "Deleting webhooks"
  ${KUBECTL} delete MutatingWebhookConfiguration namespace-admission-config-${COMMON_SERVICES_NS} --ignore-not-found
  if [[ "$REMOVE_IAM_CP_NS" == "true" ]]; then
    ${KUBECTL} delete MutatingWebhookConfiguration namespace-admission-config-${cloudpak_ns} --ignore-not-found
  fi
}
function force_delete() {
  local namespace=$1
  local remaining=$(get_remaining_resources_from_namespace "$namespace")
  if [[ "X$remaining" != "X" ]]; then
    warning "Some resources are remaining: $remaining"
    msg "Deleting finalizer for these resources ..."
    delete_operand_finalizer "${namespace}" "${remaining}" 
    # wait_for_deleted "${namespace}" "${remaining}"  5 10
  fi
}
function delete_iamcr_cloudpak_ns() {
	local crds=$2
	local namespace=$1
	for crd in ${crds}; do
		crs=$(${KUBECTL} get ${crd} --no-headers --ignore-not-found -n ${namespace} 2>/dev/null | awk '{print $1}')
		for cr in ${crs}; do
			msg "Removing the resource: ${crd}/${cr}"
			${KUBECTL} delete ${crd}  $cr -n ${namespace} --ignore-not-found &
		done
	done
}
function force_delete_iamcr_cloudpak_ns() {
	local crds=$2
	local namespace=$1
	# add finializers to resource
	for crd in ${crds}; do
    		crs=$(${KUBECTL} get ${crd} --no-headers --ignore-not-found -n ${namespace} 2>/dev/null | awk '{print $1}')
    		for cr in ${crs}; do
			msg "Removing the finalizers for resource: ${crd}/${cr}"
			${KUBECTL} patch ${crd} ${cr} -n ${namespace} --type="json" -p '[{"op": "remove", "path":"/metadata/finalizers"}]' 2>/dev/null
		done
	done
}
function wait_for_delete_iamcr_cloudpak_ns(){
	local crds=$2
	local namespace=$1
	retries=${3:-10}
	interval=${4:-30}
	index=0
	while true; do
		local new_crds=
		for crd in ${crds}; do
			if [[ "X$(${KUBECTL} get ${crd} -n ${namespace} --ignore-not-found)" != "X" ]]; then
				new_crds="${new_crds} ${crd}"
			fi
		done
		crds=${new_crds}
		if [[ "X$crds" != "X" ]]; then
			if [[ ${index} -eq ${retries} ]]; then
				error "Timeout delete resources: $crds"
				return 1
			fi
			sleep $interval
			((index++))
			wait_msg "DELETE - Waiting: resource ${crds} delete complete [$(($retries - $index)) retries left]"
		else
			break
		fi
	done
}
#-------------------------------------- Clean UP --------------------------------------#
cs_op_list="nss operand-deployment-lifecycle-manager ibm-auditlogging-operator ibm-commonui-operator ibm-events-operator ibm-healthcheck-operator ibm-iam-operator ibm-ingress-nginx-operator ibm-management-ingress-operator ibm-mongodb-operator ibm-monitoring-grafana-operator ibm-platform-api ibm-zen-operator zen-cpp-operator"  
deployments="auth-idp auth-pap auth-pdp common-web-ui default-http-backend iam-policy-controller ibm-monitoring-grafana audit-policy-controller icp-memcached management-ingress nginx-ingress-controller oidcclient-watcher platform-api secret-watcher system-healthcheck-service meta-api-deploy"
serviceaccounts="ibm-auditlogging-operand ibm-mongodb-operand ibm-platform-api-operand ibm-zen-operator-serviceaccount"
statefulsets="icp-mongodb must-gather-service"
daemonsets="audit-logging-fluentd-ds"
services="common-audit-logging common-web-ui default-http-backend iam-pap iam-pdp iam-token-service ibm-monitoring-grafana icp-management-ingress icp-mongodb memcached meta-api-svc mongodb must-gather-service nginx-ingress-controller platform-api platform-auth-service platform-identity-management platform-identity-provider system-healthcheck-service"
routes="cp-console cp-proxy"
COMMON_SERVICES_NS=
KUBECTL=$(command -v kubectl 2>/dev/null)
[[ "X$KUBECTL" == "X" ]] && error "kubectl: command not found" && exit 1
step=0
FORCE_DELETE=false
REMOVE_IAM_CP_NS=false
while [ "$#" -gt "0" ]
do
	case "$1" in
	"-h"|"--help")
		usage
		exit 0
		;;
	"-f")
		FORCE_DELETE=true
		;;
	"-n")
		COMMON_SERVICES_NS=$2
		shift
		;;
    #TODO allow users to specify multiple cloudpak_ns, may need to rethink how this variable is used to do that
    #could try setting a flag like REMOVE_IAM_CP_NS and some global variables to indicate more than one cpns and iteration over them required instead of if/else based on iam flag as it is now
	"-cpn")
		cloudpak_ns=$2
		REMOVE_IAM_CP_NS=true
		shift
		;;
	*)
		warning "invalid option -- \`$1\`"
		usage
		exit 1
		;;
	esac
	shift
done
if [[ "$COMMON_SERVICES_NS" == "" ]]; then
  error "Common service namespace flag \"-n\" was not set. Please re-run script with \"-n\" option. Re-run script with \"-h\" or \"--help\" option for more info"
  exit 1
fi
#not sure if we need to make sure they set the cpn flag yet
# if [[ "$cloudpak_ns" == "" ]]; then
#   error "CloudPak/requested from namespace flag \"-cpn\" was not set. Please re-run script with \"-cpn\" option. Re-run script with \"-h\" or \"--help\" option for more info"
#   exit 1
# fi
#check if cs namespace exists (for example on a second or third run)
csnsExist=$(${KUBECTL} get namespaces | (grep  ${COMMON_SERVICES_NS} || echo "fail") | awk '{print $1}')
if [[ "$csnsExist" == "fail" ]]; then
  msg "Creating dummy CS namespace for script to run in namespace ${COMMON_SERVICES_NS}"
  ${KUBECTL} create namespace ${COMMON_SERVICES_NS} || error "Failed to create namespace ${COMMON_SERVICES_NS}" && exit 1
fi

if [[ "$FORCE_DELETE" == "false" ]]; then
  title "Deleting ibm-common-service-operator in namespace ${COMMON_SERVICES_NS}"
  for sub in $(${KUBECTL} get sub --all-namespaces --ignore-not-found | grep ${COMMON_SERVICES_NS} | awk '{if ($3 =="ibm-common-service-operator") print $1"/"$2}'); do
    namespace=$(echo $sub | awk -F'/' '{print $1}')
    name=$(echo $sub | awk -F'/' '{print $2}')
    delete_operator "${COMMON_SERVICES_NS}" "$name" 
  done
  title "Deleting ODLM in namespace ${COMMON_SERVICES_NS}"
  for sub in $(${KUBECTL} get sub --all-namespaces --ignore-not-found | grep ${COMMON_SERVICES_NS} | awk '{if ($3 =="ibm-odlm") print $1"/"$2}'); do
    namespace=$(echo $sub | awk -F'/' '{print $1}')
    name=$(echo $sub | awk -F'/' '{print $2}')
    delete_operator "${COMMON_SERVICES_NS}" "$name" 
  done
  title "Deleting ibm-namespace-scope-operator in namespace ${COMMON_SERVICES_NS}"
  for sub in $(${KUBECTL} get sub --all-namespaces --ignore-not-found | grep ${COMMON_SERVICES_NS} | awk '{if ($3 =="ibm-namespace-scope-operator") print $1"/"$2}'); do
    namespace=$(echo $sub | awk -F'/' '{print $1}')
    name=$(echo $sub | awk -F'/' '{print $2}')
    delete_operator "${COMMON_SERVICES_NS}" "$name" 
  done
  title "Deleting other common service operators, roles, and rolebindings in namespace ${COMMON_SERVICES_NS}"
  for cs_op in $cs_op_list; do
    delete_operator "${COMMON_SERVICES_NS}" "$cs_op"
    roles=$(${KUBECTL} get roles -n ${COMMON_SERVICES_NS} --ignore-not-found | (grep $cs_op || echo fail) | awk '{print $1}')
    if [[ $roles != "fail" ]]; then
      for role in $roles; do
        ${KUBECTL} delete role $role -n ${COMMON_SERVICES_NS} --ignore-not-found || error "could not delete role $role in namesapce ${COMMON_SERVICES_NS}"
      done
    fi
    rolebindings=$(${KUBECTL} get rolebindings -n ${COMMON_SERVICES_NS} --ignore-not-found | (grep $cs_op || echo fail) | awk '{print $1}')
    if [[ $rolebindings != "fail" ]]; then
      for rolebinding in $rolebindings; do
        ${KUBECTL} delete rolebinding $rolebinding -n ${COMMON_SERVICES_NS} --ignore-not-found || error "could not delete role $rolebinding in namesapce ${COMMON_SERVICES_NS}"
      done
    fi
  done
  title "Deleting deployments, service accounts, statefulsets, daemonsets, services, and routes in ${COMMON_SERVICES_NS}"
  for deploy in $deployments; do
    ${KUBECTL} delete deploy $deploy -n ${COMMON_SERVICES_NS} --ignore-not-found
  done
  for sa in $serviceaccounts; do
    ${KUBECTL} delete sa $sa -n ${COMMON_SERVICES_NS} --ignore-not-found
  done
  for ss in $statefulsets; do
    ${KUBECTL} delete statefulset $ss -n ${COMMON_SERVICES_NS} --ignore-not-found
  done
  for ds in $daemonsets; do
    ${KUBECTL} delete ds $ds -n ${COMMON_SERVICES_NS} --ignore-not-found
  done
  for service in $services; do
    ${KUBECTL} delete service $service -n ${COMMON_SERVICES_NS} --ignore-not-found
  done
  for route in $routes; do
    ${KUBECTL} delete route $route -n ${COMMON_SERVICES_NS} --ignore-not-found
  done
  delete_iamcr_cloudpak_ns ${COMMON_SERVICES_NS} "client"
  delete_operand_finalizer "${COMMON_SERVICES_NS}" "NamespaceScope"
  delete_operand "${COMMON_SERVICES_NS}" "NamespaceScope"
  ${KUBECTL} delete pods --all -n ${COMMON_SERVICES_NS} --force
  

  #remove from cp namespace as well
  if [[ "$REMOVE_IAM_CP_NS" == "true" ]]; then
    title "Deleting ibm-common-service-operator in namespace ${cloudpak_ns}"
    for sub in $(${KUBECTL} get sub --all-namespaces --ignore-not-found | grep ${cloudpak_ns} | awk '{if ($3 =="ibm-common-service-operator") print $1"/"$2}'); do
        namespace=$(echo $sub | awk -F'/' '{print $1}')
        name=$(echo $sub | awk -F'/' '{print $2}')
        delete_operator "${cloudpak_ns}" "$name" 
    done
    title "Deleting ibm-namespace-scope-operator in namespace ${cloudpak_ns}"
    for sub in $(${KUBECTL} get sub --all-namespaces --ignore-not-found | grep ${cloudpak_ns} | awk '{if ($3 =="ibm-namespace-scope-operator") print $1"/"$2}'); do
      namespace=$(echo $sub | awk -F'/' '{print $1}')
      name=$(echo $sub | awk -F'/' '{print $2}')
      delete_operator "${cloudpak_ns}" "$name" 
    done
    title "Deleting other common service operators and resources in namespace ${cloudpak_ns}"
    for cs_op in $cs_op_list; do
      delete_operator "${cloudpak_ns}" "$cs_op"
      roles=$(${KUBECTL} get roles -n ${cloudpak_ns} --ignore-not-found | (grep $cs_op || echo fail) | awk '{print $1}')
      if [[ $roles != "fail" ]]; then
        for role in $roles; do
          ${KUBECTL} delete role $role -n ${cloudpak_ns} --ignore-not-found || error "could not delete role $role in namesapce ${cloudpak_ns}"
        done
      fi
      rolebindings=$(${KUBECTL} get rolebindings -n ${cloudpak_ns} --ignore-not-found | (grep $cs_op || echo fail) | awk '{print $1}')
      if [[ $rolebindings != "fail" ]]; then
        for rolebinding in $rolebindings; do
          ${KUBECTL} delete rolebinding $rolebinding -n ${cloudpak_ns} --ignore-not-found || error "could not delete role $rolebinding in namesapce ${cloudpak_ns}"
        done
      fi
    done
    title "Deleting deployments, service accounts, statefulsets, daemonsets, services, and routes in ${COMMON_SERVICES_NS}"
    for deploy in $deployments; do
        ${KUBECTL} delete deploy $deploy -n ${COMMON_SERVICES_NS} --ignore-not-found
    done
    for sa in $serviceaccounts; do
        ${KUBECTL} delete sa $sa -n ${COMMON_SERVICES_NS} --ignore-not-found
    done
    for ss in $statefulsets; do
        ${KUBECTL} delete statefulset $ss -n ${COMMON_SERVICES_NS} --ignore-not-found
    done
    for ds in $daemonsets; do
        ${KUBECTL} delete ds $ds -n ${COMMON_SERVICES_NS} --ignore-not-found
    done
    for service in $services; do
        ${KUBECTL} delete service $service -n ${COMMON_SERVICES_NS} --ignore-not-found
    done
    for route in $routes; do
        ${KUBECTL} delete route $route -n ${COMMON_SERVICES_NS} --ignore-not-found
    done
    delete_iamcr_cloudpak_ns ${cloudpak_ns} "client"
    delete_operand_finalizer "${cloudpak_ns}" "NamespaceScope"
    delete_operand "${cloudpak_ns}" "NamespaceScope"
  fi
fi

cleanup_cluster

if [[ "$FORCE_DELETE" == "true" ]]; then
  title "Deleting common services operand from $COMMON_SERVICES_NS namespaces"
  delete_operand_finalizer "${COMMON_SERVICES_NS}" "OperandRequest"
  delete_operand "${COMMON_SERVICES_NS}" "OperandRequest"
#   wait_for_deleted "${COMMON_SERVICES_NS}" "OperandRequest" 10 10
  delete_operand_finalizer "${COMMON_SERVICES_NS}" "CommonService OperandRegistry OperandConfig"
  delete_operand "${COMMON_SERVICES_NS}" "CommonService OperandRegistry OperandConfig" 
  delete_operand_finalizer "${COMMON_SERVICES_NS}" "NamespaceScope"
  delete_operand "${COMMON_SERVICES_NS}" "NamespaceScope" 
#   wait_for_deleted "${COMMON_SERVICES_NS}" "NamespaceScope"
  title "Deleting iam crs in ${COMMON_SERVICES_NS} namespace"

  force_delete_iamcr_cloudpak_ns ${COMMON_SERVICES_NS} "client rolebinding"
#   wait_for_delete_iamcr_cloudpak_ns ${COMMON_SERVICES_NS} "client rolebinding"  5 10
  title "Force delete remaining resources"
  force_delete "$COMMON_SERVICES_NS"
  title "Deleting namespace ${COMMON_SERVICES_NS}"
  ${KUBECTL} patch namespace ${COMMON_SERVICES_NS} --type=merge -p '{"spec": {"finalizers":null}}'
  ${KUBECTL} delete namespace ${COMMON_SERVICES_NS} --ignore-not-found
  if wait_for_namespace_deleted ${COMMON_SERVICES_NS}; then
    success "Common Services uninstall finished and successfull from namespace ${COMMON_SERVICES_NS}."
    exit 0
  fi
else
  msg "Cloud Pak and Common Services share namespace so it will not be deleted."
  success "Common Services uninstall finished and successfull from namespace ${COMMON_SERVICES_NS}."
  exit 0
fi
error "Something wrong, check namespace detail:" 
${KUBECTL} get namespace ${COMMON_SERVICES_NS} -oyaml --ignore-not-found
exit 1