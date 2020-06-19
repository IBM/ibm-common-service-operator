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

function wait_for_deleted() {
  kinds=$1
  ns=${2:---all-namespaces}
  index=0
  retries=${3:-10}
  while true; do
    rc=0
    for kind in ${kinds}; do
      cr=$(oc get ${kind} ${ns} 2>/dev/null)
      rc=$?
      [[ "${rc}" != "0" ]] && break
      [[ "X${cr}" != "X" ]] && rc=99 && break
    done

    if [[ "${rc}" != "0" ]]; then
      [[ $(($index % 5)) -eq 0 ]] && msg "Resources are deleting, waiting for complete..."
      if [[ ${index} -eq ${retries} ]]; then
        error "Timeout for wait all resource deleted"
        return 1
      fi
      sleep 60
      index=$((index + 1))
    else
      success "All resources have been deleted"
      break
    fi
  done
}

function delete_sub_csv() {
  subs=$1
  ns=$2
  for sub in ${subs}; do
    csv=$(oc get sub ${sub} -n ${ns} -o=jsonpath='{.status.installedCSV}' --ignore-not-found)
    [[ "X${csv}" != "X" ]] && oc delete csv ${csv} -n ${ns} --ignore-not-found
    oc delete sub ${sub} -n ${ns} --ignore-not-found
  done
}

function delete_operand() {
  crds=$1
  ns=$2
  for crd in ${crds}; do
    crs=$(oc get ${crd} --no-headers -n ${ns} 2>/dev/null | awk '{print $1}')
    if [[ "$?" == "0" && "X${crs}" != "X" ]]; then
      msg "Deleting ${crd} kind resource from namespace ${ns}"
      oc delete ${crd} --all -n ${ns} --ignore-not-found &
    fi
  done
}

function delete_operand_finalizer() {
  crds=$1
  ns=$2
  for crd in ${crds}; do
    crs=$(oc get ${crd} --no-headers -n ${ns} 2>/dev/null | awk '{print $1}')
    for cr in ${crs}; do
      msg "Removing the finalizers for resource: ${crd} $cr"
      oc patch ${crd} $cr -n ${ns} --type="json" -p '[{"op": "remove", "path":"/metadata/finalizers"}]' 2>/dev/null
    done
  done
}

function delete_apiservice() {
  rc=0
  apis=$(oc get apiservice | grep False | awk '{print $1}')
  if [ "X${apis}" != "X" ]; then
    warning "Found some unavailable apiservices, delete them..."
    for api in ${apis}; do
      msg "oc delete apiservice ${api}"
      oc delete apiservice ${api}
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

#-------------------------------------- Clean UP --------------------------------------#
namespace=ibm-common-services
title "Deleting common-service OperandRequest from namespace ${namespace}..."
oc delete OperandRequest common-service -n ${namespace} --ignore-not-found 2>/dev/null &
wait_for_deleted OperandRequest "-n ${namespace}" 1

title "Deleting other OperandRequest from all namespaces..."
oc delete OperandRequest --all --ignore-not-found 2>/dev/null &
wait_for_deleted OperandRequest "--all-namespaces" 1

title "Deleting ODLM sub and csv"
delete_sub_csv "operand-deployment-lifecycle-manager-app" "openshift-operators"

title "Deleting RBAC resource"
oc delete ClusterRole ibm-common-service-webhook --ignore-not-found
oc delete ClusterRoleBinding ibm-common-service-webhook --ignore-not-found
oc delete RoleBinding ibmcloud-cluster-info -n kube-public --ignore-not-found
oc delete Role ibmcloud-cluster-info -n kube-public --ignore-not-found
oc delete RoleBinding ibmcloud-cluster-ca-cert -n kube-public --ignore-not-found
oc delete Role ibmcloud-cluster-ca-cert -n kube-public --ignore-not-found

oc delete ClusterRole nginx-ingress-clusterrole --ignore-not-found
oc delete ClusterRoleBinding $(oc get ClusterRoleBinding | grep nginx-ingress-clusterrole | awk '{print $1}') --ignore-not-found
oc delete scc nginx-ingress-scc --ignore-not-found

title "Force deleting operand resources"
crds=$(oc get crd | grep ibm.com | awk '{print $1}')
msg "Delete operand resource..."
delete_operand "${crds}" "${namespace}"
wait_for_deleted "${crds}" "-n ${namespace}" 1
if [[ "$?" != "0" ]]; then
  msg "Delete operand resource finalizer..."
  delete_operand_finalizer "${crds}" "${namespace}"
  wait_for_deleted "${crds}" "-n ${namespace}" 1
fi

subs=$(oc get sub --no-headers -n ${namespace} 2>/dev/null | awk '{print $1}')
delete_sub_csv "${subs}" "${namespace}"

title "Deleting webhook"
oc delete ValidatingWebhookConfiguration -l 'app=ibm-cert-manager-webhook' --ignore-not-found

title "Deleting unavailable apiservice"
delete_apiservice

title "Deleting namespace ${namespace}"
oc delete namespace ${namespace} --ignore-not-found
[[ "$?" != "0" ]] && error "Something wrong, woooow...." && exit 1
