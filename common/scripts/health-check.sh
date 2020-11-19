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
  RC=0
  checking_sub_status "$namespace" || RC=$((RC + 1))
  checking_ip_status "$namespace" || RC=$((RC + 1))
  checking_csv_status "$namespace" || RC=$((RC + 1))
  [[ $RC -eq 0 ]] && success "All the common service operators are ready"
  return $RC
}

function checking_sub_status() {
  local namespace=$1
  output "List Subscriptions: ${OC} -n ${namespace} get sub"
  ${OC} -n ${namespace} get sub | tee -a $logfile
  pending_sub=$(${OC} -n ${namespace} get sub -o=jsonpath='{.items[?(@.status.state != "AtLatestKnown")].metadata.name}')
  if [[ "X$pending_sub" != "X" ]]; then
    for sub in $(echo $pending_sub);
    do
      output "Check subscription: ${OC} -n ${namespace} get sub ${sub} -oyaml"
      ${OC} -n ${namespace} get sub ${sub} -oyaml | tee -a $logfile
    done
    return 1
  fi
  return 0
}

function checking_ip_status() {
  local namespace=$1
  output "List InstallPlans: ${OC} -n ${namespace} get ip"
  ${OC} -n ${namespace} get ip | tee -a $logfile
  failed_ip=$(${OC} -n ${namespace} get ip -o=jsonpath='{.items[?(@.status.phase != "Complete")].metadata.name}')
  if [[ "X$failed_ip" != "X" ]]; then
    for ip in $(echo $failed_ip);
    do
      output "Check installplan: ${OC} -n ${namespace} get ip ${ip} -oyaml"
      ${OC} -n ${namespace} get ip ${ip} -oyaml | tee -a $logfile
    done
    return 1
  fi
  return 0
}

function checking_csv_status() {
  local namespace=$1
  output "List CSVs: ${OC} -n ${namespace} get csv"
  ${OC} -n ${namespace} get csv | tee -a $logfile
  failed_csv=$(${OC} -n ${namespace} get csv -o=jsonpath='{.items[?(@.status.phase != "Succeeded")].metadata.name}')
  if [[ "X$failed_csv" != "X" ]]; then
    for csv in $(echo $failed_csv);
    do
      output "Check csv: ${OC} -n ${namespace} get csv ${csv} -oyaml"
      ${OC} -n ${namespace} get csv ${csv} -oyaml | tee -a $logfile
    done
    return 1
  fi
  return 0
}

function checking_operator_logs() {
  local namespace=$1
  local name=$2
  title "Checking ${name} status"
  if ${OC} -n ${namespace} get deploy ${name} &>/dev/null; then
    msg "Fetching ${name} logs from namespace ${namespace} ..."
    templogfile=`mktemp -t operator.XXXXXX`　
    ${OC} -n ${namespace} logs deploy/${name} > $templogfile
    output "${OC} -n ${namespace} logs deploy/${name}"
    cat $templogfile | tail -n $LOG_SIZE | tee -a $logfile
    rm -f $templogfile
  else
    warning "Notfound operator deployment ${name} from namespace ${namespace}"
  fi
}

#-------------------------------------- Main --------------------------------------#
logfile=~/common-services-hc.log
[[ ! -f $logfile ]] && touch $logfile
# Check oc and tee command
OC=$(command -v oc 2>/dev/null)
[[ "X$OC" == "X" ]] && error "oc: command not found"
TEE=$(command -v tee 2>/dev/null)
[[ "X$TEE" == "X" ]] && error "tee: command not found"
LOG_SIZE=100

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

if ! checking_operator_status "ibm-common-services"; then
  checking_operator_logs "openshift-operators" "ibm-common-service-operator"
  checking_operator_logs "ibm-common-services" "ibm-common-service-operator"
  checking_operator_logs "ibm-common-services" "operand-deployment-lifecycle-manager"
  checking_operator_logs "ibm-common-services" "ibm-namespace-scope-operator"
  checking_operator_logs "openshift-operator-lifecycle-manager" "catalog-operator"
  checking_operator_logs "openshift-operator-lifecycle-manager" "olm-operator"
fi
checking_pod_status "ibm-common-services"
output "Found detail check log in file: $logfile"
