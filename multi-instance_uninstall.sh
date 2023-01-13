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
      -cpn                          specify the cloud pak namespace(s) separated by a comma and space ("cpns1, cpns2")
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
  [[ "X$3" != "X" ]] && ns="-n $3"
#   msg "remaining in update func: ${remaining}"
  for kind in ${remaining}; do
    kindExist=$(${KUBECTL} get ${kind} -n ${namespace} --ignore-not-found || echo "fail")
    if [[ "$kindExist" != "fail" ]]; then
      if [[ $new_remaining == "" ]]; then
        new_remaining="${kind}"
        # msg "first new remaining: $new_remaining"
      else
        new_remaining="${kind} ${new_remaining}"
        # msg "new remaining: $new_remaining"
      fi
    # else
    #   msg "kind ${kind} not found in namespace ${namespace}"
    fi
    
    # if [[ "X$(${KUBECTL} get ${kind} -n ${namespace} --ignore-not-found)" != "X" ]]; then
    #   new_remaining="${new_remaining} ${kind}"
    # fi
  done
  msg "${new_remaining}"
}
function wait_for_deleted() {
  local remaining=${2}
  local namespace=${1}
  msg "namespace: $namespace"
  msg "remaining: $remaining"
  retries=${3:-10}
  interval=${4:-30}
  index=0
  while true; do
    remaining=$(update_remaining_resources "$namespace" "$remaining")
    msg "remaining inside wait loop: $remaining" 
    if [[ "X$remaining" != "X" ]]; then
      if [[ ${index} -eq ${retries} ]]; then
        error "Timeout delete resources: $remaining"
        return 1
      fi
      sleep $interval
      ((index++))
      wait_msg "DELETE - Waiting: resource ${remaining} delete complete [$(($retries - $index)) retries left]"
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
    if ${KUBECTL} get namespace ${namespace} &>/dev/null; then
      if [[ ${index} -eq ${retries} ]]; then
        error "Timeout delete namespace: $namespace"
        return 1
      fi
      sleep $interval
      ((index++))
      wait_msg "DELETE - Waiting: namespace ${namespace} delete complete [$(($retries - $index)) retries left]"
    else
      break
    fi
  done
  return 0
}
function delete_operator() {
  local subs=$2 #this is a list...
  local namespace=$1 #so this is not read correctly
  for sub in ${subs}; do
    csv=$(${KUBECTL} get sub ${sub} -n ${namespace} -o=jsonpath='{.status.installedCSV}' --ignore-not-found)
    if [[ "X${csv}" != "X" ]]; then
      msg "Delete operator ${sub} from namespace ${namespace}"
      ${KUBECTL} delete csv ${csv} -n ${namespace} --ignore-not-found
      ${KUBECTL} delete sub ${sub} -n ${namespace} --ignore-not-found
    fi
  done
}
function delete_operand() { #todo change this from all namespaces to specific
  local crds=$2 #this is a list... 
  local namespace=$1 #so this is not read correctly
  echo "namespace: $namespace"
  echo "crds: $crds"
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
  local crds=$2 #this is a list...
  local ns=$1 #so this is not read correctly
  for crd in ${crds}; do
    crs=$(${KUBECTL} get ${crd} --no-headers --ignore-not-found -n ${ns} 2>/dev/null | awk '{print $1}')
    for cr in ${crs}; do
      msg "Removing the finalizers for resource: ${crd}/${cr}"
      ${KUBECTL} patch ${crd} ${cr} -n ${ns} --type="json" -p '[{"op": "remove", "path":"/metadata/finalizers"}]' 2>/dev/null
    done
  done
}
# function delete_unavailable_apiservice() {
#   rc=0
#   apis=$(${KUBECTL} get apiservice | grep False | awk '{print $1}')
#   if [ "X${apis}" != "X" ]; then
#     warning "Found some unavailable apiservices, deleting ..."
#     for api in ${apis}; do
#       msg "${KUBECTL} delete apiservice ${api}"
#       ${KUBECTL} delete apiservice ${api}
#       if [[ "$?" != "0" ]]; then
#         error "Delete apiservcie ${api} failed"
#         rc=$((rc + 1))
#         continue
#       fi
#     done
#   fi
#   return $rc
# }

function cleanup_cluster() {
  title "Deleting webhooks"
  ${KUBECTL} delete MutatingWebhookConfiguration namespace-admission-config-${COMMON_SERVICES_NS} --ignore-not-found
}
function force_delete() {
  local namespace=$1
  local remaining=$(get_remaining_resources_from_namespace "$namespace")
  if [[ "X$remaining" != "X" ]]; then
    warning "Some resources are remaining: $remaining"
    msg "Deleting finalizer for these resources ..."
    delete_operand_finalizer "${namespace}" "${remaining}" 
    wait_for_deleted "${namespace}" "${remaining}"  5 10
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
#check if cs namespace exists (for example on a second or third run)
csnsExist=$(${KUBECTL} get namespaces | (grep  ${COMMON_SERVICES_NS} || echo "fail") | awk '{print $1}')
if [[ "$csnsExist" == "fail" ]]; then
  info "Creating dummy CS namespace for script to run in namespace ${COMMON_SERVICES_NS}"
  ${KUBECTL} create namespace ${COMMON_SERVICES_NS} || error "Failed to create namespace ${COMMON_SERVICES_NS}"
fi
if [[ "$FORCE_DELETE" == "false" ]]; then
  # Before uninstall common services, we should delete some unavailable apiservice
#   delete_unavailable_apiservice
  title "Deleting ibm-common-service-operator in namespace ${cloudpak_ns}"
  for sub in $(${KUBECTL} get sub --all-namespaces --ignore-not-found | grep ${COMMON_SERVICES_NS} | awk '{if ($3 =="ibm-common-service-operator") print $1"/"$2}'); do
    namespace=$(echo $sub | awk -F'/' '{print $1}')
    name=$(echo $sub | awk -F'/' '{print $2}')
    delete_operator "${COMMON_SERVICES_NS}" "$name" 
  done
  if [[ "$REMOVE_IAM_CP_NS" == "true" ]]; then
    title "Deleting ibm-common-service-operator in namespace ${cloudpak_ns}"
    for sub in $(${KUBECTL} get sub --all-namespaces --ignore-not-found | grep ${cloudpak_ns} | awk '{if ($3 =="ibm-common-service-operator") print $1"/"$2}'); do
        namespace=$(echo $sub | awk -F'/' '{print $1}')
        name=$(echo $sub | awk -F'/' '{print $2}')
        delete_operator "${cloudpak_ns}" "$name" 
    done
  fi
  title "Deleting common services operand from $COMMON_SERVICES_NS namespaces"
  delete_operand "${COMMON_SERVICES_NS}" "OperandRequest" && echo "time to wait 1" && wait_for_deleted "${COMMON_SERVICES_NS}" "OperandRequest" 30 20
  delete_operand "${COMMON_SERVICES_NS}" "CommonService OperandRegistry OperandConfig" 
  delete_operand "${COMMON_SERVICES_NS}" "NamespaceScope" && wait_for_deleted "${COMMON_SERVICES_NS}" "NamespaceScope"
  if [[ "$REMOVE_IAM_CP_NS" == "true" ]]; then
    title "Deleting common services operand from $cloudpak_ns namespaces"
    delete_operand "${cloudpak_ns}" "OperandRequest"  && wait_for_deleted "${cloudpak_ns}" "OperandRequest"  30 20
    delete_operand "${cloudpak_ns}" "CommonService OperandRegistry OperandConfig"
    delete_operand "${cloudpak_ns}" "NamespaceScope" && wait_for_deleted "${cloudpak_ns}" "NamespaceScope" 
  fi
  cleanup_cluster
fi
if [[ "$REMOVE_IAM_CP_NS" == "true" ]]; then
  title "Deleting iam crs in cloudpak namespace"
	if [[ "$FORCE_DELETE" == "true" ]]; then
		force_delete_iamcr_cloudpak_ns $cloudpak_ns "client rolebinding"
	fi
	delete_iamcr_cloudpak_ns $cloudpak_ns "client rolebinding"
	wait_for_delete_iamcr_cloudpak_ns $cloudpak_ns "client rolebinding"  5 10
fi
title "Deleting namespace ${COMMON_SERVICES_NS}"
${KUBECTL} delete namespace ${COMMON_SERVICES_NS} --ignore-not-found &
if wait_for_namespace_deleted ${COMMON_SERVICES_NS}; then
  success "Common Services uninstall finished and successfull from namespace ${COMMON_SERVICES_NS}."
  exit 0
fi
cleanup_cluster
title "Force delete remaining resources"
# delete_unavailable_apiservice
force_delete "$COMMON_SERVICES_NS" && success "Common Services uninstall finished and successfull from namespace ${COMMON_SERVICES_NS}." && exit 0
error "Something wrong, check namespace detail:"
${KUBECTL} get namespace ${COMMON_SERVICES_NS} -oyaml
exit 1