#!/bin/bash
#
# Copyright 2020 IBM Corporation
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
	Check common services health status

	Options:
	Mandatory arguments to long options are mandatory for short options too.
	  -h, --help                    display this help and exit
	  --size=[int]                  set a size for fetch log

	Examples:
	  ${script} --size 100
	EOF
}

function msg() {
  printf '%b\n' "$1" | tee -a $logfile
}

function output() {
  echo | tee -a $logfile
  echo "------------------------------------------------------------------------------------------------------------" | tee -a $logfile
  msg "\33[94m ${1}\33[0m"
  echo "------------------------------------------------------------------------------------------------------------" | tee -a $logfile
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
  echo | tee -a $logfile
  msg "\33[1m# ${1}\33[0m"
  echo "============================================================================================================" | tee -a $logfile
}

function checking_pod_status() {
  local namespace=$1
  title "Checking common services pod status"
  msg "Check the health status of all pods in namespace $namespace"
  output "${OC} -n ${namespace} get pods --no-headers --field-selector=status.phase!=Running,status.phase!=Succeeded"
  not_running=$(${OC} -n ${namespace} get pods --no-headers --field-selector=status.phase!=Running,status.phase!=Succeeded)
  msg "$not_running"

  [[ "X$not_running" == "X" ]] && success "All common service pods are running" && return
  echo "$not_running" | while read po
  do
    pod_name=$(echo $po | awk '{print $1}')
    output "${OC} -n ${namespace} get events --field-selector involvedObject.name=${pod_name}"
    ${OC} -n ${namespace} get events --field-selector involvedObject.name=${pod_name} | tee -a $logfile
  done
}


function checking_operator_status() {
  local namespace=$1
  title "Checking common service operators status"
  msg "Check the health status of all the subscriptions, installplans and csvs in namespace $namespace"
  output "${OC} -n ${namespace} get sub,ip,csv"
  ${OC} -n ${namespace} get sub,ip,csv | tee -a $logfile
  pending_sub=$(${OC} -n ${namespace} get sub -o=jsonpath='{.items[?(@.status.state=="UpgradePending")].metadata.name}')
  echo $pending_sub | while read sub
  do
    installPlan=$(${OC} -n ${namespace} get sub ${sub} -o=jsonpath='{.status.installPlanRef.name}')
    currentCSV=$(${OC} -n ${namespace} get sub ${sub} -o=jsonpath='{.status.currentCSV}')
    approval=$(${OC} -n ${namespace} get ip ${installPlan} -o=jsonpath='{.spec.approval}')
    approved=$(${OC} -n ${namespace} get ip ${installPlan} -o=jsonpath='{.spec.approved}')
    if [[ "$approval" == "Manual" &&  "$approved" == "false" ]]; then
      warning "InstallPlan ${installPlan} need approval, it will block operator automatic upgrade, run following command to approve this installPlan:"
      output "oc -n ${namespace} patch installplan ${installPlan} -p '{"spec":{"approved":true}}' --type merge"
      break
    fi
    output "Check subscription: ${OC} -n ${namespace} get sub ${sub} -oyaml"
    ${OC} -n ${namespace} get sub ${sub} -oyaml | tee -a $logfile

    output "Check csv: ${OC} -n ${namespace} get csv ${currentCSV} -oyaml"
    ${OC} -n ${namespace} get csv ${currentCSV} -oyaml | tee -a $logfile

    output "Check installPlan: ${OC} -n ${namespace} get ip ${installPlan} -oyaml"
    ${OC} -n ${namespace} get ip ${installPlan} -oyaml | tee -a $logfile
  done
  [[ "X$pending_sub" == "X" ]] && success "All the common service operators is ready."
}

function checking_odlm_logs() {
  local namespace=$1
  title "Checking ODLM status"
  msg "Fetching ODLM logs from namespace $namespace ..."
  templogfile=`mktemp -t odlm.XXXXXX`　
  ${OC} -n ${namespace} logs deploy/operand-deployment-lifecycle-manager > $templogfile 
  output "${OC} -n ${namespace} logs deploy/operand-deployment-lifecycle-manager"
  cat $templogfile | tail -n $LOG_SIZE | tee -a $logfile
  rm -f $templogfile
}

function checking_catalog_operator_logs() {
  local namespace=$1
  title "Checking catalog operator status"
  msg "Fetching catalog operator logs from namespace $namespace ..."
  templogfile=`mktemp -t catalog.XXXXXX`　
  ${OC} -n ${namespace} logs deploy/catalog-operator> $templogfile 
  output "${OC} -n ${namespace} logs deploy/catalog-operator"
  cat $templogfile | tail -n $LOG_SIZE | tee -a $logfile
  rm -f $templogfile
}

#-------------------------------------- Main --------------------------------------#
logfile=~/common-services-hc.log
[[ ! -f $logfile ]] && touch $logfile
# Check oc and tee command
OC=$(which oc 2>/dev/null)
[[ "X$OC" == "X" ]] && error "oc: command not found"
TEE=$(which tee 2>/dev/null)
[[ "X$TEE" == "X" ]] && error "tee: command not found"

LOG_SIZE=100
CS_NS=ibm-common-services
CATALOG_NS=openshift-operator-lifecycle-manager

while [ "$#" -gt "0" ]
do
	case "$1" in
	"-h"|"--help")
		usage
		exit 0
		;;
	"--size")
		shift
		LOG_SIZE="$1"
		;;
	"--size="*)
		LOG_SIZE="${1##--size=}"
		;;
	*)
		warning "invalid option -- \`$1\`"
		usage
    exit 1
		;;
	esac
	shift
done

checking_operator_status ${CS_NS}
checking_pod_status ${CS_NS}
checking_odlm_logs ${CS_NS}
checking_catalog_operator_logs ${CATALOG_NS}

output "Found detail check log in file: $logfile"
