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
}

function title() {
  msg "\33[34m# ${1}\33[0m"
}

function apiresource_existed() {
  local kind=$1
  if ${KUBECTL} api-resources | grep $kind &>/dev/null; then
    return 0
  elif ${KUBECTL} get crd $kind &>/dev/null; then
    return 0
  else
    msg "API resource ${kind} not found"
    return 1
  fi
}

function get_remaining_resource() {
  local namespace=$1
  remaining=
  if ${KUBECTL} get namespace ${namespace} &>/dev/null; then
    for row in $(${KUBECTL} get namespace ${namespace}  -ojson | jq -r '.status.conditions[] | @base64' 2>/dev/null); do
      _jq() {
        echo ${row} | base64 --decode | jq -r ${1}
      }
      if [[ $(_jq '.type') == "NamespaceContentRemaining" ]]; then
        OLD_IFS="$IFS"
        IFS=","
        res_arr=($(_jq '.message' | awk -F': ' '{print $2}'))
        IFS="$OLD_IFS"
        for i in "${!res_arr[@]}"; do
          remaining="${remaining} $(echo ${res_arr[i]} | awk '{print $1}')"
        done
      fi
    done
  fi
}

function checking_res() {
  local kinds=$1
  local ns="--all-namespaces"
  remaining=
  [[ "X$2" != "X" ]] && ns="-n $2"
  for kind in ${kinds}; do
    res=$(${KUBECTL} get ${kind} ${ns} 2>/dev/null)
    if [[ "$?" == "0" && "X$res" != "X" ]]; then
      remaining="${remaining} ${kind}"
    fi
  done
}

function wait_for_deleted() {
  remaining=${1}
  local namespace=${2}
  retries=${3:-10}
  interval=${4:-60}
  index=0
  while true; do   
    if [[ "X$remaining" != "X" ]]; then
      [[ $(($index % 5)) -eq 0 ]] && msg "Some resources are remaining: ${remaining}, waiting for delete complete..."
      if [[ ${index} -eq ${retries} ]]; then
        error "Timeout for wait all resource deleted"
        return 1
      fi
      sleep $interval
      index=$((index + 1))
      checking_res "$remaining"
    else
      break
    fi
  done
}

function wait_for_namespace_deleted() {
  local namespace=$1
  retries=10
  index=0
  while true; do
    if ${KUBECTL} get namespace ${namespace} &>/dev/null; then
      [[ $(($index % 5)) -eq 0 ]] && msg "Deleting namespace ${namespace}, waiting for complete..."
      if [[ ${index} -eq ${retries} ]]; then
        error "Timeout for wait namespace deleted"
        RC=$((RC + 1))
        return $RC
      fi
      sleep 6
      index=$((index + 1))
    else
      break
    fi
  done
  return 0
}

function delete_operator() {
  local subs=$1
  local namespace=$2
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
  local crds=$1
  local namespace=$2
  for crd in ${crds}; do
    crs=$(${KUBECTL} get ${crd} --no-headers -n ${namespace} 2>/dev/null | awk '{print $1}')
    if [[ "$?" == "0" && "X${crs}" != "X" ]]; then
      msg "Deleting ${crd} kind resource from namespace ${namespace}"
      ${KUBECTL} delete ${crd} --all -n ${namespace} --ignore-not-found &
    fi
  done
}

function delete_operand_finalizer() {
  local crds=$1
  local namespace=$2
  for crd in ${crds}; do
    crs=$(${KUBECTL} get ${crd} --no-headers -n ${namespace} 2>/dev/null | awk '{print $1}')
    for cr in ${crs}; do
      msg "Removing the finalizers for resource: ${crd} $cr"
      ${KUBECTL} patch ${crd} $cr -n ${namespace} --type="json" -p '[{"op": "remove", "path":"/metadata/finalizers"}]' 2>/dev/null
    done
  done
}

function delete_apiservice() {
  rc=0
  apis=$(${KUBECTL} get apiservice | grep False | awk '{print $1}')
  if [ "X${apis}" != "X" ]; then
    warning "Found some unavailable apiservices, delete them..."
    for api in ${apis}; do
      msg "${KUBECTL} delete apiservice ${api}"
      ${KUBECTL} delete apiservice ${api}
      if [[ "$?" != "0" ]]; then
        error "Delete apiservcie ${api} failed"
        rc=$((rc + 1))
        continue
      fi
    done
  else
    success "All the apiservices are available, skip delete"
    return 0
  fi
  return $rc
}

function delete_rbac_resource() {
  ${KUBECTL} delete ClusterRoleBinding ibm-common-service-webhook secretshare-ibm-common-services $(${KUBECTL} get ClusterRoleBinding | grep nginx-ingress-clusterrole | awk '{print $1}') --ignore-not-found
  ${KUBECTL} delete ClusterRole ibm-common-service-webhook secretshare nginx-ingress-clusterrole --ignore-not-found
  ${KUBECTL} delete RoleBinding ibmcloud-cluster-info ibmcloud-cluster-ca-cert -n kube-public --ignore-not-found
  ${KUBECTL} delete Role ibmcloud-cluster-info ibmcloud-cluster-ca-cert -n kube-public --ignore-not-found
  ${KUBECTL} delete scc nginx-ingress-scc --ignore-not-found
}

function force_delete() {
  local namespace=$1
  # crds=$(${KUBECTL} get crd | grep ibm.com | awk '{print $1}')
  get_remaining_resource "$namespace"
  if [[ "X$remaining" != "X" ]]; then
    msg "Delete operand resource..."
    delete_operand "${remaining}" "${namespace}"
    wait_for_deleted "${remaining}" "${namespace}" 5 10
    if [[ "$?" != "0" ]]; then
      msg "Delete remaining's operand resource finalizer..."
      delete_operand_finalizer "${remaining}" "${namespace}"
      wait_for_deleted "${remaining}" "${namespace}" 5 10
    fi
  fi
  subs=$(${KUBECTL} get sub --no-headers -n ${namespace} 2>/dev/null | awk '{print $1}')
  delete_operator "${subs}" "${namespace}"  
}

#-------------------------------------- Clean UP --------------------------------------#
KUBECTL=$(which kubectl 2>/dev/null)
[[ "X$KUBECTL" == "X" ]] && error "kubectl: command not found" && exit 1
COMMON_SERVICES_NS=ibm-common-services
remaining=

title "Deleting unavailable apiservice"
delete_apiservice

title "Deleting Common Service Operators"
for sub in $(${KUBECTL} get sub --all-namespaces | awk '{if ($3 =="ibm-common-service-operator") print $1"/"$2}'); do
  namespace=$(echo $sub | awk -F'/' '{print $1}')
  name=$(echo $sub | awk -F'/' '{print $2}')
  if [[ "$namespace" != "ibm-common-services" ]]; then
    delete_operator "$name" "$namespace"
  fi
done

if apiresource_existed "CommonService"; then
  title "Deleting CommonService from namespace $COMMON_SERVICES_NS ..."
  ${KUBECTL} delete CommonService common-service -n $COMMON_SERVICES_NS --ignore-not-found 2>/dev/null &
fi

if apiresource_existed "OperandRequest"; then
  title "Deleting OperandRequest from all namespaces..."
  ${KUBECTL} delete OperandRequest --all --ignore-not-found 2>/dev/null &
  wait_for_deleted "OperandRequest" "" 20 10
fi

title "Deleting ODLM Operator"
delete_operator "operand-deployment-lifecycle-manager-app" "openshift-operators"

title "Deleting RBAC resource"
delete_rbac_resource

title "Deleting webhook"
${KUBECTL} delete ValidatingWebhookConfiguration -l 'app=ibm-cert-manager-webhook' --ignore-not-found
${KUBECTL} delete MutatingWebhookConfiguration ibm-common-service-webhook-configuration --ignore-not-found

title "Deleting namespace ${COMMON_SERVICES_NS}"
${KUBECTL} delete namespace ${COMMON_SERVICES_NS} --ignore-not-found &
if wait_for_namespace_deleted ${COMMON_SERVICES_NS}; then
  success "Common Services uninstall finished and successfull."
  exit 0
else
  title "Force deleting operand resources"
  force_delete "$COMMON_SERVICES_NS" && success "Common Services uninstall finished and successfull." && exit 0
  error "Something wrong, woooow...., checking namespace detail:"
  ${KUBECTL} get namespace ${COMMON_SERVICES_NS} -oyaml
  exit 1
fi
